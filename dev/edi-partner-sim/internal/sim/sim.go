package sim

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/emoss08/trenova/edi-partner-sim/internal/identity"
	"github.com/emoss08/trenova/edi-partner-sim/internal/journal"
	"github.com/emoss08/trenova/edi-partner-sim/internal/mailbox"
	"github.com/emoss08/trenova/edi-partner-sim/internal/oracle"
	"github.com/emoss08/trenova/edi-partner-sim/internal/partner"
	"github.com/emoss08/trenova/edi-partner-sim/internal/partner/reply"
	"github.com/emoss08/trenova/edi-partner-sim/internal/runlog"
	"github.com/emoss08/trenova/edi-partner-sim/internal/schedule"
)

const (
	sendTimeout          = 30 * time.Second
	maxRequestBody       = 32 << 20
	defaultWorkers       = 8
	defaultSnapshotEvery = 2 * time.Second
	defaultPublicURL     = "http://localhost:9210"
	defaultSFTPHost      = "localhost"
	inboundDir           = "/inbound"
	outboundDir          = "/outbound"
	archiveDir           = "archive"
)

var errNoPartners = errors.New("sim: no partners to serve")

type Options struct {
	Partners      []*partner.Definition
	IdentityDir   string
	PublicURL     string
	SFTPListen    string
	SFTPHost      string
	SFTPPassword  string
	SFTPRoot      string
	StatePath     string
	Capacity      int
	Seed          uint64
	Workers       int
	SnapshotEvery time.Duration
	Clock         schedule.Clock
	HTTPClient    *http.Client
	Logger        *slog.Logger
}

type Server struct {
	options   Options
	logger    *slog.Logger
	client    *http.Client
	clock     schedule.Clock
	scheduler *schedule.Scheduler
	journal   *journal.Journal
	runs      *runlog.Store
	mailbox   *mailbox.Server
	peers     map[string]*peer
	order     []string
	changes   atomic.Uint64
	saved     atomic.Uint64
}

func New(options Options) (*Server, error) {
	if len(options.Partners) == 0 {
		return nil, errNoPartners
	}
	if options.Logger == nil {
		options.Logger = slog.New(slog.DiscardHandler)
	}
	if options.Clock == nil {
		options.Clock = schedule.RealClock()
	}
	if options.HTTPClient == nil {
		options.HTTPClient = &http.Client{Timeout: sendTimeout}
	}
	if options.PublicURL == "" {
		options.PublicURL = defaultPublicURL
	}
	options.PublicURL = strings.TrimRight(options.PublicURL, "/")
	if options.SFTPHost == "" {
		options.SFTPHost = defaultSFTPHost
	}
	if options.Workers < 1 {
		options.Workers = defaultWorkers
	}
	if options.SnapshotEvery <= 0 {
		options.SnapshotEvery = defaultSnapshotEvery
	}
	s := &Server{
		options:   options,
		logger:    options.Logger,
		client:    options.HTTPClient,
		clock:     options.Clock,
		scheduler: schedule.New(options.Clock, options.Workers),
		peers:     map[string]*peer{},
	}
	now := func() time.Time { return options.Clock.Now() }
	s.journal = journal.New(options.Capacity, journal.NewBroker(journal.DefaultReplay, now), now)
	s.runs = runlog.New(runlog.DefaultRuns, now, s.notifyRun)
	for _, definition := range options.Partners {
		p, err := s.openPeer(definition)
		if err != nil {
			return nil, err
		}
		s.peers[definition.Name()] = p
		s.order = append(s.order, definition.Name())
	}
	if err := s.openMailbox(); err != nil {
		return nil, err
	}
	if err := s.restore(); err != nil {
		s.closeMailbox()
		return nil, err
	}
	return s, nil
}

