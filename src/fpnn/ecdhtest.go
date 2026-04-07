package main

import (
	"crypto/ecdh"
	"crypto/elliptic"
	"crypto/rand"
	"errors"
	"fmt"

	"github.com/btcsuite/btcd/btcec/v2"
)

// UnifiedCurve 统一接口，模仿 ecdh.Curve
type UnifiedCurve interface {
	GenerateKey() (priv any, pub any, err error)
	ComputeShared(priv any, pub any) ([]byte, error)
}

// ---- 标准库封装 ----

type StdCurve struct {
	c ecdh.Curve
}

func (sc *StdCurve) GenerateKey() (any, any, error) {
	priv, err := sc.c.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	return priv, priv.PublicKey(), nil
}

func (sc *StdCurve) ComputeShared(priv any, pub any) ([]byte, error) {
	return priv.(*ecdh.PrivateKey).ECDH(pub.(*ecdh.PublicKey))
}

// ---- elliptic 封装 (P192, P224) ----

type EllipticCurve struct {
	c elliptic.Curve
}

func (ec *EllipticCurve) GenerateKey() (any, any, error) {
	priv, x, y, err := elliptic.GenerateKey(ec.c, rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	return priv, elliptic.Marshal(ec.c, x, y), nil
}

func (ec *EllipticCurve) ComputeShared(priv any, pub any) ([]byte, error) {
	x, y := elliptic.Unmarshal(ec.c, pub.([]byte))
	x2, _ := ec.c.ScalarMult(x, y, priv.([]byte))
	return x2.Bytes(), nil
}

// ---- secp256k1 封装 ----

type Secp256k1Curve struct{}

func (sc *Secp256k1Curve) GenerateKey() (any, any, error) {
	priv, err := btcec.NewPrivateKey()
	if err != nil {
		return nil, nil, err
	}
	return priv, priv.PubKey(), nil
}

func (sc *Secp256k1Curve) ComputeShared(priv any, pub any) ([]byte, error) {
	return btcec.GenerateSharedSecret(priv.(*btcec.PrivateKey), pub.(*btcec.PublicKey)), nil
}

// ---- 工厂方法 ----

func GetCurve(name string) (UnifiedCurve, error) {
	switch name {
	case "secp256r1":
		return &StdCurve{ecdh.P256()}, nil
	case "secp384r1":
		return &StdCurve{ecdh.P384()}, nil
	case "secp521r1":
		return &StdCurve{ecdh.P521()}, nil
	case "secp192r1":
		return &EllipticCurve{elliptic.P192()}, nil
	case "secp224r1":
		return &EllipticCurve{elliptic.P224()}, nil
	case "secp256k1":
		return &Secp256k1Curve{}, nil
	default:
		return nil, errors.New("unsupported curve")
	}
}

func main() {
	// 你可以换成 "secp192r1", "secp224r1", "secp256r1", "secp256k1"
	curve, _ := GetCurve("secp256k1")

	alicePriv, alicePub, _ := curve.GenerateKey()
	bobPriv, bobPub, _ := curve.GenerateKey()

	secretA, _ := curve.ComputeShared(alicePriv, bobPub)
	secretB, _ := curve.ComputeShared(bobPriv, alicePub)

	fmt.Printf("Equal: %v\n", string(secretA) == string(secretB))
	fmt.Printf("Shared Secret: %x\n", secretA)
}
