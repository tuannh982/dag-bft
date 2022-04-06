package dag

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"sort"
	"time"

	"github.com/tuannh982/dag-bft/dag/commons"
	"github.com/tuannh982/dag-bft/dag/internal"
	"github.com/tuannh982/dag-bft/utils/collections"
	"github.com/tuannh982/dag-bft/utils/math"
	"github.com/tuannh982/dag-bft/utils/service"
)

// TODO FIXME not working code

const DefaultTimeout = 3 * time.Second

func must(err error) {
	if err != nil {
		panic(err)
	}
}

type BullsharkNode struct {
	service.SimpleService
	// node info
	nodeInfo *commons.NodeInfo
	// peers info
	peers []*BullsharkNode
	f     int
	// persistent info
	dag            internal.DAG
	round          commons.Round
	buffer         collections.Set[*commons.Vertex]
	ssValidatorSet internal.ValidatorSet
	fbValidatorSet internal.ValidatorSet
	decidedWave    commons.Wave
	leaderStack    collections.Stack[*commons.Vertex]
	// non-persistent info
	waitForTimer bool
	timer        *time.Timer
	timerTimeout time.Duration
	// channels
	RBcastChannel  chan internal.BroadcastMessage[*commons.Vertex, commons.Round]
	ABcastChannel  chan internal.BroadcastMessage[commons.Block, uint64]
	BlockToPropose chan commons.Block
	// log
	log *log.Logger
}

func NewBullsharkNode(addr commons.Address) *BullsharkNode {
	instance := &BullsharkNode{
		nodeInfo: &commons.NodeInfo{
			Address: addr,
		},
		peers:          make([]*BullsharkNode, 0),
		f:              0,
		dag:            internal.NewDAG(),
		round:          0,
		buffer:         collections.NewHashSet(commons.VertexHash),
		ssValidatorSet: internal.NewValidatorSet(),
		fbValidatorSet: internal.NewValidatorSet(),
		decidedWave:    0,
		leaderStack:    collections.NewStack[*commons.Vertex](),
		waitForTimer:   true,
		timer:          time.NewTimer(0),
		timerTimeout:   DefaultTimeout,
		RBcastChannel:  make(chan internal.BroadcastMessage[*commons.Vertex, commons.Round]),
		ABcastChannel:  make(chan internal.BroadcastMessage[commons.Block, uint64]),
		BlockToPropose: make(chan commons.Block),
		log:            log.New(os.Stdout, fmt.Sprintf("Node %s: ", addr), os.O_APPEND|os.O_CREATE|os.O_WRONLY),
	}
	instance.peers = append(instance.peers, instance)
	_ = instance.timer.Stop()
	instance.SimpleService = *service.NewSimpleService(instance)
	return instance
}

func (node *BullsharkNode) OnStart(ctx context.Context) error {
	err := node.Init()
	if err != nil {
		return err
	}
	node.StartRoutine(ctx)
	return nil
}

func (node *BullsharkNode) OnStop() {
	close(node.RBcastChannel)
	close(node.ABcastChannel)
	close(node.BlockToPropose)
	if !node.timer.Stop() {
		select {
		case <-node.timer.C:
		default:
		}
	}
}

func (node *BullsharkNode) Init() error {
	round0 := commons.Round(0)
	wave1 := commons.Wave(0)
	node.dag.NewRoundIfNotExists(round0)
	for _, peer := range node.peers {
		v := commons.Vertex{
			StrongEdges: make([]commons.BaseVertex, 0),
			WeakEdges:   make([]commons.BaseVertex, 0),
			Delivered:   false,
		}
		v.Source = peer.nodeInfo.Address
		v.Round = 0
		v.Block = ""
		b := node.dag.GetRound(round0).AddVertex(v)
		if !b {
			return errors.New("could not add vertex")
		}
	}
	node.ssValidatorSet.NewWaveIfNotExists(wave1)
	node.fbValidatorSet.NewWaveIfNotExists(wave1)
	for _, peer := range node.peers {
		err := node.ssValidatorSet.GetWave(wave1).Add(peer.nodeInfo.Address)
		if err != nil {
			return err
		}
	}
	return nil
}

func (node *BullsharkNode) SetPeers(peers []*BullsharkNode) {
	node.peers = peers
	node.f = len(node.peers) / 3
}

