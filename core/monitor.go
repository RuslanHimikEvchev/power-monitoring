package core

import (
	"encoding/json"
	"errors"
	"go-meshtastic-monitor/comunication"
	"log"
	"strings"
	"sync"
	"time"
)

const IntervalCoefficient = 3
const StorageKey = "devices"

const PeriodicCheck = 10

type Event struct {
	DeviceName  string `json:"device"`
	DateTime    string `json:"datetime"`
	Event       string `json:"event"`
	ComplexName string `json:"complexName"`
}

type Monitor struct {
	complexes map[string]comunication.Complex
	devices   map[string]comunication.Device
	n         *Notifier
	stopChan  chan struct{}
	rw        sync.RWMutex
	storage   *RedisStorage

	s *Schedule
}

func ToMap(complexes []comunication.Complex) map[string]comunication.Complex {
	m := make(map[string]comunication.Complex)

	for _, c := range complexes {
		m[c.Key] = c
	}

	return m
}

func NewMonitor(c []comunication.Complex, notifier *Notifier, storage *RedisStorage) *Monitor {
	m := &Monitor{
		complexes: ToMap(c),
		devices:   make(map[string]comunication.Device),
		n:         notifier,
		stopChan:  make(chan struct{}),
		storage:   storage,
	}

	return m
}

func (m *Monitor) Restore() {
	m.rw.Lock()
	defer m.rw.Unlock()
	data, err := m.storage.Get(StorageKey)

	if err != nil {
		log.Println("[ERROR] Failed to restore data from storage:", err.Error())

		return
	}

	if data == "" {
		log.Println("[ERROR] Failed to restore data from storage. Data is empty.")

		return
	}
	var devices []comunication.Device
	err = json.Unmarshal([]byte(data), &devices)

	if err != nil {
		log.Println("[ERROR] Failed to restore data from storage:", err.Error())

		return
	}

	m.devices = m.toMap(devices)
}

func (m *Monitor) Backup() {
	m.rw.RLock()
	defer m.rw.RUnlock()

	if len(m.devices) == 0 {
		log.Println("[INFO] No devices to backup")
		return
	}

	b, err := json.Marshal(m.fromMap(m.devices))

	if err != nil {
		log.Println("Error marshalling devices:", err.Error())

		return
	}

	err = m.storage.Store(StorageKey, string(b))
	if err != nil {
		log.Println("Error storing devices:", err.Error())

		return
	}

	log.Println("[INFO] Backup complete")
}

func (m *Monitor) Start() {
	t := time.NewTicker(time.Second * PeriodicCheck)

	for {
		select {
		case <-t.C:
			m.CheckDevice()
		case <-m.stopChan:
			return
		}
	}
}

func (m *Monitor) CheckDevice() {
	m.rw.Lock()
	defer m.rw.Unlock()
	for _, device := range m.devices {
		if device.IsTimeout() {
			if !device.DownNotificationSend {
				device.DownNotificationSend = true
				device.UpNotificationSend = false

				m.devices[device.MacAddress] = device

				m.n.Notify(Notification{
					Device:  device,
					Message: device.GeneratePowerOffMessage(),
				})
			}
		}
	}
}

func (m *Monitor) Stop() {
	m.stopChan <- struct{}{}
}

func (m *Monitor) UpdateComplexes(complexes []comunication.Complex) {
	m.rw.Lock()
	defer m.rw.Unlock()
	m.complexes = ToMap(complexes)
}

func (m *Monitor) AddDevice(d comunication.Device) {
	m.rw.Lock()
	defer m.rw.Unlock()
	_, exist := m.devices[d.MacAddress]
	c, err := m.findComplex(d.Key)

	if err != nil {
		if exist {
			delete(m.devices, d.MacAddress)
		}

		return
	}
	d.Complex = c

	if exist {
		device := m.devices[d.MacAddress]

		if device.IsTimeout() {
			if !device.UpNotificationSend {
				device.UpNotificationSend = true
				device.DownNotificationSend = false

				device.PowerOnAt = time.Now()

				m.n.Notify(Notification{
					Device:  m.devices[d.MacAddress],
					Message: m.devices[d.MacAddress].GeneratePowerOnMessage(),
				})
			}
		}

		device.LastSeen = time.Now()
		device.Timeout = device.Interval * IntervalCoefficient
		device.Complex = c
		m.devices[d.MacAddress] = device
	} else {
		d.LastSeen = time.Now()
		d.PowerOnAt = time.Now()
		d.Timeout = d.Interval * IntervalCoefficient

		m.devices[d.MacAddress] = d
	}
}

func (m *Monitor) findComplex(k string) (comunication.Complex, error) {
	if _, present := m.complexes[k]; !present {
		return comunication.Complex{}, errors.New("complex not found")
	}

	return m.complexes[k], nil
}

func (m *Monitor) GetStatus() []comunication.ComplexInfo {
	var complexes []comunication.ComplexInfo
	for _, complexRegistered := range m.complexes {
		c := comunication.ComplexInfo{
			Name:                complexRegistered.Name,
			NotificationEnabled: complexRegistered.NotificationEnabled,
			StatisticsEnabled:   complexRegistered.StatisticsEnabled,
		}

		for _, device := range m.devices {

			if device.Key != complexRegistered.Key {
				continue
			}

			if device.IsTimeout() {
				c.OfflineDeviceCount++
			} else {
				c.OnlineDeviceCount++
			}

			dInfo := comunication.DeviceInfo{
				Name:              device.Name,
				UpdateInterval:    device.Interval,
				CalculatedTimeout: device.Timeout,
				IsOnline:          device.IsTimeout() == false,
				LastSeen:          device.LastSeen,
			}

			c.Devices = append(c.Devices, dInfo)
		}

		complexes = append(complexes, c)
	}

	return complexes
}

func (m *Monitor) GetStatusText(c comunication.Complex) string {
	var msgs []string
	for _, device := range m.devices {
		if device.Key == c.Key {
			msgs = append(msgs, device.GenerateStatusMessage())
			msgShed := m.s.GetScheduleDescription(device.Complex.DeviceGroupMap[device.MacAddress], time.Now())

			if msgShed != "" {
				msgs = append(msgs, msgShed)
			}
		}
	}

	if len(msgs) > 0 {
		return strings.Join(msgs, "\n")
	}

	return ""
}

func (m *Monitor) fromMap(devices map[string]comunication.Device) []comunication.Device {
	var d []comunication.Device
	for _, device := range devices {
		d = append(d, device)
	}

	return d
}

func (m *Monitor) toMap(devices []comunication.Device) map[string]comunication.Device {
	d := make(map[string]comunication.Device)
	for _, device := range devices {
		d[device.MacAddress] = device
	}

	return d
}