func (s *Server) openPeer(definition *partner.Definition) (*peer, error) {
	name := definition.Name()
	guides, err := oracle.LoadGuides(filepath.Join(definition.Dir, "guide"))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	p := &peer{
		def:    definition,
		guides: guides,
		controls: partner.Controls{
			Interchange: partner.NewCounter(controlStart(s.clock.Now(), name, "isa")),
			Group:       partner.NewCounter(controlStart(s.clock.Now(), name, "gs")),
		},
		rng:       rand.New(rand.NewPCG(s.options.Seed, hash(name))),
		originals: newOriginals(s.options.Capacity),
	}
	p.trenova.AS2ID = definition.Partner.Trenova.ID
	if as2 := definition.Partner.Transport.AS2; as2 != nil {
		dir := ""
		if s.options.IdentityDir != "" {
			dir = filepath.Join(s.options.IdentityDir, name)
		}
		p.identity, err = identity.LoadOrCreate(
			dir,
			strings.ToLower(as2.ID)+"."+name+".edi-sim.local",
		)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
	}
	if definition.Partner.Role == partner.RoleCarrier {
		p.carrier, err = reply.Open(definition)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
	}
	return p, nil
}

func (s *Server) openMailbox() error {
	if s.options.SFTPListen == "" {
		for _, name := range s.order {
			if s.peers[name].def.Partner.Transport.SFTP != nil {
				return fmt.Errorf("sim: %s trades over SFTP, so the SFTP mailbox must listen", name)
			}
		}
		return nil
	}
	specs := make([]mailbox.Spec, 0, len(s.order))
	for _, name := range s.order {
		specs = append(specs, mailboxSpec(s.peers[name].def))
	}
	server, err := mailbox.New(mailbox.Options{
		Listen:      s.options.SFTPListen,
		Password:    s.options.SFTPPassword,
		RootDir:     s.options.SFTPRoot,
		IdentityDir: s.options.IdentityDir,
		Logger:      s.logger,
		Watcher:     s,
	}, specs)
	if err != nil {
		return err
	}
	s.mailbox = server
	return nil
}

func mailboxSpec(definition *partner.Definition) mailbox.Spec {
	spec := mailbox.Spec{User: definition.Name(), Inbound: inboundDir, Outbound: outboundDir}
	if sftp := definition.Partner.Transport.SFTP; sftp != nil {
		spec.Inbound, spec.Outbound = sftp.Inbound, sftp.Outbound
	}
	spec.Archive = strings.TrimSuffix(parentOf(spec.Inbound), "/") + "/" + archiveDir
	return spec
}

func parentOf(p string) string {
	index := strings.LastIndex(strings.TrimSuffix(p, "/"), "/")
	if index <= 0 {
		return "/"
	}
	return p[:index]
}

func (s *Server) Run(ctx context.Context) error {
	s.reschedule()
	done := make(chan struct{})
	go func() {
		s.scheduler.Run(ctx)
		close(done)
	}()
	if s.mailbox != nil {
		go func() {
			if err := s.mailbox.Serve(); err != nil {
				s.logger.Error("the SFTP mailbox stopped", "error", err)
			}
		}()
	}
	s.persistLoop(ctx)
	<-done
	s.closeMailbox()
	return s.persist()
}

func (s *Server) closeMailbox() {
	if s.mailbox != nil {
		_ = s.mailbox.Close()
	}
}

func (s *Server) Journal() *journal.Journal {
	return s.journal
}

func (s *Server) Mailbox() *mailbox.Server {
	return s.mailbox
}

func (s *Server) AS2URL(name string) string {
	return s.options.PublicURL + "/as2/" + name
}

func (s *Server) changed() {
	s.changes.Add(1)
}

func (s *Server) peer(name string) (*peer, bool) {
	p, ok := s.peers[name]
	return p, ok
}

func controlStart(now time.Time, name, kind string) int64 {
	return int64((uint64(now.Unix())%100_000_000+hash(name+kind)%1_000_000)%900_000_000) + 1
}

func hash(text string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(text))
	return h.Sum64()
}
