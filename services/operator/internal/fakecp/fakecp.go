package fakecp

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/bytedance/sonic"
	"github.com/google/uuid"

	"github.com/emoss08/trenova/operator/internal/credential"
	"github.com/emoss08/trenova/operator/internal/instanceapi"
)

const maxReadings = 1000

type usageKey struct {
	meter string
	hour  time.Time
}

type Server struct {
	InstanceID uuid.UUID

	caKey       *ecdsa.PrivateKey
	caCert      *x509.Certificate
	caPEM       []byte
	server      tls.Certificate
	url         string
	mu          sync.Mutex
	valid       map[string]bool
	used        map[string]bool
	revision    int64
	spec        instanceapi.InstanceSpec
	reports     []instanceapi.StatusReport
	usage       []instanceapi.UsageReport
	readings    map[usageKey]int64
	ack         *instanceapi.UsageAck
	usageFail   int
	escrowed    instanceapi.InstanceSecrets
	secretCalls int
	secretsFail int
	failStatus  int
	renewals    int
	limited     int
	unavailable int
	lifetime    time.Duration
}

type Params struct {
	Spec  instanceapi.InstanceSpec
	Hosts []string
	URL   string
}

func New(p *Params) (*Server, error) {
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "fake instance CA"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(7 * 24 * time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &caKey.PublicKey, caKey)
	if err != nil {
		return nil, err
	}
	caCert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, err
	}
	s := &Server{
		InstanceID: p.Spec.InstanceId,
		caKey:      caKey,
		caCert:     caCert,
		caPEM:      pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
		url:        p.URL,
		valid:      map[string]bool{},
		used:       map[string]bool{},
		readings:   map[usageKey]int64{},
		revision:   1,
		spec:       p.Spec,
		escrowed:   Secrets(),
		lifetime:   30 * 24 * time.Hour,
	}
	if s.server, err = s.serverCertificate(p.Hosts); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Server) serverCertificate(hosts []string) (tls.Certificate, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "fake instance API"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(7 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			tmpl.IPAddresses = append(tmpl.IPAddresses, ip)
		} else {
			tmpl.DNSNames = append(tmpl.DNSNames, h)
		}
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, s.caCert, &key.PublicKey, s.caKey)
	if err != nil {
		return tls.Certificate{}, err
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}, nil
}

func (s *Server) TLSConfig() *tls.Config {
	pool := x509.NewCertPool()
	pool.AddCert(s.caCert)
	return &tls.Config{
		MinVersion:   tls.VersionTLS13,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    pool,
		Certificates: []tls.Certificate{s.server},
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/spec", s.getSpec)
	mux.HandleFunc("PUT /v1/status", s.putStatus)
	mux.HandleFunc("POST /v1/certificate", s.renew)
	mux.HandleFunc("POST /v1/usage", s.reportUsage)
	mux.HandleFunc("GET /v1/secrets", s.getSecrets)
	return s.authenticate(mux)
}

func (s *Server) SetURL(u string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.url = u
}

func (s *Server) Bundle(lifetime time.Duration) (credential.Bundle, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return credential.Bundle{}, err
	}
	certPEM, err := s.sign(&key.PublicKey, lifetime)
	if err != nil {
		return credential.Bundle{}, err
	}
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return credential.Bundle{}, err
	}
	s.mu.Lock()
	clear(s.valid)
	apiURL := s.url
	s.mu.Unlock()
	s.markValid(certPEM)
	return credential.Bundle{
		InstanceID:  s.InstanceID.String(),
		APIURL:      apiURL,
		Certificate: string(certPEM),
		PrivateKey:  string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})),
		CABundle:    string(s.caPEM),
		NotAfter:    time.Now().Add(lifetime),
	}, nil
}

func (s *Server) SetLifetime(d time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lifetime = d
}

func (s *Server) Spec() instanceapi.InstanceSpec {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.spec
}

func (s *Server) SetSpec(spec *instanceapi.InstanceSpec) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.revision++
	s.spec = *spec
}

func (s *Server) RateLimitNext(n int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.limited = n
}

func (s *Server) UnavailableNext(n int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.unavailable = n
}

func (s *Server) RevokeAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	clear(s.valid)
}

func (s *Server) Reports() []instanceapi.StatusReport {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]instanceapi.StatusReport(nil), s.reports...)
}

func (s *Server) UsageReports() []instanceapi.UsageReport {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]instanceapi.UsageReport(nil), s.usage...)
}

func (s *Server) SetUsageAck(ack *instanceapi.UsageAck) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ack = ack
}

func (s *Server) FailUsageNext(n int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.usageFail = n
}