func (node *BullsharkNode) StartRoutine(ctx context.Context) {
	go node.ReportRoutine(ctx, 10*time.Second)
	go node.ReceiveRoutine(ctx)
	node.timer.Reset(node.timerTimeout)
	go node.TimeoutRoutine(ctx)
}

func (node *BullsharkNode) ReportRoutine(ctx context.Context, interval time.Duration) {
	timer := time.NewTimer(interval)
	for {
		select {
		case <-timer.C:
			node.log.Println("REPORT", fmt.Sprintf("round=%d\n%s", node.round, node.dag.String()))
			timer.Reset(interval)
		case <-ctx.Done():
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			return
		}
	}
}

func (node *BullsharkNode) ReceiveRoutine(ctx context.Context) {
	for {
		select {
		case rMsg := <-node.RBcastChannel:
			node.log.Println("receive message from RBcastChannel", "p=", rMsg.P, "r=", rMsg.R, "m=", rMsg.Message)
			node.rDeliver(rMsg.Message, rMsg.R, rMsg.P)
		case aMsg := <-node.ABcastChannel:
			node.log.Println("receive message from ABcastChannel", "p=", aMsg.P, "r=", aMsg.R, "m=", aMsg.Message)
			node.aBcast(aMsg.Message, aMsg.R)
		case <-ctx.Done():
			return
		default:
			continue
		}
	}
}

func (node *BullsharkNode) TimeoutRoutine(ctx context.Context) {
	for {
		select {
		case <-node.timer.C:
			node.log.Println("timer timeout")
			node.handleTimeout()
		case <-ctx.Done():
			return
		default:
			continue
		}
	}
}

func (node *BullsharkNode) createNewVertex(round commons.Round) *commons.Vertex {
	v := &commons.Vertex{}
	v.Source = node.nodeInfo.Address
	v.Round = round
	v.Block = <-node.BlockToPropose
	entries := node.dag.GetRound(round - 1).Entries()
	v.StrongEdges = make([]commons.BaseVertex, 0, len(entries))
	for _, entry := range entries {
		v.StrongEdges = append(v.StrongEdges, commons.BaseVertex{
			Source: entry.Source,
			Round:  entry.Round,
			Block:  entry.Block,
		})
	}
	node.setWeakEdges(v, round)
	node.log.Println("vertex created", v)
	return v
}

func (node *BullsharkNode) setWeakEdges(v *commons.Vertex, round commons.Round) {
	v.WeakEdges = make([]commons.BaseVertex, 0)
	for r := round - 2; r >= 1; r-- {
		roundSet := node.dag.GetRound(r)
		if roundSet != nil {
			for _, u := range roundSet.Entries() {
				if !node.dag.Path(v, &u) {
					v.WeakEdges = append(v.WeakEdges, commons.BaseVertex{
						Source: u.Source,
						Round:  u.Round,
						Block:  u.Block,
					})
				}
			}
		}
	}
}

func (node *BullsharkNode) getSteadyLeader(w commons.Wave) commons.Address {
	p := node.peers[int(w)%len(node.peers)].nodeInfo.Address // TODO update later
	return p
}

func (node *BullsharkNode) getFallbackLeader(w commons.Wave) commons.Address {
	p := node.peers[int(w)%len(node.peers)].nodeInfo.Address // TODO update later
	return p
}

func (node *BullsharkNode) getSteadyVertexLeader(w commons.Wave) *commons.Vertex {
	p := node.getSteadyLeader(w)
	return node.getVertex(p, commons.Round(2*w-1))
}

func (node *BullsharkNode) getFallbackVertexLeader(w commons.Wave) *commons.Vertex {
	p := node.getFallbackLeader(w)
	return node.getVertex(p, commons.Round(4*w-3))
}

func (node *BullsharkNode) getVertex(p commons.Address, r commons.Round) *commons.Vertex {
	if roundSet := node.dag.GetRound(r); roundSet != nil {
		if roundSet.SourceExists(p) {
			ret := roundSet.GetBySource(p)
			return &ret
		}
	}
	return nil
}

