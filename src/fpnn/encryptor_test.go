package fpnn

import (
	"bytes"
	"testing"
)

func TestNativeECDHSharedSecret(t *testing.T) {
	curveNames := []string{"secp192r1", "secp224r1", "secp256r1", "secp256k1"}

	for _, curveName := range curveNames {
		t.Run(curveName, func(t *testing.T) {
			curve, err := getNativeECCurve(curveName)
			if err != nil {
				t.Fatalf("get curve failed: %v", err)
			}
			if !curve.isOnCurve(curve.gx, curve.gy) {
				t.Fatalf("base point is not on curve")
			}
			if x, y := curve.scalarMult(curve.gx, curve.gy, curve.n.Bytes()); x != nil || y != nil {
				t.Fatalf("base point order check failed")
			}

			alicePriv, aliceX, aliceY, err := curve.makeKey()
			if err != nil {
				t.Fatalf("generate alice key failed: %v", err)
			}
			bobPriv, bobX, bobY, err := curve.makeKey()
			if err != nil {
				t.Fatalf("generate bob key failed: %v", err)
			}

			alicePublic := make([]byte, curve.size*2)
			copy(alicePublic, fixedBytes(aliceX, curve.size))
			copy(alicePublic[curve.size:], fixedBytes(aliceY, curve.size))

			bobPublic := make([]byte, curve.size*2)
			copy(bobPublic, fixedBytes(bobX, curve.size))
			copy(bobPublic[curve.size:], fixedBytes(bobY, curve.size))

			aliceSecret, err := curve.sharedSecret(bobPublic, alicePriv)
			if err != nil {
				t.Fatalf("alice shared secret failed: %v", err)
			}
			bobSecret, err := curve.sharedSecret(alicePublic, bobPriv)
			if err != nil {
				t.Fatalf("bob shared secret failed: %v", err)
			}

			if !bytes.Equal(fixedBytes(aliceSecret, curve.size), fixedBytes(bobSecret, curve.size)) {
				t.Fatalf("shared secrets mismatch")
			}
		})
	}
}

func TestMakeEcdhInfoKeepsLegacyBufferLengths(t *testing.T) {
	curve, err := getNativeECCurve("secp192r1")
	if err != nil {
		t.Fatalf("get curve failed: %v", err)
	}
	_, x, y, err := curve.makeKey()
	if err != nil {
		t.Fatalf("generate server key failed: %v", err)
	}

	serverPublic := make([]byte, curve.size*2)
	copy(serverPublic, fixedBytes(x, curve.size))
	copy(serverPublic[curve.size:], fixedBytes(y, curve.size))

	info, err := makeEcdhInfo(&eccPublicKeyInfo{
		publicKey: serverPublic,
		curveName: "secp192r1",
		keyLen:    curve.size * 2,
	})
	if err != nil {
		t.Fatalf("make ECDH info failed: %v", err)
	}

	if len(info.publicKey) != 64 || len(info.privateKey) != 32 || len(info.secret) != 32 {
		t.Fatalf("unexpected legacy buffer lengths: public=%d private=%d secret=%d",
			len(info.publicKey), len(info.privateKey), len(info.secret))
	}
	if !bytes.Equal(info.publicKey[curve.size*2:], make([]byte, 64-curve.size*2)) {
		t.Fatalf("legacy public key tail should stay zero-filled")
	}
	if !bytes.Equal(info.privateKey[curve.size:], make([]byte, 32-curve.size)) {
		t.Fatalf("legacy private key tail should stay zero-filled")
	}
	if !bytes.Equal(info.secret[curve.size:], make([]byte, 32-curve.size)) {
		t.Fatalf("legacy shared secret tail should stay zero-filled")
	}
}
