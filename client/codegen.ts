import type { CodegenConfig } from "@graphql-codegen/cli";

const config: CodegenConfig = {
  schema: "../services/tms/internal/api/graphql/schema/*.graphqls",
  documents: "src/graphql/operations/**/*.graphql",
  generates: {
    "src/graphql/generated/": {
      preset: "client",
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
  },
};

export default config;
