package provision

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/emoss08/trenova/edi-partner-sim/internal/identity"
	"github.com/emoss08/trenova/edi-partner-sim/internal/partner"
	"github.com/emoss08/trenova/edi-partner-sim/internal/poll"
	"github.com/emoss08/trenova/edi-partner-sim/internal/sim"
	"github.com/emoss08/trenova/edi-partner-sim/internal/simclient"
	"github.com/emoss08/trenova/edi-partner-sim/internal/trenova"
	"github.com/emoss08/trenova/pkg/lading"
)

const (
	CodePrefix        = "EDISIM-"
	methodAS2         = "AS2"
	MethodSFTP        = "SFTP"
	statusActive      = "Active"
	versionDraft      = "Draft"
	versionCertified  = "Certified"
	replayStatusDone  = "Completed"
	replayStatusError = "Failed"
	replayWindow      = 89 * 24 * time.Hour
	replayWithin      = 5 * time.Minute
	roleCarrier       = "carrier"
	roleBroker        = "broker"
)

var ErrNotReady = errors.New("the partner is not ready")

type Options struct {
	Trenova   *trenova.Client
	Sim       *simclient.Client
	Identity  *identity.Identity
	PublicURL string
}

type Result struct {
	Partner       string             `json:"partner"`
	Role          string             `json:"role"`
	Code          string             `json:"code"`
	PartnerID     string             `json:"-"`
	ProfileID     string             `json:"-"`
	Transport     string             `json:"transport,omitempty"`
	InboundURL    string             `json:"inboundUrl,omitempty"`
	ProjectID     string             `json:"-"`
	VersionID     string             `json:"-"`
	VersionNumber int64              `json:"versionNumber,omitempty"`
	CarrierID     string             `json:"-"`
	CustomerID    string             `json:"-"`
	CustomerLabel string             `json:"customerLabel,omitempty"`
	Unchanged     bool               `json:"unchanged,omitempty"`
	Readiness     *trenova.Readiness `json:"readiness,omitempty"`
	FailingTests  []string           `json:"failingTests,omitempty"`
	Notes         []string           `json:"notes"`
	Failure       string             `json:"failure,omitempty"`
}

func (r *Result) note(format string, args ...any) {
	r.Notes = append(r.Notes, fmt.Sprintf(format, args...))
}

func Code(name string) string {
	return CodePrefix + strings.ToUpper(name)
}

func Partner(ctx context.Context, o *Options, def *partner.Definition) *Result {
	result := &Result{
		Partner: def.Name(),
		Role:    string(def.Partner.Role),
		Code:    Code(def.Name()),
		Notes:   []string{},
	}
	if err := o.provision(ctx, def, result); err != nil {
		result.Failure = err.Error()
	}
	return result
}

func (o *Options) provision(ctx context.Context, def *partner.Definition, result *Result) error {
	info, err := o.Sim.Partner(ctx, def.Name())
	if err != nil {
		return fmt.Errorf("read %s from the simulator: %w", def.Name(), err)
	}
	record, err := o.partner(ctx, def, result)
	if err != nil {
		return err
	}
	profile, err := o.profile(ctx, def, &info, record, result)
	if err != nil {
		return err
	}
	if record.DefaultTransportID != profile.ID {
		record.DefaultTransportID = profile.ID
		updated, updateErr := o.Trenova.UpdatePartner(ctx, record)
		if updateErr != nil {
			return fmt.Errorf("make the profile the partner's transport: %w", updateErr)
		}
		record.Version = updated.Version
		result.note("made the %s profile the partner's default transport", profile.Method)
	}
	if err = o.connect(ctx, &info, profile, result); err != nil {
		return err
	}
	if def.Partner.Role == partner.RoleCarrier {
		if err = o.carrier(ctx, def, record, profile, result); err != nil {
			return err
		}
	}
	if err = o.customer(ctx, def, record, result); err != nil {
		return err
	}
	return o.project(ctx, def, record, result)
}

