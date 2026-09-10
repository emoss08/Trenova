package sftp

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	AuthModePassword = "password"

	DefaultPort = "22"
)

type Config struct {
	Host         string
	Port         string
	Username     string
	AuthMode     string
	KnownHostKey string
	Password     string
	PrivateKey   string
}

func (c Config) Validate() error {
	switch {
	case strings.TrimSpace(c.Host) == "":
		return errors.New("SFTP host is required")
	case strings.TrimSpace(c.Username) == "":
		return errors.New("SFTP username is required")
	case strings.TrimSpace(c.KnownHostKey) == "":
		return errors.New("SFTP known host key is required")
	case c.AuthMode == AuthModePassword && c.Password == "":
		return errors.New("SFTP password secret is required")
	case c.AuthMode != AuthModePassword && c.PrivateKey == "":
		return errors.New("SFTP private key secret is required")
	}

	if c.Port == "" {
		return nil
	}

	if _, err := strconv.Atoi(c.Port); err != nil {
		return fmt.Errorf("SFTP port must be numeric: %w", err)
	}

	return nil
}

func (c Config) Address() string {
	return net.JoinHostPort(c.Host, stringutils.WithDefault(c.Port, DefaultPort))
}
