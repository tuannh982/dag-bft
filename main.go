package main

import (
	"context"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/tuannh982/dag-bft/dag"
	"github.com/tuannh982/dag-bft/dag/commons"
)

func main() {
	ss := make([]*dag.RiderNode, 0)
	ctxs := make([]context.Context, 0)
	n := 3
	for i := 0; i < n; i++ {
		ss = append(ss, dag.NewRiderNode(commons.Address(strconv.Itoa(i)), 2, 1*time.Second))
		ctxs = append(ctxs, context.Background())
	}
	for i := 0; i < n; i++ {
		ss[i].SetPeers(ss)
	}
	for i := 0; i < n; i++ {
		_ = ss[i].Start(ctxs[i])
	}
	time.Sleep(1 * time.Second)
	ss[0].BlockToPropose <- "A0"
	ss[1].BlockToPropose <- "B0"
	ss[2].BlockToPropose <- "C0"
	ss[0].BlockToPropose <- "A1"
	ss[1].BlockToPropose <- "B1"
	ss[2].BlockToPropose <- "C1"
	ss[0].BlockToPropose <- "A2"
	ss[1].BlockToPropose <- "B2"
	ss[2].BlockToPropose <- "C2"
	ss[0].BlockToPropose <- "A3"
	ss[1].BlockToPropose <- "B3"
	ss[2].BlockToPropose <- "C3"
	ss[0].BlockToPropose <- "A4"
	ss[1].BlockToPropose <- "B4"
	ss[2].BlockToPropose <- "C4"
	done := make(chan struct{})
	go hookShutdownSignal(done)
	<-done
}

func hookShutdownSignal(done chan struct{}) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	close(done)
}
