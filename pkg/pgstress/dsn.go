package pgstress

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

func buildDSN(host, port, user, password, database, sslmode string) (string, error) {
	if strings.TrimSpace(host) == "" || strings.TrimSpace(user) == "" || strings.TrimSpace(database) == "" {
		return "", fmt.Errorf("host, user, and database are required")
	}

	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(user, password),
		Host:   net.JoinHostPort(host, port),
		Path:   database,
	}
	q := u.Query()
	q.Set("sslmode", sslmode)
	u.RawQuery = q.Encode()
	return u.String(), nil
}
