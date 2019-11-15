package main

import (
	"fmt"
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

func (i *incident) Update(n *notification) {
	if i.first == nil {
		i.first = n
	}
	if i.last == nil || n.Timestamp.After(i.last.Timestamp) {
		i.last = n
	}
}

func (i *incident) Duration() time.Duration {
	if i.first == nil || i.last == nil {
		return time.Duration(0)
	}
	return i.last.Timestamp.Sub(i.first.Timestamp)
}

func (i *incident) IsResolved() bool {
	return i.last.Status == statusResolved
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

func (s *store) getIncident(n *notification) *incident {
	var i *incident
	done := make(chan struct{})
	s.actionc <- func() {
		defer close(done)
		i, _ = s.incidents[n.Key()]
	}
	<-done
	return i
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
