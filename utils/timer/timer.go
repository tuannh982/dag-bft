package timer

import (
	"context"
	"time"

	"github.com/tuannh982/dag-bft/utils/service"
)

type TimeoutRequest interface {
	Duration() time.Duration
	SetTimeoutTs(t time.Time)
	TimeoutTs() time.Time
}

type Timer[R TimeoutRequest] interface {
	service.Service
	C() <-chan R
	Reset(R)
}

type timer[R TimeoutRequest] struct {
	service.SimpleService
	clock   *time.Timer
	request chan R
	timeout chan R
}

func NewTimer[R TimeoutRequest]() Timer[R] {
	t := &timer[R]{
		clock:   time.NewTimer(0),
		request: make(chan R),
		timeout: make(chan R),
	}
	t.SimpleService = *service.NewSimpleService(t)
	t.forceStopInternalTimer()
	return t
}

func (t *timer[R]) forceStopInternalTimer() {
	if !t.clock.Stop() {
		select {
		case <-t.clock.C:
		default:
		}
	}
}

func (t *timer[R]) OnStart(ctx context.Context) error {
	go func() {
		var currentReq R
		for {
			select {
			case req := <-t.request:
				// clear old request
				t.forceStopInternalTimer()
				currentReq = req
				t.clock.Reset(currentReq.Duration())
			case timeoutTs := <-t.clock.C:
				currentReq.SetTimeoutTs(timeoutTs)
				go func(req R) {
					select {
					case t.timeout <- req:
					case <-ctx.Done():
					}
				}(currentReq)
			case <-ctx.Done():
				return
			}
		}
	}()
	return nil
}

func (t *timer[R]) OnStop() {
	t.forceStopInternalTimer()
}

func (t *timer[R]) C() <-chan R {
	return t.timeout
}

func (t *timer[R]) Reset(request R) {
	t.request <- request
}
