package smime

import (
	"crypto/cipher"
	"crypto/des"
	"crypto/rsa"
	"crypto/x509"
	"encoding/asn1"
	"fmt"
	"sync"
	"testing"

	"github.com/emoss08/trenova/shared/smimetest"
	"github.com/smallstep/pkcs7"
	"github.com/stretchr/testify/require"
)

type parsedEnvelope struct {
	contentInfo envelopeContentInfo
	enveloped   envelopedData
}

func parseEnvelopeDER(t *testing.T, der []byte) parsedEnvelope {
	t.Helper()
	var info envelopeContentInfo
	rest, err := asn1.Unmarshal(der, &info)
	require.NoError(t, err)
	require.Empty(t, rest)
	require.True(t, info.ContentType.Equal(oidEnvelopedData))
	require.Equal(t, asn1.ClassContextSpecific, info.Content.Class)
	require.True(t, info.Content.IsCompound)

	var enveloped envelopedData
	rest, err = asn1.Unmarshal(info.Content.Bytes, &enveloped)
	require.NoError(t, err)
	require.Empty(t, rest)
	return parsedEnvelope{contentInfo: info, enveloped: enveloped}
}

func TestEncryptTripleDESProducesDESEDE3CBC(t *testing.T) {
	t.Parallel()

	receiver := smimetest.NewIdentity(t, "receiver.example")
	plaintext := []byte(smimetest.SamplePayload)

	der, err := Encrypt(plaintext, receiver.Certificate, EncryptionAlgorithmTripleDES)
	require.NoError(t, err)

	parsed := parseEnvelopeDER(t, der)
	eci := parsed.enveloped.EncryptedContentInfo
	require.Equal(t, "1.2.840.113549.3.7", eci.ContentEncryptionAlgorithm.Algorithm.String())
	require.False(t, eci.ContentEncryptionAlgorithm.Algorithm.Equal(oidDESCBC))

	var iv []byte
	_, err = asn1.Unmarshal(eci.ContentEncryptionAlgorithm.Parameters.FullBytes, &iv)
	require.NoError(t, err)
	require.Len(t, iv, des.BlockSize)

	require.Len(t, parsed.enveloped.RecipientInfos, 1)
	recipient := parsed.enveloped.RecipientInfos[0]
	require.True(t, recipient.KeyEncryptionAlgorithm.Algorithm.Equal(oidRSAEncryption))
	require.Equal(t, 0, recipient.IssuerAndSerialNumber.SerialNumber.Cmp(
		receiver.Certificate.SerialNumber,
	))

	contentKey, err := rsa.DecryptPKCS1v15(nil, receiver.Key, recipient.EncryptedKey)
	require.NoError(t, err)
	require.Len(t, contentKey, 24)
	require.NotEqual(t, contentKey[:8], contentKey[8:16])
	require.NotEqual(t, contentKey[8:16], contentKey[16:])

	block, err := des.NewTripleDESCipher(contentKey)
	require.NoError(t, err)
	ciphertext := eci.EncryptedContent.Bytes
	require.Zero(t, len(ciphertext)%des.BlockSize)
	decrypted := make([]byte, len(ciphertext))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(decrypted, ciphertext)
	padding := int(decrypted[len(decrypted)-1])
	require.Equal(t, plaintext, decrypted[:len(decrypted)-padding])
}

