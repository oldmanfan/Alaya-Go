package cbft

import (
	"github.com/AlayaNetwork/Alaya-Go/common/hexutil"
	"github.com/AlayaNetwork/Alaya-Go/consensus/cbft/protocols"
	"reflect"
	"sync"
	"time"
)

const (
	coordinatorWaitTimeout    = 200 * time.Millisecond
	efficientCoordinatorRatio = 15 // TODO
	defaultUnitID             = 0
)

type awaiting interface {
	GroupID() uint32
	Index() uint64
	Epoch() uint64
	ViewNumber() uint64
}

type awaitingRGBlockQC struct {
	groupID    uint32
	blockIndex uint32
	epoch      uint64
	viewNumber uint64
}

func (a *awaitingRGBlockQC) GroupID() uint32 {
	return a.groupID
}

func (a *awaitingRGBlockQC) Index() uint64 {
	return uint64(a.blockIndex)
}

func (a *awaitingRGBlockQC) Epoch() uint64 {
	return a.epoch
}

func (a *awaitingRGBlockQC) ViewNumber() uint64 {
	return a.viewNumber
}

type awaitingRGViewQC struct {
	groupID    uint32
	epoch      uint64
	viewNumber uint64
}

func (a *awaitingRGViewQC) GroupID() uint32 {
	return a.groupID
}

func (a *awaitingRGViewQC) Index() uint64 {
	return a.viewNumber
}

func (a *awaitingRGViewQC) Epoch() uint64 {
	return a.epoch
}

func (a *awaitingRGViewQC) ViewNumber() uint64 {
	return a.viewNumber
}

type awaitingJob struct {
	jobTimer *time.Timer
	awaiting awaiting
}

type RGBroadcastManager struct {
	cbft *Cbft

	delayDuration time.Duration

	// A collection of RGBlockQuorumCert messages waiting to be sent
	awaitingRGBlockQuorumCerts map[uint64]*awaitingJob

	// A collection of RGBlockQuorumCert messages that have been sent
	hadSendRGBlockQuorumCerts map[uint64]*protocols.RGBlockQuorumCert

	// A collection of RGViewChangeQuorumCert messages waiting to be sent
	awaitingRGViewChangeQuorumCerts map[uint64]*awaitingJob

	// A collection of RGViewChangeQuorumCert messages that have been sent
	hadSendRGViewChangeQuorumCerts map[uint64]*protocols.RGViewChangeQuorumCert

	broadcastCh chan awaiting

	// Termination channel to stop the broadcaster
	term chan struct{}

	// global mutex for RGBroadcast operations
	mux sync.Mutex
}

// NewBridge creates a new Bridge to update consensus state and consensus msg.
func NewRGBroadcastManager(cbft *Cbft) *RGBroadcastManager {
	//_, unitID, err := cbft.getGroupByValidatorID(cbft.state.Epoch(), cbft.Node().ID())
	//if err != nil {
	//	cbft.log.Trace("The current node is not a consensus node, no need to start RGBroadcastManager", "epoch", cbft.state.Epoch(), "nodeID", cbft.Node().ID().String())
	//	unitID = 0
	//}
	m := &RGBroadcastManager{
		cbft:                            cbft,
		delayDuration:                   time.Duration(defaultUnitID) * coordinatorWaitTimeout,
		awaitingRGBlockQuorumCerts:      make(map[uint64]*awaitingJob),
		hadSendRGBlockQuorumCerts:       make(map[uint64]*protocols.RGBlockQuorumCert),
		awaitingRGViewChangeQuorumCerts: make(map[uint64]*awaitingJob),
		hadSendRGViewChangeQuorumCerts:  make(map[uint64]*protocols.RGViewChangeQuorumCert),
		broadcastCh:                     make(chan awaiting, 20),
		term:                            make(chan struct{}),
	}
	go m.broadcastLoop()
	return m
}

func (m *RGBroadcastManager) broadcastLoop() {
	for {
		select {
		case a := <-m.broadcastCh:
			m.broadcast(a)

		case <-m.term:
			return
		}
	}
}

