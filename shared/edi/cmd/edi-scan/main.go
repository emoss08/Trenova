package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bytedance/sonic"

	"github.com/emoss08/trenova/shared/edi/internal/config"
	"github.com/emoss08/trenova/shared/edi/internal/mapper"
	tx204 "github.com/emoss08/trenova/shared/edi/internal/tx/tx204"
	"github.com/emoss08/trenova/shared/edi/internal/validation"
	"github.com/emoss08/trenova/shared/edi/internal/x12"
)

func main() {
	dir := flag.String("dir", "testdata", "directory to scan for .edi files")
	max := flag.Int("max", 2000, "maximum files to scan")
	lenient := flag.Bool("lenient", false, "use lenient validation")
	schema := flag.String("schema", "", "optional JSON schema file for validation rules")
	profile := flag.String(
		"profile",
		"",
		"optional partner profile JSON including schema and default delimiters",
	)
	outPath := flag.String(
		"out",
		"",
		"optional path to write NDJSON of shipments with validation issues",
	)
	perTx := flag.Bool(
		"per-tx",
		false,
		"when writing -out, emit one NDJSON line per ST/SE 204 transaction rather than per file",
	)
	// delimiter overrides
	optDelims := flag.String(
		"delims",
		"",
		"override delimiters as elem,comp,seg,rep (rep optional)",
	)
	optElem := flag.String("element", "", "override element separator (single char)")
	optComp := flag.String("component", "", "override component separator (single char)")
	optSeg := flag.String("segment", "", "override segment terminator (single char)")
	optRep := flag.String("repetition", "", "override repetition separator (single char)")
	flag.Parse()

	files := make([]string, 0, 64)
	_ = filepath.WalkDir(*dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(strings.ToLower(d.Name()), ".edi") {
			files = append(files, path)
		}
		return nil
	})
	sort.Strings(files)
	if len(files) > *max {
		files = files[:*max]
	}
	if len(files) == 0 {
		fmt.Println("no .edi files found")
		os.Exit(0)
	}

	fmt.Printf("Scanning %d EDI files under %s\n", len(files), *dir)

	var out *bufio.Writer
	if *outPath != "" {
		f, err := os.Create(*outPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "unable to open out file: %v\n", err)
			os.Exit(1)
		}
		defer f.Close()
		out = bufio.NewWriter(f)
		defer out.Flush()
	}
	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			fmt.Printf("%s | READ_ERROR: %v\n", f, err)
			continue
		}
		delims, err := x12.DetectDelimiters(raw)
		if err != nil {
			fmt.Printf("%s | DELIM_ERROR: %v\n", f, err)
			continue
		}
		var partner *config.PartnerConfig
		if *profile != "" {
			if pc, err := config.Load(*profile); err == nil {
				partner = pc
				pc.ApplyDelimiters(&delims)
				if *schema == "" && pc.SchemaPath != "" {
					*schema = pc.SchemaPath
				}
			} else {
				fmt.Printf("%s | PROFILE_ERROR: %v\n", f, err)
			}
		}
		// Apply overrides if provided
		if *optDelims != "" {
			parts := strings.Split(*optDelims, ",")
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
		if *optElem != "" {
			delims.Element = (*optElem)[0]
		}
		if *optComp != "" {
			delims.Component = (*optComp)[0]
		}
		if *optSeg != "" {
			delims.Segment = (*optSeg)[0]
		}
		if *optRep != "" {
			delims.Repetition = (*optRep)[0]
		}
		segs, err := x12.ParseSegments(raw, delims)
		if err != nil {
			fmt.Printf("%s | PARSE_ERROR: %v\n", f, err)
			continue
		}
		prof := validation.DefaultProfileForVersion(x12.ExtractVersion(segs))
		if partner != nil {
			prof = partner.ApplyValidation(prof)
		}
		if *lenient {
			prof.Strictness = validation.Lenient
			prof.EnforceSECount = false
			prof.RequirePickupAndDelivery = false
		}
		issues := validation.Validate204WithProfile(segs, prof)
		if *schema != "" {
			if sch, err := validation.LoadSchema(*schema); err == nil {
				issues = append(issues, validation.ValidateWithSchema(segs, sch)...)
			} else {
				fmt.Printf("%s | SCHEMA_ERROR: %v\n", f, err)
			}
		}
		errs := 0
		codes := make([]string, 0, len(issues))
		for _, is := range issues {
			if is.Severity == validation.Error {
				errs++
				codes = append(codes, is.Code)
			}
		}
		// Extract a quick header summary from typed 204
		lt := tx204.BuildFromSegments(segs)
		ver := x12.ExtractVersion(segs)
		summary := fmt.Sprintf(
			"Ver=%s SCAC=%s ShipID=%s Stops=%d",
			ver,
			lt.Header.CarrierSCAC,
			lt.Header.ShipmentID,
			len(lt.Stops),
		)
		status := "OK"
		if errs > 0 {
			if len(codes) > 5 {
				codes = codes[:5]
			}
			status = fmt.Sprintf("ERRORS(%d): %s", errs, strings.Join(codes, ","))
		}
		fmt.Printf("%s | segs=%d | %s | %s\n", f, len(segs), status, summary)

		if out != nil {
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
			// Per-file or per-transaction NDJSON
			if *perTx {
				blocks := x12.SplitTransactions(segs)
				for _, b := range blocks {
					if b.SetID != "204" {
						continue
					}
					lt := tx204.BuildFromSegments(b.Segs)
					shp := mapper.ToShipmentWithOptions(lt, opts)
					iss := issues
					// recompute issues per tx to filter envelope, using provided schema
					iss = validation.Validate204WithProfile(b.Segs, prof)
					if *schema != "" {
						if sch, err := validation.LoadSchema(*schema); err == nil {
							iss = append(iss, validation.ValidateWithSchema(b.Segs, sch)...)
						}
					}
					var segPtr *[]x12.Segment
					if partner != nil && partner.IncludeSegments {
						segPtr = &b.Segs
					}
					entry := struct {
						File     string             `json:"file"`
						Shipment any                `json:"shipment"`
						Issues   []validation.Issue `json:"issues"`
						Segments *[]x12.Segment     `json:"segments,omitempty"`
					}{File: f, Shipment: shp, Issues: iss, Segments: segPtr}
					if bts, err := sonic.ConfigFastest.Marshal(entry); err == nil {
						out.Write(bts)
						out.WriteByte('\n')
					}
				}
			} else {
				shp := mapper.ToShipmentWithOptions(lt, opts)
				type outLine struct {
					File     string             `json:"file"`
					Shipment any                `json:"shipment"`
					Issues   []validation.Issue `json:"issues"`
					Segments *[]x12.Segment     `json:"segments,omitempty"`
				}
				var segPtr *[]x12.Segment
				if partner != nil && partner.IncludeSegments {
					segPtr = &segs
				}
				line := outLine{File: f, Shipment: shp, Issues: issues, Segments: segPtr}
				b, err := sonic.ConfigFastest.Marshal(line)
				if err != nil {
					fmt.Fprintf(os.Stderr, "json encode error for %s: %v\n", f, err)
					continue
				}
				out.Write(b)
				out.WriteByte('\n')
			}
		}
	}
}
