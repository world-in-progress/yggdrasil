package queue

type (

	// A ProducerListener interface represents a produce listener.
	ProduceListener interface {
		OnProducerPause()
		OnProducerResume()
	}

	// A Producer interface represents a producer that produces messages.
	Producer[T any] interface {
		AddListener(listener ProduceListener)
		Produce() (T, bool)
	}

	// A ProducerFactory is a factory that creates a producer.
	ProducerFactory[T any] func() (Producer[T], error)
)
