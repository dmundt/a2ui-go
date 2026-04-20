package stream

import "testing"

func TestBrokerSubscribePublishAndCancel(t *testing.T) {
	b := NewBroker()
	ch, cancel := b.Subscribe()

	b.Publish("hello")
	select {
	case msg := <-ch:
		if msg != "hello" {
			t.Fatalf("unexpected message: %q", msg)
		}
	default:
		t.Fatalf("expected published message")
	}

	cancel()
	if _, ok := <-ch; ok {
		t.Fatalf("expected channel closed after cancel")
	}

	// Ensure publishing after cancel does not panic.
	b.Publish("after-cancel")
}

func TestBrokerPublishDropsWhenSubscriberBufferFull(t *testing.T) {
	b := NewBroker()
	ch, cancel := b.Subscribe()
	defer cancel()

	// Fill the subscriber buffer (size 16).
	for i := 0; i < 16; i++ {
		b.Publish("x")
	}
	// This should be dropped and must not block.
	b.Publish("dropped")

	count := 0
	for {
		select {
		case <-ch:
			count++
		default:
			if count != 16 {
				t.Fatalf("expected 16 buffered messages, got %d", count)
			}
			return
		}
	}
}
