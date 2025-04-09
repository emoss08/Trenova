// @ts-check
import eslint from "@eslint/js";
import pluginQuery from "@tanstack/eslint-plugin-query";
import eslintPluginPrettierRecommended from "eslint-plugin-prettier/recommended";
import react from "eslint-plugin-react";
import reactCompiler from 'eslint-plugin-react-compiler';
import eslintPluginReactHooks from "eslint-plugin-react-hooks";
import reactRefresh from "eslint-plugin-react-refresh";
import tseslint from "typescript-eslint";

export default tseslint.config(
  tseslint.configs.recommended,
  ...pluginQuery.configs["flat/recommended"],
  eslint.configs.recommended,
  tseslint.configs.strict,
  reactRefresh.configs.vite,
  eslintPluginPrettierRecommended,

  {
    ...react.configs.flat.recommended,
    settings: { react: { version: "detect" } },
  },
  {
    rules: {
      "prettier/prettier": ["error", { endOfLine: "auto" }],
      "@typescript-eslint/no-explicit-any": "off",
      "@typescript-eslint/no-non-null-assertion": "off",
      "react/react-in-jsx-scope": "off",
    },
  },
  {
    plugins: {
      // @ts-expect-error - react-hooks is not typed
      "react-hooks": eslintPluginReactHooks,
      'react-compiler': reactCompiler,

    },
    rules: {
      "react-hooks/rules-of-hooks": "error",
      "react-hooks/exhaustive-deps": "warn",
      "react-compiler/react-compiler": "error",
    },
  },
);
