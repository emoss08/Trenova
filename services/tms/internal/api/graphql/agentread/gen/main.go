// Command gen writes the agent read path's generated code from the tools
// listed in reads.yml: which persisted operation each tool reads through, the
// types and inputs its parameters are derived from, and an executor that
// holds only the schema shards and resolvers those operations reach.
//
// Run it from services/tms:
//
//	go run ./internal/api/graphql/agentread/gen
package main

import (
	"flag"
	"fmt"
	"os"
)

type options struct {
	reads     string
	schema    string
	documents string
	generated string
	resolver  string
	execOut   string
	toolsOut  string
	config    string
}

func main() {
	var opts options
	flag.StringVar(&opts.reads, "reads", "internal/api/graphql/agentread/reads.yml",
		"the tools that read through GraphQL")
	flag.StringVar(&opts.schema, "schema", "internal/api/graphql/schema", "the schema directory")
	flag.StringVar(&opts.documents, "documents", "internal/api/graphql/persisted-documents.json",
		"the persisted operations")
	flag.StringVar(&opts.generated, "generated", "internal/api/graphql/generated",
		"the generated executor")
	flag.StringVar(&opts.resolver, "resolver", "internal/api/graphql/resolver",
		"the resolver packages")
	flag.StringVar(&opts.execOut, "exec-out", "internal/api/graphql/agentread",
		"where the read executor is written")
	flag.StringVar(&opts.toolsOut, "tools-out",
		"internal/modules/agent/app/agentquerytoolservice", "where the tool bindings are written")
	flag.StringVar(&opts.config, "config", "gqlgen.yml", "gqlgen's configuration")
	flag.Parse()

	if err := run(&opts); err != nil {
		fmt.Fprintf(os.Stderr, "agentread: %v\n", err)
		os.Exit(1)
	}
}

func run(opts *options) error {
	reads, err := loadReads(opts.reads)
	if err != nil {
		return err
	}
	schema, err := loadSchema(opts.schema)
	if err != nil {
		return err
	}
	catalog, err := loadOperations(schema, opts.documents)
	if err != nil {
		return err
	}

	checks, err := readChecks(opts.resolver)
	if err != nil {
		return err
	}
	bindings, err := bind(schema, catalog, reads, checks)
	if err != nil {
		return err
	}

	types, err := goTypes(opts.config, bindings)
	if err != nil {
		return err
	}

	plan, err := planExecutor(schema, opts, bindings)
	if err != nil {
		return err
	}

	if err = writeExecutor(opts.execOut, plan, bindings); err != nil {
		return err
	}

	return writeTools(opts.toolsOut, schema, bindings, types)
}
