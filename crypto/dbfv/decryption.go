package dbfv

import (
	"Vote/crypto/bfv"
	"Vote/crypto/ring"
	"Vote/crypto/utils"
	"bytes"
	"encoding/binary"
	//"github.com/ldsec/lattigo/v2/bfv"
	//"github.com/ldsec/lattigo/v2/ring"
	//"github.com/ldsec/lattigo/v2/utils"
)

// CDProtocol collective decryption protocol 存储进行联合解密所需要的的参数
type CDProtocol struct {
	context       *dbfvContext
	result        *bfv.Ciphertext
	sigmaSmudging float64
	ID            uint64
	tmpNtt        *ring.Poly
	tmpDelta      *ring.Poly
	hP            *ring.Poly

	baseconverter   *ring.FastBasisExtender
	gaussianSampler *ring.GaussianSampler
	//ciphertexts    map[uint64]*bfv.Ciphertext
	//cdshares map[uint64]CDShare
}

// CDShare 存储联合解密中需要分享的内容.
type CDShare struct {
	ID uint64
	*ring.Poly
}

type CiphertextShare struct {
	ID         uint64
	Ciphertext *bfv.Ciphertext
}

// NewCKSProtocol creates a new CKSProtocol that will be used to operate a collective key-switching on
// a ciphertext encrypted under a collective public-key, whose secret-shares are distributed among j
// parties, re-encrypting the ciphertext under another public-key, whose secret-shares are also known
// to the parties.
// 该函数将一个密文转换为使用另一个密钥进行加密的 密文
func NewCDProtocol(params *bfv.Parameters, sigmaSmudging float64, ID uint64) *CDProtocol {

	context := newDbfvContext(params)

	cd := new(CDProtocol)

	//cd.cdshares = make(map[uint64]CDShare)
	//cd.ciphertexts = make(map[uint64]*bfv.Ciphertext)
	cd.context = context
	cd.result = bfv.NewCiphertext(params, 1)
	cd.sigmaSmudging = sigmaSmudging
	cd.ID = ID
	cd.tmpNtt = cd.context.ringQP.NewPoly()
	cd.tmpDelta = cd.context.ringQ.NewPoly()
	cd.hP = cd.context.ringP.NewPoly()

	cd.baseconverter = ring.NewFastBasisExtender(cd.context.ringQ, cd.context.ringP)
	prng, err := utils.NewPRNG()
	if err != nil {
		panic(err)
	}
	cd.gaussianSampler = ring.NewGaussianSampler(prng, context.ringQP, sigmaSmudging, uint64(6*sigmaSmudging))

	return cd
}

// AllocateShare allocates the shares of the CDProtocol
func (cd *CDProtocol) AllocateShare() CDShare {

	return CDShare{cd.ID, cd.context.ringQ.NewPoly()}

}

func (cd *CDProtocol) AllocateCiphertextShare(cipher *bfv.Ciphertext) CiphertextShare {
	share := CiphertextShare{cd.ID, cipher}
	return share
}

/*
   func (cd *CDProtocol) RecCiphertext(ID uint64, ciphertext *bfv.Ciphertext){
   	cd.ciphertexts[ID] = ciphertext
   }
*/

func (cd *CDProtocol) CountVotes(ciphertextShare []CiphertextShare) {
	eval := bfv.NewEvaluator(cd.context.params)
	for _, value := range ciphertextShare { //取map中的值
		eval.Add(value.Ciphertext, cd.result, cd.result)
	}
}

// 将一个数组转换成 NTT 格式表示的多项式
// 具体实现是，coeffs表示不同模数下的lambda，需要将这个数组表示成如下形式：
// coeffs[0], 0, 0, ...
// coeffs[1], 0, 0, ...
// coeffs[2], 0, 0, ...
// 之后将其转换成NTT格式的多项式即可
func (cd *CDProtocol) NewPolyFromCoeffs(coeffs []uint64) *ring.Poly {
	poly := cd.context.ringQP.NewPoly()
	cd.context.ringQP.MFormArray(coeffs, poly)
	cd.context.ringQP.NTT(poly, poly)
	return poly
}

// GenShare is the first and unique round of the CKSProtocol protocol.
// Each party holding a ciphertext ctx encrypted under a collective public-key must
// compute the following :
//
// [(skInput_i - skOutput_i) * ctx[1] + e_i]
//
// Each party then broadcast the result of this computation to the other j-1 parties.

