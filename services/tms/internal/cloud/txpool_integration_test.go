//go:build integration

package cloud_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/cloud/adapters/postgres/tenantcopyrepository"
	"github.com/emoss08/trenova/internal/cloud/domain/tenantarchive"
	"github.com/emoss08/trenova/internal/cloud/ports/archiveports"
	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/databaseconfig"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgproto3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const pooledCommand = "7a1a3c1e-8c1f-4a39-9a43-5b0d7c4e2f10"

type txPool struct {
	t         *testing.T
	ln        net.Listener
	server    string
	rotations atomic.Int64
	wg        sync.WaitGroup
}

func startTxPool(t *testing.T, server string) *txPool {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	p := &txPool{t: t, ln: ln, server: server}
	p.wg.Add(1)
	go p.accept()
	t.Cleanup(func() {
		_ = ln.Close()
		p.wg.Wait()
	})
	return p
}

func (p *txPool) port() int {
	addr, _ := p.ln.Addr().(*net.TCPAddr)
	return addr.Port
}

func (p *txPool) accept() {
	defer p.wg.Done()
	for {
		client, err := p.ln.Accept()
		if err != nil {
			return
		}
		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			defer client.Close()
			if serr := p.serve(client); serr != nil && !errors.Is(serr, io.EOF) &&
				!errors.Is(serr, io.ErrUnexpectedEOF) && !errors.Is(serr, net.ErrClosed) {
				p.t.Errorf("transaction pool: %v", serr)
			}
		}()
	}
}

type pooledServer struct {
	conn     net.Conn
	frontend *pgproto3.Frontend
	prepared map[string]bool
}

func (p *txPool) dial(ctx context.Context) (*pooledServer, map[string]string, error) {
	pg, err := pgconn.Connect(ctx, p.server)
	if err != nil {
		return nil, nil, err
	}
	hijacked, err := pg.Hijack()
	if err != nil {
		return nil, nil, err
	}
	return &pooledServer{
		conn:     hijacked.Conn,
		frontend: hijacked.Frontend,
		prepared: map[string]bool{},
	}, hijacked.ParameterStatuses, nil
}

