package fpnn

/*
#include <stdlib.h>
#include "aes/aesBridge.c"
#include "aes/rijndael.c"
#include "micro-ecc/uECC.c"
*/
import "C"

import (
	"fmt"
	"errors"
	"unsafe"
	"runtime"
	"io/ioutil"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"crypto/sha256"
	"crypto/md5"
)

var (
	oidPublicKeyECC = asn1.ObjectIdentifier{1, 2, 840, 10045, 2, 1}

	oidNamedCurve224r1 = asn1.ObjectIdentifier{1, 3, 132, 0, 33}
	oidNamedCurve192r1 = asn1.ObjectIdentifier{1,2,840,10045,3,1,1}
	oidNamedCurve256r1 = asn1.ObjectIdentifier{1,2,840,10045,3,1,7}
	oidNamedCurve256k1 = asn1.ObjectIdentifier{1,3,132,0,10}
)

type pemKeyInfo struct {
	Raw       asn1.RawContent
	Algorithm pkix.AlgorithmIdentifier
	PublicKey asn1.BitString
}

type eccPublicKeyInfo struct {
	publicKey	[]byte
	curveName	string
	keyLen		int
}

func praseCurveName(rawInfo *pemKeyInfo, keyInfo *eccPublicKeyInfo) error {

	paramsData := rawInfo.Algorithm.Parameters.FullBytes
	namedCurveOID := new(asn1.ObjectIdentifier)
	rest, err := asn1.Unmarshal(paramsData, namedCurveOID)
	if err != nil {
		return errors.New("x509: failed to parse ECC parameters as named curve")
	}
	if len(rest) != 0 {
		return errors.New("x509: trailing data after ECC parameters")
	}

	oid := *namedCurveOID

	switch {
	case oid.Equal(oidNamedCurve224r1):
		keyInfo.curveName = "secp224r1"
		keyInfo.keyLen = 28 * 2;
	case oid.Equal(oidNamedCurve192r1):
		keyInfo.curveName = "secp192r1"
		keyInfo.keyLen = 24 * 2;
	case oid.Equal(oidNamedCurve256r1):
		keyInfo.curveName = "secp256r1"
		keyInfo.keyLen = 32 * 2;
	case oid.Equal(oidNamedCurve256k1):
		keyInfo.curveName = "secp256k1"
		keyInfo.keyLen = 32 * 2;
	default:
		return errors.New("Unsupported ECC curve.")
	}
	return nil
}

func loadEccPublicKeyFromPemFile(pemFilePath string) (*eccPublicKeyInfo, error) {

	fileData, err := ioutil.ReadFile(pemFilePath)
	if err != nil {
		return nil, err
	}

	return extraEccPublicKeyFromPemData(fileData)
}

func extraEccPublicKeyFromPemData(rawPemData []byte) (*eccPublicKeyInfo, error) {

	pemData, _ := pem.Decode(rawPemData)
	if pemData == nil {
		return nil, errors.New("Invalid pem data.")
	}

	var pemKeyInfo pemKeyInfo
	_, err := asn1.Unmarshal(pemData.Bytes, &pemKeyInfo)
	if err != nil {
		return nil, err
	}

	if !pemKeyInfo.Algorithm.Algorithm.Equal(oidPublicKeyECC) {
		return nil, errors.New("PEM data is not ECC key.")
	}

	eccKeyInfo := &eccPublicKeyInfo{}
	err = praseCurveName(&pemKeyInfo, eccKeyInfo)
	if err != nil {
		return nil, err
	}

	if pemKeyInfo.PublicKey.Bytes[0] != 4 {
		return nil, errors.New("ECC public key error. Requrie uncompressed public key.")
	}

	eccKeyInfo.publicKey = pemKeyInfo.PublicKey.Bytes[1:]

	return eccKeyInfo, nil
}

type ecdhInfo struct {
	secret		[]byte
	publicKey	[]byte
	privateKey	[]byte
}