func (m *RGBroadcastManager) hadBroadcastRGBlockQuorumCert(blockIndex uint64) bool {
	if _, ok := m.hadSendRGBlockQuorumCerts[blockIndex]; ok {
		return true
	}
	return false
}

func (m *RGBroadcastManager) awaitingBroadcastRGBlockQuorumCert(blockIndex uint64) bool {
	if _, ok := m.awaitingRGBlockQuorumCerts[blockIndex]; ok {
		return true
	}
	return false
}

func (m *RGBroadcastManager) hadBroadcastRGViewChangeQuorumCert(viewNumber uint64) bool {
	if _, ok := m.hadSendRGViewChangeQuorumCerts[viewNumber]; ok {
		return true
	}
	return false
}

func (m *RGBroadcastManager) awaitingBroadcastRGViewChangeQuorumCert(viewNumber uint64) bool {
	if _, ok := m.awaitingRGViewChangeQuorumCerts[viewNumber]; ok {
		return true
	}
	return false
}

// equalsState checks if the message is currently CBFT status
func (m *RGBroadcastManager) equalsState(a awaiting) bool {
	return a.Epoch() == m.cbft.state.Epoch() && a.ViewNumber() == m.cbft.state.ViewNumber()
}

// needBroadcast to check whether the message has been sent or is being sent
func (m *RGBroadcastManager) needBroadcast(a awaiting) bool {
	switch msg := a.(type) {
	case *awaitingRGBlockQC:
		return !m.hadBroadcastRGBlockQuorumCert(msg.Index()) && !m.awaitingBroadcastRGBlockQuorumCert(msg.Index())
	case *awaitingRGViewQC:
		return !m.hadBroadcastRGViewChangeQuorumCert(msg.Index()) && !m.awaitingBroadcastRGViewChangeQuorumCert(msg.Index())
	default:
		return false
	}
}

func (m *RGBroadcastManager) broadcast(a awaiting) {
	m.mux.Lock()
	defer m.mux.Unlock()

	if !m.equalsState(a) || !m.needBroadcast(a) {
		return
	}
	m.cbft.log.Debug("Begin broadcast rg msg", "type", reflect.TypeOf(a), "groupID", a.GroupID(), "index", a.Index(), "delayDuration", m.delayDuration.String())

	timer := time.AfterFunc(m.delayDuration, func() {
		m.cbft.asyncCallCh <- func() {
			m.broadcastFunc(a)
		}
	})
	switch msg := a.(type) {
	case *awaitingRGBlockQC:
		m.awaitingRGBlockQuorumCerts[msg.Index()] = &awaitingJob{
			jobTimer: timer,
			awaiting: a,
		}
	case *awaitingRGViewQC:
		m.awaitingRGViewChangeQuorumCerts[msg.Index()] = &awaitingJob{
			jobTimer: timer,
			awaiting: a,
		}
	default:
		m.cbft.log.Error("Unsupported message type")
		return
	}
}

func (m *RGBroadcastManager) allowRGBlockQuorumCert(a awaiting) bool {
	switch a.(type) {
	case *awaitingRGBlockQC:
		if m.cbft.state.IsDeadline() {
			m.cbft.log.Debug("Current view had timeout, refuse to send RGBlockQuorumCert")
			return false
		}
	case *awaitingRGViewQC:
		return true
	}
	return true
}

