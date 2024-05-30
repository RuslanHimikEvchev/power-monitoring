package direct_wire

import (
	"encoding/json"
	"errors"
	"fmt"
	"go-meshtastic-monitor/comunication"
	"go-meshtastic-monitor/core"
	"log"
	"strings"
	"sync"
	"time"
)

const DevicesBackupKey = `backup_online_devices`
const StatusOn = `on`
const StatusOff = `off`

type DirectWireMonitor struct {
	devices   map[string]comunication.Device
	complexes map[string]comunication.Complex

	rw       sync.RWMutex
	notifier *core.Notifier
	storage  *core.RedisStorage
}

func NewDirectWireMonitor(notifier *core.Notifier, complexes []comunication.Complex, storage *core.RedisStorage) *DirectWireMonitor {

	return &DirectWireMonitor{
		devices:   make(map[string]comunication.Device),
		complexes: core.ToMap(complexes),
		notifier:  notifier,
		storage:   storage,
	}
}

func (m *DirectWireMonitor) Restore() {
	m.rw.Lock()
	defer m.rw.Unlock()
	data, err := m.storage.Get(DevicesBackupKey)

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

	fmt.Println("[INFO] Restored devices:", devices)

	m.devices = m.toMap(devices)
}

func (m *DirectWireMonitor) Backup() {
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

	err = m.storage.Store(DevicesBackupKey, string(b))
	if err != nil {
		log.Println("Error storing devices:", err.Error())

		return
	}

	log.Println("[INFO] Backup complete")
}

func (m *DirectWireMonitor) UpdateComplexes(complexes []comunication.Complex) {
	m.rw.Lock()
	defer m.rw.Unlock()
	m.complexes = core.ToMap(complexes)
}

func (m *DirectWireMonitor) HandleDevice(device comunication.Device) {
	m.rw.Lock()
	defer m.rw.Unlock()

	_, exist := m.devices[device.MacAddress]
	c, err := m.findComplex(device.Key)

	if err != nil {
		if exist {
			delete(m.devices, device.MacAddress)
		}

		return
	}
	device.Complex = c

	if exist {
		existingDevice := m.devices[device.MacAddress]

		existingDevice.Complex = c
		existingDevice.LastSeen = time.Now()
		existingDevice.IsPluggedIn = device.IsPluggedIn
		device.NotificationEnabled = existingDevice.NotificationEnabled

		m.devices[device.MacAddress] = existingDevice

		if !device.IsPluggedIn {
			if !existingDevice.DownNotificationSend {
				device.DownNotificationSend = true
				device.UpNotificationSend = false
				fmt.Printf("Detected power off %+v\n", device)

				m.notifier.Notify(core.Notification{
					Device:  existingDevice,
					Message: existingDevice.GeneratePowerOffMessageOnline(),
				})
				device.PowerOffAt = time.Now()
				m.devices[device.MacAddress] = device

				return
			}
		} else {
			if !existingDevice.UpNotificationSend {
				device.UpNotificationSend = true
				device.DownNotificationSend = false
				fmt.Printf("Detected power on %+v\n", device)

				m.notifier.Notify(core.Notification{
					Device:  existingDevice,
					Message: existingDevice.GeneratePowerOnMessageOnline(),
				})
				device.PowerOnAt = time.Now()

				m.devices[device.MacAddress] = device

				return
			}
		}
	} else {
		if device.IsPluggedIn {
			device.PowerOnAt = time.Now()
			device.UpNotificationSend = true
			device.DownNotificationSend = false
		} else {
			device.PowerOffAt = time.Now()
			device.DownNotificationSend = true
			device.UpNotificationSend = true
		}

		device.NotificationEnabled = true

		fmt.Printf("Registered %v\n", device)

		m.devices[device.MacAddress] = device
	}
}

func (m *DirectWireMonitor) UpdatePowerOnAt(mac string) {
	m.rw.Lock()
	defer m.rw.Unlock()
	device, ok := m.devices[mac]

	if !ok {
		return
	}

	device.PowerOnAt = time.Now()

	m.devices[mac] = device
}

func (m *DirectWireMonitor) UpdateNotification(mac string, notificationEnabled bool) {
	m.rw.Lock()
	defer m.rw.Unlock()
	device, ok := m.devices[mac]

	if !ok {
		return
	}

	device.NotificationEnabled = notificationEnabled

	m.devices[mac] = device
}

func (m *DirectWireMonitor) GetStatus() map[string]comunication.Device {
	m.rw.RLock()
	defer m.rw.RUnlock()
	return m.devices
}

func (m *DirectWireMonitor) findComplex(k string) (comunication.Complex, error) {
	if _, present := m.complexes[k]; !present {
		return comunication.Complex{}, errors.New("complex not found")
	}

	return m.complexes[k], nil
}

func (m *DirectWireMonitor) fromMap(devices map[string]comunication.Device) []comunication.Device {
	var d []comunication.Device
	for _, device := range devices {
		d = append(d, device)
	}

	return d
}

func (m *DirectWireMonitor) toMap(devices []comunication.Device) map[string]comunication.Device {
	d := make(map[string]comunication.Device)
	for _, device := range devices {
		d[device.MacAddress] = device
	}

	return d
}

func (m *DirectWireMonitor) GetStatusText(c comunication.Complex) string {
	var msgs []string
	for _, device := range m.devices {
		if device.Key == c.Key {
			msgs = append(msgs, device.GenerateStatusMessageOnline())
		}
	}

	if len(msgs) > 0 {
		return strings.Join(msgs, "\n")
	}

	return ""
}
