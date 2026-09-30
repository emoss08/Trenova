package main

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/99designs/gqlgen/codegen"
	"github.com/99designs/gqlgen/codegen/config"
	"github.com/99designs/gqlgen/codegen/templates"
	"github.com/99designs/gqlgen/plugin"
	"github.com/99designs/gqlgen/plugin/modelgen"
	"github.com/99designs/gqlgen/plugin/resolvergen"
)

const (
	runtimeImport = "github.com/emoss08/trenova/internal/api/graphql/gqlexec"
	rootFilename  = "root.generated.go"
)

var (
	//go:embed shard.gotpl
	shardTemplate string
	//go:embed root.gotpl
	rootTemplate string
)

type codeGenerator interface {
	plugin.Plugin
	plugin.CodeGenerator
}

func run(configPath string) error {
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if err = checkConfig(cfg); err != nil {
		return err
	}

	snap, err := takeSnapshot(cfg)
	if err != nil {
		return err
	}

	if err = generate(cfg); err != nil {
		if restoreErr := snap.restore(); restoreErr != nil {
			return errors.Join(err, restoreErr)
		}
		return err
	}
	if err = snap.discard(); err != nil {
		return err
	}

	if cfg.SkipValidation {
		return nil
	}
	return validate(cfg)
}

func generate(cfg *config.Config) error {
	data, resolvers, err := buildData(cfg)
	if err != nil {
		return err
	}
	if err = checkData(data); err != nil {
		return err
	}

	root, err := buildPlan(data, runtimeImport)
	if err != nil {
		return err
	}
	if err = writeExec(cfg, root); err != nil {
		return err
	}

	if err = resolvers.GenerateCode(data); err != nil {
		return fmt.Errorf("%s: %w", resolvers.Name(), err)
	}
	return nil
}

func buildData(cfg *config.Config) (*codegen.Data, codeGenerator, error) {
	if cfg.Model.IsDefined() {
		if err := os.Remove(cfg.Model.Filename); err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, nil, fmt.Errorf("remove models: %w", err)
		}
	}

	resolvers, ok := resolvergen.New().(codeGenerator)
	if !ok {
		return nil, nil, errors.New("resolvergen does not generate code")
	}
	plugins := make([]plugin.Plugin, 0, 2)
	if cfg.Model.IsDefined() {
		plugins = append(plugins, modelgen.New())
	}
	plugins = append(plugins, resolvers)

	if err := cfg.LoadSchema(); err != nil {
		return nil, nil, fmt.Errorf("load schema: %w", err)
	}

	codegen.ClearInlineArgsMetadata()
	if err := codegen.ExpandInlineArguments(cfg.Schema); err != nil {
		return nil, nil, fmt.Errorf("expand inline arguments: %w", err)
	}

	if err := cfg.Init(); err != nil {
		return nil, nil, fmt.Errorf("init config: %w", err)
	}

	for _, p := range plugins {
		if mut, ok := p.(plugin.SchemaMutator); ok {
			if err := mut.MutateSchema(cfg.Schema); err != nil {
				return nil, nil, fmt.Errorf("%s: %w", p.Name(), err)
			}
		}
	}
	for _, p := range plugins {
		if mut, ok := p.(plugin.ConfigMutator); ok {
			if err := mut.MutateConfig(cfg); err != nil {
				return nil, nil, fmt.Errorf("%s: %w", p.Name(), err)
			}
		}
	}

	cfg.ReloadAllPackages()

	dataPlugins := make([]any, len(plugins))
	for i := range plugins {
		dataPlugins[i] = plugins[i]
	}
	data, err := codegen.BuildData(cfg, dataPlugins...)
	if err != nil {
		return nil, nil, fmt.Errorf("bind schema to go types: %w", err)
	}

	return data, resolvers, nil
}

func checkConfig(cfg *config.Config) error {
	var problems []string
	if !cfg.OmitComplexity {
		problems = append(problems, "omit_complexity must be true")
	}
	if cfg.OmitPanicHandler {
		problems = append(problems, "omit_panic_handler is not supported")
	}
	if cfg.Federation.IsDefined() {
		problems = append(problems, "federation is not supported")
	}
	return unsupported(problems)
}

