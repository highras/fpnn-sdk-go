//go:build cgo && microecc_compare

package fpnn

import (
	"bytes"
	"crypto/rand"
	"math/big"
	"testing"
)

func TestNativeECCMatchesMicroECCPublicKey(t *testing.T) {
	for _, curveName := range []string{"secp192r1", "secp224r1", "secp256r1", "secp256k1"} {
		t.Run(curveName, func(t *testing.T) {
			nativeCurve, err := getNativeECCurve(curveName)
			if err != nil {
				t.Fatalf("get curve failed: %v", err)
			}

			for i := 0; i < 32; i++ {
				privateKey := randomECCPrivateKey(t, nativeCurve)
				nativeX, nativeY := nativeCurve.scalarMult(nativeCurve.gx, nativeCurve.gy, privateKey.Bytes())
				if nativeX == nil || nativeY == nil {
					t.Fatalf("native public key generation failed")
				}
				nativePublicKey := append(fixedBytes(nativeX, nativeCurve.size), fixedBytes(nativeY, nativeCurve.size)...)

				privateKeyBytes := fixedBytes(privateKey, nativeCurve.size)
				microPublicKey, err := microECCPublicKey(curveName, privateKeyBytes, nativeCurve.size)
				if err != nil {
					t.Fatal(err)
				}

				if !bytes.Equal(nativePublicKey, microPublicKey) {
					t.Fatalf("public key mismatch\nnative: %x\nmicro:  %x", nativePublicKey, microPublicKey)
				}
			}
		})
	}
}

func TestNativeECCMatchesMicroECCSharedSecret(t *testing.T) {
	for _, curveName := range []string{"secp192r1", "secp224r1", "secp256r1", "secp256k1"} {
		t.Run(curveName, func(t *testing.T) {
			nativeCurve, err := getNativeECCurve(curveName)
			if err != nil {
				t.Fatalf("get curve failed: %v", err)
			}

			for i := 0; i < 32; i++ {
				alicePrivateKey := randomECCPrivateKey(t, nativeCurve)
				bobPrivateKey := randomECCPrivateKey(t, nativeCurve)

				alicePrivateKeyBytes := fixedBytes(alicePrivateKey, nativeCurve.size)
				bobPrivateKeyBytes := fixedBytes(bobPrivateKey, nativeCurve.size)

				alicePublicKey, err := microECCPublicKey(curveName, alicePrivateKeyBytes, nativeCurve.size)
				if err != nil {
					t.Fatal(err)
				}
				bobPublicKey, err := microECCPublicKey(curveName, bobPrivateKeyBytes, nativeCurve.size)
				if err != nil {
					t.Fatal(err)
				}

				nativeAliceSecret, err := nativeCurve.sharedSecret(bobPublicKey, alicePrivateKey)
				if err != nil {
					t.Fatalf("native alice shared secret failed: %v", err)
				}
				nativeBobSecret, err := nativeCurve.sharedSecret(alicePublicKey, bobPrivateKey)
				if err != nil {
					t.Fatalf("native bob shared secret failed: %v", err)
				}
				nativeAliceSecretBytes := fixedBytes(nativeAliceSecret, nativeCurve.size)
				nativeBobSecretBytes := fixedBytes(nativeBobSecret, nativeCurve.size)
				if !bytes.Equal(nativeAliceSecretBytes, nativeBobSecretBytes) {
					t.Fatalf("native ECDH round trip mismatch\nalice: %x\nbob:   %x", nativeAliceSecretBytes, nativeBobSecretBytes)
				}

				microAliceSecret, err := microECCSharedSecret(curveName, bobPublicKey, alicePrivateKeyBytes, nativeCurve.size)
				if err != nil {
					t.Fatal(err)
				}
				microBobSecret, err := microECCSharedSecret(curveName, alicePublicKey, bobPrivateKeyBytes, nativeCurve.size)
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(microAliceSecret, microBobSecret) {
					t.Fatalf("micro-ecc ECDH round trip mismatch\nalice: %x\nbob:   %x", microAliceSecret, microBobSecret)
				}
				if !bytes.Equal(nativeAliceSecretBytes, microAliceSecret) {
					t.Fatalf("shared secret mismatch\nnative: %x\nmicro:  %x", nativeAliceSecretBytes, microAliceSecret)
				}
			}
		})
	}
}

func TestNativeAESMatchesOriginalCImplementation(t *testing.T) {
	testCases := []struct {
		name   string
		secret []byte
		bits   int
		chunks [][]byte
	}{
		{
			name:   "AES128 segmented",
			secret: []byte("1234567890abcdef1234567890abcdef"),
			bits:   128,
			chunks: [][]byte{
				[]byte("123456789 002 34"),
				[]byte("A"),
				[]byte("bbe"),
				[]byte("ee erewr"),
				{},
				[]byte("tail data"),
			},
		},
		{
			name:   "AES256 segmented",
			secret: []byte("short-secret-for-sha256"),
			bits:   256,
			chunks: [][]byte{
				[]byte("hello"),
				[]byte("0123456789abcdef"),
				[]byte("cross-block-boundary-data"),
				{},
				[]byte("final"),
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			nativeEncryptor := newEncryptor(testCase.secret, testCase.bits)
			microEncryptor, err := newMicroAESCipher(testCase.secret, testCase.bits)
			if err != nil {
				t.Fatal(err)
			}
			defer microEncryptor.close()

			encryptedChunks := make([][]byte, 0, len(testCase.chunks))
			for _, chunk := range testCase.chunks {
				nativeEncrypted := nativeEncryptor.encrypt(chunk)
				microEncrypted := microEncryptor.encrypt(chunk)
				if !bytes.Equal(nativeEncrypted, microEncrypted) {
					t.Fatalf("encrypt mismatch for chunk %q\nnative: %x\nmicro:  %x", chunk, nativeEncrypted, microEncrypted)
				}
				encryptedChunks = append(encryptedChunks, nativeEncrypted)
			}

			nativeDecryptor := newEncryptor(testCase.secret, testCase.bits)
			microDecryptor, err := newMicroAESCipher(testCase.secret, testCase.bits)
			if err != nil {
				t.Fatal(err)
			}
			defer microDecryptor.close()

			for i, encryptedChunk := range encryptedChunks {
				nativePlaintext := nativeDecryptor.decrypt(encryptedChunk)
				microPlaintext := microDecryptor.decrypt(encryptedChunk)
				if !bytes.Equal(nativePlaintext, microPlaintext) {
					t.Fatalf("decrypt mismatch for chunk %d\nnative: %x\nmicro:  %x", i, nativePlaintext, microPlaintext)
				}
				if !bytes.Equal(nativePlaintext, testCase.chunks[i]) {
					t.Fatalf("decrypt round trip mismatch for chunk %d\nplain: %x\nwant:  %x", i, nativePlaintext, testCase.chunks[i])
				}
			}
		})
	}
}

func randomECCPrivateKey(t *testing.T, curve *nativeECCurve) *big.Int {
	t.Helper()

	one := big.NewInt(1)
	max := new(big.Int).Sub(curve.n, one)
	privateKey, err := rand.Int(rand.Reader, max)
	if err != nil {
		t.Fatalf("random private key failed: %v", err)
	}
	return privateKey.Add(privateKey, one)
}
