package main

import (
	"crypto/ed25519"
	"github.com/yoseplee/vrf"
	"io"
	"log"
	"math"
	"math/big"
)

// GenerateKeys generate public and private keys for nodes.
func GenerateKeys(rand io.Reader) (ed25519.PublicKey, ed25519.PrivateKey) {
	pk, sk, err := ed25519.GenerateKey(rand)
	if err != nil {
		log.Panic(err)
	}
	return pk, sk
}

// Sortition implements the VRF sortition algorithm.
func (node *Node) Sortition(pk ed25519.PublicKey, sk ed25519.PrivateKey, seed HashValue, tau int64, role string, w int64, W int64) ([]byte, []byte, int64) {
	pi, hash, err := vrf.Prove(pk, sk, append(seed[:], []byte(role)...))
	if err != nil {
		log.Panic(err)
	}
	p := float64(tau) / float64(W)
	j := int64(0)
	//aa := float64(0)
	//for i := 0; i < 4; i++ {
	//	aa += float64(binary.BigEndian.Uint64(hash[i*8:i*8+8])) * math.Pow(2, float64(3-i)*8*8)
	//}
	aa := big.NewInt(0)
	aa.SetBytes(hash)
	bb := big.NewInt(0)
	bbx := big.NewInt(2)
	bby := big.NewInt(int64(len(hash) * 8))
	bbm := big.NewInt(0)
	bb.Exp(bbx, bby, bbm)

	faa := big.NewFloat(0)
	faa.SetInt(aa)
	fbb := big.NewFloat(0)
	fbb.SetInt(bb)
	res := big.NewFloat(0)
	resf, _ := res.Quo(faa, fbb).Float64()

	lowerB, upperB := sumB(j, w, p)
	lowerB = 0
	for !(resf >= lowerB && resf < upperB) {
		j++
		lowerB, upperB = sumB(j, w, p)
	}
	return hash, pi, j
}

// sumB computes the binomial bounds.
func sumB(j int64, w int64, p float64) (float64, float64) {
	lowerB := big.NewFloat(0)
	for k := int64(0); k <= j; k++ {
		coefficient := big.NewInt(0)
		coefficient.Binomial(w, k)
		lb1 := big.NewFloat(0)
		lb1.SetInt(coefficient)

		lb2 := big.NewFloat(math.Pow(p, float64(k)))
		lb3 := big.NewFloat(math.Pow(1-p, float64(w-k)))

		lb23 := big.NewFloat(0)
		lb23.Mul(lb2, lb3)

		tmpLowerBf := big.NewFloat(0)
		tmpLowerBf.Mul(lb1, lb23)
		lowerB.Add(lowerB, tmpLowerBf)
	}
	coefficient := big.NewInt(0)
	coefficient.Binomial(w, j+1)
	lb1 := big.NewFloat(0)
	lb1.SetInt(coefficient)

	lb2 := big.NewFloat(math.Pow(p, float64(j+1)))
	lb3 := big.NewFloat(math.Pow(1-p, float64(w-(j+1))))
	lb23 := big.NewFloat(0)
	lb23.Mul(lb2, lb3)

	upperB := big.NewFloat(0)
	upperB.Mul(lb1, lb23)

	upperB.Add(upperB, lowerB)

	resLowerB, _ := lowerB.Float64()
	resUpperB, _ := upperB.Float64()

	return resLowerB, resUpperB
}
