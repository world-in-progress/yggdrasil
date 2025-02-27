package queue

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const (
	consumers = 4
	rounds    = 100
)

func TestQueue(t *testing.T) {
	producer := newMockedProducer(rounds)
	consumer := newMockedConsumer()
	consumer.wait.Add(consumers)
	q := NewQueue(
		"test",
		func() (Producer[string], error) {
			return producer, nil
		},
		func() (Consumer[string], error) {
			return consumer, nil
		},
	)
	q.AddListener(new(mockedListener))
	q.SetName("mockedQueue")
	q.SetNumConsumer(consumers)
	q.SetNumProducer(1)
	q.pause()
	q.resume()
	go func() {
		producer.wait.Wait()
		q.Stop()
	}()
	q.Start()
	assert.Equal(t, int32(rounds), atomic.LoadInt32(&consumer.count))
}

type (
	mockedProducer struct {
		total    int32
		count    int32
		listener ProduceListener
		wait     sync.WaitGroup
	}

	mockedConsumer struct {
		count      int32
		events     int32
		consumeErr error
		wait       sync.WaitGroup
	}

	mockedListener struct{}
)

func newMockedProducer(total int32) *mockedProducer {
	p := &mockedProducer{
		total: total,
	}
	p.wait.Add(int(total))
	return p
}

func newMockedConsumer() *mockedConsumer {
	return &mockedConsumer{}
}

func (p *mockedProducer) AddListener(listener ProduceListener) {
	p.listener = listener
}

func (p *mockedProducer) Produce() (string, bool) {
	if atomic.AddInt32(&p.count, 1) <= p.total {
		p.wait.Done()
		return "item", true
	}

	time.Sleep(time.Second)
	return "", false
}

func (c *mockedConsumer) Consume(string) error {
	atomic.AddInt32(&c.count, 1)
	return c.consumeErr
}

func (c *mockedConsumer) OnEvent(any) {
	if atomic.AddInt32(&c.events, 1) <= consumers {
		c.wait.Done()
	}
}

func (l *mockedListener) OnPause() {
	fmt.Sprintln("Paused")
}

func (l *mockedListener) OnResume() {
	fmt.Sprintln("Resumed")
}
