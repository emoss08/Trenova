package test

import (
	"testing"

	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/api/router"
	"github.com/emoss08/trenova/internal/config"
	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/ent/enttest"
	"github.com/emoss08/trenova/internal/util"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"github.com/valyala/fasthttp"
)

// WithTestServer returns a fully configured server (using the default server config).
func WithTestServer(t *testing.T, closure func(s *api.Server)) {
	t.Helper()

	// Build the ent Client
	opts := []enttest.Option{enttest.WithOptions(ent.Log(t.Log))}
	client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&_fk=1", opts...)
	defer client.Close()

	// Build the server
	defaultConfig := config.DefaultServiceConfigFromEnv()
	execClosureNewTestServer(t, defaultConfig, client, closure)
}

// Executes closure on a new test server with a pre-provided database
func execClosureNewTestServer(t *testing.T, config config.Server, client *ent.Client, closure func(s *api.Server)) {
	t.Helper()

	// https://stackoverflow.com/questions/43424787/how-to-use-next-available-port-in-http-listenandserve
	// You may use port 0 to indicate you're not specifying an exact port, but you want a free, available port selected by the system
	config.Fiber.ListenAddress = ":0"

	s := api.NewServer(config)

	// Register gob
	s.RegisterGob()

	// attach the already initialized db
	s.Client = client
	s.Session = session.New()

	// Initialize and set some default values in the session store.
	router.Init(s)

	c := s.Fiber.AcquireCtx(&fasthttp.RequestCtx{
		Request: fasthttp.Request{
			Header:                         fasthttp.RequestHeader{},
			UseHostHeader:                  true,
			DisableRedirectPathNormalizing: true,
		},
	})
	defer s.Fiber.ReleaseCtx(c)

	sess, err := s.Session.Get(c)
	if err != nil {
		t.Fatalf("failed to get session: %v", err)
	}

	// Set the session in the context
	sess.Set(util.CTXUserID, uuid.New())
	sess.Set(util.CTXOrganizationID, uuid.New())
	sess.Set(util.CTXBusinessUnitID, uuid.New())

	if err := sess.Save(); err != nil {
		t.Fatalf("failed to save session: %v", err)
	}

	// Set the values in Context
	c.Locals(util.CTXUserID, uuid.New())
	c.Locals(util.CTXOrganizationID, uuid.New())
	c.Locals(util.CTXBusinessUnitID, uuid.New())

	closure(s)

	// fiber is managed and should close automatically after running the test
	if err := s.Fiber.Shutdown(); err != nil {
		t.Fatalf("failed to shutdown server: %v", err)
	}

	// disallow any further refs to managed object after running the test
	s = nil
}
