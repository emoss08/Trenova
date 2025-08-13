package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/bytedance/sonic"

	ackpkg "github.com/emoss08/trenova/shared/edi/internal/ack"
	"github.com/emoss08/trenova/shared/edi/internal/config"
	"github.com/emoss08/trenova/shared/edi/internal/mapper"
	tx204 "github.com/emoss08/trenova/shared/edi/internal/tx/tx204"
	"github.com/emoss08/trenova/shared/edi/internal/validation"
	"github.com/emoss08/trenova/shared/edi/internal/x12"
)

type output struct {
	Delimiters x12.Delimiters `json:"delimiters"`
	Segments   []x12.Segment  `json:"segments"`
}

func main() {
	var pretty bool
	var format string
	var doValidate bool
	var lenient bool
	var multi string
	var failOnErr bool
	var ack bool
	var schemaPath string
	var profilePath string
	var optElem string
	var optComp string
	var optSeg string
	var optRep string
	var optDelims string
	flag.BoolVar(&pretty, "pretty", true, "pretty print JSON output")
	flag.StringVar(&format, "format", "segments", "output format: segments|204|shipment")
	flag.BoolVar(&doValidate, "validate", false, "run 204 validation and include issues in output")
	flag.BoolVar(
		&lenient,
		"lenient",
		false,
		"use lenient validation (fewer required segments, SE count warning)",
	)
	flag.StringVar(
		&multi,
		"multi",
		"off",
		"when format=shipment: off|ndjson|array to emit multiple shipments per file (splitting ST/SE)",
	)
	flag.BoolVar(
		&failOnErr,
		"fail-on-error",
		false,
		"exit with non-zero status if validation errors are found",
	)
	flag.BoolVar(&ack, "ack", false, "emit an acknowledgment (997 for 004010, 999 for 0050x/0060x)")
	var ackJSON bool
	flag.BoolVar(
		&ackJSON,
		"ack-json",
		false,
		"when used with --ack and --validate, output JSON containing both ack and validation output instead of only ack",
	)
	flag.StringVar(&schemaPath, "schema", "", "optional JSON schema file for validation rules")
	flag.StringVar(
		&profilePath,
		"profile",
		"",
		"optional partner profile JSON including schema and default delimiters",
	)
	flag.StringVar(
		&optDelims,
		"delims",
		"",
		"override delimiters as elem,comp,seg,rep (rep optional)",
	)
	flag.StringVar(&optElem, "element", "", "override element separator (single char)")
	flag.StringVar(&optComp, "component", "", "override component separator (single char)")
	flag.StringVar(&optSeg, "segment", "", "override segment terminator (single char)")
	flag.StringVar(&optRep, "repetition", "", "override repetition separator (single char)")
	flag.Parse()

	if flag.NArg() < 1 {
		usage()
		os.Exit(2)
	}

	path := flag.Arg(0)
	raw, err := ioutil.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read error: %v\n", err)
		os.Exit(1)
	}

	delims, err := x12.DetectDelimiters(raw)
	if err != nil {
		fmt.Fprintf(os.Stderr, "delimiter detect error: %v\n", err)
		os.Exit(1)
	}
	var partner *config.PartnerConfig
	if profilePath != "" {
		if pc, err := config.Load(profilePath); err == nil {
			partner = pc
			pc.ApplyDelimiters(&delims)
			if schemaPath == "" && pc.SchemaPath != "" {
				schemaPath = pc.SchemaPath
			}
		} else {
			fmt.Fprintf(os.Stderr, "profile load error: %v\n", err)
		}
	}
	// Apply overrides if provided
	if optDelims != "" {
		parts := strings.Split(optDelims, ",")
		if len(parts) >= 1 && len(parts[0]) > 0 {
			delims.Element = parts[0][0]
		}
		if len(parts) >= 2 && len(parts[1]) > 0 {
			delims.Component = parts[1][0]
		}
		if len(parts) >= 3 && len(parts[2]) > 0 {
			delims.Segment = parts[2][0]
		}
		if len(parts) >= 4 && len(parts[3]) > 0 {
			delims.Repetition = parts[3][0]
		}
	}
	if optElem != "" {
		delims.Element = optElem[0]
	}
	if optComp != "" {
		delims.Component = optComp[0]
	}
	if optSeg != "" {
		delims.Segment = optSeg[0]
	}
	if optRep != "" {
		delims.Repetition = optRep[0]
	}

	segs, err := x12.ParseSegments(raw, delims)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse error: %v\n", err)
		os.Exit(1)
	}

	switch format {
	case "segments":
		out := output{Delimiters: delims, Segments: segs}
		if doValidate {
			prof := validation.DefaultProfileForVersion(x12.ExtractVersion(segs))
			if partner != nil {
				prof = partner.ApplyValidation(prof)
			}
			if lenient {
				prof.Strictness = validation.Lenient
				prof.EnforceSECount = false
				prof.RequirePickupAndDelivery = false
			}
			issues := validation.Validate204WithProfile(segs, prof)
			if schemaPath == "" && x12.ExtractVersion(segs) == "004010" {
				schemaPath = "testdata/schema/generic-204-4010.json"
			}
			if schemaPath != "" {
				if sch, err := validation.LoadSchema(schemaPath); err == nil {
					issues = append(issues, validation.ValidateWithSchema(segs, sch)...)
				} else {
					fmt.Fprintf(os.Stderr, "schema load error: %v\n", err)
				}
			}
			ver := x12.ExtractVersion(segs)
			if ack && ver == "004010" {
				if schemaPath == "" {
					schemaPath = "testdata/schema/generic-204-4010.json"
				}
				delims, _ := x12.DetectDelimiters(raw)
				edi := ackpkg.Generate997(segs, delims, issues)
				if ackJSON {
					emitJSON(struct {
						Ack    string             `json:"ack"`
						Output output             `json:"output"`
						Issues []validation.Issue `json:"issues"`
					}{Ack: edi, Output: out, Issues: issues}, pretty)
				} else {
					fmt.Println(edi)
				}
			} else if ack && (strings.HasPrefix(ver, "005") || strings.HasPrefix(ver, "006")) {
				blocks := x12.SplitTransactions(segs)
				accepted := make([]bool, len(blocks))
				for i, b := range blocks {
					if b.SetID != "204" {
						accepted[i] = true
						continue
					}
					txIssues := runValidationWithOptionalSchema(b.Segs, schemaPath, partner, lenient)
					txIssues = filterEnvelopeIssues(txIssues)
					accepted[i] = !hasError(txIssues)
				}
				delims, _ := x12.DetectDelimiters(raw)
				edi := ackpkg.Generate999(segs, delims, blocks, accepted)
				if ackJSON {
					emitJSON(struct {
						Ack    string             `json:"ack"`
						Output output             `json:"output"`
						Issues []validation.Issue `json:"issues"`
					}{Ack: edi, Output: out, Issues: issues}, pretty)
				} else {
					fmt.Println(edi)
				}
			} else {
				emitJSON(struct {
					output
					Issues []validation.Issue `json:"issues"`
				}{output: out, Issues: issues}, pretty)
			}
		} else {
			emitJSON(out, pretty)
		}
	case "204":
		lt := tx204.BuildFromSegments(segs)
		if doValidate {
			prof := validation.DefaultProfileForVersion(x12.ExtractVersion(segs))
			if partner != nil {
				prof = partner.ApplyValidation(prof)
			}
			if lenient {
				prof.Strictness = validation.Lenient
				prof.EnforceSECount = false
				prof.RequirePickupAndDelivery = false
			}
			issues := validation.Validate204WithProfile(segs, prof)
			if schemaPath == "" && x12.ExtractVersion(segs) == "004010" {
				schemaPath = "testdata/schema/generic-204-4010.json"
			}
			if schemaPath != "" {
				if sch, err := validation.LoadSchema(schemaPath); err == nil {
					issues = append(issues, validation.ValidateWithSchema(segs, sch)...)
				} else {
					fmt.Fprintf(os.Stderr, "schema load error: %v\n", err)
				}
			}
			ver := x12.ExtractVersion(segs)
			if ack && ver == "004010" {
				if schemaPath == "" {
					schemaPath = "testdata/schema/generic-204-4010.json"
				}
				delims, _ := x12.DetectDelimiters(raw)
				edi := ackpkg.Generate997(segs, delims, issues)
				if ackJSON {
					emitJSON(struct {
						Ack        string             `json:"ack"`
						LoadTender tx204.LoadTender   `json:"load_tender"`
						Issues     []validation.Issue `json:"issues"`
					}{Ack: edi, LoadTender: lt, Issues: issues}, pretty)
				} else {
					fmt.Println(edi)
				}
			} else if ack && (strings.HasPrefix(ver, "005") || strings.HasPrefix(ver, "006")) {
				blocks := x12.SplitTransactions(segs)
				accepted := make([]bool, len(blocks))
				for i, b := range blocks {
					if b.SetID != "204" {
						accepted[i] = true
						continue
					}
					txIssues := runValidationWithOptionalSchema(b.Segs, schemaPath, partner, lenient)
					txIssues = filterEnvelopeIssues(txIssues)
					accepted[i] = !hasError(txIssues)
				}
				delims, _ := x12.DetectDelimiters(raw)
				edi := ackpkg.Generate999(segs, delims, blocks, accepted)
				if ackJSON {
					emitJSON(struct {
						Ack        string             `json:"ack"`
						LoadTender tx204.LoadTender   `json:"load_tender"`
						Issues     []validation.Issue `json:"issues"`
					}{Ack: edi, LoadTender: lt, Issues: issues}, pretty)
				} else {
					fmt.Println(edi)
				}
			} else {
				emitJSON(struct {
					LoadTender tx204.LoadTender   `json:"load_tender"`
					Issues     []validation.Issue `json:"issues"`
				}{LoadTender: lt, Issues: issues}, pretty)
			}
		} else {
			emitJSON(lt, pretty)
		}
	case "shipment":
		lt := tx204.BuildFromSegments(segs)
		// Build mapper options from partner profile
		opts := mapper.DefaultOptions()
		if partner != nil {
			if len(partner.References) > 0 {
				opts.RefMap = mapper.MergeRefMaps(opts.RefMap, partner.References)
			}
			if len(partner.PartyRoles) > 0 {
				opts.PartyRoles = partner.PartyRoles
			}
			if len(partner.StopTypeMap) > 0 {
				opts.StopTypeMap = partner.StopTypeMap
			}
			if len(partner.ShipmentIDQuals) > 0 {
				opts.ShipmentIDQuals = partner.ShipmentIDQuals
			}
			if partner.ShipmentIDMode != "" {
				opts.ShipmentIDMode = partner.ShipmentIDMode
			}
			if partner.CarrierSCACFallback != "" {
				opts.CarrierSCACFallback = partner.CarrierSCACFallback
			}
			if partner.IncludeRawL11 {
				opts.IncludeRawL11 = true
			}
			if len(partner.RawL11Filter) > 0 {
				opts.RawL11Filter = partner.RawL11Filter
			}
			if len(partner.EquipmentTypeMap) > 0 {
				opts.EquipmentTypeMap = partner.EquipmentTypeMap
			}
			if partner.EmitISODateTime {
				opts.EmitISODateTime = true
			}
			if partner.Timezone != "" {
				opts.Timezone = partner.Timezone
			}
			if len(partner.ServiceLevelQuals) > 0 {
				opts.ServiceLevelQuals = partner.ServiceLevelQuals
			}
			if len(partner.ServiceLevelMap) > 0 {
				opts.ServiceLevelMap = partner.ServiceLevelMap
			}
			if len(partner.AccessorialQuals) > 0 {
				opts.AccessorialQuals = partner.AccessorialQuals
			}
			if len(partner.AccessorialMap) > 0 {
				opts.AccessorialMap = partner.AccessorialMap
			}
		}
		// Multi-transaction handling
		if multi != "off" {
			// Split ST/SE and emit multiple shipments
			blocks := x12.SplitTransactions(segs)
			// Filter to 204s only
			txs := make([]x12.TxBlock, 0, len(blocks))
			for _, b := range blocks {
				if b.SetID == "204" {
					txs = append(txs, b)
				}
			}
			if len(txs) == 0 {
				fmt.Fprintln(os.Stderr, "no 204 transactions found")
				os.Exit(0)
			}
			anyErr := false
			switch multi {
			case "ndjson":
				for _, b := range txs {
					lt := tx204.BuildFromSegments(b.Segs)
					shp := mapper.ToShipmentWithOptions(lt, opts)
					if doValidate {
						// Autoload generic schema if empty and version is 004010
						if schemaPath == "" && x12.ExtractVersion(b.Segs) == "004010" {
							schemaPath = "testdata/schema/generic-204-4010.json"
						}
						// Validate per ST..SE; filter envelope errors
						issues := runValidationWithOptionalSchema(
							b.Segs,
							schemaPath,
							partner,
							lenient,
						)
						issues = filterEnvelopeIssues(issues)
						if hasError(issues) {
							anyErr = true
						}
						// Optional segments
						if partner != nil && partner.IncludeSegments {
							emitJSON(struct {
								Shipment any                `json:"shipment"`
								Issues   []validation.Issue `json:"issues"`
								Segments *[]x12.Segment     `json:"segments,omitempty"`
							}{Shipment: shp, Issues: issues, Segments: &b.Segs}, false)
						} else {
							emitJSON(struct {
								Shipment any                `json:"shipment"`
								Issues   []validation.Issue `json:"issues"`
							}{Shipment: shp, Issues: issues}, false)
						}
					} else {
						if partner != nil && partner.IncludeSegments {
							emitJSON(struct {
								Shipment any            `json:"shipment"`
								Segments *[]x12.Segment `json:"segments,omitempty"`
							}{Shipment: shp, Segments: &b.Segs}, false)
						} else {
							emitJSON(struct {
								Shipment any `json:"shipment"`
							}{Shipment: shp}, false)
						}
					}
				}
			case "array":
				type item struct {
					Shipment any                `json:"shipment"`
					Issues   []validation.Issue `json:"issues,omitempty"`
				}
				out := make([]item, 0, len(txs))
				for _, b := range txs {
					lt := tx204.BuildFromSegments(b.Segs)
					shp := mapper.ToShipmentWithOptions(lt, opts)
					it := item{Shipment: shp}
					if doValidate {
						if schemaPath == "" && x12.ExtractVersion(b.Segs) == "004010" {
							schemaPath = "testdata/schema/generic-204-4010.json"
						}
						issues := runValidationWithOptionalSchema(
							b.Segs,
							schemaPath,
							partner,
							lenient,
						)
						it.Issues = filterEnvelopeIssues(issues)
						if hasError(it.Issues) {
							anyErr = true
						}
					}
					out = append(out, it)
				}
				emitJSON(out, pretty)
			default:
				fmt.Fprintf(os.Stderr, "invalid --multi: %s\n", multi)
				os.Exit(2)
			}
			if failOnErr && anyErr {
				os.Exit(1)
			}
			return
		}

		shp := mapper.ToShipmentWithOptions(lt, opts)
		if doValidate {
			prof := validation.DefaultProfileForVersion(x12.ExtractVersion(segs))
			if partner != nil {
				prof = partner.ApplyValidation(prof)
			}
			if lenient {
				prof.Strictness = validation.Lenient
				prof.EnforceSECount = false
				prof.RequirePickupAndDelivery = false
			}
			issues := validation.Validate204WithProfile(segs, prof)
			if schemaPath != "" {
				if sch, err := validation.LoadSchema(schemaPath); err == nil {
					issues = append(issues, validation.ValidateWithSchema(segs, sch)...)
				} else {
					fmt.Fprintf(os.Stderr, "schema load error: %v\n", err)
				}
			}
			type outWrap struct {
				Shipment any                `json:"shipment"`
				Issues   []validation.Issue `json:"issues"`
				Segments *[]x12.Segment     `json:"segments,omitempty"`
			}
			var segPtr *[]x12.Segment
			if partner != nil && partner.IncludeSegments {
				segPtr = &segs
			}
			emitJSON(outWrap{Shipment: shp, Issues: issues, Segments: segPtr}, pretty)
			if failOnErr && hasError(issues) {
				os.Exit(1)
			}
		} else {
			if partner != nil && partner.IncludeSegments {
				type outWrap struct {
					Shipment any            `json:"shipment"`
					Segments *[]x12.Segment `json:"segments,omitempty"`
				}
				segPtr := &segs
				emitJSON(outWrap{Shipment: shp, Segments: segPtr}, pretty)
			} else {
				emitJSON(shp, pretty)
			}
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown format: %s\n", format)
		os.Exit(2)
	}
}

