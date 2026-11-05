package pgstubtest

import (
	"context"
	"errors"
	"fmt"
	"net"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbpool"
	"github.com/jackc/pgx/v5/pgproto3"
	"github.com/stretchr/testify/require"
)

const (
	boolOID      = 16
	int8OID      = 20
	textOID      = 25
	binaryFormat = 1

	idle          = 'I'
	inTransaction = 'T'

	existsPrefix = "SELECT EXISTS ("

	stubOrganizationID = "org_stub"
	stubBusinessUnitID = "bu_stub"
)

var (
	parameterRef = regexp.MustCompile(`\$(\d+)`)
	countQuery   = regexp.MustCompile(`(?i)^\s*SELECT\s+COUNT\(`)
	settingQuery = regexp.MustCompile(`(?i)^\s*SELECT\s+set_config\(`)
)

func NewConnection(t testing.TB, existingTables ...string) *postgres.Connection {
	t.Helper()

	listener, err := new(net.ListenConfig).Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = listener.Close() })

	go serve(listener, existingTables)

	db, err := keel.Open(t.Context(), keel.Config{
		DSN: fmt.Sprintf(
			"postgres://stub@%s/stub?sslmode=disable",
			listener.Addr().String(),
		),
		TenantColumns: []string{dbpool.OrganizationColumn, dbpool.BusinessUnitColumn},
		Tenant: func(context.Context) ([]any, bool) {
			return []any{stubOrganizationID, stubBusinessUnitID}, true
		},
		SkipEnums: true,
	})
	require.NoError(t, err)
	t.Cleanup(db.Close)

	return postgres.WrapDB(db)
}

func serve(listener net.Listener, existingTables []string) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			return
		}

		go func() {
			defer conn.Close()

			s := &session{
				backend:        pgproto3.NewBackend(conn, conn),
				existingTables: existingTables,
				statements:     make(map[string]string),
				portals:        make(map[string]portal),
				txStatus:       idle,
			}
			_ = s.run()
		}()
	}
}

type portal struct {
	sql    string
	binary bool
}

type session struct {
	backend        *pgproto3.Backend
	existingTables []string
	statements     map[string]string
	portals        map[string]portal
	txStatus       byte
	failed         bool
}

func (s *session) run() error {
	if err := s.startup(); err != nil {
		return err
	}

	for {
		msg, err := s.backend.Receive()
		if err != nil {
			return err
		}

		switch m := msg.(type) {
		case *pgproto3.Terminate:
			return nil
		case *pgproto3.Sync:
			s.failed = false
			if err = s.ready(); err != nil {
				return err
			}
		case *pgproto3.Flush:
			if err = s.backend.Flush(); err != nil {
				return err
			}
		case *pgproto3.Query:
			s.command(m.String, false)
			if err = s.ready(); err != nil {
				return err
			}
		default:
			if !s.failed {
				s.extended(msg)
			}
		}
	}
}

func (s *session) startup() error {
	msg, err := s.backend.ReceiveStartupMessage()
	if err != nil {
		return err
	}
	if _, ok := msg.(*pgproto3.StartupMessage); !ok {
		return errors.New("pgstubtest: expected a startup message")
	}

	s.backend.Send(&pgproto3.AuthenticationOk{})
	s.backend.Send(&pgproto3.ParameterStatus{Name: "server_version", Value: "18.0"})
	s.backend.Send(&pgproto3.ParameterStatus{Name: "client_encoding", Value: "UTF8"})
	s.backend.Send(&pgproto3.ParameterStatus{Name: "standard_conforming_strings", Value: "on"})
	s.backend.Send(&pgproto3.BackendKeyData{ProcessID: 1, SecretKey: []byte{0, 0, 0, 0}})

	return s.ready()
}

func (s *session) ready() error {
	s.backend.Send(&pgproto3.ReadyForQuery{TxStatus: s.txStatus})

	return s.backend.Flush()
}

