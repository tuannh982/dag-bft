package timer

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type mockTimeoutRequest struct {
	duration  time.Duration
	timeoutTs time.Time
}

func (r *mockTimeoutRequest) Duration() time.Duration {
	return r.duration
}

func (r *mockTimeoutRequest) SetTimeoutTs(t time.Time) {
	r.timeoutTs = t
}

func (r *mockTimeoutRequest) TimeoutTs() time.Time {
	return r.timeoutTs
}

func TestTimer(t *testing.T) {
	timerService := NewTimer[*mockTimeoutRequest]()
	ctx := context.Background()
	err := timerService.Start(ctx)
	require.Equal(t, nil, err)
	go func() {
		timerService.Reset(&mockTimeoutRequest{
			duration: 2 * time.Second,
		})
		time.Sleep(3 * time.Second)
		timerService.Reset(&mockTimeoutRequest{
			duration: 3 * time.Second,
		})
		time.Sleep(5 * time.Second)
		timerService.Stop()
	}()
	go func() {
		r := <-timerService.C()
		fmt.Println(r.Duration(), r.TimeoutTs())
		r = <-timerService.C()
		fmt.Println(r.Duration(), r.TimeoutTs())
	}()
	timerService.Serve()
}
