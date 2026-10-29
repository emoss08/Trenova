package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	kindUnexportedFunc = "unexported-func"
	kindExportedFunc   = "exported-func"
	kindExportedType   = "exported-type"
	repositoryPortName = "ShipmentRepository"
)

var (
	errNoProbe        = errors.New("probe target not found")
	errAmbiguousProbe = errors.New("probe target is ambiguous")
	generatedHeader   = regexp.MustCompile(`(?m)^// Code generated .* DO NOT EDIT\.$`)
)

var plannedDomainPackages = []string{
	"usstate", "platformcatalog", "editchange", "configspec", "settingversion",
	"tenant", "iam", "permission", "apikey", "session", "onboarding",
	"instancebootstrap", "platformplan", "subscription",
	"audit", "notification", "email", "customfield", "homelayout", "pagedraft",
	"pagefavorite", "sidebarpreference", "tablelayout", "tableconfiguration",
	"tablechangealert", "bulkedit", "integration", "report", "dataentrycontrol",
	"commodity", "hazardousmaterial", "hazmatsegregationrule", "dothazmatreference",
	"servicetype", "shipmenttype", "modeprofile", "holdreason", "accessorialcharge",
	"exchangerate", "documenttype",
	"location", "locationcategory", "geofence", "jurisdictionrule",
	"distancecalculation", "distancecontrol", "distanceoverride", "distanceprofile",
	"storedmileage", "rategeo",
	"equipmenttype", "equipmentmanufacturer", "tractor", "trailer",
	"equipmentcontinuity", "fleetcode",
	"customer", "carrier", "carriercapacity", "carrierintel",
	"worker", "permit", "telematics",
	"document", "documentaiextraction", "documentcontent", "documentpacketrule",
	"documentsearchprojection", "documenttemplate",
	"documentupload", "capture", "inboundmessage", "extractioneval",
	"extractionrollout", "extractionshadow",
	"rateagreement", "rateconfirmation", "rateimport", "ratematrix", "ratequote",
	"ratesimulation", "ratezone", "fuelsurcharge", "formulatemplate",
	"shipment", "shipmentevent", "shipmentstate", "shipmentsuggestion", "order",
	"recurringshipment", "tender", "dispatchcontrol", "servicefailure", "detention",
	"weatheralert", "documentshipmentdraft",
	"edi",
	"accounttype", "glaccount", "glbalance", "journalentry", "journalreversal",
	"journalsource", "manualjournal", "fiscalclose", "fiscalperiod", "fiscalyear",
	"accountingsync",
	"invoice", "invoiceadjustment", "invoicerun", "billingqueue",
	"billingqueuefilterpreset", "billingtransfer", "customerledger",
	"customerpayment", "bankreceipt", "bankreceiptbatch", "bankreceiptworkitem",
	"latecharge", "costingcontrol",
	"driversettlement", "carriersettlement", "driverpay", "fuelpurchase", "ifta",
	"agent", "agentdefinition", "agentdryrun", "agentextension", "agentlint",
	"agentquality", "agentroster", "agentshadow", "agentwait", "aiaudit",
	"aicontrolsummary", "aicorrection", "aifeedback", "aiprovider", "airetrieval",
	"aitraining", "aituneup", "aiusage", "assistantartifact", "conversation",
	"conversationschedule", "deskcase", "insight", "watchtower", "briefing",
	"shipmentbrief", "shipmentimportchat",
}

type probe struct {
	id      string
	kind    string
	pkg     string
	file    string
	reverse int
}

type probeResolver struct {
	all    *listing
	linked *listing
	counts map[string]int
}

func runProbes(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("probes", flag.ContinueOnError)
	tags := fs.String("tags", "", "build tags")
	target := fs.String("target", "", "main package pattern, for example ./cmd/cli")
	if err := fs.Parse(args); err != nil {
		return err
	}

	all, err := goList(ctx, listRequest{tags: *tags, patterns: []string{"./..."}})
	if err != nil {
		return err
	}
	linked, err := goList(ctx, listRequest{tags: *tags, deps: true, patterns: []string{*target}})
	if err != nil {
		return err
	}

	r := &probeResolver{all: all, linked: linked, counts: all.dependentCounts()}
	probes, err := r.resolve()
	if err != nil {
		return err
	}

	for _, p := range probes {
		fmt.Printf("%s\t%s\t%s\t%s\t%d\n", p.id, p.kind, p.pkg, p.file, p.reverse)
	}

	return nil
}

