import eslint from "@eslint/js";
import pluginQuery from "@tanstack/eslint-plugin-query";
import eslintPluginPrettierRecommended from "eslint-plugin-prettier/recommended";
import react from "eslint-plugin-react";
import reactHooks from "eslint-plugin-react-hooks";
import reactRefresh from "eslint-plugin-react-refresh";
import tailwind from "eslint-plugin-tailwindcss";
import { defineConfig } from "eslint/config";
import globals from "globals";
import { dirname } from "path";
import tseslint from "typescript-eslint";
import { fileURLToPath } from "url";

const __filename = fileURLToPath(import.meta.url)
const __dirname = dirname(__filename)


export default defineConfig(
  {
    ignores: ["eslint.config.mjs"],
  },
  tseslint.configs.recommended,
  ...pluginQuery.configs["flat/recommended"],
  eslint.configs.recommended,
  tseslint.configs.strict,
  reactRefresh.configs.vite,
  eslintPluginPrettierRecommended,
  reactHooks.configs["recommended-latest"],
  ...tailwind.configs["flat/recommended"],
  {
    ...react.configs.flat.recommended,
    settings: { react: { version: "detect" } },
    languageOptions: {
      ecmaVersion: 2020,
      globals: globals.browser,
      parserOptions: {
        project: ["./tsconfig.node.json", "./tsconfig.app.json"],
        tsconfigRootDir: import.meta.dirname,
      },
    },
  },
  {
    rules: {
      "prettier/prettier": ["error", { endOfLine: "auto" }],
      "@typescript-eslint/no-explicit-any": "off",
      "@typescript-eslint/no-non-null-assertion": "off",
      "react/react-in-jsx-scope": "off",
      "react/display-name": "off",
    },
  },
  {
    rules: {
      "react-hooks/rules-of-hooks": "error",
      "react-hooks/exhaustive-deps": "error",
    },
  },
  {
  settings: {
    tailwindcss: {
      config: `${__dirname}/src/styles/app.css`
    }
  }
  }
);
