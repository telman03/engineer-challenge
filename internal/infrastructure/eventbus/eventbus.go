package eventbus

import (
	"context"
	"log/slog"
)

// Event is a marker interface for domain events.
type Event interface {
	EventName() string
}

// EventHandler processes a domain event.
type EventHandler func(ctx context.Context, event Event)

// EventBus publishes domain events to subscribers.
type EventBus interface {
	Publish(ctx context.Context, event Event)
	Subscribe(eventName string, handler EventHandler)
}

// InMemoryEventBus is a simple synchronous in-process event bus.
// In production, this would be replaced with a message broker (NATS, Kafka, RabbitMQ).
type InMemoryEventBus struct {
	handlers map[string][]EventHandler
	logger   *slog.Logger
}

func NewInMemoryEventBus(logger *slog.Logger) *InMemoryEventBus {
	return &InMemoryEventBus{
		handlers: make(map[string][]EventHandler),
		logger:   logger,
	}
}

func (b *InMemoryEventBus) Publish(ctx context.Context, event Event) {
	b.logger.Info("domain event published", "event", event.EventName())
	handlers, ok := b.handlers[event.EventName()]
	if !ok {
		return
	}
	for _, h := range handlers {
		h(ctx, event)
	}
}

func (b *InMemoryEventBus) Subscribe(eventName string, handler EventHandler) {
	b.handlers[eventName] = append(b.handlers[eventName], handler)
}
