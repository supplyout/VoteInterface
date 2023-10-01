package ui

import (
	. "Vote/crypto/party"
	. "Vote/log"
	. "Vote/message"
	. "Vote/node"
	. "Vote/vote"
	"context"
	"fmt"
	"github.com/libp2p/go-libp2p-core/peer"
	"reflect"
	"sort"
	"sync"
	"time"
)

var s *server

type server struct {
	Node     *Node
	MsgCh    map[ChanType]chan Message
	OptReqCh map[ChanType]chan OptRequest

	lSocket *LocalSocket
	ctx     context.Context
	Stat    NodeStat

	votes        []*Vote
	currentVote  *Vote
	currentParty *Party

	peers map[string]peer.IDSlice //key:topics value:参与此次投票的peers

	cancelWait context.CancelFunc

	currentNum int
}

func newApp() string {
	var err error

	//构建通信信道
	s = &server{
		MsgCh:    map[ChanType]chan Message{Read: make(chan Message, 30), Write: make(chan Message, 30)},
		OptReqCh: map[ChanType]chan OptRequest{Read: make(chan OptRequest), Write: make(chan OptRequest)},
		peers:    map[string]peer.IDSlice{},
	}
	s.ctx = context.Background()
	s.lSocket, err = NewLocalSocket("socketName")
	ErrPanic(err, Logger)

	go s.startLoop(s.ctx)
	//新建一个p2p网络的node，但并未进行初始化，即未加入最底层的pubsub网络
	isLocal := false // 是否为本地网络
	s.Node = NewNode(s.ctx, "s.etyNickName.Text", isLocal, s.MsgCh, s.OptReqCh, bootstrapInfo)
	Logger.Infof("node新建成功")
	return s.Node.GetNodeID()

}

func (s *server) InitNet(ctx context.Context) {
	// 这个函数是用来初始化网络, 即加入最底层的gossip网络
	//TODO:这里需要再设计一下
	s.Node.InitNet()
	s.Stat = NodeStatInited
	Logger.Infof("当前状态:NodeStatInited")

}

func (s *server) CreateNewVote(ctx context.Context) {
	if s.Stat != NodeStatInited {
		Logger.Infof("当前网络已经有投票在进行,无法发起新投票")
		return
	}
	voteNew := NewVote(ctx, s.Node.GetNodeID())
	s.currentVote = voteNew
}

func (s *server) SendBrief(ctx context.Context, title, brief, voteType string, deadline time.Time) {

	s.Node.StopRefreshPeers(ctx)

	// 填写当前投票信息
	s.currentVote.Title = title
	s.currentVote.Brief = brief
	s.currentVote.VoteType = voteType
	s.currentVote.Deadline = deadline
	s.currentVote.StartTime = time.Now()
	s.currentVote.From = s.Node.GetNodeID()
	s.currentVote.SetTopic()

	//生成需要发送的信息
	Logger.Infof("currentVote: %+v", s.currentVote)
	voteBriefMsg := GenVoteBrief(*s.currentVote)
	Logger.Infof("sendBrief: %+v", voteBriefMsg)

	// 加入当前vote的topic
	s.Node.Join(ctx, s.currentVote.Topic)
	s.Node.Advertise(ctx, s.currentVote.Topic)
	s.currentVote.Parties = append(s.currentVote.Parties, s.Node.GetNodeID())
	s.Stat = NodeStatConfirmed
	Logger.Infof("当前状态:NodeStatConfirmed")


	s.MsgCh[Write] <- voteBriefMsg

	var wg sync.WaitGroup
	var mctx context.Context
	mctx, s.cancelWait = context.WithCancel(ctx)
	go func(ctx context.Context) {
		wg.Add(1)
		for {
			select {
			case <-time.After(time.Until(s.currentVote.Deadline)):
				// 最多等待VotePeriod时间
				Logger.Infof("投票截至时间已到")
				break
			case <-ctx.Done():
				Logger.Infof("已提前开始投票")
				break
			}
			break
		}
	}(mctx)
	time.Sleep(time.Millisecond * 5)
	wg.Wait()

	peers := s.Node.FindPeers(ctx, s.currentVote.Topic)
	Logger.Debugf("约定的人:\n%v", peers)
	if len(peers) != len(s.currentVote.Parties) {
		Logger.Infof("约定的人数:%d\t确认参与投票的人数:%d", len(peers), len(s.currentVote.Parties))
		//TODO:约定的人数与确认参与投票的人数不一致,具体逻辑没写,现在默认以确认参与的人数为准
	}
	poll := NewPoll(uint64(len(peers)), uint64(len(peers)/2))
	s.MsgCh[Write] <- GenVoteParams(*s.currentVote, *poll)
	s.Stat = NodeStatGotParams
	Logger.Infof("当前状态:NodeStatGotParams")
}