func (cd *CDProtocol) GenShare(coeff []uint64, skInput *ring.Poly, shareOut CDShare) {
	tmpPoly := cd.context.ringQP.NewPoly()
	coeffPoly := cd.NewPolyFromCoeffs(coeff)
	cd.context.ringQP.MulCoeffsMontgomery(skInput, coeffPoly, tmpPoly)
	skOutput := cd.context.ringQ.NewPoly()
	cd.context.ringQ.Sub(tmpPoly, skOutput, cd.tmpDelta) // cks.tmpDelta = ski - skOutput
	cd.genShareDelta(cd.tmpDelta, cd.result, shareOut)
}

/*
   func (cd *CDProtocol) RecPartDec(ID uint64, share CDShare){
   	cd.cdshares[ID] = share
   }
*/
func (cd *CDProtocol) genShareDelta(skDelta *ring.Poly, ct *bfv.Ciphertext, shareOut CDShare) {

	level := uint64(len(ct.Value()[1].Coeffs) - 1)

	ringQ := cd.context.ringQ
	ringQP := cd.context.ringQP

	ringQ.NTTLazy(ct.Value()[1], cd.tmpNtt)
	// c1 * s
	ringQ.MulCoeffsMontgomeryConstant(cd.tmpNtt, skDelta, shareOut.Poly)
	ringQ.MulScalarBigint(shareOut.Poly, cd.context.ringP.ModulusBigint, shareOut.Poly)

	ringQ.InvNTTLazy(shareOut.Poly, shareOut.Poly)

	cd.gaussianSampler.ReadLvl(uint64(len(ringQP.Modulus)-1), cd.tmpNtt)
	// + e
	ringQ.AddNoMod(shareOut.Poly, cd.tmpNtt, shareOut.Poly)

	for x, i := 0, uint64(len(ringQ.Modulus)); i < uint64(len(cd.context.ringQP.Modulus)); x, i = x+1, i+1 {
		tmphP := cd.hP.Coeffs[x]
		tmpNTT := cd.tmpNtt.Coeffs[i]
		for j := uint64(0); j < ringQ.N; j++ {
			tmphP[j] += tmpNTT[j]
		}
	}

	cd.baseconverter.ModDownSplitPQ(level, shareOut.Poly, cd.hP, shareOut.Poly)

	cd.tmpNtt.Zero()
	cd.hP.Zero()
}

// AggregateShares is the second part of the unique round of the CKSProtocol protocol.
// 收到 j - 1 个元素之后，每一个参与方都进行如下计算  :
//
// shareout = share1 + share2
func (cd *CDProtocol) AggregateShares(coeffs map[uint64][]uint64, share []CDShare, shareOut CDShare) {

	for _, value := range share {
		//lambdaPoly := cd.NewPolyFromCoeffs(coeffs[value.ID])
		//tmpPoly1 := cd.context.ringQ.NewPoly()
		//zero := cd.context.ringQ.NewPoly()
		//cd.context.ringQ.Sub(lambdaPoly, zero, tmpPoly1) // tmpPoly1 = lambdaPoly
		//cd.context.ringQ.MulCoeffsMontgomery(value.Poly, tmpPoly1, tmpPoly1)
		cd.context.ringQ.Add(value.Poly, shareOut.Poly, shareOut.Poly)

	}
}

func (cd *CDProtocol) FinialDec(combined CDShare) *bfv.Plaintext {
	tmp := bfv.NewCiphertext(cd.context.params, 1)
	cd.context.ringQ.Add(cd.result.Value()[0], combined.Poly, tmp.Value()[0])
	cd.context.ringQ.Copy(cd.result.Value()[1], tmp.Value()[1])
	ctOut := tmp.Plaintext()
	return ctOut

}

// UnmarshalBinary decodes a previous marshaled share on the target share.
func (cd *CDProtocol) UnmarshalBinary(data []byte) (share CDShare) {

	share = cd.AllocateShare()
	share.ID = binary.BigEndian.Uint64(data[0:8])
	_ = share.Poly.UnmarshalBinary(data[8:])
	return share
}

// UnmarshalBinary decodes a previous marshaled share on the target share.
func (cd *CDProtocol) MarshalBinary(share CDShare) []byte {
	tmpData1 := make([]byte, 8)
	binary.BigEndian.PutUint64(tmpData1[0:8], share.ID)
	tmpData2, err := share.Poly.MarshalBinary()
	if err != nil {
		panic(err)
	}

	var buffer bytes.Buffer
	buffer.Write(tmpData1)
	buffer.Write(tmpData2)

	return buffer.Bytes()
}
