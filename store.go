package main

import (
	"fmt"
	"sort"
	"time"

	"github.com/prometheus/alertmanager/template"
)

const (
	statusFiring   = "firing"
	statusResolved = "resolved"
)

// webhookPayload represents the raw data received from Alertmanager.
type webhookPayload struct {
	*template.Data
	Version  string `json:"version"`
	GroupKey string `json:"groupKey"`
}

// notification represents a notification received from Alertmanager.
type notification struct {
	*webhookPayload
	Remote    string    `json:"remoteAddress"`
	Timestamp time.Time `json:"timestamp"`
}

func (n *notification) Key() string {
	return fmt.Sprintf("%s:%s", n.Receiver, n.GroupKey)
}

type incident struct {
	first, last *notification
}

func (i *incident) Key() string {
	return i.first.Key()
}

func (i *incident) Alerts() template.Alerts {
	return i.last.Alerts
}

func (i *incident) Duration() time.Duration {
	if i.first == nil || i.last == nil {
		return time.Duration(0)
	}
	if i.IsResolved() {
		return i.last.Timestamp.Sub(i.first.Timestamp)
	}
	return time.Now().Sub(i.first.Timestamp)
}

func (i *incident) IsResolved() bool {
	return i.last.Status == statusResolved
}

func (i *incident) Update(n *notification) {
	if i.first == nil {
		i.first = n
	}
	if i.last == nil || n.Timestamp.After(i.last.Timestamp) {
		i.last = n
	}
}

// store manages Alertmanager notifications and incidents.
type store struct {
	notifications []*notification
	incidents     map[string]*incident

	actionc chan func()
	quitc   chan struct{}
}

func newStore() *store {
	return &store{
		notifications: make([]*notification, 0),
		incidents:     make(map[string]*incident),
		actionc:       make(chan func()),
		quitc:         make(chan struct{}),
	}
}

func (s *store) stop() {
	close(s.quitc)
}

func (s *store) run() {
	for {
		select {
		case <-s.quitc:
			return
		case f := <-s.actionc:
			f()
		}
	}
}

func (s *store) addNotification(n *notification) {
	s.actionc <- func() {
		s.notifications = append(s.notifications, n)
	}
}

func (s *store) listNotifications() []*notification {
	var notifications []*notification
	done := make(chan struct{})
	s.actionc <- func() {
		defer close(done)
		notifications = s.notifications
	}
	<-done
	return notifications
}

func (s *store) getIncident(k string) *incident {
	var i *incident
	done := make(chan struct{})
	s.actionc <- func() {
		defer close(done)
		i, _ = s.incidents[k]
	}
	<-done
	return i
}

func (s *store) listIncidents() []*incident {
	var incidents []*incident
	done := make(chan struct{})
	s.actionc <- func() {
		defer close(done)
		keys := make([]string, 0, len(s.incidents))
		for k := range s.incidents {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			incidents = append(incidents, s.incidents[k])
		}
	}
	<-done
	return incidents
}

func (s *store) updateIncident(n *notification) *incident {
	var i *incident
	done := make(chan struct{})
	s.actionc <- func() {
		defer close(done)
		i, _ = s.incidents[n.Key()]
		if i == nil {
			i = &incident{}
			s.incidents[n.Key()] = i
		}
		i.Update(n)
	}
	<-done
	return i
}

func (s *store) deleteIncident(i *incident) {
	done := make(chan struct{})
	s.actionc <- func() {
		defer close(done)
		delete(s.incidents, i.Key())
	}
	<-done
	return
}