func (o *Options) partner(
	ctx context.Context,
	def *partner.Definition,
	result *Result,
) (*trenova.Partner, error) {
	existing, err := o.Trenova.FindPartner(ctx, result.Code)
	if err != nil {
		return nil, fmt.Errorf("look for partner %s: %w", result.Code, err)
	}
	wanted := &trenova.Partner{
		Kind:               "External",
		Status:             statusActive,
		Code:               result.Code,
		Name:               def.Partner.Display + " (EDI simulator)",
		Description:        "Played by edi-sim: dev/edi-partner-sim/partners/" + def.Name(),
		Timezone:           def.Partner.Zone,
		Country:            "US",
		ContactName:        "EDI simulator",
		ContactEmail:       def.Name() + "@edi-sim.example",
		EnabledForInbound:  true,
		EnabledForOutbound: true,
		Settings:           map[string]any{},
	}
	if existing == nil {
		created, createErr := o.Trenova.CreatePartner(ctx, wanted)
		if createErr != nil {
			return nil, fmt.Errorf("create partner %s: %w", result.Code, createErr)
		}
		result.PartnerID = created.ID
		result.note("created partner %s", result.Code)
		return created, nil
	}
	result.PartnerID = existing.ID
	if existing.Status == statusActive && existing.EnabledForInbound &&
		existing.EnabledForOutbound && existing.Timezone == wanted.Timezone &&
		existing.Name == wanted.Name {
		result.note("partner %s is up to date", result.Code)
		return existing, nil
	}
	wanted.ID, wanted.Version = existing.ID, existing.Version
	wanted.DefaultTransportID = existing.DefaultTransportID
	wanted.CustomerID = existing.CustomerID
	wanted.DefaultTransport = existing.DefaultTransport
	updated, err := o.Trenova.UpdatePartner(ctx, wanted)
	if err != nil {
		return nil, fmt.Errorf("refresh partner %s: %w", result.Code, err)
	}
	updated.DefaultTransport = existing.DefaultTransport
	result.note("refreshed partner %s", result.Code)
	return updated, nil
}

func (o *Options) profile(
	ctx context.Context,
	def *partner.Definition,
	info *sim.PartnerInfo,
	record *trenova.Partner,
	result *Result,
) (*trenova.Profile, error) {
	wanted, err := o.profileFor(def, info, record.ID)
	if err != nil {
		return nil, err
	}
	result.Transport = wanted.Method
	existing := record.DefaultTransport
	if existing == nil {
		existing, err = o.Trenova.PartnerProfile(ctx, record.ID, record.DefaultTransportID,
			wanted.Name)
		if err != nil {
			return nil, fmt.Errorf("look for the %s communication profile: %w", wanted.Method,
				err)
		}
	}
	if existing == nil || existing.ID == "" || existing.Method != wanted.Method {
		created, createErr := o.Trenova.CreateProfile(ctx, wanted)
		if createErr != nil {
			return nil, fmt.Errorf("create the %s communication profile: %w", wanted.Method,
				createErr)
		}
		result.ProfileID = created.ID
		result.note("created the %s communication profile", wanted.Method)
		return created, nil
	}
	wanted.ID, wanted.Version = existing.ID, existing.Version
	updated, err := o.Trenova.UpdateProfile(ctx, wanted)
	if err != nil {
		return nil, fmt.Errorf("refresh the %s communication profile: %w", wanted.Method, err)
	}
	result.ProfileID = updated.ID
	result.note("refreshed the %s communication profile with the simulator's keys",
		wanted.Method)
	return updated, nil
}

func (o *Options) profileFor(
	def *partner.Definition,
	info *sim.PartnerInfo,
	partnerID string,
) (*trenova.Profile, error) {
	profile := &trenova.Profile{
		EDIPartnerID: partnerID,
		Status:       statusActive,
		Name:         "EDI simulator " + def.Name(),
	}
	switch {
	case info.SFTP != nil && def.Partner.Transport.SFTP != nil:
		profile.Method = MethodSFTP
		profile.Config = map[string]any{
			"host":              info.SFTP.Host,
			"port":              info.SFTP.Port,
			"username":          info.SFTP.Username,
			"authMode":          "password",
			"knownHostKey":      info.SFTP.KnownHostKey,
			"inboundDirectory":  info.SFTP.InboundDirectory,
			"outboundDirectory": info.SFTP.OutboundDirectory,
			"archiveDirectory":  info.SFTP.ArchiveDirectory,
			"environment":       "test",
		}
		profile.Secrets = map[string]string{"password": info.SFTP.Password}
	case info.AS2 != nil:
		profile.Method = methodAS2
		config := map[string]any{
			"localAS2Id":                   info.Trenova.AS2ID,
			"partnerAS2Id":                 info.AS2.ID,
			"endpointUrl":                  info.AS2.URL,
			"mdnMode":                      "sync",
			"compressionAlgorithm":         "none",
			"localCertificate":             o.Identity.CertificatePEM,
			"partnerSigningCertificate":    info.AS2.CertificatePEM,
			"partnerEncryptionCertificate": info.AS2.CertificatePEM,
			"requireSignedInbound":         fmt.Sprint(info.AS2.Signed),
			"requireEncryptedInbound":      fmt.Sprint(info.AS2.Encrypted),
			"environment":                  "test",
		}
		if info.AS2.Signed {
			config["signingAlgorithm"] = "sha256"
		}
		if info.AS2.Encrypted {
			config["encryptionAlgorithm"] = "aes256-cbc"
		}
		if info.AS2.Compressed {
			config["compressionAlgorithm"] = "zlib"
		}
		profile.Config = config
		profile.Secrets = map[string]string{"privateKey": o.Identity.KeyPEM}
	default:
		return nil, fmt.Errorf("%s has neither an AS2 endpoint nor an SFTP mailbox in the "+
			"simulator (is the SFTP mailbox listening?)", def.Name())
	}
	return profile, nil
}