func (s *server) ConfirmNewVote(ctx context.Context) {
	if s.Stat != NodeStatGotNewVote {
		Logger.Infof("无新的投票进行确认")
		return
	}
	s.Node.Join(ctx, s.currentVote.Topic)
	s.Node.Advertise(ctx, s.currentVote.Topic)

	voteConfMsg := GenVoteAcceptation(*s.currentVote, true, s.Node.GetNodeID())
	s.MsgCh[Write] <- voteConfMsg
	s.Stat = NodeStatConfirmed
	Logger.Infof("当前状态:NodeStatConfirmed")
}

func (s *server) RefuseNewVote() {
	if s.Stat != NodeStatGotNewVote {
		Logger.Infof("无新的投票用于拒绝")
		return
	}
	voteRefuseMsg := GenVoteAcceptation(*s.currentVote, false, s.Node.GetNodeID())
	s.MsgCh[Write] <- voteRefuseMsg
	s.Stat = NodeStatInited
	Logger.Infof("当前状态:NodeStatInited")
}

func (s *server) BeginVote() {
	if s.Stat != NodeStatConfirmed {
		Logger.Infof("状态错误，无法开始投票")
		return
	}
	s.cancelWait()
	//TODO: t out of N 中， t 选取了固定值 N/2+1
	poll := NewPoll(uint64(len(s.currentVote.Parties)), uint64(len(s.currentVote.Parties)/2+1))
	voteParamsMsg := GenVoteParams(*s.currentVote, *poll)
	s.MsgCh[Write] <- voteParamsMsg

}

func (s *server) CancelVote() {
	if s.Stat != NodeStatConfirmed {
		Logger.Infof("状态错误，无法取消一次投票")
		return
	}
	s.Stat = NodeStatInited
}

func (s *server) SendVoteDetail(candidates []string) {

	if s.Stat != NodeStatGotPK {
		return
	}

	//设置候选人
	s.currentVote.Candidates = candidates
	voteDetailMsg := GenVoteDetail(*s.currentVote)
	// 等待其他参与方计算完自己的密钥
	time.Sleep(time.Second * 4)
	s.MsgCh[Write] <- voteDetailMsg
	s.Stat = NodeStatGotDetail
	Logger.Infof("当前状态:NodeStatGotDetail")
}

func (s *server) SendVote(ticket []uint64) {
	//TODO:可能需要保存自己的投票结果
	Logger.Infof("%v", ticket)
	if s.Stat != NodeStatGotDetail {
		Logger.Infof("未获得投票细节信息，无法进行投票")
		return
	}
	if len(s.currentVote.Candidates) != len(ticket) {
		Logger.Infof("未对所有候选者进行投票:%d,%d", len(s.currentVote.Candidates), len(ticket))
		return
	}
	ciphertext := s.currentParty.Encrypt(ticket)
	//自己存放自己的密文
	ciphertetxM := s.currentParty.MarshalCiphertext(ciphertext)
	ciphertextMsg := GenCipherText(*s.currentVote, s.Node.GetNodeID(), ciphertetxM)
	s.MsgCh[Write] <- ciphertextMsg
	s.Stat = NodeStatVoted
	Logger.Infof("当前状态:NodeStatVoted")
}

func (s *server) AbandonTicket() {
	if s.Stat != NodeStatGotDetail {
		Logger.Infof("未获得投票细节信息，无法进行投票")
		return
	}
	ticket := make([]uint64, len(s.currentVote.Candidates))
	ciphertext := s.currentParty.Encrypt(ticket)
	ciphertetxM := s.currentParty.MarshalCiphertext(ciphertext)
	ciphertextMsg := GenCipherText(*s.currentVote, s.Node.GetNodeID(), ciphertetxM)
	s.MsgCh[Write] <- ciphertextMsg
	s.Stat = NodeStatVoted
	Logger.Infof("当前状态:NodeStatVoted")
}