func (node *BullsharkNode) rDeliver(v *commons.Vertex, r commons.Round, p commons.Address) {
	node.log.Println("len(v.StrongEdges)", len(v.StrongEdges), "v.StrongEdgesContainsSource(v.Source)", v.StrongEdgesContainsSource(v.Source))
	if v.Source == p && v.Round == r && len(v.StrongEdges) >= 2*node.f+1 && v.StrongEdgesContainsSource(v.Source) {
		if !node.tryAddToDag(v) {
			must(node.buffer.Add(v))
		} else {
			for _, vp := range node.buffer.Entries() {
				if vp.Round <= r {
					node.tryAddToDag(vp)
				}
			}
			ssWave := commons.Wave(math.DivCeil(r, 2))
			if r%2 == 1 && (!node.waitForTimer || node.dag.GetRound(r).SourceExists(node.getSteadyLeader(ssWave))) {
				node.tryAdvanceRound()
			}
			count := 0
			for _, u := range node.dag.GetRound(r).Entries() {
				if node.ssValidatorSet.GetWave(ssWave).Contains(u.Source) {
					count++
				}
			}
			if r%2 == 0 && (!node.waitForTimer || count == 2*node.f+1) {
				node.tryAdvanceRound()
			}
		}
	}
}

func (node *BullsharkNode) handleTimeout() {
	node.waitForTimer = false
	node.tryAdvanceRound()
}

func (node *BullsharkNode) tryAddToDag(v *commons.Vertex) bool {
	node.log.Println("tryAddToDag", v)
	if node.dag.AllEdgesExist(v) {
		node.dag.NewRoundIfNotExists(v.Round)
		node.dag.GetRound(v.Round).AddVertex(*v)
		_ = node.buffer.Remove(v)
		node.log.Println("vertex added to DAG", v)
		node.tryOrdering(v)
		return true
	} else {
		return false
	}
}

func (node *BullsharkNode) tryAdvanceRound() {
	node.log.Println("tryAdvanceRound", node.round, "round dag size", node.dag.GetRound(node.round).Size())
	if node.dag.GetRound(node.round).Size() >= 2*node.f+1 {
		node.round = node.round + 1
		node.dag.NewRoundIfNotExists(node.round)
		v := node.createNewVertex(node.round)
		node.tryAddToDag(v)
		node.rBcast(v, node.round)
		node.waitForTimer = true
		node.timer.Reset(node.timerTimeout)
	}
}

func (node *BullsharkNode) rBcast(v *commons.Vertex, r commons.Round) {
	for _, peer := range node.peers {
		if peer.nodeInfo.Address != node.nodeInfo.Address {
			clonedPeer := peer
			go func() {
				node.log.Println("message rBcast to", clonedPeer.nodeInfo.Address, "v=", v, "r=", r)
				clonedPeer.RBcastChannel <- internal.BroadcastMessage[*commons.Vertex, commons.Round]{
					Message: v,
					R:       r,
					P:       node.nodeInfo.Address,
				}
			}()
		}
	}
}

func (node *BullsharkNode) aBcast(b commons.Block, r uint64) {
	node.BlockToPropose <- b
}

func (node *BullsharkNode) tryOrdering(v *commons.Vertex) {
	node.updateValidatorMode(v)
}

func (node *BullsharkNode) updateValidatorMode(v *commons.Vertex) {
	ssWave := commons.Wave(math.DivCeil(v.Round, 2))
	fbWave := commons.Wave(math.DivCeil(v.Round, 4))
	node.ssValidatorSet.NewWaveIfNotExists(ssWave)
	node.fbValidatorSet.NewWaveIfNotExists(fbWave)
	if node.ssValidatorSet.GetWave(ssWave).Contains(v.Source) || node.fbValidatorSet.GetWave(fbWave).Contains(v.Source) {
		return
	}
	potentialVotes := v.StrongEdgesValues()
	if node.ssValidatorSet.GetWave(ssWave-1).Contains(v.Source) && node.trySteadyCommit(potentialVotes, ssWave-1) {
		must(node.ssValidatorSet.GetWave(ssWave).Add(v.Source))
	} else if node.fbValidatorSet.GetWave(fbWave-1).Contains(v.Source) && node.tryFallbackCommit(potentialVotes, fbWave-1) {
		must(node.ssValidatorSet.GetWave(ssWave).Add(v.Source))
	} else {
		must(node.fbValidatorSet.GetWave(fbWave).Add(v.Source))
	}
}

