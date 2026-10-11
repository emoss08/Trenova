package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"slices"
	"strings"

	"github.com/bytedance/sonic"
)

const listTemplate = "{{.ImportPath}}\t{{.Dir}}\t{{.DepOnly}}\t" +
	"{{join .GoFiles \" \"}}\t{{join .Imports \" \"}}"

type pkgInfo struct {
	path    string
	dir     string
	depOnly bool
	goFiles []string
	imports []string
}

type listing struct {
	module string
	pkgs   map[string]*pkgInfo
	paths  []string
}

type listRequest struct {
	tags     string
	deps     bool
	patterns []string
}

func goList(ctx context.Context, req listRequest) (*listing, error) {
	module, err := modulePath(ctx)
	if err != nil {
		return nil, err
	}

	args := make([]string, 0, 6+len(req.patterns))
	args = append(args, "list", "-tags", req.tags, "-f", listTemplate)
	if req.deps {
		args = append(args, "-deps")
	}
	args = append(args, req.patterns...)

	out, err := goOutput(ctx, args...)
	if err != nil {
		return nil, err
	}

	l := &listing{module: module, pkgs: make(map[string]*pkgInfo)}
	scanner := bufio.NewScanner(strings.NewReader(out))
	scanner.Buffer(make([]byte, 0, 1<<20), 1<<24)
	for scanner.Scan() {
		p, perr := parseListLine(scanner.Text())
		if perr != nil {
			return nil, perr
		}
		l.pkgs[p.path] = p
		l.paths = append(l.paths, p.path)
	}
	if err = scanner.Err(); err != nil {
		return nil, fmt.Errorf("read go list output: %w", err)
	}
	slices.Sort(l.paths)

	return l, nil
}

func modulePath(ctx context.Context) (string, error) {
	out, err := goOutput(ctx, "mod", "edit", "-json")
	if err != nil {
		return "", err
	}

	var mod struct {
		Module struct {
			Path string `json:"Path"`
		} `json:"Module"`
	}
	if err = sonic.UnmarshalString(out, &mod); err != nil {
		return "", fmt.Errorf("decode go.mod: %w", err)
	}

	return mod.Module.Path, nil
}

func parseListLine(line string) (*pkgInfo, error) {
	fields := strings.Split(line, "\t")
	if len(fields) != 5 {
		return nil, fmt.Errorf("unexpected go list line %q", line)
	}

	return &pkgInfo{
		path:    fields[0],
		dir:     fields[1],
		depOnly: fields[2] == "true",
		goFiles: strings.Fields(fields[3]),
		imports: strings.Fields(fields[4]),
	}, nil
}

func goOutput(ctx context.Context, args ...string) (string, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf(
			"go %s: %w: %s",
			strings.Join(args, " "),
			err,
			strings.TrimSpace(stderr.String()),
		)
	}

	return stdout.String(), nil
}

func (l *listing) firstParty(path string) bool {
	return path == l.module || strings.HasPrefix(path, l.module+"/")
}

func (l *listing) firstPartyPaths() []string {
	out := make([]string, 0, len(l.paths))
	for _, p := range l.paths {
		if l.firstParty(p) {
			out = append(out, p)
		}
	}

	return out
}

func (l *listing) roots() []string {
	out := make([]string, 0, 1)
	for _, p := range l.paths {
		if !l.pkgs[p].depOnly {
			out = append(out, p)
		}
	}

	return out
}

func (l *listing) withSuffix(suffix string) []string {
	out := make([]string, 0, 1)
	for _, p := range l.firstPartyPaths() {
		if strings.HasSuffix(p, suffix) {
			out = append(out, p)
		}
	}

	return out
}

func (l *listing) dependentCounts() map[string]int {
	paths := l.firstPartyPaths()
	index := make(map[string]int, len(paths))
	for i, p := range paths {
		index[p] = i
	}

	reverse := make([][]int, len(paths))
	for i, p := range paths {
		for _, imp := range l.pkgs[p].imports {
			if j, ok := index[imp]; ok {
				reverse[j] = append(reverse[j], i)
			}
		}
	}

	counts := make(map[string]int, len(paths))
	seen := make([]int, len(paths))
	stack := make([]int, 0, len(paths))
	for i, p := range paths {
		stamp := i + 1
		seen[i] = stamp
		stack = append(stack[:0], i)
		n := 0
		for len(stack) > 0 {
			cur := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			for _, dep := range reverse[cur] {
				if seen[dep] != stamp {
					seen[dep] = stamp
					n++
					stack = append(stack, dep)
				}
			}
		}
		counts[p] = n
	}

	return counts
}

func (l *listing) shortestChain(from, to string) []string {
	parent := map[string]string{from: ""}
	queue := []string{from}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur == to {
			break
		}
		p, ok := l.pkgs[cur]
		if !ok {
			continue
		}
		for _, imp := range p.imports {
			if _, visited := parent[imp]; !visited {
				parent[imp] = cur
				queue = append(queue, imp)
			}
		}
	}

	if _, ok := parent[to]; !ok {
		return nil
	}

	chain := make([]string, 0, 8)
	for cur := to; cur != ""; cur = parent[cur] {
		chain = append(chain, cur)
	}
	slices.Reverse(chain)

	return chain
}

func (l *listing) trim(path string) string {
	if path == l.module {
		return path
	}

	return strings.TrimPrefix(path, l.module+"/")
}
