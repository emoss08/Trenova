package agentread

import (
	"fmt"

	"github.com/emoss08/trenova/internal/api/graphql/generated/exec/schemasource"
	"github.com/emoss08/trenova/internal/api/graphql/gqlexec"
	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
)

const workerLimit = 32

// newSchema builds the read executor from the generated wiring: the whole
// parsed schema, the shards the bound operations reach and their resolvers,
// validated against those operations at startup.
func newSchema(
	shards []*gqlexec.Shard,
	resolvers map[string]func() any,
	ops []Operation,
) Schema {
	reg, err := gqlexec.Compile(shards)
	if err != nil {
		panic(fmt.Errorf("agentread: %w", err))
	}
	docs := make([]*ast.QueryDocument, 0, len(ops))
	for _, op := range ops {
		doc, gqlErr := gqlparser.LoadQueryWithRules(schemasource.Parsed, op.Document, nil)
		if gqlErr != nil {
			panic(fmt.Errorf("agentread: operation %s does not validate: %w", op.Name, gqlErr))
		}
		docs = append(docs, doc)
	}

	return Schema{gqlexec.NewSchema(gqlexec.SchemaConfig{
		Registry:    reg,
		Parsed:      schemasource.Parsed,
		Resolvers:   resolvers,
		WorkerLimit: workerLimit,
		Operations:  docs,
	})}
}