func checkData(data *codegen.Data) error {
	var problems []string

	if data.SubscriptionRoot != nil {
		problems = append(problems, "subscriptions are not supported")
	}
	for _, location := range []string{"QUERY", "MUTATION", "FIELD"} {
		for name := range data.AllDirectives.LocationDirectives(location) {
			problems = append(problems, "runtime directive @"+name+" on "+location+" is not supported")
		}
	}
	for _, obj := range data.Objects {
		for _, f := range obj.Fields {
			name := obj.Name + "." + f.Name
			switch {
			case f.IsBatch():
				problems = append(problems, "batch resolver "+name+" is not supported")
			case f.Stream:
				problems = append(problems, "streamed field "+name+" is not supported")
			case f.HasDirectives():
				problems = append(problems, "directives on field "+name+" are not supported")
			}
			for _, arg := range f.Args {
				if len(arg.ImplDirectives()) > 0 {
					problems = append(problems, "directives on argument "+name+"("+arg.Name+") are not supported")
				}
			}
		}
	}
	for _, in := range data.Inputs {
		if len(in.InputObjectDirectives()) > 0 {
			problems = append(problems, "directives on input "+in.Name+" are not supported")
		}
		for _, f := range in.Fields {
			if len(f.ImplDirectives()) > 0 {
				problems = append(problems, "directives on input field "+in.Name+"."+f.Name+" are not supported")
			}
		}
	}

	return unsupported(problems)
}

func unsupported(problems []string) error {
	if len(problems) == 0 {
		return nil
	}
	return fmt.Errorf("unsupported gqlgen configuration: %s", strings.Join(problems, "; "))
}

func writeExec(cfg *config.Config, root *rootData) error {
	execDir := cfg.Exec.Dir()
	if err := clean(execDir); err != nil {
		return err
	}

	funcs := template.FuncMap{
		"enumMarshal":   func(t *config.TypeReference) string { return "enumMarshal" + enumKey(t) },
		"enumUnmarshal": func(t *config.TypeReference) string { return "enumUnmarshal" + enumKey(t) },
	}

	for _, s := range root.Shards {
		if err := os.MkdirAll(s.Dir, 0o755); err != nil {
			return fmt.Errorf("create %s: %w", s.Dir, err)
		}
		if err := templates.Render(templates.Options{
			PackageName:     s.Package,
			Filename:        filepath.Join(s.Dir, s.Name+".generated.go"),
			Template:        shardTemplate,
			Data:            s,
			Funcs:           funcs,
			GeneratedHeader: true,
			Packages:        cfg.Packages,
			PruneOptions:    cfg.GetPruneOptions(),
		}); err != nil {
			return fmt.Errorf("render shard %s: %w", s.Name, err)
		}
	}

	if err := templates.Render(templates.Options{
		PackageName:     cfg.Exec.Package,
		Filename:        filepath.Join(execDir, rootFilename),
		Template:        rootTemplate,
		Data:            root,
		Funcs:           funcs,
		GeneratedHeader: true,
		Packages:        cfg.Packages,
		PruneOptions:    cfg.GetPruneOptions(),
	}); err != nil {
		return fmt.Errorf("render root: %w", err)
	}

	return nil
}

func clean(execDir string) error {
	names, err := generatedEntries(execDir)
	if err != nil {
		return err
	}
	for _, name := range names {
		if err = os.RemoveAll(filepath.Join(execDir, name)); err != nil {
			return fmt.Errorf("remove %s: %w", name, err)
		}
	}
	return nil
}

func validate(cfg *config.Config) error {
	dirs := []string{cfg.Exec.Dir()}
	if cfg.Model.IsDefined() {
		dirs = append(dirs, cfg.Model.Dir())
	}
	if cfg.Resolver.IsDefined() {
		dirs = append(dirs, cfg.Resolver.Dir())
	}

	args := []string{"build"}
	for _, dir := range dirs {
		args = append(args, "./"+filepath.ToSlash(relative(dir))+"/...")
	}

	cmd := exec.Command("go", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("generated code does not build: %w", err)
	}
	return nil
}

func relative(dir string) string {
	wd, err := os.Getwd()
	if err != nil {
		return dir
	}
	rel, err := filepath.Rel(wd, dir)
	if err != nil {
		return dir
	}
	return rel
}
