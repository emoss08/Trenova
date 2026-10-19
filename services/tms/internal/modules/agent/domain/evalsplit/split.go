package evalsplit

import (
	"bytes"
	"cmp"
	"hash/fnv"
	"slices"

	"gopkg.in/yaml.v3"
)

type Split string

const (
	Development = Split("development")
	HeldOut     = Split("held-out")
	Adversarial = Split("adversarial")
)

func (s Split) IsValid() bool {
	switch s {
	case Development, HeldOut, Adversarial:
		return true
	default:
		return false
	}
}

type Mode string

const (
	Chat       = Mode("chat")
	Unattended = Mode("unattended")
)

type Stratum struct {
	Desk string `yaml:"desk" json:"desk"`
	Mode Mode   `yaml:"mode" json:"mode"`
}

func (s Stratum) String() string {
	return s.Desk + " (" + string(s.Mode) + ")"
}

func (s Stratum) Compare(other Stratum) int {
	return cmp.Or(cmp.Compare(s.Desk, other.Desk), cmp.Compare(s.Mode, other.Mode))
}

type Item struct {
	Key         string
	Stratum     Stratum
	Adversarial bool
}

func HeldOutQuota(members int) int {
	if members < 2 {
		return 0
	}

	return max(1, (members+2)/4)
}

func Rank(key string) uint64 {
	hash := fnv.New64a()
	_, _ = hash.Write([]byte(key))

	return hash.Sum64()
}

func Assign(items []Item) []Split {
	splits := make([]Split, len(items))
	strata := make(map[Stratum][]int, 16)
	for idx := range items {
		if items[idx].Adversarial {
			splits[idx] = Adversarial
			continue
		}
		splits[idx] = Development
		strata[items[idx].Stratum] = append(strata[items[idx].Stratum], idx)
	}

	for _, members := range strata {
		slices.SortFunc(members, func(a, b int) int {
			return cmp.Or(
				cmp.Compare(Rank(items[a].Key), Rank(items[b].Key)),
				cmp.Compare(items[a].Key, items[b].Key),
			)
		})
		for _, idx := range members[:HeldOutQuota(len(members))] {
			splits[idx] = HeldOut
		}
	}

	return splits
}

type Ledger map[Stratum][]string

func LedgerOf(items []Item, splits []Split) Ledger {
	ledger := make(Ledger, 16)
	for idx := range items {
		if splits[idx] != HeldOut {
			continue
		}
		ledger[items[idx].Stratum] = append(ledger[items[idx].Stratum], items[idx].Key)
	}
	for stratum := range ledger {
		slices.Sort(ledger[stratum])
	}

	return ledger
}

type LedgerEntry struct {
	Stratum Stratum  `yaml:"stratum"`
	HeldOut []string `yaml:"heldOut"`
}

func (l Ledger) Entries() []LedgerEntry {
	entries := make([]LedgerEntry, 0, len(l))
	for stratum, keys := range l {
		entries = append(entries, LedgerEntry{Stratum: stratum, HeldOut: keys})
	}
	slices.SortFunc(entries, func(a, b LedgerEntry) int {
		return a.Stratum.Compare(b.Stratum)
	})

	return entries
}

type Diff struct {
	Joined []string
	Left   []string
}

func (d Diff) Empty() bool {
	return len(d.Joined) == 0 && len(d.Left) == 0
}

func (l Ledger) Diff(recorded Ledger) Diff {
	now := l.keys()
	before := recorded.keys()
	diff := Diff{}
	for key := range now {
		if _, ok := before[key]; !ok {
			diff.Joined = append(diff.Joined, key)
		}
	}
	for key := range before {
		if _, ok := now[key]; !ok {
			diff.Left = append(diff.Left, key)
		}
	}
	slices.Sort(diff.Joined)
	slices.Sort(diff.Left)

	return diff
}

func (l Ledger) keys() map[string]struct{} {
	keys := make(map[string]struct{}, 64)
	for stratum, members := range l {
		for _, key := range members {
			keys[key+" ["+stratum.String()+"]"] = struct{}{}
		}
	}

	return keys
}

type ledgerFile struct {
	Strata []LedgerEntry `yaml:"strata"`
}

func (l Ledger) Marshal(header string) ([]byte, error) {
	var out bytes.Buffer
	out.WriteString(header)
	encoder := yaml.NewEncoder(&out)
	encoder.SetIndent(2)
	if err := encoder.Encode(ledgerFile{Strata: l.Entries()}); err != nil {
		return nil, err
	}
	if err := encoder.Close(); err != nil {
		return nil, err
	}

	return out.Bytes(), nil
}

func ParseLedger(raw []byte) (Ledger, error) {
	var file ledgerFile
	if err := yaml.Unmarshal(raw, &file); err != nil {
		return nil, err
	}

	ledger := make(Ledger, len(file.Strata))
	for _, entry := range file.Strata {
		ledger[entry.Stratum] = append(ledger[entry.Stratum], entry.HeldOut...)
	}

	return ledger, nil
}
