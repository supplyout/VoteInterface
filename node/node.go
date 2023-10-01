package node

import (
	. "Vote/log"
	. "Vote/message"
	"bufio"
	"context"
	"crypto/rand"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	discoptions "github.com/libp2p/go-libp2p-core/discovery"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/routing"
	discovery "github.com/libp2p/go-libp2p-discovery"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	ddht "github.com/libp2p/go-libp2p-kad-dht/dual"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	swarm "github.com/libp2p/go-libp2p-swarm"
	"github.com/libp2p/go-libp2p/p2p/protocol/ping"
	ma "github.com/multiformats/go-multiaddr"
	"os"
	"sort"
	"sync"
	"time"
)

const (
	RootTopic      = "P2P_this_is_my_root_topic"
	RootRendezvous = "P2P_this_is_my_root_rendezvous"
	DhtPrefix      = "/this_is_my_dht_prefix"
	ProtocolID     = "/this_is_my_protocol_id"
)

type NodeStat int

const (
	NodeStatInited NodeStat = iota
	NodeStatGotNewVote
	NodeStatConfirmed
	NodeStatGotParams
	NodeStatSentPki
	NodeStatGotPK
	NodeStatGotDetail
	NodeStatVoted
	NodeStatSentCts
	NodeStatSentResult
	NodeStatSentID
)

var (
	defaultBootstraps, _ = peer.AddrInfosFromP2pAddrs(dht.DefaultBootstrapPeers...) //公网启动节点

	relayNodeID1       = "QmYCWGYGhTfuK89u2nEZk73WLBxbvpHJ1bQpty14DrupR4"
	relayNodeID2       = "QmW3F1jgJMmo5TBd2hip1M6vLN9uwADcphgpy87p8Chww4"
	txyunAddr1, _      = ma.NewMultiaddr("/ip4/152.136.116.202/tcp/8000/p2p/" + relayNodeID1)
	txyunAddr2, _      = ma.NewMultiaddr("/ip4/152.136.116.202/tcp/8001/p2p/" + relayNodeID2)
	txyunBootstraps, _ = peer.AddrInfosFromP2pAddrs(txyunAddr1, txyunAddr2)
	localBootstrapInfo *peer.AddrInfo
)

type ChanType int

const (
	Read = iota
	Write
)

type Node struct {
	ctx context.Context

	host host.Host
	dual *ddht.DHT
	disc *discovery.RoutingDiscovery

	gossip *pubsub.PubSub
	topics []*pubsub.Topic
	subs   []*pubsub.Subscription

	pingService *ping.PingService

	// 保存当前网络中所有节点
	allPeers []peer.AddrInfo

	allPeersCh chan []string
	msgCh      map[ChanType]chan Message
	optCh      map[ChanType]chan OptRequest

	gotMsgFromNet chan []byte

	Nickname string // 用户的昵称

	findPeersTicker *time.Ticker
}

// Node 的构造工厂方法, Node 创建之后连接到bootstrap节点
func NewNode(ctx context.Context, nickname string, isLocal bool,
	msgCh map[ChanType]chan Message, optCh map[ChanType]chan OptRequest, bootInfo *peer.AddrInfo) *Node {
	//log.SetLogLevel("dht","debug")
	//log.SetLogLevel("relay","debug")
	//log.SetLogLevel("autorelay","debug")

	localBootstrapInfo = bootInfo

	var err error
	node := &Node{
		ctx:             ctx,
		Nickname:        nickname,
		msgCh:           map[ChanType]chan Message{Read: msgCh[Write], Write: msgCh[Read]}, // node与ui的读写刚好反过来
		optCh:           map[ChanType]chan OptRequest{Read: optCh[Write], Write: optCh[Read]},
		gotMsgFromNet:   make(chan []byte),
		findPeersTicker: time.NewTicker(time.Second * 5),
	}

	sk, _, _ := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, rand.Reader)
	identity := libp2p.Identity(sk)

	routing := libp2p.Routing(func(h host.Host) (routing.PeerRouting, error) {
		if isLocal {
			node.dual, err = ddht.New(ctx, h, ddht.DHTOption(dht.ProtocolPrefix(DhtPrefix), dht.Mode(dht.ModeServer), dht.Resiliency(1), dht.MaxRecordAge(time.Minute*8)))
			ErrPanic(err, Logger)
		} else {
			node.dual, err = ddht.New(ctx, h, ddht.DHTOption(dht.ProtocolPrefix(DhtPrefix), dht.Resiliency(1), dht.MaxRecordAge(time.Minute*8)))
			ErrPanic(err, Logger)
		}
		node.disc = discovery.NewRoutingDiscovery(node.dual)
		return node.dual, err
	})

	//if isLocal{
	//	node.host, err = libp2p.New(ctx,
	//		identity,
	//		routing,
	//		libp2p.EnableRelay(relay.OptHop),
	//		libp2p.EnableAutoRelay(),
	//		libp2p.NATPortMap(),
	//	)
	//	ErrPanic(err, Logger)
	//}else {
	//
	//	ErrPanic(err, Logger)
	//}
	node.host, err = libp2p.New(ctx,
		identity,
		routing,
		libp2p.EnableRelay(),
		libp2p.EnableAutoRelay(),
		libp2p.StaticRelays(txyunBootstraps),
		libp2p.ForceReachabilityPrivate(),
		libp2p.NATPortMap(),
	)

	node.host.SetStreamHandler(ProtocolID, node.handleStream)
	err = node.dual.Bootstrap(ctx)
	ErrPanic(err, Logger)

	// 连接到bootstrap节点
	node.bootstrap(ctx, isLocal)

	// 约定集合点 RootRendezvous
	discovery.Advertise(ctx, node.disc, RootRendezvous, discoptions.TTL(time.Minute*5))
	// 新建gossip
	node.gossip, err = pubsub.NewGossipSub(ctx, node.host)
	ErrPanic(err, Logger)

	//加入 rootTopic
	topic, err := node.gossip.Join(RootTopic)
	node.topics = append(node.topics, topic)
	ErrPanic(err, Logger)
	// 订阅
	sub, err := node.topics[0].Subscribe()
	node.subs = append(node.subs, sub)
	ErrPanic(err, Logger)
	// 开始接收网络中的信息
	go node.readMsgFromTopic(ctx, sub)
	ErrPanic(err, Logger)
	return node
}