func (s *server) GetID() uint64 {

	//对peerid进行排序
	var temp = s.currentVote.Topic
	myid, err := peer.Decode(s.Node.GetNodeID())
	ErrPanic(err, Logger)
	var peers = s.peers[temp]
	sort.Stable(peers)
	//获取自己的id（用于shamir以及之后的方案)
	for i := 0; i < len(peers); i++ {
		if myid == peers[i] {

			return uint64(i + 1)

		}
	}
	return 0
}

func (s *server) GetPeerIDFromId(myid int) string {
	//对peerid进行排序
	var temp = s.currentVote.Topic
	var peers = s.peers[temp]
	sort.Sort(peers)
	return peers[myid-1].Pretty()

}
func (s *server) startLoop(ctx context.Context) {

	for {
		select {
		case <-ctx.Done():
			Logger.Infof("应用层停止监听事件。。。")

		case msg := <-s.MsgCh[Read]:
			s.HandleMessage(ctx, msg)
		}
	}
}

func (s *server) SendMsgToKot(m msg4Kot) {
	data, _ := m.MarshalJSON()
	Logger.Infof("准备向android发送消息：%s", string(data))
	_, err := s.lSocket.Write(data)
	if err != nil {
		Logger.Errorf(err.Error())
	}
}

func IDSlice2Parties(slice peer.IDSlice) []string {
	var s []string
	for _, id := range slice {
		s = append(s, id.Pretty())
	}
	return s
}
func (s *server) HandleMessage(ctx context.Context, msg Message) {
	switch msg.GetType() {
	case VoteBriefType:
		s.handleVoteBrief(ctx, msg)
	case VoteParamsType:
		s.handleVoteParams(ctx, msg)
	case VoteDetailType:
		s.handleVoteDetail(ctx, msg)
	case VoteResultType:
		s.handleVoteResult(ctx, msg)
	case VoteAcceptationType:
		s.handleVoteAccept(ctx, msg)
	case PolyResultType:
		s.handlePolyResult(ctx, msg)
	case PubKeyPartType:
		s.handlePubKeyPart(ctx, msg)
	case PlainTextPartType:
		s.handlePlainText(ctx, msg)
	case CipherTextType:
		s.handleCipherText(ctx, msg)
	case IDType:
		s.handleCoffin(ctx, msg)
	default:
		Logger.Infof("消息分发失败，收到未识别的消息,消息类型码:%d", msg.GetType())
	}
}

func (s *server) handleTestMessage(ctx context.Context, msg Message) {
	Logger.Infof("收到测试消息")
}

func (s *server) handleVoteBrief(ctx context.Context, msg Message) {
	m := msg.(*VoteBrief)
	if m.From == s.Node.GetNodeID() {
		//Logger.Infof("收到自己的信息")
		return
	}
	if s.Stat != NodeStatInited {
		Logger.Infof("当前节点未处在初始化状态，无法参与新投票")
		return
	}
	//TODO:收到投票简介之后停止更新allPeers
	s.Node.StopRefreshPeers(ctx)
	s.currentVote = &Vote{
		From:        m.From,
		Topic:       m.Topic,
		Title:       m.Title,
		Brief:       m.Brief,
		VoteType:    m.VoteType,
		StartTime:   m.StartTime,
		Deadline:    m.Deadline,
		OtherResult: make(map[string]map[string]int),
	}
	s.Stat = NodeStatGotNewVote
	Logger.Infof("当前状态:NodeStatGotNewVote")

	msg4K := MsgNewVote{
		MType:    MsgNewVoteType,
		Starter:  m.From,
		Title:    m.Title,
		Brief:    m.Brief,
		VoteType: m.VoteType,
		Start:    m.StartTime,
		Deadline: m.Deadline,
	}
	s.SendMsgToKot(&msg4K)

}