func (o *Options) connect(
	ctx context.Context,
	info *sim.PartnerInfo,
	profile *trenova.Profile,
	result *Result,
) error {
	test, err := o.Trenova.TestProfile(ctx, profile.ID)
	switch {
	case err != nil:
		result.note("the connection test could not run: %v", err)
	case !test.Success:
		var failed []string
		for _, check := range test.Checks {
			if !strings.EqualFold(check.Status, "pass") && !strings.EqualFold(check.Status, "ok") &&
				!strings.EqualFold(check.Status, "success") {
				failed = append(failed, check.Name+": "+check.Message)
			}
		}
		return fmt.Errorf("Trenova's connection test failed: %s", strings.Join(failed, "; "))
	default:
		result.note("Trenova's connection test passed")
	}
	if profile.Method != methodAS2 {
		return nil
	}
	if profile.AS2InboundPath == "" {
		return errors.New("the AS2 profile has no inbound URL")
	}
	base := strings.TrimRight(o.PublicURL, "/")
	if base == "" {
		base = o.Trenova.Host()
	}
	result.InboundURL = base + profile.AS2InboundPath
	if _, err = o.Sim.Configure(ctx, info.Name, &sim.TrenovaRequest{
		CertificatePEM: o.Identity.CertificatePEM,
		InboundURL:     result.InboundURL,
		AS2ID:          info.Trenova.AS2ID,
	}); err != nil {
		return fmt.Errorf("give the simulator Trenova's side: %w", err)
	}
	result.note("gave the simulator the profile's inbound URL and Trenova's certificate")
	return nil
}

func (o *Options) carrier(
	ctx context.Context,
	def *partner.Definition,
	record *trenova.Partner,
	profile *trenova.Profile,
	result *Result,
) error {
	scac := def.Partner.Them.SCAC
	carrier, err := o.Trenova.FindCarrier(ctx, scac)
	if err != nil {
		return fmt.Errorf("look for the carrier %s: %w", scac, err)
	}
	if carrier == nil {
		carrier, err = o.Trenova.CreateCarrier(ctx, map[string]any{
			"code":             scac,
			"name":             def.Partner.Display + " (EDI simulator)",
			"status":           statusActive,
			"carrierType":      "Common",
			"complianceStatus": "Qualified",
			"safetyRating":     "Satisfactory",
			"paymentMethod":    "Check",
			"scac":             scac,
		})
		if err != nil {
			return fmt.Errorf("create the carrier %s: %w", scac, err)
		}
		carrier, err = o.Trenova.Carrier(ctx, fmt.Sprint(carrier["id"]))
		if err != nil {
			return fmt.Errorf("read the carrier %s: %w", scac, err)
		}
		result.note("created the carrier %s", scac)
	}
	result.CarrierID = fmt.Sprint(carrier["id"])
	channels, _ := carrier["ediChannels"].([]any)
	for _, raw := range channels {
		channel, _ := raw.(map[string]any)
		if channel["ediPartnerId"] == record.ID && channel["isDefault"] == true {
			result.note("carrier %s already tenders to the partner by EDI", scac)
			return nil
		}
	}
	profileID := profile.ID
	kept := make([]any, 0, len(channels)+1)
	for _, raw := range channels {
		channel, _ := raw.(map[string]any)
		if channel == nil || channel["ediPartnerId"] == record.ID {
			continue
		}
		channel["isDefault"] = false
		kept = append(kept, channel)
	}
	kept = append(kept, map[string]any{
		"ediPartnerId":           record.ID,
		"communicationProfileId": profileID,
		"isDefault":              true,
	})
	carrier["ediChannels"] = kept
	if _, err = o.Trenova.UpdateCarrier(ctx, carrier); err != nil {
		return fmt.Errorf("link carrier %s to the partner: %w", scac, err)
	}
	result.note("linked carrier %s to the partner as its default EDI channel", scac)
	return nil
}

