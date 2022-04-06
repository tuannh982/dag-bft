package service

import (
	"context"
)

type Service interface {
	Start(ctx context.Context) error
	IsRunning() bool
	Serve()
	Stop()
}

type StartStopCallback interface {
	OnStart(ctx context.Context) error
	OnStop()
}
