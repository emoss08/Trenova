package oracle

import (
	"strings"
	"testing"
)

func clean990(ctl string) []string {
	return transaction("990", ctl, "B1*TRNV*SIM000042*20261119*A")
}

func TestEnvelopeFindings(t *testing.T) {
	t.Parallel()

	good := interchange("GF", clean990("0001")...)
	tests := []struct {
		name  string
		text  string
		level Level
		code  string
		seg   string
	}{
		{
			name:  "IEA02 differs from ISA13",
			text:  strings.Replace(good, "IEA*1*000000101", "IEA*1*000000999", 1),
			level: LevelInterchange, code: "001", seg: "IEA",
		},
		{
			name:  "IEA01 miscounts groups",
			text:  strings.Replace(good, "IEA*1*", "IEA*2*", 1),
			level: LevelInterchange, code: "021", seg: "IEA",
		},
		{
			name:  "segment between groups",
			text:  strings.Replace(good, "GE*1*101~\n", "GE*1*101~\nN9*CN*X~\n", 1),
			level: LevelInterchange, code: "022", seg: "N9",
		},
		{
			name:  "ST outside a group",
			text:  strings.Replace(good, "GE*1*101~\n", "GE*1*101~\nST*990*0009~\nSE*2*0009~\n", 1),
			level: LevelInterchange, code: "022", seg: "ST",
		},
		{
			name:  "missing IEA",
			text:  strings.Replace(good, "IEA*1*000000101~\n", "", 1),
			level: LevelInterchange, code: "023", seg: "IEA",
		},
		{
			name:  "duplicate interchange control number",
			text:  good + good,
			level: LevelInterchange, code: "025", seg: "ISA",
		},
		{
			name:  "element separator is a letter",
			text:  strings.ReplaceAll(good, "*", "Q"),
			level: LevelInterchange, code: "026", seg: "ISA",
		},
		{
			name:  "component separator equals element separator",
			text:  strings.Replace(good, "*T*>~", "*T**~", 1),
			level: LevelInterchange, code: "027", seg: "ISA",
		},
		{
			name:  "terminator is a digit",
			text:  strings.ReplaceAll(good, "~", "7"),
			level: LevelInterchange, code: "004", seg: "ISA",
		},
		{
			name:  "ISA06 too wide",
			text:  strings.Replace(good, "TRENOVA        ", "TRENOVA         ", 1),
			level: LevelInterchange, code: "006", seg: "ISA",
		},
		{
			name:  "ISA09 not a date",
			text:  strings.Replace(good, "*261119*", "*261340*", 1),
			level: LevelInterchange, code: "014", seg: "ISA",
		},
		{
			name:  "GE missing",
			text:  strings.Replace(good, "GE*1*101~\n", "", 1),
			level: LevelGroup, code: "3", seg: "GE",
		},
		{
			name:  "GE02 differs from GS06",
			text:  strings.Replace(good, "GE*1*101", "GE*1*102", 1),
			level: LevelGroup, code: "4", seg: "GE",
		},
		{
			name:  "GE01 miscounts sets",
			text:  strings.Replace(good, "GE*1*101", "GE*3*101", 1),
			level: LevelGroup, code: "5", seg: "GE",
		},
		{
			name: "GS06 not digits",
			text: strings.NewReplacer("*101*X*", "*A1*X*", "GE*1*101", "GE*1*A1").
				Replace(good),
			level: LevelGroup, code: "6", seg: "GS",
		},
		{
			name: "duplicate group control number",
			text: strings.Replace(
				strings.Replace(good, "GE*1*101~\n",
					"GE*1*101~\nGS*GF*TRENOVA*SIMPARTNER*20261119*1200*101*X*004010~\n"+
						strings.Join(clean990("0002"), "~\n")+"~\nGE*1*101~\n", 1),
				"IEA*1*", "IEA*2*", 1),
			level: LevelGroup, code: "19", seg: "GS",
		},
		{
			name:  "no guide for the set",
			text:  interchange("FA", transaction("997", "0001", "AK1*GF*1", "AK9*A*1*1*1")...),
			level: LevelTransaction, code: "1", seg: "ST",
		},
		{
			name:  "SE missing",
			text:  strings.Replace(good, "SE*3*0001~\n", "", 1),
			level: LevelTransaction, code: "2", seg: "SE",
		},
		{
			name:  "SE02 differs from ST02",
			text:  strings.Replace(good, "SE*3*0001", "SE*3*0002", 1),
			level: LevelTransaction, code: "3", seg: "SE",
		},
		{
			name:  "SE01 miscounts segments",
			text:  strings.Replace(good, "SE*3*0001", "SE*4*0001", 1),
			level: LevelTransaction, code: "4", seg: "SE",
		},
		{
			name:  "ST01 not a set identifier",
			text:  strings.Replace(good, "ST*990*", "ST*99*", 1),
			level: LevelTransaction, code: "6", seg: "ST",
		},
		{
			name: "ST02 too short",
			text: strings.NewReplacer("ST*990*0001", "ST*990*1", "SE*3*0001", "SE*3*1").
				Replace(good),
			level: LevelTransaction, code: "7", seg: "ST",
		},
		{
			name:  "duplicate transaction control number",
			text:  interchange("GF", append(clean990("0001"), clean990("0001")...)...),
			level: LevelTransaction, code: "23", seg: "ST",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			j := judge(t, tt.text)
			fs := allFindings(j)
			f := findFinding(fs, tt.level, tt.code)
			if f == nil || f.SegmentID != tt.seg {
				t.Fatalf("want %s %s on %s, got:\n%s", tt.level, tt.code, tt.seg, errorSummary(fs))
			}
			if j.Accepted() {
				t.Fatal("a file with an envelope error must not be accepted")
			}
		})
	}
}

func TestGoodEnvelopeIsAccepted(t *testing.T) {
	t.Parallel()

	j := judge(t, interchange("GF", clean990("0001")...))
	if !j.Accepted() {
		t.Fatalf("rejected:\n%s", errorSummary(allFindings(j)))
	}
}

func TestGroupMismatchWarns(t *testing.T) {
	t.Parallel()

	j := judge(t, interchange("QM", clean990("0001")...))
	fs := allFindings(j)
	if !j.Accepted() || len(fs) != 1 || fs[0].Severity != SeverityWarning ||
		fs[0].Rule != "GS01 GF" {
		t.Fatalf("want one GS01 warning, got:\n%s", errorSummary(fs))
	}
}
