package dag

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/tuannh982/dag-bft/dag/commons"
	"github.com/tuannh982/dag-bft/dag/internal"
	"github.com/tuannh982/dag-bft/utils/collections"
	"github.com/tuannh982/dag-bft/utils/service"

	log "github.com/sirupsen/logrus"
)

type RiderNode struct {
	service.SimpleService
	// node info
	nodeInfo *commons.NodeInfo
	// peers info
	peers []*RiderNode
	f     int
	w     int
	// persistent info
	dag         internal.DAG
	round       commons.Round
	buffer      collections.Set[*commons.Vertex]
	decidedWave commons.Wave
	leaderStack collections.Stack[*commons.Vertex]
	// non-persistent info
	timer        *time.Timer
	timerTimeout time.Duration
	// channels
	RBcastChannel  chan internal.BroadcastMessage[*commons.Vertex, commons.Round]
	ABcastChannel  chan internal.BroadcastMessage[commons.Block, uint64]
	BlockToPropose chan commons.Block
	// log
	log *log.Entry
}

func NewRiderNode(addr commons.Address, waveSize int, timerTimeout time.Duration) *RiderNode {
	logger := log.WithFields(log.Fields{"node": string(addr)})
	logger.Logger.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})
	logger.Level = log.InfoLevel
	instance := &RiderNode{
		nodeInfo: &commons.NodeInfo{
			Address: addr,
		},
		peers:          make([]*RiderNode, 0),
		f:              0,
		w:              waveSize,
		dag:            internal.NewDAG(),
		round:          0,
		buffer:         collections.NewHashSet(commons.VertexHash),
		decidedWave:    0,
		leaderStack:    collections.NewStack[*commons.Vertex](),
		timer:          time.NewTimer(0),
		timerTimeout:   timerTimeout,
		RBcastChannel:  make(chan internal.BroadcastMessage[*commons.Vertex, commons.Round], 65535),
		ABcastChannel:  make(chan internal.BroadcastMessage[commons.Block, uint64], 65535),
		BlockToPropose: make(chan commons.Block, 65535),
		log:            logger,
	}
	instance.peers = append(instance.peers, instance)
	_ = instance.timer.Stop()
	instance.SimpleService = *service.NewSimpleService(instance)
	return instance
}

func (node *RiderNode) OnStart(ctx context.Context) error {
	err := node.Init()
	if err != nil {
		return err
	}
	node.StartRoutine(ctx)
	return nil
}