func (m *RGBroadcastManager) upgradeCoordinator(a awaiting) bool {
	// Check whether the current node is the first group member
	groupID, unitID, err := m.cbft.getGroupByValidatorID(m.cbft.state.Epoch(), m.cbft.Node().ID())
	if err != nil || groupID != a.GroupID() {
		return false
	}
	if unitID == uint32(0) { // the first echelon,Send by default
		return true
	}

	coordinatorIndexes, err := m.cbft.validatorPool.GetCoordinatorIndexesByGroupID(m.cbft.state.Epoch(), groupID)
	if err != nil || len(coordinatorIndexes) <= 0 {
		m.cbft.log.Error("Get coordinator indexes by groupID error")
		return false
	}

	var receiveIndexes []uint32

	switch msg := a.(type) {
	case *awaitingRGBlockQC:
		receiveIndexes = m.cbft.state.RGBlockQuorumCertsIndexes(msg.blockIndex, groupID)
	case *awaitingRGViewQC:
		receiveIndexes = m.cbft.state.RGViewChangeQuorumCertsIndexes(groupID)
	default:
		return false
	}
	if !m.enoughCoordinator(groupID, unitID, coordinatorIndexes, receiveIndexes) {
		m.cbft.log.Warn("Upgrade the current node to Coordinator", "type", reflect.TypeOf(a), "groupID", groupID, "index", a.Index(), "unitID", unitID, "coordinatorIndexes", coordinatorIndexes, "receiveIndexes", receiveIndexes)
		m.recordMetrics(a)
		return true
	}
	m.cbft.log.Debug("Enough Coordinator, no need to upgrade to Coordinator", "groupID", groupID, "unitID", unitID, "coordinatorIndexes", coordinatorIndexes, "receiveIndexes", receiveIndexes)
	return false
}

func (m *RGBroadcastManager) recordMetrics(a awaiting) {
	switch a.(type) {
	case *awaitingRGBlockQC:
		upgradeCoordinatorBlockCounter.Inc(1)
	case *awaitingRGViewQC:
		upgradeCoordinatorViewCounter.Inc(1)
	default:
	}
}

func (m *RGBroadcastManager) enoughCoordinator(groupID, unitID uint32, coordinatorIndexes [][]uint32, receiveIndexes []uint32) bool {
	enough := func() int {
		// The total number of validators in the current group
		total := m.cbft.groupLen(m.cbft.state.Epoch(), groupID)
		return total * efficientCoordinatorRatio / 100
	}()

	return m.countCoordinator(unitID, coordinatorIndexes, receiveIndexes) >= enough
}

func (m *RGBroadcastManager) countCoordinator(unitID uint32, coordinatorIndexes [][]uint32, receiveIndexes []uint32) int {
	receiveIndexesMap := make(map[uint32]struct{})
	for i := 0; i < len(receiveIndexes); i++ {
		receiveIndexesMap[receiveIndexes[i]] = struct{}{}
	}

	c := 0
	for i := 0; i < len(coordinatorIndexes); i++ {
		if uint32(i) >= unitID {
			break
		}
		for _, v := range coordinatorIndexes[i] {
			if _, ok := receiveIndexesMap[v]; ok {
				c++
			}
		}
	}
	return c
}