func (o *Options) project(
	ctx context.Context,
	def *partner.Definition,
	record *trenova.Partner,
	result *Result,
) error {
	files, samples, err := localProject(def)
	if err != nil {
		return err
	}
	project, err := o.Trenova.PartnerProject(ctx, record.ID)
	if err != nil {
		return fmt.Errorf("read the partner's Lading project: %w", err)
	}
	if project == nil {
		if project, err = o.Trenova.CreateProject(ctx, starter(def, record.ID)); err != nil {
			return fmt.Errorf("create the partner's Lading project: %w", err)
		}
		result.note("created the Lading project")
	}
	result.ProjectID = project.ID
	versions, err := o.Trenova.Versions(ctx, project.ID)
	if err != nil {
		return fmt.Errorf("list the project's versions: %w", err)
	}
	if project.ActiveVersionID != "" {
		active, detailErr := o.Trenova.VersionDetail(ctx, project.ActiveVersionID)
		if detailErr != nil {
			return fmt.Errorf("read the active version: %w", detailErr)
		}
		if sameProject(active.Version.Files, active.Samples, files, samples) {
			result.VersionID, result.VersionNumber = active.Version.ID, active.Version.Number
			result.Readiness, result.Unchanged = active.Version.Readiness, true
			result.note("the active version %d already runs these files", active.Version.Number)
			return nil
		}
	}
	draft, err := o.draft(ctx, project, versions, result)
	if err != nil {
		return err
	}
	saved, err := o.Trenova.SaveDraft(ctx, draft.ID, &trenova.SaveDraft{
		Files:   files,
		Samples: samples,
		Notes:   "Saved by edi-sim run from partners/" + def.Name() + "/lading",
		Version: draft.Version,
	})
	if err != nil {
		return fmt.Errorf("save the draft: %w", err)
	}
	result.VersionID, result.VersionNumber = saved.ID, saved.Number
	result.note("saved %d files and %d samples to draft %d", len(files), len(samples),
		saved.Number)
	if err = o.test(ctx, saved, result); err != nil {
		return err
	}
	if project.ActiveVersionID != "" {
		if err = o.replay(ctx, saved, result); err != nil {
			return err
		}
	}
	if _, err = o.Trenova.VersionAction(ctx, saved.ID, "certify"); err != nil {
		return fmt.Errorf("%w: certifying version %d failed: %w", ErrNotReady, saved.Number, err)
	}
	result.note("certified version %d", saved.Number)
	if _, err = o.Trenova.VersionAction(ctx, saved.ID, "activate"); err != nil {
		return fmt.Errorf("activate version %d: %w", saved.Number, err)
	}
	result.note("activated version %d", saved.Number)
	return nil
}

func (o *Options) draft(
	ctx context.Context,
	project *trenova.Project,
	versions []*trenova.Version,
	result *Result,
) (*trenova.Version, error) {
	for _, version := range versions {
		if version.Status == versionDraft {
			result.note("reusing draft %d", version.Number)
			return version, nil
		}
	}
	from := project.ActiveVersionID
	if from == "" && len(versions) > 0 {
		latest := slices.MaxFunc(versions, func(a, b *trenova.Version) int {
			return int(a.Number - b.Number)
		})
		from = latest.ID
	}
	if from == "" {
		return nil, errors.New("the project has no version to start a draft from")
	}
	draft, err := o.Trenova.NewDraft(ctx, project.ID, from)
	if err != nil {
		return nil, fmt.Errorf("start a draft: %w", err)
	}
	result.note("started draft %d", draft.Number)
	return draft, nil
}

