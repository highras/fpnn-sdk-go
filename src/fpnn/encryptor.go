package fpnn

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"math/big"
)

var (
	oidPublicKeyECC = asn1.ObjectIdentifier{1, 2, 840, 10045, 2, 1}

	oidNamedCurve224r1 = asn1.ObjectIdentifier{1, 3, 132, 0, 33}
	oidNamedCurve192r1 = asn1.ObjectIdentifier{1, 2, 840, 10045, 3, 1, 1}
	oidNamedCurve256r1 = asn1.ObjectIdentifier{1, 2, 840, 10045, 3, 1, 7}
	oidNamedCurve256k1 = asn1.ObjectIdentifier{1, 3, 132, 0, 10}
)

type pemKeyInfo struct {
	Raw       asn1.RawContent
	Algorithm pkix.AlgorithmIdentifier
	PublicKey asn1.BitString
}

type eccPublicKeyInfo struct {
	publicKey []byte
	curveName string
	keyLen    int
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
		keyInfo.keyLen = 28 * 2
	case oid.Equal(oidNamedCurve192r1):
		keyInfo.curveName = "secp192r1"
		keyInfo.keyLen = 24 * 2
	case oid.Equal(oidNamedCurve256r1):
		keyInfo.curveName = "secp256r1"
		keyInfo.keyLen = 32 * 2
	case oid.Equal(oidNamedCurve256k1):
		keyInfo.curveName = "secp256k1"
		keyInfo.keyLen = 32 * 2
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
	secret     []byte
	publicKey  []byte
	privateKey []byte
}

type nativeECCurve struct {
	size int
	p    *big.Int
	n    *big.Int
	a    *big.Int
	b    *big.Int
	gx   *big.Int
	gy   *big.Int
}

func makeEcdhInfo(serverKeyInfo *eccPublicKeyInfo) (*ecdhInfo, error) {

	curve, err := getNativeECCurve(serverKeyInfo.curveName)
	if err != nil {
		return nil, err
	}

	info := &ecdhInfo{}
	info.secret = make([]byte, 32)
	info.publicKey = make([]byte, 64)
	info.privateKey = make([]byte, 32)

	privateKey, x, y, err := curve.makeKey()
	if err != nil {
		return nil, fmt.Errorf("Generate ECC key pair failed: %w", err)
	}
	copy(info.privateKey, fixedBytes(privateKey, curve.size))
	copy(info.publicKey, fixedBytes(x, curve.size))
	copy(info.publicKey[curve.size:], fixedBytes(y, curve.size))

	secret, err := curve.sharedSecret(serverKeyInfo.publicKey, privateKey)
	if err != nil {
		return nil, fmt.Errorf("Generate ECC shared secret failed: %w", err)
	}
	copy(info.secret, fixedBytes(secret, curve.size))

	return info, nil
}