func (r *probeResolver) resolve() ([]probe, error) {
	service, err := r.uniqueSuffix("/shipmentwrite")
	if err != nil {
		return nil, err
	}
	resolver, err := r.uniqueSuffix("/shipmentresolver")
	if err != nil {
		return nil, err
	}
	domain, err := r.uniqueSuffix("/domain/shipment")
	if err != nil {
		return nil, err
	}
	port, portFile, err := r.repositoryPort(service)
	if err != nil {
		return nil, err
	}
	smallest, err := r.smallestPlannedDomain()
	if err != nil {
		return nil, err
	}

	specs := []struct {
		id, kind, pkg, file string
	}{
		{"S2", kindUnexportedFunc, service, ""},
		{"S3", kindUnexportedFunc, resolver, ""},
		{"S4", kindExportedFunc, domain, ""},
		{"S5", kindExportedType, port, portFile},
		{"S6", kindExportedFunc, smallest, ""},
	}

	probes := make([]probe, 0, len(specs))
	for _, s := range specs {
		file := s.file
		if file == "" {
			if file, err = r.sourceFile(s.pkg); err != nil {
				return nil, err
			}
		}
		probes = append(probes, probe{
			id:      s.id,
			kind:    s.kind,
			pkg:     r.all.trim(s.pkg),
			file:    file,
			reverse: r.counts[s.pkg],
		})
	}

	return probes, nil
}

func (r *probeResolver) uniqueSuffix(suffix string) (string, error) {
	matches := r.all.withSuffix(suffix)
	switch len(matches) {
	case 0:
		return "", fmt.Errorf("%w: no package ends with %q", errNoProbe, suffix)
	case 1:
		return matches[0], nil
	default:
		return "", fmt.Errorf("%w: %q matches %s", errAmbiguousProbe, suffix, matches)
	}
}

func (r *probeResolver) repositoryPort(service string) (pkg, file string, err error) {
	var found []string
	for _, imp := range r.all.pkgs[service].imports {
		info, listed := r.all.pkgs[imp]
		if !listed {
			continue
		}
		for _, name := range info.goFiles {
			path := filepath.Join(info.dir, name)
			ok, derr := declaresInterface(path, repositoryPortName)
			if derr != nil {
				return "", "", derr
			}
			if ok {
				found = append(found, imp+"\t"+path)
			}
		}
	}

	switch len(found) {
	case 0:
		return "", "", fmt.Errorf(
			"%w: no direct import of %s declares %s",
			errNoProbe,
			service,
			repositoryPortName,
		)
	case 1:
		var path string
		pkg, path, _ = strings.Cut(found[0], "\t")
		file, err = relativeToWorkdir(path)
		return pkg, file, err
	default:
		return "", "", fmt.Errorf(
			"%w: %s declared in %s",
			errAmbiguousProbe,
			repositoryPortName,
			found,
		)
	}
}

func declaresInterface(path, name string) (bool, error) {
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.SkipObjectResolution)
	if err != nil {
		return false, fmt.Errorf("parse %s: %w", path, err)
	}

	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.TYPE {
			continue
		}
		for _, spec := range gen.Specs {
			ts, isType := spec.(*ast.TypeSpec)
			if !isType || ts.Name.Name != name {
				continue
			}
			if _, isIface := ts.Type.(*ast.InterfaceType); isIface {
				return true, nil
			}
		}
	}

	return false, nil
}

func (r *probeResolver) smallestPlannedDomain() (string, error) {
	best, bestCount := "", -1
	for _, name := range plannedDomainPackages {
		matches := r.all.withSuffix("/domain/" + name)
		if len(matches) == 0 {
			fmt.Fprintf(os.Stderr, "buildbench: no package for planned domain %q\n", name)
			continue
		}
		if len(matches) > 1 {
			return "", fmt.Errorf("%w: domain %q matches %s", errAmbiguousProbe, name, matches)
		}
		pkg := matches[0]
		if _, isLinked := r.linked.pkgs[pkg]; !isLinked {
			continue
		}
		count := r.counts[pkg]
		if bestCount < 0 || count < bestCount || (count == bestCount && pkg < best) {
			best, bestCount = pkg, count
		}
	}

	if best == "" {
		return "", fmt.Errorf("%w: no planned domain package is linked by the target", errNoProbe)
	}

	return best, nil
}

func (r *probeResolver) sourceFile(pkg string) (string, error) {
	info := r.all.pkgs[pkg]
	for _, name := range info.goFiles {
		path := filepath.Join(info.dir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("read %s: %w", path, err)
		}
		if !generatedHeader.Match(data) {
			return relativeToWorkdir(path)
		}
	}

	return "", fmt.Errorf("%w: %s has no hand-written Go file", errNoProbe, pkg)
}

func relativeToWorkdir(path string) (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}
	rel, err := filepath.Rel(wd, path)
	if err != nil {
		return "", fmt.Errorf("relativize %s: %w", path, err)
	}

	return rel, nil
}
