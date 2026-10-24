package pgxdsn

import (
	"fmt"
	"math"
	"net/url"
	"strconv"
	"time"
)

const (
	dialTimeoutParam     = "dial_timeout"
	connectTimeoutParam  = "connect_timeout"
	applicationNameParam = "application_name"
)

type Options struct {
	ApplicationName string
}

func Normalize(dsn string, opts Options) (string, error) {
	u, err := url.Parse(dsn)
	if err != nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") {
		return "", fmt.Errorf("pgxdsn: %q is not a postgres URL", redact(dsn))
	}

	query := u.Query()
	if raw := query.Get(dialTimeoutParam); raw != "" {
		timeout, parseErr := time.ParseDuration(raw)
		if parseErr != nil {
			return "", fmt.Errorf("pgxdsn: %s %q: %w", dialTimeoutParam, raw, parseErr)
		}
		seconds := max(int(math.Ceil(timeout.Seconds())), 1)
		query.Set(connectTimeoutParam, strconv.Itoa(seconds))
		query.Del(dialTimeoutParam)
	}
	if opts.ApplicationName != "" {
		query.Set(applicationNameParam, opts.ApplicationName)
	}

	u.RawQuery = query.Encode()

	return u.String(), nil
}

func redact(dsn string) string {
	u, err := url.Parse(dsn)
	if err != nil {
		return "<unparseable>"
	}

	return u.Redacted()
}