func (s *session) extended(msg pgproto3.FrontendMessage) {
	switch m := msg.(type) {
	case *pgproto3.Parse:
		s.statements[m.Name] = m.Query
		s.backend.Send(&pgproto3.ParseComplete{})
	case *pgproto3.Bind:
		s.portals[m.DestinationPortal] = portal{
			sql:    s.statements[m.PreparedStatement],
			binary: slices.Contains(m.ResultFormatCodes, binaryFormat),
		}
		s.backend.Send(&pgproto3.BindComplete{})
	case *pgproto3.Describe:
		s.describe(m)
	case *pgproto3.Execute:
		bound := s.portals[m.Portal]
		s.command(bound.sql, bound.binary)
	case *pgproto3.Close:
		s.backend.Send(&pgproto3.CloseComplete{})
	default:
		s.fail(fmt.Sprintf("unsupported message %T", msg))
	}
}

func (s *session) describe(m *pgproto3.Describe) {
	sql := s.portals[m.Name].sql
	if m.ObjectType == 'S' {
		sql = s.statements[m.Name]
		s.backend.Send(&pgproto3.ParameterDescription{ParameterOIDs: parameterOIDs(sql)})
	}

	if strings.HasPrefix(sql, existsPrefix) {
		s.backend.Send(&pgproto3.RowDescription{Fields: []pgproto3.FieldDescription{{
			Name:         []byte("exists"),
			DataTypeOID:  boolOID,
			DataTypeSize: 1,
			TypeModifier: -1,
		}}})
		return
	}

	if countQuery.MatchString(sql) {
		s.backend.Send(&pgproto3.RowDescription{Fields: []pgproto3.FieldDescription{{
			Name:         []byte("count"),
			DataTypeOID:  int8OID,
			DataTypeSize: 8,
			TypeModifier: -1,
		}}})
		return
	}

	s.backend.Send(&pgproto3.NoData{})
}

func (s *session) command(sql string, binary bool) {
	if strings.HasPrefix(sql, existsPrefix) {
		s.backend.Send(&pgproto3.DataRow{Values: [][]byte{s.existsValue(sql, binary)}})
		s.backend.Send(&pgproto3.CommandComplete{CommandTag: []byte("SELECT 1")})
		return
	}

	if countQuery.MatchString(sql) {
		s.backend.Send(&pgproto3.DataRow{Values: [][]byte{zeroCount(binary)}})
		s.backend.Send(&pgproto3.CommandComplete{CommandTag: []byte("SELECT 1")})
		return
	}

	if settingQuery.MatchString(sql) {
		s.backend.Send(&pgproto3.CommandComplete{CommandTag: []byte("SELECT 0")})
		return
	}

	fields := strings.Fields(strings.ToUpper(sql))
	if len(fields) == 0 {
		s.backend.Send(&pgproto3.EmptyQueryResponse{})
		return
	}

	switch fields[0] {
	case "BEGIN", "START":
		s.txStatus = inTransaction
		s.backend.Send(&pgproto3.CommandComplete{CommandTag: []byte("BEGIN")})
	case "COMMIT", "ROLLBACK", "END":
		s.txStatus = idle
		s.backend.Send(&pgproto3.CommandComplete{CommandTag: []byte(fields[0])})
	case "SET":
		s.backend.Send(&pgproto3.CommandComplete{CommandTag: []byte("SET")})
	default:
		s.fail("unsupported statement: " + sql)
	}
}

func (s *session) existsValue(sql string, binary bool) []byte {
	found := slices.ContainsFunc(s.existingTables, func(table string) bool {
		return strings.Contains(sql, `"`+table+`"`)
	})

	switch {
	case binary && found:
		return []byte{1}
	case binary:
		return []byte{0}
	case found:
		return []byte("t")
	default:
		return []byte("f")
	}
}

func zeroCount(binary bool) []byte {
	if binary {
		return make([]byte, 8)
	}

	return []byte("0")
}

func (s *session) fail(message string) {
	s.failed = true
	s.backend.Send(&pgproto3.ErrorResponse{
		Severity: "ERROR",
		Code:     "0A000",
		Message:  "pgstubtest: " + message,
	})
}

func parameterOIDs(sql string) []uint32 {
	count := 0
	for _, match := range parameterRef.FindAllStringSubmatch(sql, -1) {
		if n, err := strconv.Atoi(match[1]); err == nil {
			count = max(count, n)
		}
	}

	oids := make([]uint32, count)
	for i := range oids {
		oids[i] = textOID
	}

	return oids
}