func (p *txPool) serve(client net.Conn) error {
	backend := pgproto3.NewBackend(client, client)
	for {
		msg, err := backend.ReceiveStartupMessage()
		if err != nil {
			return err
		}
		switch msg.(type) {
		case *pgproto3.SSLRequest, *pgproto3.GSSEncRequest:
			if _, err = client.Write([]byte{'N'}); err != nil {
				return err
			}
			continue
		case *pgproto3.CancelRequest:
			return nil
		}
		break
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	servers := make([]*pooledServer, 2)
	var statuses map[string]string
	for i := range servers {
		s, params, err := p.dial(ctx)
		if err != nil {
			return err
		}
		defer s.conn.Close()
		servers[i], statuses = s, params
	}
	backend.Send(&pgproto3.AuthenticationOk{})
	for name, value := range statuses {
		backend.Send(&pgproto3.ParameterStatus{Name: name, Value: value})
	}
	backend.Send(&pgproto3.BackendKeyData{ProcessID: 1, SecretKey: []byte{0, 0, 0, 1}})
	backend.Send(&pgproto3.ReadyForQuery{TxStatus: 'I'})
	if err := backend.Flush(); err != nil {
		return err
	}
	return p.relay(backend, servers)
}

func (p *txPool) relay(backend *pgproto3.Backend, servers []*pooledServer) error {
	parses := map[string]*pgproto3.Parse{}
	current, idle := 0, false
	for {
		var injected []bool
		for {
			msg, err := backend.Receive()
			if err != nil {
				return err
			}
			if idle {
				current, idle = 1-current, false
				p.rotations.Add(1)
			}
			s := servers[current]
			done := false
			switch m := msg.(type) {
			case *pgproto3.Terminate:
				return nil
			case *pgproto3.Parse:
				if m.Name != "" {
					parses[m.Name] = &pgproto3.Parse{
						Name:          m.Name,
						Query:         m.Query,
						ParameterOIDs: append([]uint32(nil), m.ParameterOIDs...),
					}
					s.prepared[m.Name] = true
				}
				injected = append(injected, false)
			case *pgproto3.Bind:
				injected = p.prepare(s, parses, m.PreparedStatement, injected)
			case *pgproto3.Describe:
				if m.ObjectType == 'S' {
					injected = p.prepare(s, parses, m.Name, injected)
				}
			case *pgproto3.Close:
				if m.ObjectType == 'S' {
					delete(parses, m.Name)
					for _, other := range servers {
						delete(other.prepared, m.Name)
					}
				}
			case *pgproto3.Sync, *pgproto3.Query:
				done = true
			case *pgproto3.Execute:
			default:
				return fmt.Errorf("transaction pool: unexpected client message %T", msg)
			}
			s.frontend.Send(msg)
			if done {
				break
			}
		}
		s := servers[current]
		if err := s.frontend.Flush(); err != nil {
			return err
		}
		status, err := p.answer(backend, s, injected)
		if err != nil {
			return err
		}
		idle = status == 'I'
	}
}

func (p *txPool) prepare(
	s *pooledServer,
	parses map[string]*pgproto3.Parse,
	name string,
	injected []bool,
) []bool {
	if name == "" || s.prepared[name] {
		return injected
	}
	parse, ok := parses[name]
	if !ok {
		return injected
	}
	s.frontend.Send(parse)
	s.prepared[name] = true
	return append(injected, true)
}

func (p *txPool) answer(
	backend *pgproto3.Backend,
	s *pooledServer,
	injected []bool,
) (byte, error) {
	for {
		msg, err := s.frontend.Receive()
		if err != nil {
			return 0, err
		}
		if _, parsed := msg.(*pgproto3.ParseComplete); parsed && len(injected) > 0 {
			swallow := injected[0]
			injected = injected[1:]
			if swallow {
				continue
			}
		}
		backend.Send(msg)
		if ready, ok := msg.(*pgproto3.ReadyForQuery); ok {
			return ready.TxStatus, backend.Flush()
		}
		if err = backend.Flush(); err != nil {
			return 0, err
		}
	}
}

func pooledDatabase(database *databaseconfig.Config, port int) *databaseconfig.Config {
	pooled := *database
	pooled.Host = "127.0.0.1"
	pooled.Port = port
	return &pooled
}

func TestAnArchiveTakenThroughATransactionPoolCarriesItsTailSealAndLeavesNothingBehind(
	t *testing.T,
) {
	src := newInstall(t, true, sourceKey)
	r := &trailRig{src: src, facts: prepareSource(t, src), keys: archiveKeys(0x41)}

	time.Sleep(2 * time.Second)
	after := time.Now()
	r.write(t, after, "organization.updated", "written after the last hourly seal")

	pool := startTxPool(t, src.database.DSNForRole("trenova-tx-pool", src.database.System))
	pooled := *src
	pooled.rows = tenantcopyrepository.New(tenantcopyrepository.Params{
		DB:       src.conn,
		App:      &appconfig.Config{Name: "archive-test"},
		Database: pooledDatabase(src.database, pool.port()),
		AIAudit:  src.chain,
	})
	pooled.engine = pooled.engineWith(src.buckets)

	stated := export(t, &pooled, r.facts.tenant, tenantarchive.PurposeFinal, pooledCommand, r.keys)
	assert.Positive(t, pool.rotations.Load(),
		"a statement outside the snapshot's transaction ran on another server connection")

	trail := stated.Manifest.Trails.Audit
	require.NotNil(t, trail)
	assert.True(t, trail.Verified)
	assert.Greater(t, trail.SealedThrough.Unix(), after.Unix(),
		"the archive carries the seal over the entry written after the last hourly seal")

	held := newInstallWithChain(t, false, destinationKey, holding(destinationKey, sourceKey))
	result := r.loadStated(t, held, pooledCommand, nil)
	assert.Nil(t, result.Trails.Tail, "no entry arrived unsealed")
	assert.Equal(t, archiveports.EstablishedHere, result.Trails.AuditSeals.Established)
	assert.Equal(t, audit.VerificationVerified, verifyTrails(t, held, r.facts.tenant).audit)

	for _, in := range []*install{src, held} {
		assert.Equal(t, []string{"0"}, in.query(t,
			`SELECT count(*)::text FROM pg_catalog.pg_class WHERE relpersistence = 't'`),
			"no temporary table is left on any connection")
	}
	assert.Equal(t, []string{"0"}, src.query(t,
		`SELECT count(*)::text FROM pg_catalog.pg_class c
         JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
         WHERE n.nspname LIKE 'pg_temp%' AND c.relname LIKE 'archive_audit%'`),
		"the archive keeps no session state of its own")
}
