package service

import (
	"context"
	"errors"
	"sync"
)

var (
	ErrServiceAlreadyStopped = errors.New("service already stopped")
)

type SimpleService struct {
	cancel      context.CancelFunc
	closeChan   <-chan struct{}
	mu          sync.Mutex
	startStopCb StartStopCallback
}

func NewSimpleService(startStopCb StartStopCallback) *SimpleService {
	return &SimpleService{
		startStopCb: startStopCb,
	}
}

func (s *SimpleService) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closeChan == nil {
		select {
		case <-s.closeChan:
			return ErrServiceAlreadyStopped
		default:
			err := s.startStopCb.OnStart(ctx)
			if err != nil {
				return err
			}
			wrappedCtx, cancel := context.WithCancel(ctx)
			s.cancel = cancel
			s.closeChan = wrappedCtx.Done()
			go func(ctx context.Context) {
				select {
				case <-wrappedCtx.Done():
					return
				case <-ctx.Done():
					s.Stop()
				}
			}(ctx)
			return nil
		}
	}
	return nil
}

func (s *SimpleService) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closeChan != nil {
		select {
		case <-s.closeChan:
			return false
		default:
			return true
		}
	} else {
		return false
	}
}

func (s *SimpleService) Serve() {
	var ch <-chan struct{}
	s.mu.Lock()
	if s.closeChan != nil {
		ch = s.closeChan
	}
	s.mu.Unlock()
	if ch != nil {
		<-ch
	}
}

func (s *SimpleService) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closeChan != nil {
		select {
		case <-s.closeChan:
			return
		default:
			s.startStopCb.OnStop()
			s.cancel()
		}
	}
}
