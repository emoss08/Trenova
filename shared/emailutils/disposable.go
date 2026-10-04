package emailutils

import (
	"bufio"
	_ "embed"
	"strings"
	"sync"
)

//go:embed disposable_domains.txt
var disposableDomainsSource string

var disposableDomains = sync.OnceValue(func() map[string]struct{} {
	domains := make(map[string]struct{}, strings.Count(disposableDomainsSource, "\n")+1)
	scanner := bufio.NewScanner(strings.NewReader(disposableDomainsSource))
	for scanner.Scan() {
		line := strings.ToLower(strings.TrimSpace(scanner.Text()))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		domains[line] = struct{}{}
	}

	return domains
})

func IsDisposableDomain(domain string) bool {
	domain = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(domain)), ".")
	if domain == "" {
		return false
	}

	blocked := disposableDomains()
	for {
		if _, ok := blocked[domain]; ok {
			return true
		}
		dot := strings.IndexByte(domain, '.')
		if dot < 0 || strings.IndexByte(domain[dot+1:], '.') < 0 {
			return false
		}
		domain = domain[dot+1:]
	}
}

func IsDisposable(address string) bool {
	return IsDisposableDomain(Domain(address))
}

func DisposableDomainCount() int {
	return len(disposableDomains())
}
