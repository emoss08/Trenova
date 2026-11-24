package minio

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
)

const (
	unsignedPayload  = "UNSIGNED-PAYLOAD"
	s3Service        = "s3"
	amzQueryPrefix   = "x-amz-"
	credentialParam  = "X-Amz-Credential"
	expiresParam     = "X-Amz-Expires"
	credentialFields = 5
)

type stableSigner struct {
	credentials aws.Credentials
	signer      *v4.Signer
}

func newStableSigner(accessKey, secretKey, sessionToken string) *stableSigner {
	return &stableSigner{
		credentials: aws.Credentials{
			AccessKeyID:     accessKey,
			SecretAccessKey: secretKey,
			SessionToken:    sessionToken,
		},
		signer: v4.NewSigner(func(o *v4.SignerOptions) {
			o.DisableURIPathEscaping = true
		}),
	}
}

func (s *stableSigner) resign(
	ctx context.Context,
	presigned *url.URL,
	expiry time.Duration,
	signedAt time.Time,
) (string, error) {
	query := presigned.Query()
	region := regionOf(query.Get(credentialParam))
	if region == "" || s.credentials.AccessKeyID == "" {
		return presigned.String(), nil
	}

	for name := range query {
		if strings.HasPrefix(strings.ToLower(name), amzQueryPrefix) {
			query.Del(name)
		}
	}
	query.Set(expiresParam, strconv.FormatInt(int64(expiry/time.Second), 10))

	target := *presigned
	target.RawQuery = query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), http.NoBody)
	if err != nil {
		return "", fmt.Errorf("build the request to sign: %w", err)
	}

	signed, _, err := s.signer.PresignHTTP(
		ctx,
		s.credentials,
		req,
		unsignedPayload,
		s3Service,
		region,
		signedAt.UTC(),
	)
	if err != nil {
		return "", fmt.Errorf("sign the request: %w", err)
	}
	return signed, nil
}

func regionOf(credential string) string {
	parts := strings.Split(credential, "/")
	if len(parts) != credentialFields {
		return ""
	}
	return parts[2]
}
