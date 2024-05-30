package core

import (
	"encoding/json"
	"fmt"
	"go-meshtastic-monitor/comunication"
	"log"
	"os"
	"strconv"
	"sync"
	"time"
)

const Yes = "yes"
const No = "no"
const Maybe = "maybe"

type Schedule struct {
	groups []comunication.Groups
	rw     sync.RWMutex
}

func ParseSchedules(file string) []comunication.Groups {
	var groups []comunication.Groups
	b, err := os.ReadFile(file)

	if err != nil {
		log.Fatal(err.Error())
	}

	err = json.Unmarshal(b, &groups)

	if err != nil {
		log.Fatal(err.Error())
	}

	return groups
}

func NewSchedule(groups []comunication.Groups) *Schedule {
	return &Schedule{groups: groups}
}

func (s *Schedule) Update(groups []comunication.Groups) {
	s.rw.Lock()
	defer s.rw.Unlock()
	s.groups = groups
}

func (s *Schedule) GetScheduleStatus(groupId int64, date time.Time) string {
	s.rw.Lock()
	defer s.rw.Unlock()
	hour, _ := strconv.Atoi(date.Format("15"))
	hour += 1
	hourString := fmt.Sprintf("%d", hour)

	dayOfWeek := date.Format("Monday")

	for _, group := range s.groups {
		if group.Id == groupId {
			return group.Week[dayOfWeek][hourString]
		}
	}

	return ""
}

func (s *Schedule) GetScheduleDescription(groupId int64, date time.Time) string {
	status := s.GetScheduleStatus(groupId, date)

	if status == Yes {
		return fmt.Sprintf("Згідно розкладу групи %d, світло є", groupId)
	}

	if status == No {
		return fmt.Sprintf("Згідно розкладу групи %d, світла нема", groupId)
	}

	if status == Maybe {
		return fmt.Sprintf("Згідно розкладу групи %d, світла може не бути", groupId)
	}

	return ""
}
