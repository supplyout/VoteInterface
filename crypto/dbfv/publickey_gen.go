//Package dbfv implements a distributed (or threshold) version of the BFV scheme that enables secure
//multiparty computation solutions with secret-shared secret keys.
package dbfv

import (
	"bytes"
	"encoding/binary"
	//"github.com/ldsec/lattigo/v2/bfv"
	//"github.com/ldsec/lattigo/v2/ring"
	//"github.com/ldsec/lattigo/v2/utils"
	"Vote/crypto/bfv"
	"Vote/crypto/ring"
	"Vote/crypto/utils"
)

// CKGProtocol is the structure storing the parameters and state for a party in the collective key
// generation protocol.
type CKGProtocol struct {
	ID              uint64
	ringQP          *ring.Ring
	sigma           float64
	gaussianSampler *ring.GaussianSampler
	//ckgshares       map[uint64]CKGShare
}

// CKGShare is a struct holding a CKG share.
type CKGShare struct {
	ID   uint64
	Poly *ring.Poly
}

// NewCKGProtocol creates a new CKGProtocol instance
func NewCKGProtocol(params *bfv.Parameters, ID uint64) *CKGProtocol {
	context := newDbfvContext(params)
	ckg := new(CKGProtocol)
	ckg.ringQP = context.ringQP
	ckg.sigma = params.Sigma()
	ckg.ID = ID
	//ckg.ckgshares = make(map[uint64]CKGShare)
	prng, err := utils.NewPRNG()
	if err != nil {
		panic(err)
	}
	ckg.gaussianSampler = ring.NewGaussianSampler(prng, ckg.ringQP, params.Sigma(), uint64(6*params.Sigma()))
	return ckg
}

// AllocateShares allocates the CKG shares.
func (ckg *CKGProtocol) AllocateShares() CKGShare {
	return CKGShare{ckg.ID, ckg.ringQP.NewPoly()}
}

// GenShare generates the party's public key share from its secret key as:
//
// lambda_poly * crs * s_i + e_i
//
// for the receiver protocol. Has no effect is the share was already generated.

func (ckg *CKGProtocol) GenShare(sk, a *ring.Poly, shareOut CKGShare) {
	ckg.gaussianSampler.Read(shareOut.Poly) // e
	ckg.ringQP.NTT(shareOut.Poly, shareOut.Poly)
	ckg.ringQP.MulCoeffsMontgomeryAndSub(sk, a, shareOut.Poly) // shareout.Poly = e - a * ski
}

// 将一个数组转换成NTT格式表示的多项式
// 具体实现是，coeffs表示不同模数下的lambda，需要将这个数组表示成如下形式：
// coeffs[0], 0, 0, ...
// coeffs[1], 0, 0, ...
// coeffs[2], 0, 0, ...
// 之后将其转换成NTT格式的多项式即可
func (ckg *CKGProtocol) NewPolyFromCoeffs(coeffs []uint64) *ring.Poly {
	poly := ckg.ringQP.NewPoly()
	ckg.ringQP.MFormArray(coeffs, poly)
	ckg.ringQP.NTT(poly, poly)
	return poly
}

// AggregateShares aggregates a new share to the aggregate key
// share_out = share1 + share2
func (ckg *CKGProtocol) AggregateShares(lambda []uint64, share1, share2, shareOut CKGShare) {
	lambdaPoly := ckg.NewPolyFromCoeffs(lambda)
	tmpPoly1 := ckg.ringQP.NewPoly()

	ckg.ringQP.MulCoeffsMontgomery(share1.Poly, lambdaPoly, tmpPoly1)
	ckg.ringQP.Add(tmpPoly1, share2.Poly, shareOut.Poly)
}

// GenPublicKey return the current aggregation of the received shares as a bfv.PublicKey.
/*
func (ckg *CKGProtocol) GenPublicKey(roundShare CKGShare, crs *ring.Poly, pubkey *bfv.PublicKey) {
	pubkey.Set([2]*ring.Poly{roundShare.Poly, crs})
}
*/

func (ckg *CKGProtocol) GenPublicKey(crs *ring.Poly, coeffs map[uint64][]uint64, pk *bfv.PublicKey, shares []CKGShare) {
	ckgCombined := ckg.AllocateShares()
	for _, share := range shares {
		ckg.AggregateShares(coeffs[share.ID], share, ckgCombined, ckgCombined)
	}
	pk.Set([2]*ring.Poly{ckgCombined.Poly, crs})
}

// UnmarshalBinary decode a marshaled CKG share on the target CKG share.
func (ckg *CKGProtocol) UnMarshalBinary(data []byte) (share CKGShare) {
	share = ckg.AllocateShares()
	share.ID = binary.BigEndian.Uint64(data[0:8])
	if share.Poly == nil {
		share.Poly = new(ring.Poly)
	}
	err := share.Poly.UnmarshalBinary(data[8:])
	if err != nil {
		panic(err)
	}
	return share
}

// UnmarshalBinary decode a marshaled CKG share on the target CKG share.
func (ckg *CKGProtocol) MarshalBinary(share CKGShare) []byte {
	tmpData1 := make([]byte, 8)
	binary.BigEndian.PutUint64(tmpData1[0:8], share.ID)
	if share.Poly == nil {
		share.Poly = new(ring.Poly)
	}
	tmpData2, err := share.Poly.MarshalBinary()
	if err != nil {
		panic(err)
	}
	var buffer bytes.Buffer
	buffer.Write(tmpData1)
	buffer.Write(tmpData2)
	return buffer.Bytes()
}