func (s *Server) SetSecrets(v instanceapi.InstanceSecrets) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.escrowed = v
}

func (s *Server) FailSecretsNext(n, status int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.secretsFail, s.failStatus = n, status
}

func (s *Server) SecretCalls() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.secretCalls
}

func (s *Server) Renewals() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.renewals
}

func (s *Server) Revision() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.revision
}

func fingerprint(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func (s *Server) markValid(certPEM []byte) {
	block, _ := pem.Decode(certPEM)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.valid[fingerprint(block.Bytes)] = true
}

func (s *Server) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fp := fingerprint(r.TLS.PeerCertificates[0].Raw)
		s.mu.Lock()
		ok := s.valid[fp]
		if ok && !s.used[fp] {
			s.used[fp] = true
			for other := range s.valid {
				if other != fp && s.used[other] {
					delete(s.valid, other)
				}
			}
		}
		limited := s.limited > 0
		if limited {
			s.limited--
		}
		s.mu.Unlock()
		if !ok {
			problem(w, http.StatusUnauthorized, "unauthenticated")
			return
		}
		if limited {
			w.Header().Set("Retry-After", "1")
			problem(w, http.StatusTooManyRequests, "rate_limited")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) getSpec(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	rev, spec := s.revision, s.spec
	unavailable := s.unavailable > 0
	if unavailable {
		s.unavailable--
	}
	s.mu.Unlock()
	if unavailable {
		w.Header().Set("Retry-After", "1")
		problem(w, http.StatusServiceUnavailable, "unavailable")
		return
	}
	etag := strconv.Quote(strconv.FormatInt(rev, 10))
	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("ETag", etag)
	write(w, instanceapi.DesiredSpec{Revision: rev, Spec: spec})
}

func (s *Server) getSecrets(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	s.secretCalls++
	escrowed, status := s.escrowed, s.failStatus
	fail := s.secretsFail > 0
	if fail {
		s.secretsFail--
	}
	s.mu.Unlock()
	if fail {
		failSecrets(w, status)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	write(w, escrowed)
}

func failSecrets(w http.ResponseWriter, status int) {
	switch status {
	case http.StatusUnauthorized:
		problem(w, status, "unauthenticated")
	case http.StatusTooManyRequests:
		w.Header().Set("Retry-After", "1")
		problem(w, status, "rate_limited")
	default:
		problem(w, status, "internal")
	}
}

func (s *Server) putStatus(w http.ResponseWriter, r *http.Request) {
	var report instanceapi.StatusReport
	if err := sonic.ConfigDefault.NewDecoder(r.Body).Decode(&report); err != nil {
		problem(w, http.StatusBadRequest, "invalid_request")
		return
	}
	s.mu.Lock()
	s.reports = append(s.reports, report)
	s.mu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) reportUsage(w http.ResponseWriter, r *http.Request) {
	var report instanceapi.UsageReport
	if err := sonic.ConfigDefault.NewDecoder(r.Body).Decode(&report); err != nil ||
		len(report.Readings) > maxReadings {
		problem(w, http.StatusBadRequest, "invalid_request")
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.usageFail > 0 {
		s.usageFail--
		problem(w, http.StatusInternalServerError, "internal")
		return
	}
	s.usage = append(s.usage, report)
	for _, rd := range report.Readings {
		s.readings[usageKey{meter: rd.Meter, hour: rd.Hour.UTC()}] = rd.Quantity
	}
	if s.ack != nil {
		write(w, s.ack)
		return
	}
	write(w, s.defaultAck(time.Now().UTC()))
}

func (s *Server) defaultAck(now time.Time) instanceapi.UsageAck {
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	window := instanceapi.UsageWindow{Start: start, End: start.AddDate(0, 1, 0)}
	ack := instanceapi.UsageAck{
		Window:  &window,
		Overage: true,
		Meters:  make([]instanceapi.MeterUsage, 0, len(s.spec.Allowances)),
	}
	for _, a := range s.spec.Allowances {
		var quantity int64
		var latest time.Time
		for k, q := range s.readings {
			if k.meter != a.Meter || k.hour.Before(window.Start) || !k.hour.Before(window.End) {
				continue
			}
			if a.Unit != instanceapi.SpecAllowanceUnitBytes {
				quantity += q
				continue
			}
			if !k.hour.Before(latest) {
				quantity, latest = q, k.hour
			}
		}
		ack.Meters = append(ack.Meters, instanceapi.MeterUsage{
			Meter:     a.Meter,
			Unit:      instanceapi.MeterUsageUnit(a.Unit),
			Quantity:  quantity,
			Allowance: a.Quantity,
			HardCap:   a.HardCap,
			Stopped:   a.HardCap && quantity >= a.Quantity,
		})
	}
	return ack
}

func (s *Server) renew(w http.ResponseWriter, r *http.Request) {
	var req instanceapi.RenewCertificateRequest
	if err := sonic.ConfigDefault.NewDecoder(r.Body).Decode(&req); err != nil {
		problem(w, http.StatusBadRequest, "invalid_request")
		return
	}
	pub, err := csrKey(req.Csr)
	if err != nil || pub.Equal(r.TLS.PeerCertificates[0].PublicKey) {
		problem(w, http.StatusBadRequest, "validation_failed")
		return
	}
	s.mu.Lock()
	lifetime := s.lifetime
	s.renewals++
	s.mu.Unlock()
	certPEM, err := s.sign(pub, lifetime)
	if err != nil {
		problem(w, http.StatusInternalServerError, "internal")
		return
	}
	s.markValid(certPEM)
	write(w, instanceapi.IssuedCertificate{
		Certificate: string(certPEM),
		CaBundle:    string(s.caPEM),
		NotAfter:    time.Now().Add(lifetime),
	})
}

func csrKey(csrPEM string) (*ecdsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(csrPEM))
	if block == nil {
		return nil, errors.New("no PEM block")
	}
	csr, err := x509.ParseCertificateRequest(block.Bytes)
	if err != nil {
		return nil, err
	}
	if err = csr.CheckSignature(); err != nil {
		return nil, err
	}
	pub, ok := csr.PublicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, errors.New("not an ECDSA key")
	}
	return pub, nil
}

func (s *Server) sign(pub *ecdsa.PublicKey, lifetime time.Duration) ([]byte, error) {
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 127))
	if err != nil {
		return nil, err
	}
	uri, err := url.Parse("urn:trenova:instance:" + s.InstanceID.String())
	if err != nil {
		return nil, err
	}
	now := time.Now()
	der, err := x509.CreateCertificate(rand.Reader, &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: fmt.Sprintf("instance %s", s.InstanceID)},
		NotBefore:    now.Add(-5 * time.Minute),
		NotAfter:     now.Add(lifetime),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		URIs:         []*url.URL{uri},
	}, s.caCert, pub, s.caKey)
	if err != nil {
		return nil, err
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), nil
}