// 初始化网络，找到所有加入 RootRendezvous 的节点，再创建 gossipsub 网络，并每隔10s更新allPeers
func (node *Node) InitNet() {
	peers, _ := discovery.FindPeers(node.ctx, node.disc, RootRendezvous)
	sort.SliceStable(peers, func(i, j int) bool {
		return string(peers[i].ID) < string(peers[j].ID)
	})
	Logger.Infof("初始化时找到的peers：%v", peers)
	node.connectToPeers(node.ctx, peers)
	node.updateAllPeers(peers)

	//peerNm := len(node.allPeers)
	//partPeers := peers
	//
	//if peerNm >= 6 {
	//	// 从allPeers中随机选出 peerNm/3 个节点进行连接
	//	//TODO
	//	indexes := mrand.Perm(peerNm)[0 : peerNm/3]
	//	var partPeers []peer.AddrInfo
	//	for _, index := range indexes {
	//		partPeers = append(partPeers, node.allPeers[index])
	//	}
	//}

	go node.startLoop(node.ctx)

}

func (node *Node) Join(ctx context.Context, voteTopic string) {
	topic, err := node.gossip.Join(voteTopic)
	ErrPanic(err, Logger)
	sub, err := topic.Subscribe()
	node.topics = append(node.topics, topic)
	node.subs = append(node.subs, sub)
	go node.readMsgFromTopic(ctx, sub)
}

func (node *Node) Advertise(ctx context.Context, rendezvous string) {
	discovery.Advertise(ctx, node.disc, rendezvous, discoptions.TTL(time.Minute*20))
}

func (node *Node) FindPeers(ctx context.Context, rendezvous string) peer.IDSlice {
	peerAddrs, err := discovery.FindPeers(ctx, node.disc, rendezvous)
	ErrPanic(err, Logger)
	slice := AddrInfo2IDSlice(peerAddrs)
	sort.Stable(slice)
	return slice
}

func (node *Node) StopRefreshPeers(ctx context.Context) {
	node.findPeersTicker.Stop()
	Logger.Infof("已停止刷新allPeers")
}
func (node *Node) RestartRefreshPeers(ctx context.Context) {
	node.findPeersTicker.Reset(time.Second * 5)
	Logger.Infof("已重新开始刷新allPeers")
}

func (node *Node) GetNodeID() string {
	return node.host.ID().Pretty()
}

func AddrInfo2IDSlice(addrInfos []peer.AddrInfo) peer.IDSlice {
	var slice peer.IDSlice
	for _, addr := range addrInfos {
		slice = append(slice, addr.ID)
	}
	return slice
}

