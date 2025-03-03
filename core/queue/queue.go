package queue

import (
	"fmt"
	"log"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/world-in-progress/yggdrasil/core/rescue"
	"github.com/world-in-progress/yggdrasil/core/threading"
)

type (
	// Listener interface represents a listener that can be notified with queue events.
	Listener interface {
		OnPause()
		OnResume()
	}

	// Queue is the structure for task queue.
	Queue[T any] struct {
		name                 string
		producerFactory      ProducerFactory[T]
		producerRoutineGroup *threading.RoutineGroup
		consumerFactory      ConsumeFactory[T]
		consumerRoutineGroup *threading.RoutineGroup
		producerCount        int
		consumerCount        int
		active               int32
		channel              chan T
		quit                 chan struct{}
		listeners            []Listener
		eventLock            sync.Mutex
		eventChannels        []chan any
	}

	routineListener[T any] struct {
		queue *Queue[T]
	}
)

// NewQueue returns a new queue.
func NewQueue[T any](name string, producerFactory ProducerFactory[T], consumerFactory ConsumeFactory[T]) *Queue[T] {

	q := &Queue[T]{
		producerFactory:      producerFactory,
		producerRoutineGroup: threading.NewRoutineGroup(),
		consumerFactory:      consumerFactory,
		consumerRoutineGroup: threading.NewRoutineGroup(),
		producerCount:        runtime.NumCPU(),
		consumerCount:        runtime.NumCPU() << 1,
		channel:              make(chan T),
		quit:                 make(chan struct{}),
	}

	q.SetName(name)
	return q
}

// SetName sets the name of task queue.
func (q *Queue[T]) SetName(name string) {
	q.name = name
}

// SetNumProducer sets the numer of producers.
func (q *Queue[T]) SetNumProducer(count int) {
	q.producerCount = count
}

// SetNumConsumer sets the numer of consumers.
func (q *Queue[T]) SetNumConsumer(count int) {
	q.consumerCount = count
}

// AddListener adds a listener to the queue.
func (q *Queue[T]) AddListener(listener Listener) {
	q.listeners = append(q.listeners, listener)
}

// Broadcast broadcasts the message to all event channels.
func (q *Queue[T]) Broadcast(message any) {
	go func() {
		defer q.eventLock.Unlock()

		q.eventLock.Lock()
		for _, channel := range q.eventChannels {
			channel <- message
		}
	}()
}

// Start starts the task queue.
func (q *Queue[T]) Start() {
	q.startProducers(q.producerCount)
	q.startConsumers(q.consumerCount)

	q.producerRoutineGroup.Wait()
	close(q.channel)
	q.consumerRoutineGroup.Wait()
}

// Stop stops the task queue.
func (q *Queue[T]) Stop() {
	close(q.quit)
}

func (q *Queue[T]) produceOne(producer Producer[T]) (T, bool) {
	defer rescue.Recover()

	return producer.Produce()
}

func (q *Queue[T]) produce() {
	var producer Producer[T]

	for {
		var err error
		if producer, err = q.producerFactory(); err != nil {
			fmt.Println("Error occurred while creating producer: ", err)
			return
		} else {
			break
		}
	}

	atomic.AddInt32(&q.active, 1)
	producer.AddListener(routineListener[T]{
		queue: q,
	})

	for {
		select {
		case <-q.quit:
			fmt.Println("Quitting producer")
			return
		default:
			if v, ok := q.produceOne(producer); ok {
				q.channel <- v
			}
		}
	}
}

func (q *Queue[T]) startProducers(number int) {
	for range number {
		q.producerRoutineGroup.Run(func() {
			q.produce()
		})
	}
}

func (q *Queue[T]) consumeOne(consumer Consumer[T], task T) {
	threading.RunSafe(func() {
		if err := consumer.Consume(task); err != nil {
			log.Fatal("Error occurred while consuming: ", err)
		}
	})
}

func (q *Queue[T]) consume(eventChan chan any) {
	var consumer Consumer[T]

	for {
		var err error
		if consumer, err = q.consumerFactory(); err != nil {
			fmt.Println("Error occurred while creating consumer: ", err)
			return
		} else {
			break
		}
	}

	for {
		select {
		case message, ok := <-q.channel:
			if ok {
				q.consumeOne(consumer, message)
			} else {
				fmt.Println("Task channel was closed, quitting consumer...")
				return
			}
		case event := <-eventChan:
			consumer.OnEvent(event)
		}
	}
}

func (q *Queue[T]) startConsumers(number int) {
	for range number {
		eventChan := make(chan any)
		q.eventLock.Lock()
		q.eventChannels = append(q.eventChannels, eventChan)
		q.eventLock.Unlock()
		q.consumerRoutineGroup.RunSafe(func() {
			q.consume(eventChan)
		})
	}
}

func (q *Queue[T]) pause() {
	for _, listener := range q.listeners {
		listener.OnPause()
	}
}

func (q *Queue[T]) resume() {
	for _, listener := range q.listeners {
		listener.OnResume()
	}
}

func (rl routineListener[T]) OnProducerPause() {
	if atomic.AddInt32(&rl.queue.active, -1) <= 0 {
		rl.queue.pause()
	}
}

func (rl routineListener[T]) OnProducerResume() {
	if atomic.AddInt32(&rl.queue.active, 1) > 0 {
		rl.queue.resume()
	}
}