func (s *server) handleVoteParams(ctx context.Context, msg Message) {
	m := msg.(*VoteParams)
	if s.Stat != NodeStatConfirmed {
		Logger.Infof("当前节点未处在已确认参与投票状态，无法参与该投票")
		return
	}
	peers := s.Node.FindPeers(ctx, s.currentVote.Topic)
	Logger.Infof("参与节点发现的peers：%v", peers)
	if len(peers) != int(m.N) {
		// 不一致时直接退出程序
		Logger.Infof(fmt.Sprintf("当前的本地节点发现的peers人数:%d与发送方发送过来的人数:%d不相同", len(peers), m.N))
		m.N = uint64(len(peers))
	}

	// 保存当前投票的所有参与方
	s.peers[s.currentVote.Topic] = peers

	// TODO:需要将所有的参与方写入到Vote中,未测试
	s.currentVote.Parties = IDSlice2Parties(peers)

	poll := NewPoll(m.N, m.T)
	Logger.Infof("本地ID:%d", s.GetID())
	s.currentParty = NewParty(s.GetID(), poll)

	//改变状态
	s.Stat = NodeStatGotParams
	Logger.Info("当前状态：NodeStatGotParams")

	//生成为了ski所需的信息
	infForSki := s.currentParty.GenSecretShare()
	//将对应的信息转发给对应的人

	//TODO:存疑，可能出错
	s.currentVote.SkInf = append(s.currentVote.SkInf, infForSki[s.currentParty.ID-1])

	for i := 1; i <= int(m.N); i++ {
		if i != int(s.currentParty.ID) {
			time.Sleep(time.Millisecond * 300)
			s.MsgCh[Write] <- GenPolyResult(*s.currentVote, s.Node.GetNodeID(), s.GetPeerIDFromId(i), s.currentParty.MarshalSecretShare(infForSki[i-1]))
			Logger.Infof("polyResult:%d", i)
		}
	}

}

func (s *server) handleVoteAccept(ctx context.Context, msg Message) {
	m := msg.(*VoteAcceptation)
	if s.Stat != NodeStatConfirmed {
		Logger.Infof("当前发起节点未处接收他人确定结果状态，无法进行统计")
		return
	}
	s.currentVote.Parties = append(s.currentVote.Parties, m.From)

	msg4K := MsgPartiesNum{
		MType: MsgPartiesNumType,
		Num:   len(s.currentVote.Parties),
	}
	s.SendMsgToKot(&msg4K)
}

/*
获取他人发来的信息，用于生成ski，并用ski生成pki
*/
func (s *server) handlePolyResult(ctx context.Context, msg Message) {
	m := msg.(*PolyResult)
	if s.Stat != NodeStatGotParams && s.Stat != NodeStatConfirmed {
		Logger.Infof("当前节点暂未收到参数或者未确认投票，无法处理多项式结果")
		return
	}

	share := s.currentParty.UnmarshalSecretShare(m.Result)

	s.currentVote.SkInf = append(s.currentVote.SkInf, share)

	//判断获得其他所有人的信息
	if len(s.currentVote.Parties) == len(s.currentVote.SkInf) {

		//生成每个人的SKi
		s.currentParty.GenSki(s.currentVote.SkInf)

		Logger.Infof("成功生成sk_i")
		//生成pki
		pki := s.currentParty.GenPki()
		//s.currentVote.PartPubKey = append(s.currentVote.PartPubKey, pki) //将自己的pki保存到Vote中
		Logger.Infof("成功生成pk_i")
		//将pki广播
		pkiMarshaled := s.currentParty.MarshalPki(pki)
		s.MsgCh[Write] <- GenPubKeyPart(*s.currentVote, s.Node.GetNodeID(), pkiMarshaled)
		s.Stat = NodeStatSentPki
		Logger.Infof("当前状态:NodeStatSentPki")

	}

}

func (s *server) handlePubKeyPart(ctx context.Context, msg Message) {
	m := msg.(*PubKeyPart)
	if s.Stat != NodeStatSentPki && s.Stat != NodeStatGotParams {
		Logger.Infof("当前节点未发送自己PKi或者未获取参数，无法处理他人的PKi")
		return
	}
	//解码得到他人发过来的pk_i
	Logger.Infof("收到pki")
	pki := s.currentParty.UnMarshalPki(m.Part)
	s.currentVote.PartPubKey = append(s.currentVote.PartPubKey, pki)

	//获得t个pk_i之后
	if len(s.currentVote.PartPubKey) >= len(s.currentVote.Parties) {
		//生成完整的pk
		s.currentParty.GenPk(s.currentVote.PartPubKey)
		s.Stat = NodeStatGotPK
		Logger.Infof("当前状态:NodeStatGotPK")
		msg4K := MsgGotPK{
			MType: MsgGotPKType,
		}
		s.SendMsgToKot(&msg4K)
	}

}