func (node *Node) startLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Second * 25)
	for {
		select {
		case <-ctx.Done():
			Logger.Infof("node循环已停止...")
			return
		case <-node.findPeersTicker.C:
			go node.refreshAllPeers(ctx)

		case msg := <-node.msgCh[Read]:
			go node.handleMessage(ctx, msg)

		case req := <-node.optCh[Read]:
			go node.handleOptReq(ctx, req)

		case msg := <-node.gotMsgFromNet:
			// 从网络中收到“发起投票”消息,则暂停更新allPeers;如果整个投票结束,则重新开始更新allPeers
			// 直接对从网络中收到的消息进行转发
			m := ParseMsg(msg)
			node.msgCh[Write] <- m
		case <-ticker.C:
			// 定时约定集合点 RootRendezvous
			discovery.Advertise(ctx, node.disc, RootRendezvous, discoptions.TTL(time.Minute*5))
			//tmp,err:=node.anat.PublicAddr()
			//if err != nil {
			//	Logger.Infof("\nNat：状态 %s; 公共地址 %s",node.anat.Status(),err.Error())
			//}else{
			//	Logger.Infof("\nNat：状态 %s; 公共地址 %v",node.anat.Status(),tmp)
			//}
			Logger.Infof("本地ID%s:", node.host.ID().ShortString())
			//Logger.Infof("\n可以通过以下地址进行连接:")
			//for _, addr := range node.host.Addrs() {
			//	Logger.Infof("%s", addr)
			//}
			Logger.Infof("Mode：Wan %s; Lan %s", node.dual.WAN.Mode(), node.dual.LAN.Mode())
			Logger.Infof("************network数据********************************************************")
			ps := node.host.Network().Peers()
			for _, p := range ps {
				if p == node.host.ID() {
					continue
				}
				Logger.Infof("与%s连接状态：%s", p.ShortString(), node.host.Network().Connectedness(p))
			}

			//Logger.Infof("************peerstores数据********************************************************")
			//peers:=node.host.Peerstore().Peers()
			//for _,p:= range peers{
			//	if p == node.host.ID(){
			//		continue
			//	}
			//	addr:=node.host.Peerstore().Addrs(p)
			//	Logger.Infof("%s的地址:%v",p.ShortString(),addr)
			//	Logger.Infof("与%s连接状态：%s",p.ShortString(),node.host.Network().Connectedness(p))
			//}

			Logger.Infof("************LAN********************************************************")
			node.dual.LAN.RoutingTable().Print()

			Logger.Infof("\n************WAN********************************************************")
			Logger.Infof("WAN状态：%s", node.dual.WANActive())
			node.dual.WAN.RoutingTable().Print()
			Logger.Infof("***********************************************************************")

		}
	}
}

// 监听gossip网络中的消息
func (node *Node) readMsgFromTopic(ctx context.Context, sub *pubsub.Subscription) {
	Logger.Infof("已开始接收来自Topic:%s的信息", sub.Topic())
	for {
		m, err := sub.Next(ctx)
		ErrPanic(err, Logger)
		node.gotMsgFromNet <- m.Data
	}
}

// 监听点对点的消息
func (node *Node) handleStream(s network.Stream) {
	reader := bufio.NewReader(s)
	data, err := reader.ReadBytes('}')
	Logger.Infof("收到点对点消息：%s", data[0:108])
	ErrPanic(err, Logger)
	node.gotMsgFromNet <- data

}

// 连接到bootstrap节点
func (node *Node) bootstrap(ctx context.Context, isLocal bool) {
	var success int
	if isLocal {
		// TODO: 需要填写一个固定的bootstrap节点
		success = node.connectToPeers(ctx, []peer.AddrInfo{*localBootstrapInfo})
	} else {
		success = node.connectToPeers(ctx, txyunBootstraps)
		ps := node.host.Network().Peers()
		for _, p := range ps {
			if p == node.host.ID() {
				continue
			}
			if node.host.Network().Connectedness(p) == network.Connected {
				if _, err := node.dual.WAN.RoutingTable().TryAddPeer(p, false, false); err != nil {
					Logger.Infof("添加失败")
				}
			}

		}
	}

	if success > 0 {
		Logger.Infof("成功连接%d个bootstrap节点", success)
	} else {
		Logger.Infof("bootstrap节点连接失败")
		os.Exit(1)
	}
}

// 更新allPeers,如果新找到的peers无变化则不更新，否则更新
func (node *Node) refreshAllPeers(ctx context.Context) {
	peers, _ := discovery.FindPeers(node.ctx, node.disc, RootRendezvous)
	sort.SliceStable(peers, func(i, j int) bool {
		return string(peers[i].ID) < string(peers[j].ID)
	})
	Logger.Infof("新peers:\n%v", AddrInfo2IDSlice(peers))
	node.connectToPeers(ctx, peers)
	node.updateAllPeers(peers)
	//isEqual := true
	//if len(peers) == len(node.allPeers) {
	//	for i := range peers {
	//		if peers[i].ID != node.allPeers[i].ID {
	//			isEqual = false
	//		}
	//	}
	//} else {
	//	isEqual = false
	//}
	//if isEqual {
	//	//Logger.Infof("allPeers无需更新")
	//	return
	//} else {
	//	Logger.Infof("新找到的peers与原来的allPeers不相同,需要更新")
	//	Logger.Infof("原有allPeers:\n%v", AddrInfo2IDSlice(node.allPeers))
	//	Logger.Infof("新peers:\n%v", AddrInfo2IDSlice(peers))
	//	Logger.Infof("需要重新连接")
	//}
}

