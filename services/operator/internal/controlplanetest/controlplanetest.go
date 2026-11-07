package controlplanetest

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/emoss08/trenova/operator/internal/credential"
	"github.com/emoss08/trenova/operator/internal/fakecp"
	"github.com/emoss08/trenova/operator/internal/instanceapi"
)

type Server struct {
	*fakecp.Server
	URL string
	t   testing.TB
}

func New(t testing.TB, spec instanceapi.InstanceSpec) *Server {
	t.Helper()
	fake, err := fakecp.New(&fakecp.Params{Spec: spec, Hosts: []string{"127.0.0.1"}})
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewUnstartedServer(fake.Handler())
	srv.TLS = fake.TLSConfig()
	srv.StartTLS()
	t.Cleanup(srv.Close)
	fake.SetURL(srv.URL)
	return &Server{Server: fake, URL: srv.URL, t: t}
}

func (s *Server) Bundle(lifetime time.Duration) credential.Bundle {
	s.t.Helper()
	b, err := s.Server.Bundle(lifetime)
	if err != nil {
		s.t.Fatal(err)
	}
	return b
}

func (s *Server) SetSpec(spec instanceapi.InstanceSpec) {
	s.Server.SetSpec(&spec)
}

func Spec(instanceID uuid.UUID) instanceapi.InstanceSpec {
	return fakecp.Spec(instanceID)
}

func Secrets() instanceapi.InstanceSecrets {
	return fakecp.Secrets()
}
