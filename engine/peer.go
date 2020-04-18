package engine

import (
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/MixinNetwork/mixin/logger"
	"github.com/gofrs/uuid"
	"github.com/pion/webrtc/v2"
)

type Peer struct {
	sync.Mutex
	rid     string
	uid     string
	cid     string
	pc      *webrtc.PeerConnection
	track   *webrtc.Track
	senders map[string]*webrtc.RTPSender
	buffer  chan []byte
}

func (engine *Engine) BuildPeer(rid, uid string, pc *webrtc.PeerConnection) *Peer {
	cid, err := uuid.NewV4()
	if err != nil {
		panic(err)
	}
	peer := &Peer{rid: rid, uid: uid, cid: cid.String(), pc: pc}
	peer.buffer = make(chan []byte, 1024)
	peer.senders = make(map[string]*webrtc.RTPSender)
	peer.handle()
	return peer
}

func (p *Peer) id() string {
	return fmt.Sprintf("%s:%s:%s", p.rid, p.uid, p.cid)
}

func (p *Peer) Close() error {
	logger.Printf("PeerClose(%s) now\n", p.id())
	err := p.pc.Close()
	logger.Printf("PeerClose(%s) with %v\n", p.id(), err)
	return err
}

func (peer *Peer) handle() {
	peer.pc.OnSignalingStateChange(func(state webrtc.SignalingState) {
		logger.Printf("HandlePeer(%s) OnSignalingStateChange(%s)\n", peer.id(), state)
	})
	peer.pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		logger.Printf("HandlePeer(%s) OnConnectionStateChange(%s)\n", peer.id(), state)
	})
	peer.pc.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		logger.Printf("HandlePeer(%s) OnICEConnectionStateChange(%s)\n", peer.id(), state)
	})
	peer.pc.OnTrack(func(rt *webrtc.Track, receiver *webrtc.RTPReceiver) {
		logger.Printf("HandlePeer(%s) OnTrack(%d, %d)\n", peer.id(), rt.PayloadType(), rt.SSRC())
		if webrtc.DefaultPayloadTypeOpus != rt.PayloadType() {
			return
		}

		peer.Lock()
		lt, err := peer.pc.NewTrack(rt.PayloadType(), rt.SSRC(), peer.cid, peer.uid)
		if err != nil {
			panic(err)
		}
		peer.track = lt
		peer.Unlock()

		err = peer.copyTrack(rt, lt)
		logger.Printf("HandlePeer(%s) OnTrack(%d, %d) end with %s\n", peer.id(), rt.PayloadType(), rt.SSRC(), err.Error())
		peer.pc.Close()
		peer.track = nil
	})
}

func (peer *Peer) copyTrack(src, dst *webrtc.Track) error {
	go func() error {
		for {
			buf := make([]byte, 256)
			i, err := src.Read(buf)
			if err == io.EOF {
				return nil
			}
			if err != nil {
				return err
			}
			if i >= len(buf) {
				return fmt.Errorf("track packet size too large %d", i)
			}
			peer.buffer <- buf[:i]
		}
	}()

	for {
		timer := time.NewTimer(3 * time.Second)
		select {
		case buf := <-peer.buffer:
			dst.Write(buf)
		case <-timer.C:
			return fmt.Errorf("peer track read timeout")
		}
		timer.Stop()
	}
}