// 更新allPeers,并通知ui更新数据
func (node *Node) updateAllPeers(peers []peer.AddrInfo) {
	node.allPeers = peers
}

// 连接给定的peers，返回成功连接的数量
func (node *Node) connectToPeers(ctx context.Context, peers []peer.AddrInfo) int {
	var wg sync.WaitGroup
	ch := make(chan error, len(peers))
	for _, p := range peers {
		wg.Add(1)
		p := p
		go func() {
			ctx, cancel := context.WithTimeout(ctx, time.Second*2)
			defer cancel()
			err := node.host.Connect(ctx, p)
			if err != nil {
				// 如果连接失败，则使用中继节点进行连接
				// TODO:可能会引入多个中继，需要进行负载均衡，现在默认是一个固定的
				relayaddr, err := ma.NewMultiaddr("/p2p/" + relayNodeID1 + "/p2p-circuit/p2p/" + p.ID.Pretty())
				if err != nil {
					panic(err)
				}
				node.host.Network().(*swarm.Swarm).Backoff().Clear(p.ID)
				relayInfo := peer.AddrInfo{
					ID:    p.ID,
					Addrs: []ma.Multiaddr{relayaddr},
				}
				err = node.host.Connect(ctx, relayInfo)
				if err != nil {
					ch <- err
					Logger.Errorf("连接到%s失败：%s", p.ID.ShortString(), err.Error())
				} else {
					Logger.Infof("成功连接到%s", p.ID.ShortString())
				}
			} else {
				Logger.Infof("成功连接到%s", p.ID.ShortString())
			}
			wg.Done()
		}()
	}
	time.Sleep(time.Millisecond * 5)
	wg.Wait()
	close(ch)
	count := 0
	for range ch {
		count++
	}
	success := len(peers) - count
	Logger.Infof("成功连接%d个peer", success)
	return success
}

func (node *Node) publish(ctx context.Context, topic *pubsub.Topic, msg Message) {
	sendData, err := msg.MarshalJSON()
	ErrPanic(err, Logger)
	err = topic.Publish(ctx, sendData)
	ErrPanic(err, Logger)
	Logger.Infof("向%sTopic广播信息：%s", topic.String(), string(sendData)[:200])
}

func (node *Node) sendToPeer(ctx context.Context, msg Message) {
	peerId, err := peer.Decode(msg.GetTo())
	ErrPanic(err, Logger)
	Logger.Infof("准备发送到peerID:%s", peerId)
	stat := node.host.Network().Connectedness(peerId)
	Logger.Errorf("与peer：%s的连接状态：%s", peerId, stat.String())
	if stat != network.Connected {
		addrInfo := node.host.Peerstore().PeerInfo(peerId)
		success := node.connectToPeers(ctx, []peer.AddrInfo{addrInfo})
		if success == 0 {
			// TODO: 现在点对点发送失败直接返回,让程序不闪退
			Logger.Infof("连接到%s失败", peerId.ShortString())
			return
		}
	}
	s, err := node.host.NewStream(ctx, peerId, ProtocolID)
	defer s.Close()
	if err != nil {
		time.Sleep(time.Millisecond * 100)
		s, err = node.host.NewStream(ctx, peerId, ProtocolID)
		if err != nil {
			// TODO: 现在点对点发送失败直接返回,让程序不闪退
			Logger.Infof("%s:与%s创建新流失败", err.Error(), peerId.ShortString())
			return
		}
	}
	writer := bufio.NewWriter(s)
	sendData, _ := msg.MarshalJSON()
	Logger.Infof("准备向%s发送信息：%s", msg.GetTo(), string(sendData)[0:150])
	_, _ = writer.Write(sendData)
	_ = writer.Flush()

}

func (node *Node) handleMessage(ctx context.Context, msg Message) {
	Logger.Infof("msg的TO%s", msg.GetTo())
	if msg.GetTo()[0] == 'Q' {
		go node.sendToPeer(ctx, msg)
	} else if msg.GetTo()[0] == 'P' {
		go node.publish(ctx, node.stringToTopic(msg.GetTo()), msg)
	} else {
		Logger.Infof("分发消息失败")
	}
}

func (node *Node) handleOptReq(ctx context.Context, req OptRequest) {
	switch req.OptType {
	//TODO:
	}
}

func (node *Node) stringToTopic(s string) *pubsub.Topic {
	for _, t := range node.topics {
		if t.String() == s {
			Logger.Infof("当前topic的string为:%s", t.String())
			return t
		}
	}
	return nil
}
