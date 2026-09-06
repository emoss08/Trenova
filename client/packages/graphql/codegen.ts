import type { CodegenConfig } from "@graphql-codegen/cli";
import type { DocumentNode } from "graphql";

function operationMeta(documentNode: DocumentNode): Record<string, string> | undefined {
  for (const definition of documentNode.definitions) {
    if (definition.kind === "OperationDefinition") {
      return {
        kind: definition.operation,
        ...(definition.name ? { name: definition.name.value } : {}),
      };
    }
  }
  return undefined;
}

const config: CodegenConfig = {
  schema: "../../../services/tms/internal/api/graphql/schema/*.graphqls",
  documents: "src/operations/**/*.graphql",
  hooks: {
    afterAllFileWrite: [
      "node scripts/sync-graphql-persisted-documents.mjs",
      "node scripts/generate-graphql-catalog.mjs",
      "node scripts/generate-error-enums.mjs",
      "node scripts/generate-permission-resources.mjs",
    ],
  },
  generates: {
    "src/generated/": {
      preset: "client",
      presetConfig: {
        persistedDocuments: { mode: "replaceDocumentWithHash" },
        onExecutableDocumentNode: operationMeta,
        fragmentMasking: { unmaskFunctionName: "getFragmentData" },
      },
      config: {
        documentMode: "string",
        enumsAsTypes: true,
        useTypeImports: true,
        scalars: {
          Any: "unknown",
          JSON: "unknown",
          Timestamp: { input: "number", output: "number" },
          Decimal: { input: "string", output: "string" },
        },
      },
    },
    "src/schema.graphql": {
      plugins: ["schema-ast"],
      config: {
        includeDirectives: true,
        sort: true,
      },
    },
  },
};

export default config;