func TestEncryptEveryAlgorithmUsesItsOIDAndKeyLength(t *testing.T) {
	t.Parallel()

	receiver := smimetest.NewIdentity(t, "receiver.example")
	cases := []struct {
		algorithm string
		oid       string
		keyLength int
	}{
		{EncryptionAlgorithmTripleDES, "1.2.840.113549.3.7", 24},
		{EncryptionAlgorithmAES128CBC, "2.16.840.1.101.3.4.1.2", 16},
		{EncryptionAlgorithmAES256CBC, "2.16.840.1.101.3.4.1.42", 32},
		{EncryptionAlgorithmAES128GCM, "2.16.840.1.101.3.4.1.6", 16},
		{EncryptionAlgorithmAES256GCM, "2.16.840.1.101.3.4.1.46", 32},
		{"", "2.16.840.1.101.3.4.1.42", 32},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("algorithm=%q", tc.algorithm), func(t *testing.T) {
			t.Parallel()

			der, err := Encrypt([]byte(smimetest.SamplePayload), receiver.Certificate, tc.algorithm)
			require.NoError(t, err)

			parsed := parseEnvelopeDER(t, der)
			algorithm := parsed.enveloped.EncryptedContentInfo.ContentEncryptionAlgorithm.Algorithm
			require.Equal(t, tc.oid, algorithm.String())

			contentKey, err := rsa.DecryptPKCS1v15(
				nil,
				receiver.Key,
				parsed.enveloped.RecipientInfos[0].EncryptedKey,
			)
			require.NoError(t, err)
			require.Len(t, contentKey, tc.keyLength)

			detected, err := envelopeContentAlgorithm(der)
			require.NoError(t, err)
			require.Equal(t, tc.oid, detected.String())

			library, err := pkcs7.Parse(der)
			require.NoError(t, err)
			fromLibrary, err := library.Decrypt(receiver.Certificate, receiver.Key)
			require.NoError(t, err)
			require.Equal(t, smimetest.SamplePayload, string(fromLibrary))

			decrypted, err := Decrypt(der, receiver.Certificate, receiver.Key)
			require.NoError(t, err)
			require.Equal(t, smimetest.SamplePayload, string(decrypted))
		})
	}
}

func TestNoSupportedAlgorithmEmitsSingleDES(t *testing.T) {
	t.Parallel()

	for name, selected := range contentCiphers {
		require.False(t, selected.oid.Equal(oidDESCBC), name)
		require.GreaterOrEqual(t, selected.keySize, 16, name)
	}
}

func TestEncryptRejectsUnknownAlgorithm(t *testing.T) {
	t.Parallel()

	receiver := smimetest.NewIdentity(t, "receiver.example")
	for _, algorithm := range []string{"des", "des-cbc", "rc2", "aes192-cbc"} {
		_, err := Encrypt([]byte(smimetest.SamplePayload), receiver.Certificate, algorithm)
		require.ErrorIs(t, err, ErrUnsupportedEncryptionAlgorithm, algorithm)
	}
}

var singleDESMutex sync.Mutex

func encryptWithLibrarySingleDES(t *testing.T, receiver *smimetest.Identity) []byte {
	t.Helper()
	singleDESMutex.Lock()
	defer singleDESMutex.Unlock()
	previous := pkcs7.ContentEncryptionAlgorithm
	pkcs7.ContentEncryptionAlgorithm = pkcs7.EncryptionAlgorithmDESCBC
	defer func() { pkcs7.ContentEncryptionAlgorithm = previous }()
	der, err := pkcs7.Encrypt(
		[]byte(smimetest.SamplePayload),
		[]*x509.Certificate{receiver.Certificate},
	)
	require.NoError(t, err)
	return der
}

func TestDecryptRejectsSingleDES(t *testing.T) {
	t.Parallel()

	receiver := smimetest.NewIdentity(t, "receiver.example")
	der := encryptWithLibrarySingleDES(t, receiver)

	detected, err := envelopeContentAlgorithm(der)
	require.NoError(t, err)
	require.True(t, detected.Equal(oidDESCBC))

	_, err = Decrypt(der, receiver.Certificate, receiver.Key)
	require.ErrorIs(t, err, ErrInsecureEncryptionAlgorithm)

	_, err = Decrypt(toIndefiniteLength(t, der), receiver.Certificate, receiver.Key)
	require.ErrorIs(t, err, ErrInsecureEncryptionAlgorithm)
}

func TestDecryptReadsIndefiniteLengthEnvelopes(t *testing.T) {
	t.Parallel()

	receiver := smimetest.NewIdentity(t, "receiver.example")
	der, err := Encrypt(
		[]byte(smimetest.SamplePayload),
		receiver.Certificate,
		EncryptionAlgorithmTripleDES,
	)
	require.NoError(t, err)

	ber := toIndefiniteLength(t, der)
	require.NotEqual(t, der, ber)

	detected, err := envelopeContentAlgorithm(ber)
	require.NoError(t, err)
	require.True(t, detected.Equal(oidDESEDE3CBC))

	decrypted, err := Decrypt(ber, receiver.Certificate, receiver.Key)
	require.NoError(t, err)
	require.Equal(t, smimetest.SamplePayload, string(decrypted))
}

