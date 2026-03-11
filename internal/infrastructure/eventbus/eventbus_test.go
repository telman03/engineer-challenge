package eventbus_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/atls-academy/engineer-challenge/internal/infrastructure/eventbus"
)

type testEvent struct {
	name string
	data string
}

func (e testEvent) EventName() string { return e.name }

func TestInMemoryEventBus_PublishSubscribe(t *testing.T) {
	bus := eventbus.NewInMemoryEventBus(slog.Default())

	var received []string
	bus.Subscribe("user.registered", func(ctx context.Context, event eventbus.Event) {
		received = append(received, event.EventName())
	})

	bus.Publish(context.Background(), testEvent{name: "user.registered", data: "test"})

	if len(received) != 1 {
		t.Fatalf("expected 1 event, got %d", len(received))
	}
	if received[0] != "user.registered" {
		t.Errorf("event name = %q, want %q", received[0], "user.registered")
	}
}

func TestInMemoryEventBus_MultipleSubscribers(t *testing.T) {
	bus := eventbus.NewInMemoryEventBus(slog.Default())

	count := 0
	handler := func(ctx context.Context, event eventbus.Event) {
		count++
	}
	bus.Subscribe("test.event", handler)
	bus.Subscribe("test.event", handler)

	bus.Publish(context.Background(), testEvent{name: "test.event"})

	if count != 2 {
		t.Errorf("expected 2 handler calls, got %d", count)
	}
}

func TestInMemoryEventBus_NoSubscribers(t *testing.T) {
	bus := eventbus.NewInMemoryEventBus(slog.Default())
	bus.Publish(context.Background(), testEvent{name: "unsubscribed.event"})
}

func TestInMemoryEventBus_DifferentEvents(t *testing.T) {
	bus := eventbus.NewInMemoryEventBus(slog.Default())

	eventACount := 0
	eventBCount := 0

	bus.Subscribe("event.a", func(ctx context.Context, event eventbus.Event) {
		eventACount++
	})
	bus.Subscribe("event.b", func(ctx context.Context, event eventbus.Event) {
		eventBCount++
	})

	bus.Publish(context.Background(), testEvent{name: "event.a"})

	if eventACount != 1 {
		t.Errorf("eventA handler count = %d, want 1", eventACount)
	}
	if eventBCount != 0 {
		t.Errorf("eventB handler count = %d, want 0", eventBCount)
	}
}
