package stream

import "sync"

// Broker distributes html updates to SSE clients.
type Broker struct {
	mu      sync.RWMutex
	nextID  int
	clients map[int]chan string
}

// NewBroker creates a broker.
func NewBroker() *Broker {
	return &Broker{clients: make(map[int]chan string)}
}

// Subscribe returns a channel and cancel callback.
func (b *Broker) Subscribe() (<-chan string, func()) {
	b.mu.Lock()
	defer b.mu.Unlock()
	id := b.nextID
	b.nextID++
	ch := make(chan string, 16)
	b.clients[id] = ch
	return ch, func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		if c, ok := b.clients[id]; ok {
			close(c)
			delete(b.clients, id)
		}
	}
}

// Publish broadcasts non-blocking to all subscribers.
func (b *Broker) Publish(msg string) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, ch := range b.clients {
		select {
		case ch <- msg:
		default:
		}
	}
}
