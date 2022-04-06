package internal

import (
	"github.com/tuannh982/dag-bft/dag/commons"
	"golang.org/x/exp/constraints"
)

type BroadcastMessage[T any, S constraints.Ordered] struct {
	Message T               // message
	R       S               // sequence number
	P       commons.Address // sender
}
