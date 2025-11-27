package audit

import (
	"sync"
	"time"
)

// Action тип действия аудита
type Action string

const (
	ActionShorten Action = "shorten"
	ActionFollow  Action = "follow"
)

// Event структура события аудита
type Event struct {
	Timestamp int64  `json:"ts"`
	Action    Action `json:"action"`
	UserID    string `json:"user_id,omitempty"`
	URL       string `json:"url"`
}

// NewEvent создаёт новое событие аудита
func NewEvent(action Action, userID, url string) Event {
	return Event{
		Timestamp: time.Now().Unix(),
		Action:    action,
		UserID:    userID,
		URL:       url,
	}
}

type Observer interface {
	Notify(event Event)
	Close() error
}

type Publisher struct {
	mu          sync.Mutex
	subscribers []Observer
}

func NewPublisher() *Publisher {
	return &Publisher{}
}

func (p *Publisher) Subscribe(o Observer) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.subscribers = append(p.subscribers, o)
}

func (p *Publisher) Publish(event Event) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, s := range p.subscribers {
		s.Notify(event)
	}
}

// Close закрывает всех наблюдателей
func (p *Publisher) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, obs := range p.subscribers {
		if err := obs.Close(); err != nil {
			return err
		}
	}
	return nil
}
