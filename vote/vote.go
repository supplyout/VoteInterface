package vote

import (
	"Vote/crypto/dbfv"
	"context"
	util "github.com/ipfs/go-ipfs-util"
	"time"
)

type Vote struct {
	Title       string    //投票标题
	From        string    //发起者ID
	StartTime   time.Time //创建时间
	Deadline    time.Time
	Brief       string                    //投票简介
	VoteType    string                    //投票类型
	Topic       string                    //投票的topic=hash(StartTime+Title+From),唯一标识选票
	Parties     []string                  //参与方
	DecParties  []string                  //参与解密方
	Candidates  []string                  //候选人
	Result      map[string]int            //最终结果
	Partdec     []dbfv.CDShare            //存放受到他人的部分解密的结果
	EvalCipher  []dbfv.CiphertextShare    //存放其他方发过来的密文
	EvalResult  []dbfv.CiphertextShare    //密文相加的效果
	PartPubKey  []dbfv.CKGShare           //存放收到的Pk_i
	SkInf       []dbfv.DKGShare           //存放生成Sk_i所需要的信息
	OtherResult map[string]map[string]int // 收集不同人的投票结果
	IsCorrect   bool                      // 判断最终解密结果是否一致

	DecPartNum []uint64 //参与解密方下标
	Flag       bool     //是否参与投票
	LengthDec  int      // 新加的trueId长度
}

func NewVote(ctx context.Context, from string) *Vote {
	t, _ := time.Parse("2006-01-02 15:04:05", time.Now().Format("2006-01-02 15:04:05"))
	voteNew := &Vote{
		From:        from,
		StartTime:   t,
		OtherResult: make(map[string]map[string]int),
	}
	return voteNew
}

func (v *Vote) SetTopic() {
	// Topic=hash(StartTime+Title+From),唯一标识选票
	data := []byte(v.Title + v.StartTime.String() + v.From)
	v.Topic = "P2P" + util.Hash(data).B58String()
}
