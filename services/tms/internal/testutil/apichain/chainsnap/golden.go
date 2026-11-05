package chainsnap

import (
	"bytes"
	"errors"
	"fmt"
	"maps"
	"os"
	"reflect"
	"slices"
	"strings"

	"github.com/bytedance/sonic"
)

const (
	UpdateEnv = "CHAIN_UPDATE"
	maxDiffs  = 40
)

var encoder = sonic.Config{SortMapKeys: true, EscapeHTML: false}.Froze()

type Snapshot struct {
	Chain  string        `json:"chain"`
	Steps  []StepRecord  `json:"steps"`
	Probes []ProbeRecord `json:"probes,omitempty"`
}

type StepRecord struct {
	ID         string         `json:"id"`
	Call       string         `json:"call"`
	Write      bool           `json:"write"`
	Request    any            `json:"request,omitempty"`
	Status     int            `json:"status"`
	Code       string         `json:"code,omitempty"`
	Response   any            `json:"response,omitempty"`
	Effects    []EffectRecord `json:"effects,omitempty"`
	Side       any            `json:"sideEffects,omitempty"`
	RoundTrips int            `json:"roundTrips"`
}

type EffectRecord struct {
	Read   string `json:"read"`
	Path   string `json:"path"`
	Rule   string `json:"rule"`
	Before any    `json:"before,omitempty"`
	After  any    `json:"after"`
}

type ProbeRecord struct {
	Probe  string `json:"probe"`
	Step   string `json:"step"`
	Status int    `json:"status"`
	Code   string `json:"code,omitempty"`
}

func Encode(snapshot *Snapshot) ([]byte, error) {
	raw, err := encoder.Marshal(snapshot)
	if err != nil {
		return nil, err
	}
	var tree any
	if err = sonic.Unmarshal(raw, &tree); err != nil {
		return nil, err
	}
	out, err := encoder.MarshalIndent(tree, "", "  ")
	if err != nil {
		return nil, err
	}

	return append(out, '\n'), nil
}

type Outcome struct {
	Updated bool
	Diffs   []Diff
}

type Diff struct {
	Step string
	Call string
	Path string
	Want any
	Got  any
}

func (d Diff) String() string {
	return fmt.Sprintf("%s want=%s got=%s", d.Path, short(d.Want), short(d.Got))
}

func Check(path string, snapshot *Snapshot) (Outcome, error) {
	actual, err := Encode(snapshot)
	if err != nil {
		return Outcome{}, fmt.Errorf("encode the snapshot: %w", err)
	}

	if os.Getenv(UpdateEnv) != "" {
		if err = os.WriteFile(path, actual, 0o600); err != nil {
			return Outcome{}, err
		}
		return Outcome{Updated: true}, nil
	}

	golden, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Outcome{}, fmt.Errorf(
			"%s does not exist: run with %s=1 to write it, then review it like code", path, UpdateEnv,
		)
	}
	if err != nil {
		return Outcome{}, err
	}
	if bytes.Equal(golden, actual) {
		return Outcome{}, nil
	}

	var want, got any
	if err = sonic.Unmarshal(golden, &want); err != nil {
		return Outcome{}, fmt.Errorf("%s: %w", path, err)
	}
	if err = sonic.Unmarshal(actual, &got); err != nil {
		return Outcome{}, err
	}

	var diffs []Diff
	compare(nil, want, got, &diffs)
	label(snapshot, diffs)

	return Outcome{Diffs: diffs}, nil
}

func compare(path []string, want, got any, out *[]Diff) {
	if len(*out) >= maxDiffs || reflect.DeepEqual(want, got) {
		return
	}

	wantMap, wantIsMap := want.(map[string]any)
	gotMap, gotIsMap := got.(map[string]any)
	if wantIsMap && gotIsMap {
		keys := map[string]bool{}
		for key := range wantMap {
			keys[key] = true
		}
		for key := range gotMap {
			keys[key] = true
		}
		for _, key := range slices.Sorted(maps.Keys(keys)) {
			compare(append(slices.Clip(path), key), wantMap[key], gotMap[key], out)
		}
		return
	}

	wantList, wantIsList := want.([]any)
	gotList, gotIsList := got.([]any)
	if wantIsList && gotIsList {
		for i := range max(len(wantList), len(gotList)) {
			var w, g any = missing{}, missing{}
			if i < len(wantList) {
				w = wantList[i]
			}
			if i < len(gotList) {
				g = gotList[i]
			}
			compare(append(slices.Clip(path), fmt.Sprint(i)), w, g, out)
		}
		return
	}

	*out = append(*out, Diff{Path: strings.Join(path, "."), Want: want, Got: got})
}

type missing struct{}

func label(snapshot *Snapshot, diffs []Diff) {
	for i := range diffs {
		segments := strings.Split(diffs[i].Path, ".")
		if len(segments) < 2 || segments[0] != "steps" {
			continue
		}
		var index int
		if _, err := fmt.Sscan(segments[1], &index); err != nil || index >= len(snapshot.Steps) {
			continue
		}
		diffs[i].Step = snapshot.Steps[index].ID
		diffs[i].Call = snapshot.Steps[index].Call
		diffs[i].Path = strings.Join(segments[2:], ".")
	}
}

func short(value any) string {
	if _, ok := value.(missing); ok {
		return "<absent>"
	}
	if value == nil {
		return "null"
	}
	raw, err := encoder.Marshal(value)
	if err != nil {
		return fmt.Sprint(value)
	}
	const limit = 160
	if len(raw) > limit {
		return string(raw[:limit]) + "…"
	}

	return string(raw)
}
