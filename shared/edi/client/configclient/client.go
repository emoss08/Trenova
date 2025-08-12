package configclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"os"
	"time"

	"github.com/emoss08/trenova/shared/edi/adapter/configproto"
	"github.com/emoss08/trenova/shared/edi/pkg/configtypes"
	configpb "github.com/emoss08/trenova/shared/edi/proto/config/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type Client struct {
	cc   *grpc.ClientConn
	svc  configpb.EDIConfigServiceClient
	auth struct {
		bearer string
		apiKey string
	}
	// defaults
	defaultTimeout time.Duration
	defaultRetries int
	defaultBackoff time.Duration
}

type DialOptions struct {
	Address  string
	Bearer   string
	APIKey   string
	Insecure bool
	// TLS options (client-side TLS or mTLS)
	TLSCertFile        string
	TLSKeyFile         string
	CACertFile         string
	InsecureSkipVerify bool
	DialTimeout        time.Duration
}

func Dial(ctx context.Context, opt DialOptions, grpcOpts ...grpc.DialOption) (*Client, error) {
	if opt.DialTimeout <= 0 {
		opt.DialTimeout = 5 * time.Second
	}
	if opt.Insecure {
		grpcOpts = append(grpcOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	} else {
		// Load TLS credentials if provided
		if creds, err := loadClientTLS(opt); err == nil && creds != nil {
			grpcOpts = append(grpcOpts, grpc.WithTransportCredentials(creds))
		}
	}
	cctx, cancel := context.WithTimeout(ctx, opt.DialTimeout)
	defer cancel()
	conn, err := grpc.DialContext(cctx, opt.Address, grpcOpts...)
	if err != nil {
		return nil, err
	}
	c := &Client{
		cc:             conn,
		svc:            configpb.NewEDIConfigServiceClient(conn),
		defaultTimeout: 5 * time.Second,
		defaultRetries: 2,
		defaultBackoff: 200 * time.Millisecond,
	}
	c.auth.bearer = opt.Bearer
	c.auth.apiKey = opt.APIKey
	return c, nil
}

func (c *Client) Close() error { return c.cc.Close() }

// Get fetches a partner config; at least one of id or (bu,org,name) should be provided.
func (c *Client) Get(
	ctx context.Context,
	id, bu, org, name string,
) (*configtypes.PartnerConfig, error) {
	return c.GetWithOptions(ctx, id, bu, org, name, 0, 0, 0)
}

func (c *Client) GetWithOptions(
	ctx context.Context,
	id, bu, org, name string,
	timeout time.Duration,
	retries int,
	backoff time.Duration,
) (*configtypes.PartnerConfig, error) {
	ctx = c.injectAuth(ctx)
	if timeout <= 0 {
		timeout = c.defaultTimeout
	}
	if retries < 0 {
		retries = c.defaultRetries
	}
	if backoff <= 0 {
		backoff = c.defaultBackoff
	}
	req := &configpb.GetPartnerConfigRequest{
		Id:             id,
		BusinessUnitId: bu,
		OrganizationId: org,
		Name:           name,
	}
	var resp *configpb.GetPartnerConfigResponse
	err := c.withRetry(ctx, timeout, retries, backoff, func(callCtx context.Context) error {
		var err error
		resp, err = c.svc.GetPartnerConfig(callCtx, req)
		return err
	})
	if err != nil {
		return nil, err
	}
	return configproto.FromProto(resp.GetConfig()), nil
}

// List returns a page of partner configs along with next page token.
func (c *Client) List(
	ctx context.Context,
	bu, org string,
	pageSize int32,
	pageToken string,
) ([]*configtypes.PartnerConfig, string, error) {
	return c.ListWithOptions(ctx, bu, org, pageSize, pageToken, 0, 0, 0)
}

func (c *Client) ListWithOptions(
	ctx context.Context,
	bu, org string,
	pageSize int32,
	pageToken string,
	timeout time.Duration,
	retries int,
	backoff time.Duration,
) ([]*configtypes.PartnerConfig, string, error) {
	ctx = c.injectAuth(ctx)
	if timeout <= 0 {
		timeout = c.defaultTimeout
	}
	if retries < 0 {
		retries = c.defaultRetries
	}
	if backoff <= 0 {
		backoff = c.defaultBackoff
	}
	req := &configpb.ListPartnerConfigsRequest{
		BusinessUnitId: bu,
		OrganizationId: org,
		PageSize:       pageSize,
		PageToken:      pageToken,
	}
	var resp *configpb.ListPartnerConfigsResponse
	err := c.withRetry(ctx, timeout, retries, backoff, func(callCtx context.Context) error {
		var err error
		resp, err = c.svc.ListPartnerConfigs(callCtx, req)
		return err
	})
	if err != nil {
		return nil, "", err
	}
	out := make([]*configtypes.PartnerConfig, 0, len(resp.GetItems()))
	for _, it := range resp.GetItems() {
		out = append(out, configproto.FromProto(it.GetConfig()))
	}
	return out, resp.GetNextPageToken(), nil
}

func (c *Client) injectAuth(ctx context.Context) context.Context {
	if c.auth.bearer == "" && c.auth.apiKey == "" {
		return ctx
	}
	md := metadata.MD{}
	if c.auth.bearer != "" {
		md.Append("authorization", "Bearer "+c.auth.bearer)
	}
	if c.auth.apiKey != "" {
		md.Append("x-api-key", c.auth.apiKey)
	}
	return metadata.NewOutgoingContext(ctx, md)
}

func loadClientTLS(opt DialOptions) (credentials.TransportCredentials, error) {
	// If nothing is provided, return nil to let caller decide
	if opt.TLSCertFile == "" && opt.CACertFile == "" {
		return nil, nil
	}
	var certs []tls.Certificate
	if opt.TLSCertFile != "" && opt.TLSKeyFile != "" {
		if cert, err := tls.LoadX509KeyPair(opt.TLSCertFile, opt.TLSKeyFile); err == nil {
			certs = []tls.Certificate{cert}
		} else {
			return nil, err
		}
	}
	tlsCfg := &tls.Config{
		Certificates:       certs,
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: opt.InsecureSkipVerify,
	}
	if opt.CACertFile != "" {
		if caBytes, err := os.ReadFile(opt.CACertFile); err == nil {
			caPool := x509.NewCertPool()
			_ = caPool.AppendCertsFromPEM(caBytes)
			tlsCfg.RootCAs = caPool
		}
	}
	return credentials.NewTLS(tlsCfg), nil
}

func (c *Client) withRetry(
	ctx context.Context,
	timeout time.Duration,
	retries int,
	backoff time.Duration,
	fn func(context.Context) error,
) error {
	var attempt int
	for {
		callCtx, cancel := context.WithTimeout(ctx, timeout)
		err := fn(callCtx)
		cancel()
		if err == nil {
			return nil
		}
		// Retry on transient codes
		st, _ := status.FromError(err)
		switch st.Code() {
		case codes.Unavailable, codes.DeadlineExceeded, codes.ResourceExhausted:
			if attempt < retries {
				attempt++
				time.Sleep(time.Duration(attempt) * backoff)
				continue
			}
		}
		return err
	}
}
