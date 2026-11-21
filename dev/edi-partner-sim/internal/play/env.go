package play

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/emoss08/trenova/edi-partner-sim/internal/identity"
	"github.com/emoss08/trenova/edi-partner-sim/internal/partner"
	"github.com/emoss08/trenova/edi-partner-sim/internal/provision"
	"github.com/emoss08/trenova/edi-partner-sim/internal/simclient"
	"github.com/emoss08/trenova/edi-partner-sim/internal/trenova"
)

const (
	MaxRun            = 899
	runStride         = 1000
	resourceLimit     = 100
	assignAttempts    = 5
	trenovaCommonName = "trenova.edi-sim.local"
)

type Options struct {
	API         string
	Sim         string
	PublicURL   string
	Email       string
	Password    string
	Root        string
	Partners    []string
	Scenarios   []string
	Run         uint64
	CLI         trenova.CLI
	IdentityDir string
	KeepGoing   bool
	Verbose     bool
	Scale       float64
	OnScenario  func(partner string, report *ScenarioReport)
	Out         io.Writer
	HTTPClient  *http.Client
}

type Party struct {
	Def   *partner.Definition
	Setup *provision.Result
	Happy *partner.LoadNeed
}

func (p *Party) Name() string {
	return p.Def.Name()
}

func (p *Party) Carrier() bool {
	return p.Def.Partner.Role == partner.RoleCarrier
}

type Env struct {
	Trenova   *trenova.Client
	Sim       *simclient.Client
	Watch     *simclient.Watch
	CLI       *trenova.CLI
	Run       uint64
	identity  *identity.Identity
	publicURL string
	root      string
	out       io.Writer
	verbose   bool
	fleet     fleet
	template  shipmentTemplate
	mapper    *provision.Mapper
	api       string
	email     string
	progress  *progress
	scale     float64
	finished  func(partner string, report *ScenarioReport)
}

func Open(ctx context.Context, o *Options) (*Env, error) {
	if o.Run < 1 || o.Run > MaxRun {
		return nil, fmt.Errorf("the run is a number from 1 to %d, not %d", MaxRun, o.Run)
	}
	scale := o.Scale
	if scale == 0 {
		scale = 1
	}
	if scale < 0 {
		return nil, fmt.Errorf("the deadline scale multiplies every wait; %v is not above 0",
			o.Scale)
	}
	out := o.Out
	if out == nil {
		out = io.Discard
	}
	client, err := trenova.New(o.API, o.HTTPClient)
	if err != nil {
		return nil, err
	}
	simulator := simclient.New(o.Sim, o.HTTPClient)
	if err = simulator.Healthy(ctx); err != nil {
		return nil, fmt.Errorf("the simulator is not answering (start it with task sim): %w",
			err)
	}
	if err = client.Login(ctx, o.Email, o.Password); err != nil {
		return nil, err
	}
	dir := ""
	if o.IdentityDir != "" {
		dir = filepath.Join(o.IdentityDir, "trenova")
	}
	trenovaIdentity, err := identity.LoadOrCreate(dir, trenovaCommonName)
	if err != nil {
		return nil, fmt.Errorf("Trenova's AS2 identity: %w", err)
	}
	watch, err := simulator.Watch(ctx)
	if err != nil {
		return nil, err
	}
	cli := o.CLI
	return &Env{
		Trenova:   client,
		Sim:       simulator,
		Watch:     watch,
		CLI:       &cli,
		Run:       o.Run,
		identity:  trenovaIdentity,
		publicURL: o.PublicURL,
		root:      o.Root,
		out:       out,
		verbose:   o.Verbose,
		api:       o.API,
		email:     o.Email,
		mapper:    provision.NewMapper(client),
		scale:     scale,
		finished:  o.OnScenario,
	}, nil
}

func (e *Env) Close() {
	if e.Watch != nil {
		e.Watch.Close()
	}
}

func (e *Env) logf(format string, args ...any) {
	fmt.Fprintf(e.out, format+"\n", args...)
}

func (e *Env) Definitions(only []string) ([]*partner.Definition, error) {
	all, err := partner.OpenAll(e.root)
	if err != nil {
		return nil, err
	}
	if len(only) == 0 || (len(only) == 1 && only[0] == "all") {
		return all, nil
	}
	var picked []*partner.Definition
	for _, name := range only {
		name = strings.TrimSpace(name)
		index := slices.IndexFunc(all, func(d *partner.Definition) bool { return d.Name() == name })
		if index < 0 {
			return nil, fmt.Errorf("no partner %q in %s", name, e.root)
		}
		picked = append(picked, all[index])
	}
	return picked, nil
}

func (e *Env) Provision(ctx context.Context, def *partner.Definition) *Party {
	e.logf("provisioning %s", def.Name())
	setup := provision.Partner(ctx, &provision.Options{
		Trenova:   e.Trenova,
		Sim:       e.Sim,
		Identity:  e.identity,
		PublicURL: e.publicURL,
	}, def)
	for _, note := range setup.Notes {
		e.logf("  %s", note)
	}
	if setup.Failure != "" {
		e.logf("  %s is not ready: %s", def.Name(), setup.Failure)
		for _, test := range setup.FailingTests {
			e.logf("    failing test: %s", test)
		}
	}
	return &Party{Def: def, Setup: setup, Happy: happyNeed(def)}
}

func SeedOf(run uint64, offset uint64) uint64 {
	return run*runStride + offset
}

type shipmentTemplate struct {
	mu    sync.Mutex
	ready bool
	value trenova.NewShipment
}

func (e *Env) shipmentTemplate(ctx context.Context) (trenova.NewShipment, error) {
	t := &e.template
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.ready {
		return t.value, nil
	}
	recent, err := e.Trenova.RecentShipments(ctx, 25)
	if err != nil {
		return trenova.NewShipment{}, fmt.Errorf("read shipments to copy a customer and "+
			"lane from: %w", err)
	}
	for _, s := range recent {
		stops := s.Stops()
		if s.CustomerID == "" || s.ServiceTypeID == "" || len(stops) < 2 {
			continue
		}
		pickup, delivery := "", ""
		for _, stop := range stops {
			if pickup == "" && strings.Contains(stop.Type, "Pickup") {
				pickup = stop.LocationID
			}
			if strings.Contains(stop.Type, "Delivery") {
				delivery = stop.LocationID
			}
		}
		if pickup == "" || delivery == "" || pickup == delivery {
			continue
		}
		t.value = trenova.NewShipment{
			CustomerID:     s.CustomerID,
			ServiceTypeID:  s.ServiceTypeID,
			ShipmentTypeID: s.ShipmentTypeID,
			Pickup:         pickup,
			Delivery:       delivery,
			Weight:         24000,
			Pieces:         12,
		}
		t.ready = true
		return t.value, nil
	}
	return trenova.NewShipment{}, errors.New("no shipment in the tenant has a customer, a " +
		"service type and a pickup and delivery to copy (seed the database: task db-seed)")
}
