// @ts-check
import eslint from "@eslint/js";
import pluginQuery from "@tanstack/eslint-plugin-query";
import eslintPluginPrettierRecommended from "eslint-plugin-prettier/recommended";
import react from "eslint-plugin-react";
import reactCompiler from "eslint-plugin-react-compiler";
import reactRefresh from "eslint-plugin-react-refresh";
import tailwind from "eslint-plugin-tailwindcss";
import tseslint from "typescript-eslint";

export default tseslint.config(
  tseslint.configs.recommended,
  ...pluginQuery.configs["flat/recommended"],
  ...tailwind.configs["flat/recommended"],
  eslint.configs.recommended,
  tseslint.configs.strict,
  reactRefresh.configs.vite,
  eslintPluginPrettierRecommended,
  {
    ...react.configs.flat.recommended,
    settings: { react: { version: "18" } },
  },
  reactCompiler.configs.recommended,
  {
    rules: {
      "prettier/prettier": ["error", { endOfLine: "auto" }],
      "@typescript-eslint/no-explicit-any": "off",
      "react/react-in-jsx-scope": "off",
    },
  },
);