func getNativeECCurve(name string) (*nativeECCurve, error) {
	params := map[string]struct {
		size        int
		p, n, b     string
		gx, gy      string
		secpK1Curve bool
	}{
		"secp192r1": {
			24,
			"FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEFFFFFFFFFFFFFFFF",
			"FFFFFFFFFFFFFFFFFFFFFFFF99DEF836146BC9B1B4D22831",
			"64210519E59C80E70FA7E9AB72243049FEB8DEECC146B9B1",
			"188DA80EB03090F67CBF20EB43A18800F4FF0AFD82FF1012",
			"07192B95FFC8DA78631011ED6B24CDD573F977A11E794811",
			false,
		},
		"secp224r1": {
			28,
			"FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF000000000000000000000001",
			"FFFFFFFFFFFFFFFFFFFFFFFFFFFF16A2E0B8F03E13DD29455C5C2A3D",
			"B4050A850C04B3ABF54132565044B0B7D7BFD8BA270B39432355FFB4",
			"B70E0CBD6BB4BF7F321390B94A03C1D356C21122343280D6115C1D21",
			"BD376388B5F723FB4C22DFE6CD4375A05A07476444D5819985007E34",
			false,
		},
		"secp256r1": {
			32,
			"FFFFFFFF00000001000000000000000000000000FFFFFFFFFFFFFFFFFFFFFFFF",
			"FFFFFFFF00000000FFFFFFFFFFFFFFFFBCE6FAADA7179E84F3B9CAC2FC632551",
			"5AC635D8AA3A93E7B3EBBD55769886BC651D06B0CC53B0F63BCE3C3E27D2604B",
			"6B17D1F2E12C4247F8BCE6E563A440F277037D812DEB33A0F4A13945D898C296",
			"4FE342E2FE1A7F9B8EE7EB4A7C0F9E162BCE33576B315ECECBB6406837BF51F5",
			false,
		},
		"secp256k1": {
			32,
			"FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEFFFFFC2F",
			"FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEBAAEDCE6AF48A03BBFD25E8CD0364141",
			"7",
			"79BE667EF9DCBBAC55A06295CE870B07029BFCDB2DCE28D959F2815B16F81798",
			"483ADA7726A3C4655DA4FBFC0E1108A8FD17B448A68554199C47D08FFB10D4B8",
			true,
		},
	}

	param, ok := params[name]
	if !ok {
		return nil, fmt.Errorf("Unsupported ECC curve: %s", name)
	}

	curve := &nativeECCurve{
		size: param.size,
		p:    hexToBig(param.p),
		n:    hexToBig(param.n),
		b:    hexToBig(param.b),
		gx:   hexToBig(param.gx),
		gy:   hexToBig(param.gy),
	}
	if param.secpK1Curve {
		curve.a = big.NewInt(0)
	} else {
		curve.a = new(big.Int).Sub(curve.p, big.NewInt(3))
	}
	return curve, nil
}

func (curve *nativeECCurve) makeKey() (*big.Int, *big.Int, *big.Int, error) {
	one := big.NewInt(1)
	max := new(big.Int).Sub(curve.n, one)
	privateKey, err := rand.Int(rand.Reader, max)
	if err != nil {
		return nil, nil, nil, err
	}
	privateKey.Add(privateKey, one)

	x, y := curve.scalarMult(curve.gx, curve.gy, privateKey.Bytes())
	if x == nil || y == nil {
		return nil, nil, nil, errors.New("generated invalid ECC public key")
	}

	return privateKey, x, y, nil
}

func (curve *nativeECCurve) sharedSecret(publicKey []byte, privateKey *big.Int) (*big.Int, error) {
	if len(publicKey) < curve.size*2 {
		return nil, errors.New("invalid ECC public key length")
	}

	x := new(big.Int).SetBytes(publicKey[:curve.size])
	y := new(big.Int).SetBytes(publicKey[curve.size : curve.size*2])
	if !curve.isOnCurve(x, y) {
		return nil, errors.New("server ECC public key is not on curve")
	}

	secret, _ := curve.scalarMult(x, y, privateKey.Bytes())
	if secret == nil {
		return nil, errors.New("invalid ECC shared point")
	}
	return secret, nil
}

func (curve *nativeECCurve) scalarMult(x, y *big.Int, scalar []byte) (*big.Int, *big.Int) {
	var rx, ry *big.Int
	infinity := true

	for _, b := range scalar {
		for bit := 7; bit >= 0; bit-- {
			if !infinity {
				rx, ry, infinity = curve.double(rx, ry)
			}
			if ((b >> uint(bit)) & 1) == 1 {
				rx, ry, infinity = curve.add(rx, ry, infinity, x, y, false)
			}
		}
	}

	if infinity {
		return nil, nil
	}
	return rx, ry
}

