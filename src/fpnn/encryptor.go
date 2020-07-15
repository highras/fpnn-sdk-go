package fpnn

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/elliptic"
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

func parseCurveName(rawInfo *pemKeyInfo, keyInfo *eccPublicKeyInfo) error {

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
	err = parseCurveName(&pemKeyInfo, eccKeyInfo)
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

func makeEcdhInfo(serverKeyInfo *eccPublicKeyInfo) (*ecdhInfo, error) {
	var curve elliptic.Curve
	switch serverKeyInfo.curveName {
	case "secp224r1":
		curve = elliptic.P224()
	case "secp256r1":
		curve = elliptic.P256()
	default:
		return nil, fmt.Errorf("Unsupported ECC curve: %s", serverKeyInfo.curveName)
	}
	priv, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		fmt.Printf("Failed to generate Alice's private key pair: %s\n", err)
	}

	pub := priv.PublicKey
	x := new(big.Int).SetBytes(serverKeyInfo.publicKey[0:32])
	y := new(big.Int).SetBytes(serverKeyInfo.publicKey[32:])

	secret, _ := pub.Curve.ScalarMult(x, y, priv.D.Bytes())

	info := &ecdhInfo{}
	info.secret = secret.Bytes()
	xy := make([]byte, 64)
	copy(xy, append(pub.X.Bytes(), pub.Y.Bytes()...))
	info.publicKey = xy
	info.privateKey = priv.D.Bytes()

	return info, nil
}

type encryptor struct {
	stream cipher.Stream
	iv     []byte
}

func newEncryptor(secret []byte, bits int, encrypt bool) *encryptor {
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
	enc.iv = make([]byte, 16)

	rawIv := md5.Sum(secret)
	copy(enc.iv, rawIv[:])

	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}
	var stream cipher.Stream
	if encrypt {
		stream = cipher.NewCFBEncrypter(block, enc.iv)
	} else {
		stream = cipher.NewCFBDecrypter(block, enc.iv)
	}

	enc.stream = stream
	return enc
}

func (enc *encryptor) encrypt(data []byte) []byte {
	enc.stream.XORKeyStream(data, data)
	return data
}

func (dec *encryptor) decrypt(data []byte) []byte {
	dec.stream.XORKeyStream(data, data)
	return data
}
