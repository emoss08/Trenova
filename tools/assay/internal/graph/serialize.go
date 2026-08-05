package graph

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"sort"
)

const snapshotVersion = 1

type snapshot struct {
	Version  int
	Root     string
	Packages []Package
}

func (g *Graph) MarshalBinary() ([]byte, error) {
	snap := snapshot{
		Version:  snapshotVersion,
		Root:     g.Root,
		Packages: make([]Package, 0, len(g.pkgs)),
	}
	for _, pkg := range g.pkgs {
		snap.Packages = append(snap.Packages, *pkg)
	}
	sort.Slice(snap.Packages, func(i, j int) bool {
		return snap.Packages[i].ImportPath < snap.Packages[j].ImportPath
	})

	return encodeSnapshot(snap)
}

func encodeSnapshot(snap snapshot) ([]byte, error) {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(snap); err != nil {
		return nil, fmt.Errorf("encode graph: %w", err)
	}

	return buf.Bytes(), nil
}

func UnmarshalGraph(payload []byte) (*Graph, error) {
	var snap snapshot
	if err := gob.NewDecoder(bytes.NewReader(payload)).Decode(&snap); err != nil {
		return nil, fmt.Errorf("decode graph: %w", err)
	}
	if snap.Version != snapshotVersion {
		return nil, fmt.Errorf("unsupported graph snapshot version %d", snap.Version)
	}

	g := newGraph(snap.Root)
	for i := range snap.Packages {
		pkg := snap.Packages[i]
		g.pkgs[pkg.ImportPath] = &pkg
	}
	g.index()

	return g, nil
}
