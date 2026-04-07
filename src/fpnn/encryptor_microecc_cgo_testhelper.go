//go:build cgo && microecc_compare

package fpnn

/*
#include <stdlib.h>
#include "aes/aesBridge.c"
#include "aes/rijndael.c"
#include "micro-ecc/uECC.c"
*/
import "C"

import (
	"crypto/md5"
	"crypto/sha256"
	"fmt"
	"unsafe"
)

type microAESCipher struct {
	ctx unsafe.Pointer
	iv  []byte
	pos C.size_t
}

func newMicroAESCipher(secret []byte, bits int) (*microAESCipher, error) {
	var key []byte

	if bits == 256 {
		if len(secret) >= 32 {
			key = secret[:32]
		} else {
			srcKey := sha256.Sum256(secret)
			key = srcKey[:]
		}
	} else if bits == 128 {
		key = secret[:16]
	} else {
		return nil, fmt.Errorf("invalid AES bits: %d", bits)
	}

	aesCtx, _ := C.mallocAESCtx()
	C.rijndael_setup_encrypt(aesCtx, (*C.uchar)(unsafe.Pointer(&key[0])), C.size_t(len(key)))

	rawIv := md5.Sum(secret)
	iv := make([]byte, 16)
	copy(iv, rawIv[:])

	return &microAESCipher{
		ctx: unsafe.Pointer(aesCtx),
		iv:  iv,
	}, nil
}

func (cipher *microAESCipher) close() {
	C.free(cipher.ctx)
}

func (cipher *microAESCipher) encrypt(data []byte) []byte {
	return cipher.crypt(true, data)
}

func (cipher *microAESCipher) decrypt(data []byte) []byte {
	return cipher.crypt(false, data)
}

func (cipher *microAESCipher) crypt(encrypt bool, data []byte) []byte {
	output := make([]byte, len(data))
	if len(data) == 0 {
		return output
	}

	aesCtx := C.ctxPtr(cipher.ctx)
	C.rijndael_cfb_encrypt(aesCtx, C.bool(encrypt),
		(*C.uchar)(unsafe.Pointer(&data[0])),
		(*C.uchar)(unsafe.Pointer(&output[0])),
		C.size_t(len(data)),
		(*C.uchar)(unsafe.Pointer(&cipher.iv[0])),
		(*C.size_t)(unsafe.Pointer(&cipher.pos)))

	return output
}

func microECCPublicKey(curveName string, privateKey []byte, keySize int) ([]byte, error) {
	curve, err := microECCCurve(curveName)
	if err != nil {
		return nil, err
	}

	publicKey := make([]byte, keySize*2)
	if ret, _ := C.uECC_compute_public_key(
		(*C.uchar)(unsafe.Pointer(&privateKey[0])),
		(*C.uchar)(unsafe.Pointer(&publicKey[0])),
		curve,
	); ret == 0 {
		return nil, fmt.Errorf("micro-ecc public key generation failed")
	}
	return publicKey, nil
}

func microECCSharedSecret(curveName string, publicKey []byte, privateKey []byte, keySize int) ([]byte, error) {
	curve, err := microECCCurve(curveName)
	if err != nil {
		return nil, err
	}

	secret := make([]byte, keySize)
	if ret, _ := C.uECC_shared_secret(
		(*C.uchar)(unsafe.Pointer(&publicKey[0])),
		(*C.uchar)(unsafe.Pointer(&privateKey[0])),
		(*C.uchar)(unsafe.Pointer(&secret[0])),
		curve,
	); ret == 0 {
		return nil, fmt.Errorf("micro-ecc shared secret generation failed")
	}
	return secret, nil
}

func microECCCurve(curveName string) (C.uECC_Curve, error) {
	switch curveName {
	case "secp192r1":
		curve, _ := C.uECC_secp192r1()
		return curve, nil
	case "secp224r1":
		curve, _ := C.uECC_secp224r1()
		return curve, nil
	case "secp256r1":
		curve, _ := C.uECC_secp256r1()
		return curve, nil
	case "secp256k1":
		curve, _ := C.uECC_secp256k1()
		return curve, nil
	default:
		return nil, fmt.Errorf("unsupported curve: %s", curveName)
	}
}
