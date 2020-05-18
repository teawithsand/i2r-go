package util

import (
	"context"
	"errors"
	"sync"
)

var ErrSinkClosed = errors.New("i2r/util: Given message sink is closed")

type SinkID string

type MessageRouter struct {
	SinkLock *sync.RWMutex
	Sinks    map[SinkID]chan interface{}
	Fallback chan interface{}
}

func NewMessageRouter(fallback chan interface{}) *MessageRouter {
	return &MessageRouter{
		SinkLock: &sync.RWMutex{},
		Sinks:    make(map[SinkID]chan interface{}),
		Fallback: fallback,
	}
}

// SendToID sends message to receiver with given id or times out due to context.
// If there is no valid receiver for this id message is sent to fallback.
func (mr *MessageRouter) SendToID(ctx context.Context, id SinkID, msg interface{}) (err error) {
	mr.SinkLock.RLock()
	sink, ok := mr.Sinks[id]
	mr.SinkLock.Unlock()
	if !ok {
		sink = mr.Fallback
	}
	// either:
	// send to appropriate sink
	// or timeout trying
	// Cancelation can be added using appropriate method in context package
	select {
	case sink <- msg:
	case <-ctx.Done():
		err = ctx.Err()
	}
	return
}

func (mr *MessageRouter) RegisterSink(id SinkID, sink chan interface{}) {
	mr.SinkLock.Lock()
	defer mr.SinkLock.Unlock()

	mr.Sinks[id] = sink
}

func (mr *MessageRouter) DropSink(id SinkID) bool {
	mr.SinkLock.Lock()
	defer mr.SinkLock.Unlock()

	_, ok := mr.Sinks[id]
	if ok {
		delete(mr.Sinks, id)
	}

	return ok
}

/*
// MessageSink is like channel but can be closed without worrying about
// panics on send
type MessageSink struct {
	// IsDone is atomic variable, which should be operated on with atomic.* only
	IsDone  int32
	Channel chan interface{}
}

func (s *MessageSink) Close() {
	atomic.StoreInt32(&s.IsDone, 1)
}

func (s *MessageSink) Receive(ctx context.Context, interrupt chan error) (msg interface{}, err error) {
	if atomic.LoadInt32(&s.IsDone) == 1 {
		err = ErrSinkClosed
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if interrupt == nil {
		interrupt = make(chan error)
	}
	select {
	case err = <-interrupt:
		return
	case <-ctx.Done():
		err = ctx.Err()
		return
	case msg = <-s.Channel:
		return
	}
}
*/
