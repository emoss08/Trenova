package cli

import (
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/emoss08/assay/internal/cover"
	"github.com/emoss08/assay/internal/index"
	"github.com/emoss08/assay/internal/vcs"
)

func newExplainCommand(opts *options) *cobra.Command {
	return &cobra.Command{
		Use:   "explain <file>:<line>",
		Short: "Show which tests cover a source line",
		Long: "explain answers the question the index exists to answer: if I change this line,\n" +
			"which tests would notice? It also reports whether the line is attributable at all.",
		Args:    cobra.ExactArgs(1),
		Example: "  assay explain internal/core/domain/shipment/shipment.go:412",
		RunE: func(cmd *cobra.Command, args []string) error {
			path, line, err := parseFileLine(args[0])
			if err != nil {
				return err
			}

			session, err := opts.open(cmd.Context())
			if err != nil {
				return err
			}
			head, err := vcs.HeadCommit(cmd.Context(), session.root)
			if err != nil {
				return err
			}
			loader, err := session.loaderFor(cmd.Context(), head, opts.tags)
			if err != nil {
				return err
			}
			if loader == nil {
				return fmt.Errorf("no index available; run assay index first")
			}

			if !filepath.IsAbs(path) {
				path = filepath.Join(session.root, path)
			}
			path = filepath.Clean(path)

			owner, ok := session.graph.PackageForFile(path)
			if !ok {
				return fmt.Errorf("%s does not belong to any package in the workspace", path)
			}

			classification, err := cover.ClassifyFile(path, []int{line})
			if err != nil {
				return err
			}
			if !classification.Narrowable() {
				cmd.Printf("%s:%d is a declaration; a change here always runs the whole package\n",
					filepath.Base(path), line)

				return nil
			}
			if len(classification.Executable) == 0 {
				cmd.Printf("%s:%d is blank or a plain comment; a change here affects no tests\n",
					filepath.Base(path), line)

				return nil
			}

			// The shared query applies the same staleness rule mutation and narrowing
			// use: records must be indexed at this commit. explain used to skip that
			// guard and would report coverage measured against source that is not
			// the source being asked about.
			query := index.NewCoverageQuery(session.graph, loader, head)
			covering, usable := query.TestsCovering(path, []int{line})

			if !usable {
				return fmt.Errorf(
					"no index record for %s is usable at %s; run assay index at this commit",
					owner.ImportPath, vcs.ShortSHA(head))
			}
			if len(covering) == 0 {
				cmd.Printf("%s:%d is executable but no indexed test covers it\n", filepath.Base(path), line)

				return nil
			}

			packages := make([]string, 0, len(covering))
			for pkg := range covering {
				packages = append(packages, pkg)
			}
			sort.Strings(packages)

			total := 0
			for _, pkg := range packages {
				total += len(covering[pkg])
			}
			cmd.Printf("%s:%d is covered by %d tests across %d packages\n",
				filepath.Base(path), line, total, len(packages))
			for _, pkg := range packages {
				cmd.Printf("  %s\n", pkg)
				for _, test := range covering[pkg] {
					cmd.Printf("    %s\n", test)
				}
			}

			return nil
		},
	}
}

func parseFileLine(spec string) (string, int, error) {
	colon := strings.LastIndex(spec, ":")
	if colon <= 0 || colon == len(spec)-1 {
		return "", 0, fmt.Errorf("expected <file>:<line>, got %q", spec)
	}

	line, err := strconv.Atoi(spec[colon+1:])
	if err != nil || line < 1 {
		return "", 0, fmt.Errorf("invalid line number in %q", spec)
	}

	return spec[:colon], line, nil
}