func (m *RGBroadcastManager) broadcastFunc(a awaiting) {
	m.cbft.log.Debug("Broadcast rg msg", "type", reflect.TypeOf(a), "groupID", a.GroupID(), "index", a.Index())
	if !m.equalsState(a) || !m.allowRGBlockQuorumCert(a) {
		return
	}

	if !m.upgradeCoordinator(a) {
		return
	}

	node, err := m.cbft.isCurrentValidator()
	if err != nil || node == nil {
		m.cbft.log.Debug("Current node is not validator, no need to send RGQuorumCert")
		return
	}

	m.mux.Lock()
	defer m.mux.Unlock()

	switch msg := a.(type) {
	case *awaitingRGBlockQC:
		// Query the QuorumCert with the largest number of signatures in the current group
		blockQC, parentQC := m.cbft.state.FindMaxGroupRGQuorumCert(msg.blockIndex, msg.GroupID())
		if blockQC == nil {
			m.cbft.log.Error("Cannot find the RGBlockQuorumCert of the current group", "blockIndex", msg.blockIndex, "groupID", msg.GroupID())
			return
		}
		if blockQC.BlockNumber != 1 && parentQC == nil {
			m.cbft.log.Error("Cannot find the ParentQC corresponding to the current blockQC", "blockIndex", msg.blockIndex, "blockNumber", blockQC.BlockNumber, "groupID", msg.GroupID())
			return
		}
		rg := &protocols.RGBlockQuorumCert{
			GroupID:        msg.groupID,
			BlockQC:        blockQC,
			ValidatorIndex: node.Index,
			ParentQC:       parentQC,
		}
		if err := m.cbft.signMsgByBls(rg); err != nil {
			m.cbft.log.Error("Sign RGBlockQuorumCert failed", "err", err, "rgmsg", rg.String())
			return
		}
		m.cbft.network.Broadcast(rg)
		// todo just for log
		digest, _ := rg.CannibalizeBytes()
		pubKey := m.cbft.config.Option.BlsPriKey.GetPublicKey()
		blsPublicKey := hexutil.Encode(pubKey.Serialize())
		m.cbft.log.Debug("Success to broadcast RGBlockQuorumCert", "msg", rg.String(), "data", hexutil.Encode(digest), "sign", hexutil.Encode(rg.Sign()), "pubkey", blsPublicKey)
		m.hadSendRGBlockQuorumCerts[msg.Index()] = rg
		delete(m.awaitingRGBlockQuorumCerts, msg.Index())
		m.cbft.state.AddRGBlockQuorumCert(node.Index, rg)
	case *awaitingRGViewQC:
		viewChangeQC, prepareQCs := m.cbft.state.FindMaxGroupRGViewChangeQuorumCert(msg.GroupID())
		if viewChangeQC == nil {
			m.cbft.log.Error("Cannot find the RGViewChangeQuorumCert of the current group", "groupID", msg.GroupID())
			return
		}
		rg := &protocols.RGViewChangeQuorumCert{
			GroupID:        msg.groupID,
			ViewChangeQC:   viewChangeQC,
			ValidatorIndex: node.Index,
			PrepareQCs:     prepareQCs,
		}
		if err := m.cbft.signMsgByBls(rg); err != nil {
			m.cbft.log.Error("Sign RGViewChangeQuorumCert failed", "err", err, "rgmsg", rg.String())
			return
		}
		m.cbft.network.Broadcast(rg)
		m.cbft.log.Debug("Success to broadcast RGViewChangeQuorumCert", "msg", rg.String())
		m.hadSendRGViewChangeQuorumCerts[msg.Index()] = rg
		delete(m.awaitingRGViewChangeQuorumCerts, msg.Index())
		m.cbft.state.AddRGViewChangeQuorumCert(node.Index, rg)
	}
}

// AsyncSendRGQuorumCert queues list of RGQuorumCert propagation to a remote peer.
// Before calling this function, it will be judged whether the current node is validator
func (m *RGBroadcastManager) AsyncSendRGQuorumCert(a awaiting) {
	select {
	case m.broadcastCh <- a:
		m.cbft.log.Debug("Async send RGQuorumCert", "groupID", a.GroupID(), "index", a.Index(), "type", reflect.TypeOf(a))
	case <-m.term:
		m.cbft.log.Debug("Dropping RGQuorumCert propagation")
	}
}

func (m *RGBroadcastManager) Reset() {
	m.mux.Lock()
	defer m.mux.Unlock()

	for _, await := range m.awaitingRGBlockQuorumCerts {
		await.jobTimer.Stop() // Some JobTimers are already running and may fail to stop
	}
	for _, await := range m.awaitingRGViewChangeQuorumCerts {
		await.jobTimer.Stop() // Some JobTimers are already running and may fail to stop
	}
	_, unitID, err := m.cbft.getGroupByValidatorID(m.cbft.state.Epoch(), m.cbft.Node().ID())
	m.cbft.log.Debug("RGBroadcastManager Reset", "unitID", unitID)
	if err != nil {
		m.cbft.log.Trace("The current node is not a consensus node, no need to start RGBroadcastManager", "epoch", m.cbft.state.Epoch(), "nodeID", m.cbft.Node().ID().String())
		unitID = defaultUnitID
	}
	m.delayDuration = time.Duration(unitID) * coordinatorWaitTimeout
	m.awaitingRGBlockQuorumCerts = make(map[uint64]*awaitingJob)
	m.hadSendRGBlockQuorumCerts = make(map[uint64]*protocols.RGBlockQuorumCert)
	m.awaitingRGViewChangeQuorumCerts = make(map[uint64]*awaitingJob)
	m.hadSendRGViewChangeQuorumCerts = make(map[uint64]*protocols.RGViewChangeQuorumCert)
}

// close signals the broadcast goroutine to terminate.
func (m *RGBroadcastManager) Close() {
	close(m.term)
}