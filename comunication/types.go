package comunication

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"time"
)

type Device struct {
	Name                 string    `json:"n"`
	Key                  string    `json:"k"`
	Interval             int64     `json:"i"`
	MacAddress           string    `json:"m"`
	LastSeen             time.Time `json:"lastSeen"`
	PowerOffAt           time.Time `json:"powerOffAt"`
	Timeout              int64     `json:"timeout,omitempty"`
	DownNotificationSend bool      `json:"downNotificationSend"`
	UpNotificationSend   bool      `json:"upNotificationSend"`
	PowerOnAt            time.Time `json:"powerOnAt"`
	Complex              Complex   `json:"complex"`
	HasDirectWire        bool      `json:"h"`
	IsPluggedIn          bool      `json:"p"`
	NotificationEnabled  bool      `json:"notification_enabled"`
}

type Complex struct {
	Key                 string           `json:"key" yaml:"key"`
	Name                string           `json:"name" yaml:"name"`
	BotToken            string           `json:"bot_token" yaml:"bot_token"`
	BotChannels         []int64          `json:"bot_channels" yaml:"bot_channels"`
	BotIdentity         string           `json:"bot_identity" yaml:"bot_identity"`
	NotificationEnabled bool             `json:"notification_enabled" yaml:"notification_enabled"`
	StatisticsEnabled   bool             `json:"statistics_enabled" yaml:"statistics_enabled"`
	StatisticsKey       string           `json:"statistics_key" yaml:"statistics_key"`
	DeviceGroupMap      map[string]int64 `json:"device_group_map" yaml:"device_group_map"`
	IsDirectWire        bool             `json:"is_direct_wire" yaml:"is_direct_wire"`
}

type DeviceInfo struct {
	Name              string    `json:"name"`
	UpdateInterval    int64     `json:"updateInterval"`
	CalculatedTimeout int64     `json:"calculatedTimeout"`
	IsOnline          bool      `json:"isOnline"`
	LastSeen          time.Time `json:"lastSeen"`
}

type ComplexInfo struct {
	Name                string       `json:"name"`
	NotificationEnabled bool         `json:"notificationEnabled"`
	StatisticsEnabled   bool         `json:"statisticsEnabled"`
	OnlineDeviceCount   int          `json:"onlineDeviceCount"`
	OfflineDeviceCount  int          `json:"offlineDeviceCount"`
	Devices             []DeviceInfo `json:"devices"`
}

type ComplexStat struct {
	TotalSecondsOffline int64      `json:"totalSecondsOffline"`
	TotalSecondsOnline  int64      `json:"totalSecondsOnline"`
	Name                string     `json:"name"`
	StatisticKey        string     `json:"statisticKey"`
	Lines               []LineStat `json:"lines"`
}

type LineStat struct {
	TotalSecondsOffline int64      `json:"totalSecondsOffline"`
	TotalSecondsOnline  int64      `json:"totalSecondsOnline"`
	Name                string     `json:"name"`
	Dates               []DateStat `json:"dates"`
}

type DateStat struct {
	TotalSecondsOffline int64  `json:"totalSecondsOffline"`
	TotalSecondsOnline  int64  `json:"totalSecondsOnline"`
	Date                string `json:"date"`
}

type Groups struct {
	Name string                       `json:"name"`
	Id   int64                        `json:"id"`
	Week map[string]map[string]string `json:"week"`
}

func (d Device) IsTimeout() bool {
	return (time.Now().Unix() - d.LastSeen.Unix()) > d.Timeout
}

func (d Device) GeneratePowerOffMessageOnline() string {
	since := time.Since(d.PowerOnAt)

	return fmt.Sprintf("\"%s %s\" живлення зникло о %s. Світло було %s", d.Name, d.Complex.Name, time.Now().Format("15:04"), since.Round(time.Second).String())
}

func (d Device) GeneratePowerOnMessageOnline() string {
	since := time.Since(d.PowerOffAt)

	return fmt.Sprintf("\"%s %s\" живлення з'явилось о %s. Світла не було %s", d.Name, d.Complex.Name, time.Now().Format("15:04"), since.Round(time.Second).String())
}

func (d Device) GeneratePowerOffMessage() string {
	since := time.Since(d.PowerOnAt)

	return fmt.Sprintf("\"%s %s\" живлення зникло о %s. Світло було %s", d.Name, d.Complex.Name, d.LastSeen.Format("15:04:05"), since.Round(time.Second).String())
}

func (d Device) GeneratePowerOnMessage() string {
	since := time.Since(d.LastSeen)

	return fmt.Sprintf("\"%s %s\" живлення з'явилось о %s. Світла не було %s", d.Name, d.Complex.Name, time.Now().Format("15:04:05"), since.Round(time.Second).String())
}

func (d Device) GenerateStatusMessage() string {
	if d.IsTimeout() {
		return fmt.Sprintf("%s вимкнена з %s", d.Name, d.LastSeen.Format("2006-01-02 15:04:05"))
	} else {
		return fmt.Sprintf("%s увімкнена з %s", d.Name, d.PowerOnAt.Format("2006-01-02 15:04:05"))
	}
}

func (d Device) GenerateStatusMessageOnline() string {
	if d.IsPluggedIn {
		return fmt.Sprintf("%s увімкнена з %s", d.Name, d.PowerOnAt.Format("2006-01-02 15:04:05"))
	} else {
		return fmt.Sprintf("%s вимкнена з %s", d.Name, d.PowerOffAt.Format("2006-01-02 15:04:05"))
	}
}

func (d Device) Hash() string {
	hash := md5.Sum([]byte(d.MacAddress))
	return hex.EncodeToString(hash[:])
}