func (node *BullsharkNode) trySteadyCommit(potentialVotes []commons.BaseVertex, w commons.Wave) bool {
	node.log.Println("trySteadyCommit", "potentialVotes=", potentialVotes, "wave=", w)
	v := node.getSteadyVertexLeader(w)
	count := 0
	for _, vp := range potentialVotes {
		vvp := node.dag.GetRound(vp.Round).GetBySource(vp.Source)
		if node.ssValidatorSet.GetWave(w).Contains(vp.Source) && node.dag.StrongPath(&vvp, v) {
			count++
		}
	}
	if count >= 2*node.f+1 {
		node.commitLeader(v)
		return true
	} else {
		return false
	}
}

func (node *BullsharkNode) tryFallbackCommit(potentialVotes []commons.BaseVertex, w commons.Wave) bool {
	node.log.Println("tryFallbackCommit", "potentialVotes=", potentialVotes, "wave=", w)
	v := node.getFallbackVertexLeader(w)
	count := 0
	for _, vp := range potentialVotes {
		vvp := node.dag.GetRound(vp.Round).GetBySource(vp.Source)
		if node.fbValidatorSet.GetWave(w).Contains(vp.Source) && node.dag.StrongPath(&vvp, v) {
			count++
		}
	}
	node.log.Println("count", count)
	if count >= 2*node.f+1 {
		node.commitLeader(v)
		return true
	} else {
		return false
	}
}

func (node *BullsharkNode) commitLeader(v *commons.Vertex) {
	node.leaderStack.Push(v)
	ssWave := commons.Wave(math.DivCeil(v.Round, 2))
	for w := ssWave - 1; w >= node.decidedWave+1; w-- {
		potentialVotes := make([]commons.BaseVertex, 0)
		if node.dag.GetRound(commons.Round(2*w + 1)).SourceExists(v.Source) {
			potentialVotes = node.dag.GetRound(commons.Round(2*w + 1)).GetBySource(v.Source).StrongEdges
		}
		vs := node.getSteadyVertexLeader(w)
		ssVotes := make([]commons.Vertex, 0)
		for _, vp := range potentialVotes {
			vvp := node.dag.GetRound(vp.Round).GetBySource(vp.Source)
			if node.ssValidatorSet.GetWave(w).Contains(vp.Source) && node.dag.StrongPath(&vvp, vs) {
				ssVotes = append(ssVotes, vvp)
			}
		}
		fbVotes := make([]commons.Vertex, 0)
		var vf *commons.Vertex = nil
		if w%2 == 0 {
			vf = node.getFallbackVertexLeader(w / 2)
			for _, vp := range potentialVotes {
				vvp := node.dag.GetRound(vp.Round).GetBySource(vp.Source)
				if node.fbValidatorSet.GetWave(w/2).Contains(vp.Source) && node.dag.StrongPath(&vvp, vf) {
					fbVotes = append(fbVotes, vvp)
				}
			}
		}
		if len(ssVotes) >= node.f+1 && len(fbVotes) < node.f+1 {
			node.leaderStack.Push(vs)
			v = vs
		}
		if len(ssVotes) < node.f+1 && len(fbVotes) >= node.f+1 {
			node.leaderStack.Push(vf)
			v = vf
		}
	}
	node.decidedWave = ssWave
	node.orderVertices()
}

func (node *BullsharkNode) orderVertices() {
	for node.leaderStack.Size() > 0 {
		v := node.leaderStack.Pop()
		verticesToDeliver := make([]commons.Vertex, 0)
		for r := commons.Round(1); r < node.round; r++ {
			for _, vp := range node.dag.GetRound(r).Entries() {
				if node.dag.Path(v, &vp) && !v.Delivered {
					verticesToDeliver = append(verticesToDeliver, vp)
				}
			}
		}
		/*
			for every vp in verticesToDeliver in some deterministic order do
				output a_deliver(v.block, v.round, v.source)
				deliveredVertices = deliveredVertices + vp
		*/
		sort.Slice(verticesToDeliver, func(i, j int) bool {
			return commons.VertexHash(&verticesToDeliver[i]) < commons.VertexHash(&verticesToDeliver[j])
		})
		for _, vp := range verticesToDeliver {
			node.dag.SetDelivered(&vp, true)
			node.log.Println("deliver vertex", vp)
		}
	}
}
