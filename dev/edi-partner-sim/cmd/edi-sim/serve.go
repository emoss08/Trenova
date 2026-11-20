package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/emoss08/trenova/edi-partner-sim/internal/journal"
	"github.com/emoss08/trenova/edi-partner-sim/internal/partner"
	"github.com/emoss08/trenova/edi-partner-sim/internal/reportload"
	"github.com/emoss08/trenova/edi-partner-sim/internal/sim"
)

const (
	serveSummary    = "host every partner at once: AS2, SFTP mailboxes and the control API"
	shutdownTimeout = 10 * time.Second
	readTimeout     = 10 * time.Second
)

type serveFlags struct {
	root     string
	partners string
	listen   string
	report   string
	options  sim.Options
}

func runServe(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	set := flag.NewFlagSet(programName+" serve", flag.ContinueOnError)
	set.SetOutput(stderr)
	var f serveFlags
	set.StringVar(&f.root, "partners", "partners", "the folder holding the partners")
	set.StringVar(&f.partners, "partner", "", "serve only these partners (comma separated)")
	set.StringVar(
		&f.listen,
		"listen",
		":9210",
		"the HTTP address: AS2 endpoints and the control API",
	)
	set.StringVar(&f.options.PublicURL, "public-url", "",
		"the base URL Trenova reaches the simulator at (AS2 URLs, asynchronous MDN returns; "+
			"default: http://localhost:<the --listen port>)")
	set.StringVar(&f.options.IdentityDir, "identity-dir", "",
		"where each partner's AS2 keypair and the SFTP host key persist (empty: new ones each run)")
	set.StringVar(&f.options.SFTPListen, "sftp-listen", ":9222",
		"the SFTP mailbox address (empty: no mailbox)")
	set.StringVar(&f.options.SFTPHost, "sftp-host", "localhost",
		"the host Trenova reaches the SFTP mailbox at")
	set.StringVar(&f.options.SFTPPassword, "sftp-password", "trenova-sim",
		"the password every partner's SFTP user signs in with")
	set.StringVar(&f.options.SFTPRoot, "sftp-root", "",
		"the folder holding each partner's mailbox (empty: a temporary folder)")
	set.StringVar(&f.options.StatePath, "state", "",
		"the JSON snapshot the simulator restarts from (empty: nothing kept)")
	set.IntVar(&f.options.Capacity, "history", journal.DefaultCapacity,
		"messages kept per partner")
	set.Uint64Var(
		&f.options.Seed,
		"seed",
		1,
		"the seed behind withheld acknowledgments and answers",
	)
	set.IntVar(&f.options.Workers, "workers", 8, "deliveries made at once")
	set.StringVar(&f.report, "report", "",
		"a run or conform report.json to show in the viewer at / (its wire and scenarios)")
	if err := set.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return exitOK
		}
		return exitUsage
	}
	if set.NArg() > 0 {
		fmt.Fprintf(stderr, "serve: unexpected argument %q\n", set.Arg(0))
		return exitUsage
	}
	if f.options.PublicURL == "" {
		f.options.PublicURL = "http://" + localHost(f.listen)
	}
	definitions, err := openPartners(f.root, f.partners)
	if err != nil {
		fmt.Fprintf(stderr, "serve: %v\n", err)
		return exitFailed
	}
	logger := slog.New(slog.NewTextHandler(stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	f.options.Partners, f.options.Logger = definitions, logger
	server, err := sim.New(f.options)
	if err != nil {
		fmt.Fprintf(stderr, "serve: %v\n", err)
		return exitFailed
	}
	if f.report != "" {
		if err = loadReport(server, f.report, logger); err != nil {
			fmt.Fprintf(stderr, "serve: %v\n", err)
			return exitFailed
		}
	}
	if err := listenAndRun(ctx, &f, server, logger); err != nil {
		fmt.Fprintf(stderr, "serve: %v\n", err)
		return exitFailed
	}
	return exitOK
}

func loadReport(server *sim.Server, path string, logger *slog.Logger) error {
	result, err := reportload.Load(server, path)
	if err != nil {
		return fmt.Errorf("--report %s: %w", path, err)
	}
	for _, name := range result.Skipped {
		logger.Warn("the report names a partner this simulator does not serve", "partner", name)
	}
	if result.Already {
		logger.Info("the report is already in the saved state", "report", path)
		return nil
	}
	logger.Info("report loaded", "report", path, "kind", result.Kind,
		"scenarios", result.Scenarios, "messages", result.Messages)
	return nil
}

func openPartners(root, only string) ([]*partner.Definition, error) {
	definitions, err := partner.OpenAll(root)
	if err != nil {
		return nil, err
	}
	if only == "" {
		return definitions, nil
	}
	wanted := strings.Split(only, ",")
	var picked []*partner.Definition
	for _, name := range wanted {
		name = strings.TrimSpace(name)
		index := slices.IndexFunc(definitions, func(d *partner.Definition) bool {
			return d.Name() == name
		})
		if index < 0 {
			return nil, fmt.Errorf("no partner %q in %s", name, root)
		}
		picked = append(picked, definitions[index])
	}
	return picked, nil
}

func listenAndRun(
	ctx context.Context,
	f *serveFlags,
	server *sim.Server,
	logger *slog.Logger,
) error {
	listener, err := (&net.ListenConfig{}).Listen(ctx, "tcp", f.listen)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", f.listen, err)
	}
	runCtx, stop := context.WithCancel(ctx)
	defer stop()
	streams, closeStreams := context.WithCancel(context.WithoutCancel(ctx))
	defer closeStreams()
	httpServer := &http.Server{
		Handler:           server.Handler(),
		ReadHeaderTimeout: readTimeout,
		BaseContext:       func(net.Listener) context.Context { return streams },
	}
	httpServer.RegisterOnShutdown(closeStreams)
	ran := make(chan error, 1)
	go func() { ran <- server.Run(runCtx) }()
	served := make(chan error, 1)
	go func() { served <- httpServer.Serve(listener) }()
	for _, info := range server.Partners() {
		attrs := []any{"partner", info.Name, "role", info.Role, "transport", info.Transport}
		if info.AS2 != nil {
			attrs = append(attrs, "as2", info.AS2.URL)
		}
		if info.SFTP != nil {
			attrs = append(attrs, "sftpUser", info.SFTP.Username)
		}
		logger.Info("partner ready", attrs...)
	}
	logger.Info("EDI simulator listening", "address", listener.Addr().String(),
		"sftp", f.options.SFTPListen, "publicUrl", f.options.PublicURL,
		"viewer", "http://"+localHost(listener.Addr().String())+"/")
	var serveErr error
	select {
	case <-ctx.Done():
	case serveErr = <-served:
	}
	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
	defer cancel()
	shutdownErr := httpServer.Shutdown(shutdownCtx)
	stop()
	runErr := <-ran
	if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
		return serveErr
	}
	return errors.Join(shutdownErr, runErr)
}

func localHost(addr string) string {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	if host == "" || host == "::" || host == "0.0.0.0" || host == "[::]" {
		host = "localhost"
	}
	return net.JoinHostPort(host, port)
}
