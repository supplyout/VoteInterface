package ui

import (
	. "Vote/log"
	"fmt"
	"github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"strconv"
	"strings"
	"time"
)

func Test() {
	fmt.Println("666")
}

func CreateNewVoteForKotlin() {
	Logger.Infof("调用了CreateNewVote")
	s.CreateNewVote(s.ctx)
}

func RefuseNewVoteForKotlin() {
	Logger.Infof("调用了RefuseNewVote")
	s.RefuseNewVote()
}

func BeginVoteForKotlin() {
	Logger.Infof("调用了BeginVote")
	s.BeginVote()
}

func CancelVoteForKotlin() {
	Logger.Infof("调用了CancelVote")
	s.CancelVote()
}

func SendVoteForKotlin(ticket string) {
	Logger.Infof("调用了SendVote")
	strs := strings.Split(ticket, ",")
	var mTicket []uint64
	for _, s := range strs {
		tmp, _ := strconv.Atoi(s)
		mTicket = append(mTicket, uint64(tmp))
	}
	Logger.Infof("%+v", mTicket)
	s.SendVote(mTicket)
}

func AbandonTicketForKotlin() {
	Logger.Infof("调用了AbandonTicket")
	s.AbandonTicket()
}

func SendBriefForKotlin(title, brief, voteType, deadline string) {
	Logger.Infof("调用了SendBrief")
	dl, _ := time.Parse("2006-01-02 15:04:05", deadline)
	s.SendBrief(s.ctx, title, brief, voteType, dl)
}

func SendVoteDetailForKotlin(candidates string) {
	Logger.Infof("调用了SendVoteDetail")
	strs := strings.Split(candidates, ",")
	s.SendVoteDetail(strs)
}
func ConfirmNewVoteForKotlin() {
	Logger.Infof("调用了ConfirmNewVote")
	s.ConfirmNewVote(s.ctx)
}
func NewAppForKotlin() string {
	InitLogger()
	Logger.Infof("调用了NewApp")
	peerID := newApp()
	return peerID
}
func InitNetForKotlin() {
	Logger.Infof("调用了InitNet")
	s.InitNet(s.ctx)
}

var bootstrapInfo *peer.AddrInfo

func SetAddress(addr string) {
	bootAddr, _ := ma.NewMultiaddr(addr + "/p2p/QmQcb8b4GTbDSAgzTKZMUMnpKgWrg5w9haZSJDKrmC5Dqt")
	bootstrapInfo, _ = peer.AddrInfoFromP2pAddr(bootAddr)
}