func makeEcdhInfo(serverKeyInfo *eccPublicKeyInfo) (*ecdhInfo, error) {
	
	var curve C.uECC_Curve

	switch serverKeyInfo.curveName {
	case "secp192r1":
		curve, _ = C.uECC_secp192r1()
	case "secp224r1":
		curve, _ = C.uECC_secp224r1()
	case "secp256r1":
		curve, _ = C.uECC_secp256r1()
	case "secp256k1":
		curve, _ = C.uECC_secp256k1()
	default:
		return nil, fmt.Errorf("Unsupported ECC curve: %s", serverKeyInfo.curveName)
	}

	info := &ecdhInfo{}
	info.secret = make([]byte, 32)
	info.publicKey = make([]byte, 64)
	info.privateKey = make([]byte, 32)

	rev, _ := C.uECC_make_key((*C.uchar)(unsafe.Pointer(&info.publicKey[0])), (*C.uchar)(unsafe.Pointer(&info.privateKey[0])), curve)
	if rev == 0 {
		return nil, fmt.Errorf("Generate ECC key pair failed, uECC_make_key() failed.")
	}

	rev, _ = C.uECC_shared_secret((*C.uchar)(unsafe.Pointer(&serverKeyInfo.publicKey[0])),
		(*C.uchar)(unsafe.Pointer(&info.privateKey[0])), (*C.uchar)(unsafe.Pointer(&info.secret[0])), curve)
	if rev == 0 {
		return nil, fmt.Errorf("Generate ECC shared secret failed, uECC_shared_secret() failed.")
	}

	return info, nil
}

type encryptor struct {
	aesCtx		unsafe.Pointer
	goPos		uint64
	iv			[]byte
}

func newEncryptor(secret []byte, bits int) *encryptor {

	var key []byte

	if bits == 256 {
		if len(secret) >= 32 {
			key = secret[:32]
		} else {
			srcKey := sha256.Sum256(secret)
			key = make([]byte, 32)
			copy(key, srcKey[:])
		}
	} else if bits == 128 {
		key = secret[:16]
	} else {
		panic("Invalid bits for AES encryption.")
	}

	enc := &encryptor{}
	enc.goPos = 0
	enc.iv = make([]byte, 16)
	
	aesCtx, _ := C.mallocAESCtx()
	C.rijndael_setup_encrypt(aesCtx, (*C.uchar)(unsafe.Pointer(&key[0])), C.size_t(len(key)))

	rawIv := md5.Sum(secret)
	copy(enc.iv, rawIv[:])
	enc.aesCtx = unsafe.Pointer(aesCtx)

	runtime.SetFinalizer(enc, closeEncryptor)
	return enc
}

func closeEncryptor(enc *encryptor) {
	C.free(enc.aesCtx)
}

func (enc *encryptor) encrypt(data []byte) []byte {

	srclen := len(data)
	encBuf := make([]byte, srclen)
	aesCtx := C.ctxPtr(enc.aesCtx)

	pos := (C.size_t)(enc.goPos)
	C.rijndael_cfb_encrypt(aesCtx, true, (*C.uchar)(unsafe.Pointer(&data[0])),
		(*C.uchar)(unsafe.Pointer(&encBuf[0])), C.size_t(srclen),
		(*C.uchar)(unsafe.Pointer(&enc.iv[0])), (*C.size_t)(unsafe.Pointer(&pos)))

	enc.goPos = uint64(pos)
	
	return encBuf
}

func (dec *encryptor) decrypt(data []byte) []byte {
	
	srclen := len(data)
	decBuf := make([]byte, srclen)
	aesCtx := C.ctxPtr(dec.aesCtx)

	pos := (C.size_t)(dec.goPos)
	C.rijndael_cfb_encrypt(aesCtx, false, (*C.uchar)(unsafe.Pointer(&data[0])),
		(*C.uchar)(unsafe.Pointer(&decBuf[0])), C.size_t(srclen),
		(*C.uchar)(unsafe.Pointer(&dec.iv[0])), (*C.size_t)(unsafe.Pointer(&pos)))

	dec.goPos = uint64(pos)
	
	return decBuf
}
