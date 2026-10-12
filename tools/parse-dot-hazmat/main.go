package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

const ecfrURL = "https://ecfr.gov/api/versioner/v1/full/2025-01-01/title-49.xml?chapter=I&subchapter=C&part=172"

type HazmatEntry struct {
	Symbols             string
	ProperShippingName  string
	HazardClass         string
	UnNumber            string
	PackingGroup        string
	SubsidiaryHazard    string
	SpecialProvisions   string
	PackagingExceptions string
	PackagingNonBulk    string
	PackagingBulk       string
	QuantityPassenger   string
	QuantityCargo       string
	VesselStowage       string
	ErgGuide            string
}

func main() {
	var data []byte
	var err error

	if len(os.Args) > 1 && os.Args[1] == "--local" {
		localPath := "/tmp/hazmat_test.xml"
		if len(os.Args) > 2 {
			localPath = os.Args[2]
		}
		fmt.Printf("Reading from local file: %s\n", localPath)
		data, err = os.ReadFile(localPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading local file: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Println("Fetching DOT Hazmat Table from eCFR...")
		client := &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return nil
			},
		}
		resp, fetchErr := client.Get(ecfrURL)
		if fetchErr != nil {
			fmt.Fprintf(os.Stderr, "Error fetching data: %v\n", fetchErr)
			os.Exit(1)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			fmt.Fprintf(os.Stderr, "HTTP error: %s\n", resp.Status)
			os.Exit(1)
		}

		data, err = io.ReadAll(resp.Body)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading response: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Printf("Loaded %d bytes, parsing XML...\n", len(data))

	entries := parseHazmatTable(string(data))
	fmt.Printf("Parsed %d entries\n", len(entries))

	outputPath := "services/tms/internal/infrastructure/database/seeds/base/data/dot_hazmat_references.yaml"

	if err := writeYAML(entries, outputPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing YAML: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Wrote %d entries to %s\n", len(entries), outputPath)
}

func parseHazmatTable(xmlData string) []HazmatEntry {
	var entries []HazmatEntry

	tableStart := strings.Index(xmlData, "172.101 Hazardous Materials Table")
	if tableStart == -1 {
		fmt.Println(
			"WARNING: Could not find '172.101 Hazardous Materials Table' marker, parsing entire document",
		)
		tableStart = 0
	}

	tableData := xmlData[tableStart:]

	rows := extractTableRows(tableData)
	fmt.Printf("Found %d total TR rows in table section\n", len(rows))

	for _, row := range rows {
		cells := extractCells(row)
		if len(cells) < 12 {
			continue
		}

		rawID := cleanText(cells[3])
		unNumber := extractUnNumber(rawID)
		if unNumber == "" {
			continue
		}

		properName := cleanText(cells[1])
		if properName == "" || strings.Contains(strings.ToLower(properName), ", see") {
			continue
		}

		hazardClass := cleanText(cells[2])
		if hazardClass == "" || hazardClass == "Forbidden" {
			continue
		}

		entry := HazmatEntry{
			Symbols:             cleanText(cells[0]),
			ProperShippingName:  properName,
			HazardClass:         hazardClass,
			UnNumber:            unNumber,
			PackingGroup:        cleanText(cells[4]),
			SubsidiaryHazard:    cleanText(cells[5]),
			SpecialProvisions:   cleanText(cells[6]),
			PackagingExceptions: cleanText(cells[7]),
			PackagingNonBulk:    cleanText(cells[8]),
			PackagingBulk:       cleanText(cells[9]),
			QuantityPassenger:   cleanText(cells[10]),
			QuantityCargo:       cleanText(cells[11]),
		}

		if len(cells) > 12 {
			entry.VesselStowage = cleanText(cells[12])
		}
		if len(cells) > 13 {
			entry.ErgGuide = cleanText(cells[13])
		}

		entries = append(entries, entry)
	}

	return entries
}

func extractUnNumber(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	num := raw
	if strings.HasPrefix(raw, "UN") {
		num = raw[2:]
	} else if strings.HasPrefix(raw, "NA") {
		num = raw[2:]
	}

	num = strings.TrimSpace(num)
	if len(num) != 4 {
		return ""
	}
	if !isNumeric(num) {
		return ""
	}

	return num
}

func extractTableRows(data string) []string {
	var rows []string
	for {
		start := strings.Index(data, "<TR")
		if start == -1 {
			break
		}
		end := strings.Index(data[start:], "</TR>")
		if end == -1 {
			break
		}
		rows = append(rows, data[start:start+end+5])
		data = data[start+end+5:]
	}
	return rows
}

func extractCells(row string) []string {
	var cells []string
	data := row
	for {
		start := strings.Index(data, "<TD")
		if start == -1 {
			break
		}
		closeTag := strings.Index(data[start:], ">")
		if closeTag == -1 {
			break
		}
		contentStart := start + closeTag + 1
		end := strings.Index(data[contentStart:], "</TD>")
		if end == -1 {
			break
		}
		cells = append(cells, data[contentStart:contentStart+end])
		data = data[contentStart+end+5:]
	}
	return cells
}

func cleanText(s string) string {
	result := s
	for {
		start := strings.Index(result, "<")
		if start == -1 {
			break
		}
		end := strings.Index(result[start:], ">")
		if end == -1 {
			break
		}
		result = result[:start] + result[start+end+1:]
	}

	result = strings.ReplaceAll(result, "&amp;", "&")
	result = strings.ReplaceAll(result, "&lt;", "<")
	result = strings.ReplaceAll(result, "&gt;", ">")
	result = strings.ReplaceAll(result, "&#x2003;", " ")
	result = strings.ReplaceAll(result, "&#xA0;", " ")
	result = strings.ReplaceAll(result, "&#x2014;", "-")
	result = strings.ReplaceAll(result, "&#x201C;", "\"")
	result = strings.ReplaceAll(result, "&#x201D;", "\"")
	result = strings.ReplaceAll(result, "\n", " ")
	result = strings.ReplaceAll(result, "\r", "")

	for strings.Contains(result, "  ") {
		result = strings.ReplaceAll(result, "  ", " ")
	}

	return strings.TrimSpace(result)
}

func isNumeric(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return len(s) > 0
}

func yamlEscape(s string) string {
	if s == "" {
		return `""`
	}
	escaped := strings.ReplaceAll(s, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `"`, `\"`)
	return `"` + escaped + `"`
}

func writeYAML(entries []HazmatEntry, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	fmt.Fprintln(f, "references:")
	for _, e := range entries {
		fmt.Fprintf(f, "  - un_number: %s\n", yamlEscape(e.UnNumber))
		fmt.Fprintf(f, "    proper_shipping_name: %s\n", yamlEscape(e.ProperShippingName))
		fmt.Fprintf(f, "    hazard_class: %s\n", yamlEscape(e.HazardClass))
		fmt.Fprintf(f, "    subsidiary_hazard: %s\n", yamlEscape(e.SubsidiaryHazard))
		fmt.Fprintf(f, "    packing_group: %s\n", yamlEscape(e.PackingGroup))
		fmt.Fprintf(f, "    special_provisions: %s\n", yamlEscape(e.SpecialProvisions))
		fmt.Fprintf(f, "    packaging_exceptions: %s\n", yamlEscape(e.PackagingExceptions))
		fmt.Fprintf(f, "    packaging_non_bulk: %s\n", yamlEscape(e.PackagingNonBulk))
		fmt.Fprintf(f, "    packaging_bulk: %s\n", yamlEscape(e.PackagingBulk))
		fmt.Fprintf(f, "    quantity_passenger: %s\n", yamlEscape(e.QuantityPassenger))
		fmt.Fprintf(f, "    quantity_cargo: %s\n", yamlEscape(e.QuantityCargo))
		fmt.Fprintf(f, "    vessel_stowage: %s\n", yamlEscape(e.VesselStowage))
		fmt.Fprintf(f, "    erg_guide: %s\n", yamlEscape(e.ErgGuide))
		fmt.Fprintf(f, "    symbols: %s\n", yamlEscape(e.Symbols))
	}

	return nil
}
