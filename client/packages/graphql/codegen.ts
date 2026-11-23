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
  hooks: {
    afterAllFileWrite: [
      "bun scripts/strip-fragment-text.mjs",
      "bun scripts/split-documents.mjs",
      "bun scripts/sync-graphql-persisted-documents.mjs",
      "bun scripts/generate-error-enums.mjs",
      "bun scripts/generate-permission-resources.mjs",
      "bun scripts/generate-record-kinds.mjs",
      "bun scripts/generate-tool-namespaces.mjs",
    ],
  },
  generates: {
    "src/generated/": {
      documents: "src/operations/**/*.graphql",
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
