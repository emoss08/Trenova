package smime

import (
	"bytes"
	"crypto"
	"crypto/hmac"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/asn1"
	"io"
)

const rejectionKeyLabel = "trenova smime implicit rejection"

type implicitRejectionKey struct {
	crypto.Decrypter
	sessionKeyLen int
}

func (k implicitRejectionKey) Decrypt(
	random io.Reader,
	ciphertext []byte,
	opts crypto.DecrypterOpts,
) ([]byte, error) {
	if _, ok := opts.(*rsa.PKCS1v15DecryptOptions); ok {
		opts = &rsa.PKCS1v15DecryptOptions{SessionKeyLen: k.sessionKeyLen}
		if key, isRSA := k.Decrypter.(*rsa.PrivateKey); isRSA {
			random = rejectionKeySource(key, ciphertext, k.sessionKeyLen)
		}
	}

	return k.Decrypter.Decrypt(random, ciphertext, opts)
}

func rejectionKeySource(key *rsa.PrivateKey, ciphertext []byte, size int) io.Reader {
	secret := sha256.Sum256(append([]byte(rejectionKeyLabel), key.D.Bytes()...))
	mac := hmac.New(sha256.New, secret[:])
	mac.Write(ciphertext)

	return bytes.NewReader(mac.Sum(nil)[:size])
}

func withImplicitRejection(
	key crypto.PrivateKey,
	algorithm asn1.ObjectIdentifier,
) crypto.PrivateKey {
	decrypter, ok := key.(crypto.Decrypter)
	if !ok {
		return key
	}

	for _, selected := range contentCiphers {
		if selected.oid.Equal(algorithm) {
			return implicitRejectionKey{Decrypter: decrypter, sessionKeyLen: selected.keySize}
		}
	}

	return key
}
