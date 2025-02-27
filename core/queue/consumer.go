package queue

type (
	// A Consumer interface represents a consumer that can consume string messages
	Consumer[T any] interface {
		Consume(T) error
		OnEvent(event any)
	}

	// ConsumeFactory is a factory that creates a consumer
	ConsumeFactory[T any] func() (Consumer[T], error)
)
