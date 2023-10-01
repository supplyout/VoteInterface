package party

import (
	"Vote/crypto/ring"
	"Vote/crypto/utils"
	"bytes"
	"encoding/binary"

	//"github.com/ldsec/lattigo/v2/bfv"
	//"github.com/ldsec/lattigo/v2/ring"
	//"github.com/ldsec/lattigo/v2/utils"
	"Vote/crypto/bfv"
)

// Poll is the main data type representing an encrypted poll
// 表示一次问卷调查
type Poll struct {
	N      uint64          // 选民人数
	T      uint64          // 成功解密所需的最少选民数
	Params *bfv.Parameters // 加密所需的参数
	A      *ring.Poly
	// 可能还需要有投票内容
}

func NewPoll(N, t uint64) *Poll {
	params := bfv.DefaultParams[bfv.PN13QP202pq]
	lattigoPRNG, err := utils.NewKeyedPRNG([]byte{'l', 'a', 't', 't', 'i', 'g', 'o'})
	if err != nil {
		panic(err)
	}
	ringQP, _ := ring.NewRing(1<<params.LogN(), append(params.Qi(), params.Pi()...))
	crsGen := ring.NewUniformSampler(lattigoPRNG, ringQP)
	a := crsGen.ReadNew() // 生成公钥需要的多项式a
	poll := &Poll{
		N:      N,
		T:      t,
		Params: params,
		A:      a,
	}

	return poll
}

// UnmarshalBinary decodes a slice of byte on the target polynomial.
func (poll *Poll) UnmarshalBinary(data []byte) {

	poll.N = binary.BigEndian.Uint64(data[8:16])
	poll.T = binary.BigEndian.Uint64(data[16:24])

	pointer := (3 + poll.N) << 3
	index := binary.BigEndian.Uint64(data[pointer : pointer+8])
	poll.Params = bfv.DefaultParams[index]
	pointer += 8
	poll.A = ring.NewPoly(poll.Params.N(), poll.Params.QPiCount())
	_ = poll.A.UnmarshalBinary(data[pointer:])
	return
}

// MarshalBinary encodes the target polynomial on a slice of bytes.
func (poll *Poll) MarshalBinary() (data []byte) {
	tmpDataLen := (4 + poll.N) << 3 // 没有包含多项式的字节数组大小

	tmpData := make([]byte, tmpDataLen)

	binary.BigEndian.PutUint64(tmpData[8:16], poll.N)
	binary.BigEndian.PutUint64(tmpData[16:24], poll.T)

	pointer := (3 + poll.N) << 3
	binary.BigEndian.PutUint64(tmpData[pointer:pointer+8], 5)
	/*
		binary.BigEndian.PutUint64(tmpData[pointer : pointer + 8], poll.Params.QiCount())
		binary.BigEndian.PutUint64(tmpData[pointer + 8 : pointer + 16], poll.Params.PiCount())
		binary.BigEndian.PutUint64(tmpData[pointer + 16 : pointer + 24], poll.Params.LogN())
		pointer += 24
		for i := uint64(0); i < poll.Params.QiCount(); i++{
			binary.BigEndian.PutUint64(tmpData[pointer + i << 3 : pointer + (i + 1) << 3], poll.Params.Qi()[i])
		}
		pointer += poll.Params.QiCount() << 3
		for i := uint64(0); i < poll.Params.PiCount(); i++{
			binary.BigEndian.PutUint64(tmpData[pointer + i << 3 : pointer + (i + 1) << 3], poll.Params.Pi()[i])
		}
		pointer += poll.Params.PiCount() << 3

		bits := math.Float64bits(poll.Params.Sigma())
		binary.BigEndian.PutUint64(tmpData[pointer : pointer + 8], bits)
	*/
	/*
		ID   8字节
		存储选民的个数N   8字节
		存储t       8字节
		选民       8N字节

		qi\ pi的个数    8 * 2字节
		Params *bfv.Parameters
			logN      8字节
			qi, pi    8 * 个数字节
			t         8字节
		    sigma     8字节

		之后是多项式，直接按照多项式的存储规则来进行存储即可
	*/
	tmp, err := poll.A.MarshalBinary()
	if err != nil {
		panic(err)
	}

	var buffer bytes.Buffer
	buffer.Write(tmpData)
	buffer.Write(tmp)

	return buffer.Bytes()
}
