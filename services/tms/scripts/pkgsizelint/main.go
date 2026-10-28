package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	repoRoot      = "../.."
	configPath    = ".go-pkgsize.yml"
	guideLink     = "docs/engineering/backend/go.md#small-packages"
	exitViolation = 1
)

var (
	sourceRoots   = []string{"services/tms", "shared"}
	errViolations = errors.New("package size rules violated")
	errConfig     = errors.New("invalid " + configPath)
)

type budgets struct {
	Foundation int `yaml:"foundation"`
	Default    int `yaml:"default"`
}

type exception struct {
	Ceiling int    `yaml:"ceiling"`
	Reason  string `yaml:"reason"`
}

type config struct {
	Budgets    budgets              `yaml:"budgets"`
	Foundation []string             `yaml:"foundation"`
	Exceptions map[string]exception `yaml:"exceptions"`
	Debt       map[string]int       `yaml:"debt"`
}

type checker struct {
	cfg        *config
	foundation []*regexp.Regexp
	sizes      map[string]int
	update     bool
	problems   []string
}

func main() {
	update := flag.Bool(
		"update",
		false,
		"lower debt ceilings to current sizes and drop entries that are within budget",
	)
	flag.Parse()
	if err := run(*update); err != nil {
		fmt.Fprintln(os.Stderr, "pkgsizelint:", err)
		os.Exit(exitViolation)
	}
}

func run(update bool) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	foundation, err := compilePatterns(cfg.Foundation)
	if err != nil {
		return err
	}
	sizes, err := measure()
	if err != nil {
		return err
	}

	c := &checker{cfg: cfg, foundation: foundation, sizes: sizes, update: update}
	c.checkExceptions()
	c.checkDebt()
	c.checkBudgets()

	if update {
		if err = writeConfig(cfg); err != nil {
			return err
		}
	}
	if len(c.problems) == 0 {
		return nil
	}
	slices.Sort(c.problems)
	for _, p := range c.problems {
		fmt.Println(p)
	}

	return fmt.Errorf("%w: %d (see %s)", errViolations, len(c.problems), guideLink)
}

func (c *checker) budget(pkg string) (int, string) {
	for _, re := range c.foundation {
		if re.MatchString(pkg) {
			return c.cfg.Budgets.Foundation, "foundation"
		}
	}

	return c.cfg.Budgets.Default, "default"
}

func (c *checker) report(format string, args ...any) {
	c.problems = append(c.problems, fmt.Sprintf(format, args...))
}

func (c *checker) withinBudget(pkg string) bool {
	size, ok := c.sizes[pkg]
	limit, _ := c.budget(pkg)

	return !ok || size <= limit
}

func (c *checker) checkExceptions() {
	for _, pkg := range slices.Sorted(maps.Keys(c.cfg.Exceptions)) {
		entry := c.cfg.Exceptions[pkg]
		if _, ok := c.cfg.Debt[pkg]; ok {
			c.report("%s: listed under both exceptions and debt", pkg)
		}
		if strings.TrimSpace(entry.Reason) == "" {
			c.report("%s: exception has no reason", pkg)
		}
		if c.withinBudget(pkg) {
			c.retire(pkg, "exception", func() { delete(c.cfg.Exceptions, pkg) })
			continue
		}
		if size := c.sizes[pkg]; size > entry.Ceiling {
			c.report("%s: %d lines, past its %d-line exception ceiling", pkg, size, entry.Ceiling)
		}
	}
}

func (c *checker) checkDebt() {
	for _, pkg := range slices.Sorted(maps.Keys(c.cfg.Debt)) {
		ceiling := c.cfg.Debt[pkg]
		if c.withinBudget(pkg) {
			c.retire(pkg, "debt", func() { delete(c.cfg.Debt, pkg) })
			continue
		}
		size := c.sizes[pkg]
		switch {
		case size > ceiling:
			c.report(
				"%s: %d lines, grew past its %d-line debt ceiling; split it before adding to it",
				pkg,
				size,
				ceiling,
			)
		case size < ceiling && c.update:
			c.cfg.Debt[pkg] = size
		}
	}
}

func (c *checker) retire(pkg, kind string, drop func()) {
	if c.update {
		drop()
		return
	}
	c.report(
		"%s: within budget or gone; remove its %s entry with `go run ./scripts/pkgsizelint -update`",
		pkg,
		kind,
	)
}

func (c *checker) checkBudgets() {
	for _, pkg := range slices.Sorted(maps.Keys(c.sizes)) {
		size := c.sizes[pkg]
		limit, tier := c.budget(pkg)
		if size <= limit {
			continue
		}
		if _, ok := c.cfg.Debt[pkg]; ok {
			continue
		}
		if _, ok := c.cfg.Exceptions[pkg]; ok {
			continue
		}
		c.report("%s: %d lines, over the %d-line %s budget; split it", pkg, size, limit, tier)
	}
}

func loadConfig() (*config, error) {
	raw, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", configPath, err)
	}
	cfg := new(config)
	if err = yaml.Unmarshal(raw, cfg); err != nil {
		return nil, fmt.Errorf("%w: %w", errConfig, err)
	}
	if cfg.Budgets.Foundation <= 0 || cfg.Budgets.Default <= 0 {
		return nil, fmt.Errorf("%w: both budgets must be positive", errConfig)
	}
	if cfg.Exceptions == nil {
		cfg.Exceptions = make(map[string]exception)
	}
	if cfg.Debt == nil {
		cfg.Debt = make(map[string]int)
	}

	return cfg, nil
}

func writeConfig(cfg *config) error {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(cfg); err != nil {
		return fmt.Errorf("encode %s: %w", configPath, err)
	}
	if err := enc.Close(); err != nil {
		return fmt.Errorf("encode %s: %w", configPath, err)
	}
	if err := os.WriteFile(configPath, buf.Bytes(), 0o600); err != nil {
		return fmt.Errorf("write %s: %w", configPath, err)
	}

	return nil
}

func compilePatterns(patterns []string) ([]*regexp.Regexp, error) {
	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, p := range patterns {
		re, err := regexp.Compile(p)
		if err != nil {
			return nil, fmt.Errorf("%w: foundation pattern %q: %w", errConfig, p, err)
		}
		compiled = append(compiled, re)
	}

	return compiled, nil
}

func measure() (map[string]int, error) {
	sizes := make(map[string]int)
	for _, root := range sourceRoots {
		err := filepath.WalkDir(
			filepath.Join(repoRoot, root),
			func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if d.IsDir() {
					if skipDir(d.Name()) {
						return filepath.SkipDir
					}
					return nil
				}
				if !strings.HasSuffix(d.Name(), ".go") || strings.HasSuffix(d.Name(), "_test.go") {
					return nil
				}
				lines, err := countLines(path)
				if err != nil {
					return err
				}
				rel, err := filepath.Rel(repoRoot, filepath.Dir(path))
				if err != nil {
					return err
				}
				sizes[filepath.ToSlash(rel)] += lines
				return nil
			},
		)
		if err != nil {
			return nil, fmt.Errorf("walk %s: %w", root, err)
		}
	}

	return sizes, nil
}

func skipDir(name string) bool {
	return name == "testdata" || name == "vendor" || name == "node_modules" ||
		strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_")
}

func countLines(path string) (int, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	file, err := parser.ParseFile(
		token.NewFileSet(),
		path,
		src,
		parser.PackageClauseOnly|parser.ParseComments,
	)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", path, err)
	}
	if ast.IsGenerated(file) {
		return 0, nil
	}

	return bytes.Count(src, []byte{'\n'}), nil
}
