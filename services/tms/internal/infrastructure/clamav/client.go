package clamav

import (
	"bufio"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"slices"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/modules/documents/ports/contentscanports"
)

const (
	chunkSize        = 32 << 10
	maxResponseBytes = 4 << 10
	unixPrefix       = "unix:"
	foundSuffix      = " FOUND"
	streamPrefix     = "stream: "
)

var errUnexpectedReply = errors.New("clamd sent a reply it does not send")

type Client struct {
	network        string
	address        string
	timeout        time.Duration
	maxStreamBytes int64
	dialer         net.Dialer
}

var _ contentscanports.Scanner = (*Client)(nil)

func New(cfg config.ClamAVConfig) *Client {
	network, address := "tcp", strings.TrimSpace(cfg.Address)
	if strings.HasPrefix(address, unixPrefix) {
		network, address = "unix", strings.TrimPrefix(address, unixPrefix)
	}
	return &Client{
		network:        network,
		address:        address,
		timeout:        cfg.GetTimeout(),
		maxStreamBytes: cfg.GetMaxStreamBytes(),
	}
}

func (c *Client) Ping(ctx context.Context) error {
	reply, err := c.exchange(ctx, func(w io.Writer) error {
		_, err := io.WriteString(w, "zPING\x00")
		return err
	})
	if err != nil {
		return err
	}
	if reply != "PONG" {
		return fmt.Errorf(
			"%w: %w: %q",
			contentscanports.ErrScannerUnavailable,
			errUnexpectedReply,
			reply,
		)
	}
	return nil
}

func (c *Client) Scan(ctx context.Context, content []byte) (*contentscanports.ScanResult, error) {
	if int64(len(content)) > c.maxStreamBytes {
		return nil, fmt.Errorf(
			"%w: the file is %d bytes and the scanner takes at most %d",
			contentscanports.ErrScannerUnavailable, len(content), c.maxStreamBytes,
		)
	}

	reply, err := c.exchange(ctx, func(w io.Writer) error {
		return writeStream(w, content)
	})
	if err != nil {
		return nil, err
	}
	return parseReply(reply)
}

func (c *Client) exchange(ctx context.Context, send func(io.Writer) error) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	conn, err := c.dialer.DialContext(ctx, c.network, c.address)
	if err != nil {
		return "", fmt.Errorf("%w: %w", contentscanports.ErrScannerUnavailable, err)
	}
	defer func() { _ = conn.Close() }()

	if deadline, ok := ctx.Deadline(); ok {
		if err = conn.SetDeadline(deadline); err != nil {
			return "", fmt.Errorf("%w: %w", contentscanports.ErrScannerUnavailable, err)
		}
	}

	writer := bufio.NewWriterSize(conn, chunkSize+4)
	if err = send(writer); err != nil {
		return "", fmt.Errorf("%w: %w", contentscanports.ErrScannerUnavailable, err)
	}
	if err = writer.Flush(); err != nil {
		return "", fmt.Errorf("%w: %w", contentscanports.ErrScannerUnavailable, err)
	}

	reply, err := bufio.NewReader(io.LimitReader(conn, maxResponseBytes)).ReadString(0)
	if err != nil && !errors.Is(err, io.EOF) {
		return "", fmt.Errorf("%w: %w", contentscanports.ErrScannerUnavailable, err)
	}
	return strings.TrimSpace(strings.TrimRight(reply, "\x00")), nil
}

func writeStream(w io.Writer, content []byte) error {
	if _, err := io.WriteString(w, "zINSTREAM\x00"); err != nil {
		return err
	}
	var size [4]byte
	for chunk := range slices.Chunk(content, chunkSize) {
		binary.BigEndian.PutUint32(size[:], uint32(len(chunk))) //nolint:gosec // chunkSize fits
		if _, err := w.Write(size[:]); err != nil {
			return err
		}
		if _, err := w.Write(chunk); err != nil {
			return err
		}
	}
	binary.BigEndian.PutUint32(size[:], 0)
	_, err := w.Write(size[:])
	return err
}

func parseReply(reply string) (*contentscanports.ScanResult, error) {
	switch {
	case reply == streamPrefix+"OK":
		return &contentscanports.ScanResult{Verdict: contentscanports.VerdictClean}, nil
	case strings.HasPrefix(reply, streamPrefix) && strings.HasSuffix(reply, foundSuffix):
		signature := strings.TrimSuffix(strings.TrimPrefix(reply, streamPrefix), foundSuffix)
		return &contentscanports.ScanResult{
			Verdict:   contentscanports.VerdictInfected,
			Signature: strings.TrimSpace(signature),
		}, nil
	case strings.HasSuffix(reply, "ERROR"):
		return nil, fmt.Errorf("%w: clamd: %s", contentscanports.ErrScannerUnavailable, reply)
	default:
		return nil, fmt.Errorf("%w: %w: %q", contentscanports.ErrScannerUnavailable,
			errUnexpectedReply, reply)
	}
}