func usage() {
	msg := `Usage: edi-cli [--pretty] [--validate] [--lenient] [--schema path] [--profile path] [--delims elem,comp,seg,rep] [--element c --component c --segment c --repetition c] [--format segments|204|shipment] [--multi off|ndjson|array] [--ack] <path-to-edi>

Parses an X12 EDI file (e.g., 204) and prints JSON. Default format is segments; use --format 204 to emit a minimal typed 204 load tender. Use --validate to include validation issues.`
	fmt.Fprintln(os.Stderr, msg)
}

func emitJSON(v any, pretty bool) {
	var b []byte
	var err error
	if pretty {
		b, err = sonic.ConfigFastest.MarshalIndent(v, "", "  ")
	} else {
		b, err = sonic.ConfigFastest.Marshal(v)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "json encode error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(b))
}

// runValidationWithOptionalSchema validates a slice of segments with profile and optional schema path,
// honoring lenient flag.
func runValidationWithOptionalSchema(
	segs []x12.Segment,
	schemaPath string,
	partner *config.PartnerConfig,
	lenient bool,
) []validation.Issue {
	prof := validation.DefaultProfileForVersion(x12.ExtractVersion(segs))
	if partner != nil {
		prof = partner.ApplyValidation(prof)
	}
	if lenient {
		prof.Strictness = validation.Lenient
		prof.EnforceSECount = false
		prof.RequirePickupAndDelivery = false
	}
	issues := validation.Validate204WithProfile(segs, prof)
	if schemaPath != "" {
		if sch, err := validation.LoadSchema(schemaPath); err == nil {
			issues = append(issues, validation.ValidateWithSchema(segs, sch)...)
		} else {
			fmt.Fprintf(os.Stderr, "schema load error: %v\n", err)
		}
	}
	return issues
}

// filterEnvelopeIssues removes validation findings that are specific to envelopes (ISA/GS/GE/IEA)
// so that per-transaction (ST..SE) validation remains meaningful.
func filterEnvelopeIssues(issues []validation.Issue) []validation.Issue {
	filtered := make([]validation.Issue, 0, len(issues))
	skip := map[string]struct{}{
		"ISA.MISSING":       {},
		"GS.MISSING":        {},
		"GE.MISSING":        {},
		"IEA.MISSING":       {},
		"GE.COUNT":          {},
		"GE.CTRL.MISMATCH":  {},
		"IEA.CTRL.MISMATCH": {},
	}
	for _, is := range issues {
		if _, ok := skip[is.Code]; ok {
			continue
		}
		filtered = append(filtered, is)
	}
	return filtered
}

func hasError(issues []validation.Issue) bool {
	for _, is := range issues {
		if is.Severity == validation.Error {
			return true
		}
	}
	return false
}
