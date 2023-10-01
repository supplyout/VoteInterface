package party

import (
	"Vote/crypto/bfv"
	"Vote/crypto/dbfv"
	"bytes"
	"encoding/binary"
	"fmt"
	//"github.com/ldsec/lattigo/v2/bfv"
	//"github.com/ldsec/lattigo/v2/dbfv"
)

// 表示参与投票的参与方
type Party struct {
	ID   uint64         //选民身份的标识
	ski  *bfv.SecretKey // 选民的ski
	pk   *bfv.PublicKey //本次投票的公钥
	poll *Poll
	//result           *bfv.Ciphertext
	resultPlaintext *bfv.Plaintext
	genSecretKey    *dbfv.DKGProtocol // 负责分布式生成私钥
	genPubKey       *dbfv.CKGProtocol
	disDec          *dbfv.CDProtocol
}

// 构造函数:生成一个投票参与方实例
// ID ： 该参与方的编号，是参与方身份的唯一标识
// poll : 标识一次投票该参与方需要掌握的信息
func NewParty(ID uint64, poll *Poll) *Party {
	// 没有进行错误检查
	keygen := bfv.NewKeyGenerator(poll.Params)
	sk, pk := keygen.GenKeyPair()
	return &Party{
		ID:   ID,
		ski:  sk,
		pk:   pk,
		poll: poll,
		//result: bfv.NewCiphertext(poll.Params, 1),
		resultPlaintext: bfv.NewPlaintext(poll.Params),
		genSecretKey:    dbfv.NewDKGProtocol(poll.Params, poll.N, poll.T, ID),
		genPubKey:       dbfv.NewCKGProtocol(poll.Params, ID),
		disDec:          dbfv.NewCDProtocol(poll.Params, 3.19, ID),
	}
}

//获取本次投票的t
func (voter *Party) GetThreshold() uint64 {
	return voter.poll.T
}

// 生成需要分发给其他参与方的secret-share
func (voter *Party) GenSecretShare() (secretShare []dbfv.DKGShare) {
	secretShare = voter.genSecretKey.DKGGenMatrixSingle()
	return secretShare
}

// 将需要分发的secret-share进行编码，以便传输
// 因该参与方分发给不同参与方的share不同，因此需要调用 N (N为选民总数）次该函数
func (voter *Party) MarshalSecretShare(share dbfv.DKGShare) (data []byte) {
	data = voter.genSecretKey.MarshalBinary(share)
	return data
}

// 将经过传输的secret-share进行解码
func (voter *Party) UnmarshalSecretShare(data []byte) (share dbfv.DKGShare) {
	share = voter.genSecretKey.UnmarshalBinary(data)
	return share
}

// 利用各参与方分享的secret-share生成ski
func (voter *Party) GenSki(dkgShare []dbfv.DKGShare) {
	voter.ski = voter.genSecretKey.Combine(dkgShare)
}

// 生成PK的第一步，生成需要分享给其他参与方的 pk-share
func (voter *Party) GenPki() (share dbfv.CKGShare) {
	share = voter.genPubKey.AllocateShares()
	voter.genPubKey.GenShare(voter.ski.Get(), voter.poll.A, share)
	return share
}

// 将该参与方生成的pk-share进行编码，以便传输
func (voter *Party) MarshalPki(share dbfv.CKGShare) []byte {
	return voter.genPubKey.MarshalBinary(share)
}

// 将经过传输的pk-share进行解码
// ID : 该pk-share发送方的编号
// share : pk-share内容
func (voter *Party) UnMarshalPki(data []byte) (share dbfv.CKGShare) {
	return voter.genPubKey.UnMarshalBinary(data)
}

// 根据各参与方分享的pk-share生成公钥 pk
func (voter *Party) GenPk(share []dbfv.CKGShare) {
	index := make([]int64, len(share))
	for i, s := range share {
		index[i] = int64(s.ID)
	}
	coeffs := voter.genSecretKey.GenCoffee(index)
	voter.genPubKey.GenPublicKey(voter.poll.A, coeffs, voter.pk, share)
}