func (node *RiderNode) OnStop() {
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

func (node *RiderNode) Init() error {
	round0 := commons.Round(0)
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
	return nil
}

func (node *RiderNode) SetPeers(peers []*RiderNode) {
	node.peers = peers
	node.f = len(node.peers) / 3
}

func (node *RiderNode) StartRoutine(ctx context.Context) {
	go node.ReportRoutine(ctx, 10*time.Second)
	go node.ReceiveRoutine(ctx)
	node.timer.Reset(node.timerTimeout)
	go node.TimeoutRoutine(ctx)
}

func (node *RiderNode) ReportRoutine(ctx context.Context, interval time.Duration) {
	timer := time.NewTimer(interval)
	for {
		select {
		case <-timer.C:
			fmt.Println("REPORT", fmt.Sprintf("round=%d\n%s", node.round, node.dag.String()))
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

func (node *RiderNode) ReceiveRoutine(ctx context.Context) {
	for {
		select {
		case rMsg := <-node.RBcastChannel:
			node.log.Debug("receive message from RBcastChannel", "p=", rMsg.P, "r=", rMsg.R, "m=", rMsg.Message)
			node.rDeliver(rMsg.Message, rMsg.R, rMsg.P)
		case aMsg := <-node.ABcastChannel:
			node.log.Debug("receive message from ABcastChannel", "p=", aMsg.P, "r=", aMsg.R, "m=", aMsg.Message)
			node.aBcast(aMsg.Message, aMsg.R)
		case <-ctx.Done():
			return
		default:
			continue
		}
	}
}

func (node *RiderNode) TimeoutRoutine(ctx context.Context) {
	for {
		select {
		case <-node.timer.C:
			node.log.Debug("timer timeout")
			node.handleTimeout()
		case <-ctx.Done():
			return
		default:
			continue
		}
	}
}

func (node *RiderNode) createNewVertex(round commons.Round) *commons.Vertex {
	select {
	case <-time.After(100 * time.Millisecond):
		break
	case block := <-node.BlockToPropose:
		v := &commons.Vertex{}
		v.Source = node.nodeInfo.Address
		v.Round = round
		v.Block = block
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
		node.log.Debug("vertex created", v)
		return v
	}
	return nil
}

func (node *RiderNode) setWeakEdges(v *commons.Vertex, round commons.Round) {
	v.WeakEdges = make([]commons.BaseVertex, 0)
	for r := round - 2; r >= 1; r-- {
		roundSet := node.dag.GetRound(r)
		if roundSet != nil {
			for _, u := range roundSet.Entries() {
				for _, vs := range v.StrongEdgesValues() {
					vss := node.dag.GetRound(vs.Round).GetBySource(vs.Source)
					if !node.dag.Path(&vss, &u) {
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
}

func (node *RiderNode) getVertex(p commons.Address, r commons.Round) *commons.Vertex {
	if roundSet := node.dag.GetRound(r); roundSet != nil {
		if roundSet.SourceExists(p) {
			ret := roundSet.GetBySource(p)
			return &ret
		}
	}
	return nil
}

func (node *RiderNode) rDeliver(v *commons.Vertex, r commons.Round, p commons.Address) {
	if len(v.StrongEdges) >= 2*node.f+1 && v.StrongEdgesContainsSource(v.Source) {
		must(node.buffer.Add(v))
		node.log.Debug("buffer", node.buffer.Entries())
	}
}

func (node *RiderNode) handleTimeout() {
	node.log.Debug("buffer", node.buffer.Entries(), "round", node.round, "node.dag.GetRound(node.round).Size()", node.dag.GetRound(node.round).Size())
	for _, v := range node.buffer.Entries() {
		if v.Round <= node.round && node.dag.AllEdgesExist(v) {
			node.dag.NewRoundIfNotExists(v.Round)
			node.dag.GetRound(v.Round).AddVertex(*v)
			must(node.buffer.Remove(v))
			node.log.Debug("vertex added to DAG", v)
		}
	}
	if node.dag.GetRound(node.round).Size() >= 2*node.f+1 {
		if int64(node.round)%int64(node.w) == 0 {
			w := commons.Wave(int64(node.round) / int64(node.w))
			node.log.Debug("wave ready", w)
			node.waveReady(w)
		}
		node.dag.NewRoundIfNotExists(node.round + 1)
		v := node.createNewVertex(node.round + 1)
		if v != nil {
			node.rBcast(v, node.round+1)
			node.round = node.round + 1
		}
	}
	node.timer.Reset(node.timerTimeout)
}

func (node *RiderNode) rBcast(v *commons.Vertex, r commons.Round) {
	for _, peer := range node.peers {
		clonedPeer := peer
		go func() {
			node.log.Debug("message rBcast to", clonedPeer.nodeInfo.Address, "v=", v, "r=", r)
			clonedPeer.RBcastChannel <- internal.BroadcastMessage[*commons.Vertex, commons.Round]{
				Message: v,
				R:       r,
				P:       node.nodeInfo.Address,
			}
		}()
	}

}

func (node *RiderNode) aBcast(b commons.Block, r uint64) {
	node.BlockToPropose <- b
}

func (node *RiderNode) wRound(w commons.Wave, k int) commons.Round {
	return commons.Round(int64(node.w)*(int64(w-1)) + int64(k))
}

func (node *RiderNode) getWaveLeader(w commons.Wave) commons.Address {
	p := node.peers[int(w)%len(node.peers)].nodeInfo.Address // TODO update later
	return p
}

func (node *RiderNode) getWaveVertexLeader(w commons.Wave) *commons.Vertex {
	p := node.getWaveLeader(w)
	return node.getVertex(p, commons.Round(node.wRound(w, 1)))
}

func (node *RiderNode) waveReady(w commons.Wave) {
	v := node.getWaveVertexLeader(w)
	if v == nil {
		return
	}
	node.dag.NewRoundIfNotExists(node.wRound(w, node.w))
	count := 0
	for _, vp := range node.dag.GetRound(node.wRound(w, node.w)).Entries() {
		if node.dag.StrongPath(&vp, v) {
			count++
		}
	}
	if count < 2*node.f+1 {
		return
	}
	node.leaderStack.Push(v)
	for wp := w - 1; wp >= node.decidedWave+1; wp-- {
		vp := node.getWaveVertexLeader(wp)
		if vp != nil && node.dag.StrongPath(v, vp) {
			node.leaderStack.Push(vp)
			v = vp
		}
	}
	node.decidedWave = w
	node.orderVertices()
}

func (node *RiderNode) orderVertices() {
	for node.leaderStack.Size() > 0 {
		v := node.leaderStack.Pop()
		verticesToDeliver := make([]commons.Vertex, 0)
		for r := commons.Round(1); r < node.round; r++ {
			for _, vp := range node.dag.GetRound(r).Entries() {
				if node.dag.Path(v, &vp) && !vp.Delivered {
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
			node.log.Info("deliver vertex", vp.BaseVertex)
		}
	}
}
