import type { CodegenConfig } from "@graphql-codegen/cli";

const config: CodegenConfig = {
  schema: "../services/tms/internal/api/graphql/schema/*.graphqls",
  documents: "src/graphql/operations/**/*.graphql",
  hooks: {
    afterAllFileWrite: ["node scripts/sync-graphql-persisted-documents.mjs"],
  },
  generates: {
    "src/graphql/generated/": {
      preset: "client",
      presetConfig: {
        persistedDocuments: true,
      },
      config: {
        documentMode: "string",
        enumsAsTypes: true,
        useTypeImports: true,
        scalars: {
          Any: "unknown",
          JSON: "unknown",
        },
      },
    },
    "src/graphql/schema.graphql": {
      plugins: ["schema-ast"],
      config: {
        includeDirectives: true,
        sort: true,
      },
    },
  },
};

export default config;