func (o *Options) test(ctx context.Context, version *trenova.Version, result *Result) error {
	report, err := o.Trenova.TestVersion(ctx, version.ID)
	if err != nil {
		return fmt.Errorf("test version %d: %w", version.Number, err)
	}
	readiness := report.Readiness
	result.Readiness = &trenova.Readiness{
		Errors:               readiness.Errors,
		Failing:              readiness.Failing,
		Tests:                readiness.Tests,
		UncoveredFlowClauses: readiness.UncoveredFlowClauses,
		Ready:                readiness.Ready,
	}
	for _, test := range report.Tests {
		if test.Passed {
			continue
		}
		reason := ""
		if len(test.Failures) > 0 {
			reason = ": " + test.Failures[0].Message.String()
		}
		result.FailingTests = append(result.FailingTests,
			fmt.Sprintf("%s %s%s", test.Path, test.Name, reason))
	}
	result.note("tested version %d: %d tests, %d failing, %d errors, %d uncovered flow clauses",
		version.Number, readiness.Tests, readiness.Failing, readiness.Errors,
		readiness.UncoveredFlowClauses)
	if readiness.Ready {
		return nil
	}
	return fmt.Errorf("%w: version %d is not ready to certify: %d errors in the files, "+
		"%d of %d tests failing, %d flow clauses no test reaches",
		ErrNotReady, version.Number, readiness.Errors, readiness.Failing, readiness.Tests,
		readiness.UncoveredFlowClauses)
}

func (o *Options) replay(ctx context.Context, version *trenova.Version, result *Result) error {
	detail, err := o.Trenova.VersionDetail(ctx, version.ID)
	if err != nil {
		return fmt.Errorf("read version %d: %w", version.Number, err)
	}
	now := time.Now()
	for _, item := range detail.Version.Manifest.Maps {
		if item.Direction != lading.DirectionInbound || item.Source != "x12" {
			continue
		}
		replay, startErr := o.Trenova.StartReplay(ctx, version.ID, item.Name,
			now.Add(-replayWindow).Unix(), now.Unix())
		if startErr != nil {
			return fmt.Errorf("replay %s: %w", item.Name, startErr)
		}
		var finished *trenova.Replay
		waitErr := poll.Until(ctx, replayWithin, func(ctx context.Context) (bool, error) {
			current, readErr := o.Trenova.Replay(ctx, replay.ID)
			if readErr != nil {
				return false, readErr
			}
			finished = current
			return current.Status == replayStatusDone || current.Status == replayStatusError, nil
		})
		if waitErr != nil {
			return fmt.Errorf("replay %s did not finish: %w", item.Name, waitErr)
		}
		if finished.Status == replayStatusError {
			return fmt.Errorf("%w: the replay of %s failed: %s", ErrNotReady, item.Name,
				finished.Error)
		}
		if summary := finished.Summary; summary != nil {
			result.note("replayed %s over %d messages: %d differ, %d newly rejected",
				item.Name, summary.Total, summary.Differ, summary.NewlyRejected)
		}
	}
	return nil
}

func starter(def *partner.Definition, partnerID string) *trenova.CreateProject {
	role := roleCarrier
	if def.Partner.Role == partner.RoleCarrier {
		role = roleBroker
	}
	them, us := def.Partner.Them, def.Partner.Trenova
	separators := def.Partner.Envelope.Separators
	return &trenova.CreateProject{
		PartnerID: partnerID,
		Role:      role,
		Sets:      []string{"204", "990", "214", "210"},
		Them:      &[2]string{them.Qualifier, them.ID},
		Us:        &[2]string{us.Qualifier, us.ID},
		Group:     &[2]string{them.Group, us.Group},
		Separators: []string{
			separators.Element, separators.Segment, separators.Component, separators.Repetition,
		},
		Usage: def.Partner.Envelope.Usage,
	}
}

func localProject(def *partner.Definition) ([]lading.File, []lading.Sample, error) {
	sourceFiles, sourceSamples, err := def.LadingProject()
	if err != nil {
		return nil, nil, err
	}
	if len(sourceFiles) == 0 {
		return nil, nil, fmt.Errorf("%s has no Lading files in %s/lading", def.Name(), def.Dir)
	}
	files := make([]lading.File, 0, len(sourceFiles))
	for _, file := range sourceFiles {
		files = append(files, lading.File{Path: file.Path, Source: file.Content})
	}
	samples := make([]lading.Sample, 0, len(sourceSamples))
	for _, sample := range sourceSamples {
		samples = append(samples, lading.Sample{Path: sample.Path, Content: sample.Content})
	}
	return files, samples, nil
}

func sameProject(
	files []lading.File,
	samples []lading.Sample,
	wantFiles []lading.File,
	wantSamples []lading.Sample,
) bool {
	byPath := func(a, b lading.File) int { return strings.Compare(a.Path, b.Path) }
	samplePath := func(a, b lading.Sample) int { return strings.Compare(a.Path, b.Path) }
	files, samples = slices.Clone(files), slices.Clone(samples)
	slices.SortFunc(files, byPath)
	slices.SortFunc(samples, samplePath)
	return slices.Equal(files, wantFiles) && slices.Equal(samples, wantSamples)
}