/*
存疑，生成pk后，接受类似候选人名单等的投票信息
*/
func (s *server) handleVoteDetail(ctx context.Context, msg Message) {
	m := msg.(*VoteDetail)
	// 忽略自己的广播信息
	if m.From == s.Node.GetNodeID() {
		return
	}
	if s.Stat != NodeStatGotPK && s.Stat != NodeStatSentPki {
		Logger.Infof("当前节点没有生成pk或没有发送pki，无法接收候选人信息")
		return
	}
	s.currentVote.Title = m.Title
	s.currentVote.Candidates = m.Candidates

	//改变状态
	s.Stat = NodeStatGotDetail
	Logger.Infof("当前状态:NodeStatGotDetail")

	//跳转到投票界面
	msg4K := MsgVoteDetails{
		MType:      MsgVoteDetailsType,
		Title:      m.Title,
		Brief:      m.Brief,
		Candidates: m.Candidates,
	}
	s.SendMsgToKot(&msg4K)
}

/*
接受他人的密文，对这些密文进行运算
*/
func (s *server) handleCipherText(ctx context.Context, msg Message) {
	m := msg.(*CipherText)
	Logger.Infof("收到他人的密文")
	if s.Stat != NodeStatVoted && s.Stat != NodeStatGotDetail {
		Logger.Infof("当前发起节点还不能收到其他人的密文")
		return
	}
	//解码得到他人发过来的密文
	cipher := s.currentParty.UnMarshalCiphertext(m.Text)
	s.currentVote.EvalCipher = append(s.currentVote.EvalCipher, cipher)
	//判断获得其他所有人的信息
	if len(s.currentVote.Parties) == len(s.currentVote.EvalCipher) {

		//对密文运算
		s.currentParty.EvalAdd(s.currentVote.EvalCipher)

		////部分解密
		//partdec := s.currentParty.PartDec()
		//
		////发广播
		//s.MsgCh[Write] <- GenPlainTextPart(*s.currentVote, s.Node.GetNodeID(), s.currentParty.MarshalCDShare(partdec))
		////s.MsgCh[Write] <- Gen
		//if
		//rand.Seed(int64(s.currentParty.ID))
		//randnum := rand.Intn(10)

		//if randnum <= 7 {
		text := s.currentParty.MarshalID(s.currentParty.ID)
		Logger.Infof("当前节点参与投票 %s", text)
		s.MsgCh[Write] <- GenPersonID(*s.currentVote, s.Node.GetNodeID(), text)
		s.currentVote.Flag = true
		//} else {
		//	s.MsgCh[Write] <- GenPersonID(*s.currentVote, s.Node.GetNodeID(), s.currentParty.MarshalID(0))
		//	Logger.Info("此节点不参与解密")
		//	s.currentVote.Flag = false
		//}
		s.Stat = NodeStatSentID
		Logger.Infof("当前状态:NodeStatSentID")
	}

}

/*
接受他人的ID，计算出对应的拉格朗日系数，进行部分解密
*/
func (s *server) handleCoffin(ctx context.Context, msg Message) {
	m := msg.(*ID)
	Logger.Infof("收到ID")
	if s.Stat != NodeStatSentID && s.Stat != NodeStatVoted {
		Logger.Infof("当前发起节点还不能接收ID")
		return
	}

	//解码得到他人发过来的ID
	id := s.currentParty.UnmarshalID(m.Text)
	s.currentVote.DecPartNum = append(s.currentVote.DecPartNum, id)

	//cipher := s.currentParty.UnMarshalCiphertext(m.Text)
	//如果当前节点参与投票，进行部分解密
	if s.currentVote.Flag {

		//判断获得其他所有人的信息
		if len(s.currentVote.Parties) == len(s.currentVote.DecPartNum) {

			//将trueid作为参数传入部分解密的结果
			var trueId []int64
			for i := 0; i < len(s.currentVote.DecPartNum); i++ {
				if s.currentVote.DecPartNum[i] > 0 {
					trueId = append(trueId, int64(s.currentVote.DecPartNum[i]))
				}
			}
			s.currentVote.LengthDec = len(trueId)
			////部分解密
			//println("输出trueid长度: ", len(trueId))
			//fmt.Println(trueId)
			partdec := s.currentParty.PartDec(trueId)
			//
			//发广播
			s.MsgCh[Write] <- GenPlainTextPart(*s.currentVote, s.Node.GetNodeID(), s.currentParty.MarshalCDShare(partdec))
			s.Stat = NodeStatSentCts
			Logger.Infof("当前状态:NodeStatSentCts")
		}
	} //else {
	//s.Stat = NodeStatSentCts
	//Logger.Infof("当前状态:NodeStatSentCts")
	//}

}

