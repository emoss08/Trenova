package clamav

import (
	"bufio"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
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
	sizeLimitReply   = "INSTREAM size limit exceeded"
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

func (c *Client) Scan(
	ctx context.Context,
	content io.Reader,
	size int64,
) (*contentscanports.ScanResult, error) {
	if size > c.maxStreamBytes {
		return nil, &contentscanports.TooLargeError{Size: size, Limit: c.maxStreamBytes}
	}

	reply, err := c.exchange(ctx, func(w io.Writer) error {
		return writeStream(w, content, c.maxStreamBytes)
	})
	if err != nil {
		return nil, err
	}
	if strings.HasPrefix(reply, sizeLimitReply) {
		return nil, &contentscanports.TooLargeError{Size: size}
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
		var tooLarge *contentscanports.TooLargeError
		if errors.As(err, &tooLarge) {
			return "", err
		}
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

func writeStream(w io.Writer, content io.Reader, limit int64) error {
	if _, err := io.WriteString(w, "zINSTREAM\x00"); err != nil {
		return err
	}
	var size [4]byte
	chunk := make([]byte, chunkSize)
	sent := int64(0)
	for {
		n, readErr := io.ReadFull(content, chunk)
		if n > 0 {
			sent += int64(n)
			if sent > limit {
				return &contentscanports.TooLargeError{Size: sent, Limit: limit}
			}
			binary.BigEndian.PutUint32(size[:], uint32(n)) //nolint:gosec // n <= chunkSize
			if _, err := w.Write(size[:]); err != nil {
				return err
			}
			if _, err := w.Write(chunk[:n]); err != nil {
				return err
			}
		}
		if errors.Is(readErr, io.EOF) || errors.Is(readErr, io.ErrUnexpectedEOF) {
			break
		}
		if readErr != nil {
			return readErr
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