func (curve *nativeECCurve) add(x1, y1 *big.Int, inf1 bool, x2, y2 *big.Int, inf2 bool) (*big.Int, *big.Int, bool) {
	if inf1 {
		return new(big.Int).Set(x2), new(big.Int).Set(y2), inf2
	}
	if inf2 {
		return new(big.Int).Set(x1), new(big.Int).Set(y1), inf1
	}
	if x1.Cmp(x2) == 0 {
		sumY := new(big.Int).Add(y1, y2)
		sumY.Mod(sumY, curve.p)
		if sumY.Sign() == 0 {
			return nil, nil, true
		}
		return curve.double(x1, y1)
	}

	numerator := new(big.Int).Sub(y2, y1)
	denominator := new(big.Int).Sub(x2, x1)
	return curve.finishAdd(x1, y1, x2, numerator, denominator)
}

func (curve *nativeECCurve) double(x, y *big.Int) (*big.Int, *big.Int, bool) {
	if y.Sign() == 0 {
		return nil, nil, true
	}

	numerator := new(big.Int).Mul(x, x)
	numerator.Mul(numerator, big.NewInt(3))
	numerator.Add(numerator, curve.a)
	denominator := new(big.Int).Mul(y, big.NewInt(2))
	return curve.finishAdd(x, y, x, numerator, denominator)
}

func (curve *nativeECCurve) finishAdd(x1, y1, x2, numerator, denominator *big.Int) (*big.Int, *big.Int, bool) {
	denominator.Mod(denominator, curve.p)
	inverse := new(big.Int).ModInverse(denominator, curve.p)
	if inverse == nil {
		return nil, nil, true
	}

	lambda := numerator.Mul(numerator, inverse)
	lambda.Mod(lambda, curve.p)

	x3 := new(big.Int).Mul(lambda, lambda)
	x3.Sub(x3, x1)
	x3.Sub(x3, x2)
	x3.Mod(x3, curve.p)

	y3 := new(big.Int).Sub(x1, x3)
	y3.Mul(lambda, y3)
	y3.Sub(y3, y1)
	y3.Mod(y3, curve.p)

	return x3, y3, false
}

func (curve *nativeECCurve) isOnCurve(x, y *big.Int) bool {
	if x.Sign() < 0 || x.Cmp(curve.p) >= 0 || y.Sign() < 0 || y.Cmp(curve.p) >= 0 {
		return false
	}

	left := new(big.Int).Mul(y, y)
	left.Mod(left, curve.p)

	right := new(big.Int).Mul(x, x)
	right.Mul(right, x)
	ax := new(big.Int).Mul(curve.a, x)
	right.Add(right, ax)
	right.Add(right, curve.b)
	right.Mod(right, curve.p)

	return left.Cmp(right) == 0
}

func hexToBig(value string) *big.Int {
	result, ok := new(big.Int).SetString(value, 16)
	if !ok {
		panic("invalid ECC curve parameter")
	}
	return result
}

func fixedBytes(value *big.Int, size int) []byte {
	data := value.Bytes()
	if len(data) == size {
		return data
	}

	result := make([]byte, size)
	if len(data) > size {
		copy(result, data[len(data)-size:])
	} else {
		copy(result[size-len(data):], data)
	}
	return result
}

type encryptor struct {
	encrypter cipher.Stream
	decrypter cipher.Stream
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

	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}
	rawIv := md5.Sum(secret)

	encIv := make([]byte, aes.BlockSize)
	decIv := make([]byte, aes.BlockSize)
	copy(encIv, rawIv[:])
	copy(decIv, rawIv[:])

	return &encryptor{
		encrypter: cipher.NewCFBEncrypter(block, encIv),
		decrypter: cipher.NewCFBDecrypter(block, decIv),
	}
}

func (enc *encryptor) encrypt(data []byte) []byte {
	encBuf := make([]byte, len(data))
	enc.encrypter.XORKeyStream(encBuf, data)
	return encBuf
}

func (dec *encryptor) decrypt(data []byte) []byte {
	decBuf := make([]byte, len(data))
	dec.decrypter.XORKeyStream(decBuf, data)
	return decBuf
}