func write(w http.ResponseWriter, body any) {
	out, err := sonic.Marshal(body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(out)
}

func problem(w http.ResponseWriter, status int, code string) {
	out, _ := sonic.Marshal(map[string]any{
		"type":   "about:blank",
		"title":  code,
		"status": status,
		"code":   code,
	})
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_, _ = w.Write(out)
}

func Secrets() instanceapi.InstanceSecrets {
	return instanceapi.InstanceSecrets{
		EncryptionKey:         "fake-encryption-key-0123456789abcdefghijklmnopqrstuvwxyz",
		RlsScopeKeyId:         "k20261107",
		RlsScopeKey:           "ZmFrZS1zY29wZS1rZXktMDEyMzQ1Njc4OWFiY2RlZmdoaWprbG1ub3A=",
		MfaRecoveryCodePepper: "fake-recovery-code-pepper-0123456789abcdefghijklmnop",
	}
}

func Spec(instanceID uuid.UUID) instanceapi.InstanceSpec {
	return instanceapi.InstanceSpec{
		SchemaVersion:  1,
		Lifecycle:      instanceapi.Active,
		InstanceId:     instanceID,
		OrganizationId: uuid.New(),
		Name:           "Acme Freight",
		Hostname:       "k7q2m9x4p1r8t3v6.trenova.cloud",
		Release:        instanceapi.SpecRelease{Channel: instanceapi.Stable, Version: "0.11.0"},
		Plan: instanceapi.SpecPlan{
			Key:                 "production",
			Name:                "Production",
			Vcpu:                8,
			MemoryGiB:           32,
			DiskGiB:             200,
			Backup:              instanceapi.PointInTime,
			BackupRetentionDays: 30,
			Support:             instanceapi.BusinessHours,
		},
		Placement: instanceapi.SpecPlacement{
			Provider:        "ovh",
			Region:          "us-east",
			Location:        "US-EAST-VA-1",
			StorageLocation: "US-EAST-VA",
			Country:         "US",
			Runtime:         instanceapi.SingleNode,
		},
		Addons:     []string{},
		Features:   []string{},
		Components: []string{},
		AiProvider: instanceapi.Trenova,
		Allowances: []instanceapi.SpecAllowance{},
	}
}