func TestEnvelopeContentAlgorithmRejectsMalformedInput(t *testing.T) {
	t.Parallel()

	receiver := smimetest.NewIdentity(t, "receiver.example")
	der, err := Encrypt(
		[]byte(smimetest.SamplePayload),
		receiver.Certificate,
		EncryptionAlgorithmAES256CBC,
	)
	require.NoError(t, err)

	cases := map[string][]byte{
		"empty":          nil,
		"truncated":      der[:len(der)/2],
		"not a sequence": {0x04, 0x01, 0x00},
		"wrong content type": mustMarshal(t, envelopeContentInfo{
			ContentType: oidData,
			Content: asn1.RawValue{
				Class:      asn1.ClassContextSpecific,
				IsCompound: true,
				Bytes:      []byte{0x30, 0x00},
			},
		}),
		"unterminated indefinite": {0x30, 0x80, 0x06, 0x01, 0x2a},
		"deep nesting":            deeplyNested(maxBERDepth + 4),
	}
	for name, input := range cases {
		_, err = envelopeContentAlgorithm(input)
		require.ErrorIs(t, err, errMalformedEnvelope, name)
	}
}

func TestEncryptIsSafeForConcurrentMixedAlgorithms(t *testing.T) {
	t.Parallel()

	receiver := smimetest.NewIdentity(t, "receiver.example")
	algorithms := []string{
		EncryptionAlgorithmTripleDES,
		EncryptionAlgorithmAES128CBC,
		EncryptionAlgorithmAES256CBC,
		EncryptionAlgorithmAES128GCM,
		EncryptionAlgorithmAES256GCM,
	}
	expected := map[string]asn1.ObjectIdentifier{
		EncryptionAlgorithmTripleDES: oidDESEDE3CBC,
		EncryptionAlgorithmAES128CBC: oidAES128CBC,
		EncryptionAlgorithmAES256CBC: oidAES256CBC,
		EncryptionAlgorithmAES128GCM: oidAES128GCM,
		EncryptionAlgorithmAES256GCM: oidAES256GCM,
	}

	const perAlgorithm = 8
	var wg sync.WaitGroup
	errs := make(chan error, len(algorithms)*perAlgorithm)
	for _, algorithm := range algorithms {
		for range perAlgorithm {
			wg.Go(func() {
				der, err := Encrypt(
					[]byte(smimetest.SamplePayload),
					receiver.Certificate,
					algorithm,
				)
				if err != nil {
					errs <- err
					return
				}
				detected, err := envelopeContentAlgorithm(der)
				if err != nil {
					errs <- err
					return
				}
				if !detected.Equal(expected[algorithm]) {
					errs <- fmt.Errorf("%s produced %s", algorithm, detected)
					return
				}
				decrypted, err := Decrypt(der, receiver.Certificate, receiver.Key)
				if err != nil {
					errs <- err
					return
				}
				if string(decrypted) != smimetest.SamplePayload {
					errs <- fmt.Errorf("%s did not round trip", algorithm)
				}
			})
		}
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
}

func mustMarshal(t *testing.T, value any) []byte {
	t.Helper()
	der, err := asn1.Marshal(value)
	require.NoError(t, err)
	return der
}

func deeplyNested(depth int) []byte {
	nested := []byte{0x06, 0x01, 0x2a}
	for range depth {
		nested = append(append([]byte{0x30, 0x80}, nested...), 0x00, 0x00)
	}
	return nested
}

func toIndefiniteLength(t *testing.T, der []byte) []byte {
	t.Helper()
	root, rest, err := readBERElement(der, 0)
	require.NoError(t, err)
	require.Empty(t, rest)
	return reencodeIndefinite(t, root)
}

func reencodeIndefinite(t *testing.T, element berElement) []byte {
	t.Helper()
	if !element.constructed {
		return encodeDefinite(element)
	}
	children, err := element.children(len(element.content))
	require.NoError(t, err)
	out := []byte{byte(element.class<<6) | 0x20 | byte(element.tag), 0x80}
	for _, child := range children {
		out = append(out, reencodeIndefinite(t, child)...)
	}
	return append(out, 0x00, 0x00)
}

func encodeDefinite(element berElement) []byte {
	identifier := byte(element.class << 6)
	if element.constructed {
		identifier |= 0x20
	}
	identifier |= byte(element.tag)
	raw, _ := asn1.Marshal(asn1.RawValue{
		Class:      element.class,
		Tag:        element.tag,
		IsCompound: element.constructed,
		Bytes:      element.content,
	})
	raw[0] = identifier
	return raw
}
