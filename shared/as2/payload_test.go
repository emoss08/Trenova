package as2

import (
	"bytes"
	"mime"
	"testing"

	"github.com/emoss08/trenova/shared/smime"
	"github.com/emoss08/trenova/shared/smimetest"
	"github.com/stretchr/testify/require"
)

const lfTender = "ISA*00*          *00*          *ZZ*KESTRELVW      *ZZ*TRENOVA        " +
	"*260424*0920*^*00401*000000501*0*T*>~\n" +
	"GS*SM*KESTRELVW*TRENOVA*20260424*0920*501*X*004010~\n" +
	"ST*204*0001~\nB2**TRNV**KVW-100037**PP~\nSE*3*0001~\n" +
	"GE*1*501~\nIEA*1*000000501~\n"

func TestThePayloadReachesTheReceiverByteForByte(t *testing.T) {
	t.Parallel()

	sender := smimetest.NewIdentity(t, "sender.example")
	receiver := smimetest.NewIdentity(t, "receiver.example")
	payloads := map[string]string{
		"line feeds": lfTender,
		"crlf": string(
			bytes.ReplaceAll([]byte(lfTender), []byte("\n"), []byte("\r\n")),
		),
		"no terminator":    string(bytes.ReplaceAll([]byte(lfTender), []byte("\n"), nil)),
		"mixed and binary": "ISA*00~\r\nGS*SM~\nST\r*204~\x00\xff\n\n",
	}
	layers := []struct {
		name     string
		sign     bool
		encrypt  bool
		compress bool
	}{
		{name: "plain"},
		{name: "signed", sign: true},
		{name: "encrypted", encrypt: true},
		{name: "compressed", compress: true},
		{name: "signed and encrypted", sign: true, encrypt: true},
		{name: "compressed and encrypted", encrypt: true, compress: true},
		{name: "compressed and signed", sign: true, compress: true},
		{name: "compressed, signed and encrypted", sign: true, encrypt: true, compress: true},
	}
	for payloadName, payload := range payloads {
		for _, layer := range layers {
			t.Run(payloadName+"/"+layer.name, func(t *testing.T) {
				t.Parallel()

				options := &BuildMessageOptions{
					From:     "SENDER-AS2",
					To:       "RECEIVER-AS2",
					FileName: "tender.edi",
					Payload:  []byte(payload),
					Compress: layer.compress,
				}
				if layer.sign {
					options.SigningCertificate, options.SigningKey = sender.Certificate, sender.Key
				}
				if layer.encrypt {
					options.EncryptionCertificate = receiver.Certificate
					options.EncryptionAlgorithm = smime.EncryptionAlgorithmAES256CBC
				}
				built, err := BuildMessage(options)
				require.NoError(t, err)

				parseOptions := &ParseMessageOptions{
					EntityHeaders:     built.Headers,
					RequireSignature:  layer.sign,
					RequireEncryption: layer.encrypt,
				}
				if layer.sign {
					parseOptions.PartnerCertificate = sender.Certificate
				}
				if layer.encrypt {
					parseOptions.DecryptionCertificate = receiver.Certificate
					parseOptions.DecryptionKey = receiver.Key
				}
				parsed, err := ParseMessage(built.ContentType, built.Body, parseOptions)
				require.NoError(t, err)
				require.Equal(t, []byte(payload), parsed.Payload)
				require.Equal(t, layer.sign, parsed.Signed)
				require.Equal(t, layer.encrypt, parsed.Encrypted)
				require.Equal(t, layer.compress, parsed.Compressed)
				require.Equal(t, "tender.edi", parsed.FileName)
				require.True(t, MICMatches(built.MIC, parsed.MIC),
					"sent %s, received %s", built.MIC, parsed.MIC)
			})
		}
	}
}

func TestTheMICOfACompressedSignedMessageCoversWhatWasSigned(t *testing.T) {
	t.Parallel()

	sender := smimetest.NewIdentity(t, "sender.example")
	built, err := BuildMessage(&BuildMessageOptions{
		From:               "SENDER-AS2",
		To:                 "RECEIVER-AS2",
		Payload:            []byte(lfTender),
		SigningCertificate: sender.Certificate,
		SigningKey:         sender.Key,
		Compress:           true,
	})
	require.NoError(t, err)

	_, params, err := mime.ParseMediaType(built.ContentType)
	require.NoError(t, err)
	parts, err := splitMultipart(built.Body, params["boundary"])
	require.NoError(t, err)
	signed, err := ComputeMIC(parts[0], MICAlgorithmSHA256)
	require.NoError(t, err)
	require.True(t, MICMatches(signed, built.MIC))
}

func TestThePlainMessageMICCoversTheContentWithoutHeaders(t *testing.T) {
	t.Parallel()

	built, err := BuildMessage(&BuildMessageOptions{
		From:     "SENDER-AS2",
		To:       "RECEIVER-AS2",
		FileName: "tender.edi",
		Payload:  []byte(lfTender),
	})
	require.NoError(t, err)

	content, err := ComputeMIC([]byte(lfTender), MICAlgorithmSHA256)
	require.NoError(t, err)
	require.True(t, MICMatches(content, built.MIC))
	require.Equal(t, "binary", built.Headers.Get("Content-Transfer-Encoding"))
	require.Contains(t, built.Headers.Get("Content-Disposition"), `filename=tender.edi`)
}

func TestEntityHeadersKeepsOnlyTheMIMEEntityHeaders(t *testing.T) {
	t.Parallel()

	headers := EntityHeaders(map[string][]string{
		"Content-Transfer-Encoding": {"binary"},
		"Content-Disposition":       {"attachment; filename=a.edi"},
		"As2-From":                  {"SENDER-AS2"},
		"Content-Type":              {"application/edi-x12"},
	})
	require.Equal(t, "binary", headers.Get("Content-Transfer-Encoding"))
	require.Equal(t, "attachment; filename=a.edi", headers.Get("Content-Disposition"))
	require.Len(t, headers, 2)
}

func TestTheSentEntityCarriesThePayloadUnchanged(t *testing.T) {
	t.Parallel()

	built, err := BuildMessage(&BuildMessageOptions{
		From:    "SENDER-AS2",
		To:      "RECEIVER-AS2",
		Payload: []byte(lfTender),
	})
	require.NoError(t, err)
	require.Equal(t, []byte(lfTender), built.Body)
	require.NotContains(t, string(built.Body), "\r")
}
