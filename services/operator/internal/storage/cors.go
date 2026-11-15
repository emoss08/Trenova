package storage

import (
	"bytes"
	"context"
	"crypto/md5" //nolint:gosec // S3 requires Content-MD5 on PutBucketCors: an integrity check, not a secret
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
)

const maxErrorBody = 512

type S3CORS struct {
	Client *http.Client
	Now    func() time.Time
}

func (c S3CORS) Allow(ctx context.Context, b Bucket, origin string) error {
	body, err := corsDocument(origin)
	if err != nil {
		return err
	}
	url := strings.TrimSuffix(b.Endpoint, "/") + "/" + b.Name + "?cors"
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("the documents bucket's CORS request: %w", err)
	}
	sum := sha256.Sum256(body)
	payload := hex.EncodeToString(sum[:])
	digest := md5.Sum(body) //nolint:gosec // see the import
	req.ContentLength = int64(len(body))
	req.Header.Set("Content-Type", "application/xml")
	req.Header.Set("Content-MD5", base64.StdEncoding.EncodeToString(digest[:]))
	req.Header.Set("X-Amz-Content-Sha256", payload)
	creds := aws.Credentials{AccessKeyID: b.AccessKeyID, SecretAccessKey: b.SecretAccessKey}
	if err = v4.NewSigner().SignHTTP(ctx, creds, req, payload, "s3", b.Region, c.now()); err != nil {
		return fmt.Errorf("sign the documents bucket's CORS request: %w", err)
	}
	res, err := c.client().Do(req)
	if err != nil {
		return fmt.Errorf("set the documents bucket's CORS rules: %w", err)
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode/100 != 2 {
		_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, maxErrorBody))
		return fmt.Errorf("the documents bucket refused its CORS rules: %s", res.Status)
	}
	return nil
}

func (c S3CORS) client() *http.Client {
	if c.Client != nil {
		return c.Client
	}
	return &http.Client{Timeout: corsTimeout}
}

func (c S3CORS) now() time.Time {
	if c.Now != nil {
		return c.Now().UTC()
	}
	return time.Now().UTC()
}

func corsDocument(origin string) ([]byte, error) {
	var escaped bytes.Buffer
	if err := xml.EscapeText(&escaped, []byte(origin)); err != nil {
		return nil, fmt.Errorf("the CORS origin: %w", err)
	}
	return []byte(`<?xml version="1.0" encoding="UTF-8"?>` +
		`<CORSConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><CORSRule>` +
		`<AllowedOrigin>` + escaped.String() + `</AllowedOrigin>` +
		`<AllowedMethod>GET</AllowedMethod><AllowedMethod>HEAD</AllowedMethod><AllowedMethod>PUT</AllowedMethod>` +
		`<AllowedHeader>*</AllowedHeader><ExposeHeader>ETag</ExposeHeader><MaxAgeSeconds>3600</MaxAgeSeconds>` +
		`</CORSRule></CORSConfiguration>`), nil
}