// 将 ticket 表示的投票信息经过编码，加密成密文输出
func (voter *Party) Encrypt(ticket []uint64) (ciphertextShare dbfv.CiphertextShare) {

	fmt.Printf("%v", ticket)
	ciphertext := bfv.NewCiphertext(voter.poll.Params, 1)
	encryptor := bfv.NewEncryptorFromPk(voter.poll.Params, voter.pk)
	plaintext := bfv.NewPlaintext(voter.poll.Params)
	encoder := bfv.NewEncoder(voter.poll.Params)
	encoder.EncodeUint(ticket, plaintext)
	encryptor.Encrypt(plaintext, ciphertext)
	ciphertextShare = voter.disDec.AllocateCiphertextShare(ciphertext)
	return ciphertextShare
}

// 将上面函数返回的密文进行编码，以便传输
func (voter *Party) MarshalCiphertext(ciphertextShare dbfv.CiphertextShare) []byte {
	tmpData1 := make([]byte, 8)
	binary.BigEndian.PutUint64(tmpData1[0:8], ciphertextShare.ID)
	tmpData2, _ := ciphertextShare.Ciphertext.MarshalBinary()

	var buffer bytes.Buffer
	buffer.Write(tmpData1)
	buffer.Write(tmpData2)

	return buffer.Bytes()
}

// 将经过传输的密文进行解码
func (voter *Party) UnMarshalCiphertext(data []byte) (ciphertextShare dbfv.CiphertextShare) {
	ciphertextShare.ID = binary.BigEndian.Uint64(data[0:8])
	ciphertextShare.Ciphertext = new(bfv.Ciphertext)
	err := ciphertextShare.Ciphertext.UnmarshalBinary(data[8:])
	if err != nil {
		panic(err)
	}
	return ciphertextShare
}

// 统计选票
// 接收其他参与方发送的加密后的选票信息进行计票
func (voter *Party) EvalAdd(ciphertextShare []dbfv.CiphertextShare) {
	voter.disDec.CountVotes(ciphertextShare)
}

// 部分解密函数，生成需要进行传输的 dec-share
func (voter *Party) PartDec(index []int64) dbfv.CDShare {
	share := voter.disDec.AllocateShare()
	coeffs := voter.genSecretKey.GenCoffee(index)
	//println("系数长度", len(coeffs[voter.ID]))
	//println("该选民的系数")
	//fmt.Println(coeffs[voter.ID])
	voter.disDec.GenShare(coeffs[voter.ID], voter.ski.Get(), share)
	return share
}

// 将 dec-share 进行编码，以便传输
func (voter *Party) MarshalCDShare(share dbfv.CDShare) (data []byte) {
	data = voter.disDec.MarshalBinary(share)
	return data
}

// 将接收到的dec-share进行解码
func (voter *Party) UnMarshalCDShare(data []byte) (partdec dbfv.CDShare) {
	partdec = voter.disDec.UnmarshalBinary(data)
	//println("检测部分解密函数的解码函数", " ", partdec.ID)
	return partdec
}

// 根据各参与方分享的部分解密信息，进行解密操作的第二步，返回投票信息
func (voter *Party) FinaDec(share []dbfv.CDShare) (ans []uint64) {
	index := make([]int64, len(share))
	for i := 0; i < len(share); i++ {
	}
	for i, s := range share {
		index[i] = int64(s.ID)
	}
	coeffs := voter.genSecretKey.GenCoffee(index) // 拉格朗日系数
	combined := voter.disDec.AllocateShare()
	voter.disDec.AggregateShares(coeffs, share, combined)
	voter.resultPlaintext = voter.disDec.FinialDec(combined)
	encoder := bfv.NewEncoder(voter.poll.Params)
	ans = encoder.DecodeUintNew(voter.resultPlaintext)
	return ans
}

/*
func (voter *Party) MarshalID() (data []byte){
	data = make([]byte, 8)
	binary.BigEndian.PutUint64(data[0:8], voter.ID)
	return
}

func (voter *Party) UnmarshalID(data []byte) uint64{
	ID := binary.BigEndian.Uint64(data[0:8])
	return ID
}
*/
func (voter *Party) MarshalID(id uint64) (data []byte) {
	data = make([]byte, 8)
	binary.BigEndian.PutUint64(data[0:8], id)
	return data
}

func (voter *Party) UnmarshalID(data []byte) uint64 {
	ID := binary.BigEndian.Uint64(data[0:8])
	return ID
}