/*
接受他人发来的部分解密的信息
*/
func (s *server) handlePlainText(ctx context.Context, msg Message) {
	m := msg.(*PlainTextPart)
	if s.Stat != NodeStatSentCts && s.Stat != NodeStatSentID {
		Logger.Infof("进行最终的结果，无法进行统计")
		return
	}
	Logger.Infof("收到别人部分解密的结果")
	//解码部分解密的信息
	partdec := s.currentParty.UnMarshalCDShare(m.Text)

	//将部分解密结果存入此应用
	s.currentVote.Partdec = append(s.currentVote.Partdec, partdec)
	s.currentVote.DecParties = append(s.currentVote.DecParties, m.From)

	//判断a.Partdec的长度是否>=t，若满足条件，则可以进行Findec
	if len(s.currentVote.Partdec) == s.currentVote.LengthDec {
		Logger.Infof("进行最终解密")
		s.currentVote.Result = make(map[string]int)
		//保存投票最终结果的票数
		fin := s.currentParty.FinaDec(s.currentVote.Partdec)
		Logger.Infof("result:%v", fin[:len(s.currentVote.Candidates)])

		//按对应的顺序将候选人和票数一一对应存入result
		l := len(s.currentVote.Candidates)
		for i := 0; i < l; i++ {
			s.currentVote.Result[s.currentVote.Candidates[i]] = int(fin[i])
		}
		Logger.Infof("最终解密成功")
		s.MsgCh[Write] <- GenVoteResult(*s.currentVote, s.Node.GetNodeID())
		s.Stat = NodeStatSentResult
		Logger.Infof("当前状态:NodeStatSentResult")

	}
	//s.MsgCh[Write] <- GenVoteResult(*s.currentVote, s.Node.GetNodeID())
	//s.Stat = NodeStatGotPts
}

/*
每个人计算出最终结果之后，将结果广播，比对他人发来的结果，观察是否一致
*/
func (s *server) handleVoteResult(ctx context.Context, msg Message) {
	m := msg.(*VoteResult)
	if s.Stat != NodeStatSentResult && s.Stat != NodeStatSentCts {
		Logger.Infof("当前发起节点未处接收他人确定结果状态，无法进行统计")
		return
	}
	// 先添加
	result := m.Result
	s.currentVote.OtherResult[m.From] = result
	Logger.Infof("收到结果的个数：%d,参与投票的人数：%d,参与解密的人数:%d",
		len(s.currentVote.OtherResult), len(s.currentVote.Parties), len(s.currentVote.Partdec))
	if (len(s.currentVote.OtherResult) == len(s.currentVote.Partdec)) && s.currentVote.Result != nil {
		// 收到的结果数量等于参与解密方的数量时，结果全部比对完，通知安卓
		s.currentVote.IsCorrect = true
		Logger.Infof("自己结果：%v", s.currentVote.Result)
		for _, r := range s.currentVote.OtherResult {
			if !reflect.DeepEqual(r, s.currentVote.Result) {
				Logger.Infof("他人结果%v，自己结果%v", result, s.currentVote.Result)
				Logger.Infof("投票结果不一致，出现错误")
				s.currentVote.IsCorrect = false
			}
			Logger.Infof("他人结果：%v", r)
		}
		msg4K := MsgResult{
			MType:     MsgResultType,
			Result:    s.currentVote.Result,
			IsCorrect: s.currentVote.IsCorrect,
		}
		s.SendMsgToKot(&msg4K)
		Logger.Infof("%v", s.currentVote.Result)
		Logger.Infof("参与投票方：%v", s.currentVote.DecParties)
		s.Stat = NodeStatInited
	}

}
