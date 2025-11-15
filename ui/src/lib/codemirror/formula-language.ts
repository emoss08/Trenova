/*
 * Formula Expression Language Definition for CodeMirror
 *
 * Provides syntax highlighting, autocomplete, and validation for formula templates
 */

import { parser } from "@lezer/highlight";
import { LRLanguage, LanguageSupport } from "@codemirror/language";
import { styleTags, tags as t } from "@lezer/highlight";

// Define the formula language grammar tokens
export const formulaLanguage = LRLanguage.define({
  parser: parser.configure({
    props: [
      styleTags({
        Number: t.number,
        String: t.string,
        Boolean: t.bool,
        Variable: t.variableName,
        Function: t.function(t.variableName),
        Operator: t.operator,
        Comma: t.separator,
        ParenOpen: t.paren,
        ParenClose: t.paren,
        BracketOpen: t.squareBracket,
        BracketClose: t.squareBracket,
        Comment: t.lineComment,
      }),
    ],
  }),
  languageData: {
    commentTokens: { line: "//" },
    closeBrackets: { brackets: ["(", "[", "{", "'", '"'] },
  },
});

// Keywords and operators for formula expressions
export const formulaKeywords = [
  "if",
  "true",
  "false",
  "null",
  "and",
  "or",
  "not",
];

export const formulaOperators = [
  "+",
  "-",
  "*",
  "/",
  "%",
  "^",
  "==",
  "!=",
  ">",
  "<",
  ">=",
  "<=",
  "&&",
  "||",
  "!",
  "?",
  ":",
];

// Built-in function names
export const builtInFunctions = [
  // Math
  "abs",
  "min",
  "max",
  "round",
  "floor",
  "ceil",
  "sqrt",
  "pow",
  "log",
  "exp",
  "sin",
  "cos",
  "tan",
  // Type conversion
  "number",
  "string",
  "bool",
  // Array
  "len",
  "sum",
  "avg",
  "slice",
  "concat",
  "contains",
  "indexOf",
  // Conditional
  "if",
  "coalesce",
];

/**
 * Create a language support instance for formula expressions
 */
export function formula(): LanguageSupport {
  return new LanguageSupport(formulaLanguage);
}
