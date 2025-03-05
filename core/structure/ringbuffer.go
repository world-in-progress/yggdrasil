package structure

import (
	"runtime"
	"sync"
	"sync/atomic"
)

// Nillable defines types that can be nil.
type Nillable interface {
	~*struct{} | *any | any
}

// RingBuffer is a fixed-size, thread-safe ring buffer.
type RingBuffer[T Nillable] struct {
	buffer   []T
	capacity uint64
	writePos atomic.Uint64
	readPos  atomic.Uint64
}

func NewRingBuffer[T Nillable](size int) *RingBuffer[T] {
	if size <= 0 {
		size = 1 // Prevent zero-size buffer
	}
	return &RingBuffer[T]{
		buffer:   make([]T, size),
		capacity: uint64(size),
	}
}

func (r *RingBuffer[T]) Push(entry T) bool {
	for {
		read := r.readPos.Load()
		write := r.writePos.Load()

		if write-read >= r.capacity { // Buffer full
			return false
		}

		nextWrite := write + 1
		if r.writePos.CompareAndSwap(write, nextWrite) {
			r.buffer[write%r.capacity] = entry
			return true
		}
	}
}

func (r *RingBuffer[T]) Pop() (T, bool) {
	var zero T
	for {
		read := r.readPos.Load()
		write := r.writePos.Load()

		if read > write { // Buffer empty
			return zero, false
		}

		nextRead := read + 1
		if r.readPos.CompareAndSwap(read, nextRead) {
			entry := r.buffer[read%r.capacity]
			return entry, true
		}
	}
}

func (r *RingBuffer[T]) Len() uint64 {
	read := r.readPos.Load()
	write := r.writePos.Load()
	if write < read {
		return 0
	}
	return write - read
}

type node[T any] struct {
	data T
	next *node[T]
}

// ElasticBuffer extends RingBuffer with an overflow slice.
type ElasticBuffer[T any] struct {
	nodes    atomic.Pointer[node[T]]
	nodePool sync.Pool
}

func NewElasticBuffer[T any]() *ElasticBuffer[T] {
	return &ElasticBuffer[T]{
		nodePool: sync.Pool{
			New: func() any { return new(node[T]) },
		},
	}
}

func (erb *ElasticBuffer[T]) Push(entry T) {

	newNode := erb.nodePool.Get().(*node[T])
	newNode.data = entry
	newNode.next = nil

	for {
		head := erb.nodes.Load()
		newNode.next = head
		if erb.nodes.CompareAndSwap(head, newNode) {
			return
		}
	}
}

func (erb *ElasticBuffer[T]) Pop() (T, bool) {
	var zero T

	for {
		head := erb.nodes.Load()
		if head == nil {
			return zero, false
		}
		next := head.next
		if erb.nodes.CompareAndSwap(head, next) {
			data := head.data
			head.data = zero
			head.next = nil
			erb.nodePool.Put(head)
			return data, true
		}
		runtime.Gosched()
	}
}

func (erb *ElasticBuffer[T]) Len() uint64 {
	var length uint64 = 0
	head := erb.nodes.Load()
	for head != nil {
		length++
		head = head.next
	}
	return length
}
