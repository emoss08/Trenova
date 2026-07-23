import {
  AVAILABLE_FUNCTIONS,
  SHIPMENT_VARIABLES,
  type VariableDefinitionInput,
} from "@/types/formula-template";
import { type CompletionContext } from "@codemirror/autocomplete";
import {
  HighlightStyle,
  LanguageSupport,
  StreamLanguage,
  syntaxHighlighting,
} from "@codemirror/language";
import { tags as t } from "@lezer/highlight";

const KEYWORDS = new Set(["true", "false", "nil", "in", "not", "and", "or"]);
const BUILTIN_FUNCTIONS = new Set<string>(
  AVAILABLE_FUNCTIONS.map((f) => f.name),
);
const BUILTIN_VARIABLES = new Set<string>(
  SHIPMENT_VARIABLES.map((v) => {
    const dot = v.name.indexOf(".");
    return dot === -1 ? v.name : v.name.slice(0, dot);
  }),
);

type ExprState = {
  inBlockComment: boolean;
};

const exprLanguage = StreamLanguage.define<ExprState>({
  name: "expr",

  token(stream, state: ExprState) {
    if (stream.eatSpace()) {
      return null;
    }

    if (stream.match(/\/\/.*/)) {
      return "lineComment";
    }

    if (stream.match(/\/\*/)) {
      state.inBlockComment = true;
      return "blockComment";
    }

    if (state.inBlockComment) {
      if (stream.match(/.*?\*\//)) {
        state.inBlockComment = false;
      } else {
        stream.skipToEnd();
      }
      return "blockComment";
    }

    if (stream.match(/"(?:[^"\\]|\\.)*"/)) {
      return "string";
    }
    if (stream.match(/'(?:[^'\\]|\\.)*'/)) {
      return "string";
    }
    if (stream.match(/`(?:[^`\\]|\\.)*`/)) {
      return "string";
    }

    if (
      stream.match(/\d+\.?\d*([eE][+-]?\d+)?/) ||
      stream.match(/\.\d+([eE][+-]?\d+)?/)
    ) {
      return "number";
    }

    if (stream.match(/[+\-*/%^]=?|[<>!=]=?|&&|\|\||[?:]/)) {
      return "operator";
    }

    if (stream.match(/[()[\]{},;.]/)) {
      return "punctuation";
    }

    if (stream.match(/[a-zA-Z_][a-zA-Z0-9_]*/)) {
      const word = stream.current();

      if (KEYWORDS.has(word)) {
        return "keyword";
      }

      if (BUILTIN_FUNCTIONS.has(word)) {
        return "function";
      }

      if (BUILTIN_VARIABLES.has(word)) {
        return "variableName";
      }

      return "variableName";
    }

    stream.next();
    return null;
  },

  startState() {
    return { inBlockComment: false };
  },

  copyState(state: ExprState): ExprState {
    return { inBlockComment: state.inBlockComment };
  },

  languageData: {
    commentTokens: { line: "//", block: { open: "/*", close: "*/" } },
    closeBrackets: { brackets: ["(", "[", "{", '"', "'", "`"] },
  },
});

export const exprHighlightStyle = HighlightStyle.define([
  { tag: t.keyword, color: "var(--expr-keyword)", fontWeight: "500" },
  { tag: t.operator, color: "var(--expr-operator)" },
  { tag: t.number, color: "var(--expr-number)" },
  { tag: t.string, color: "var(--expr-string)" },
  { tag: t.variableName, color: "var(--expr-variable)" },
  { tag: t.function(t.variableName), color: "var(--expr-function)" },
  { tag: t.bool, color: "var(--expr-keyword)" },
  { tag: t.null, color: "var(--expr-keyword)" },
  {
    tag: t.comment,
    color: "var(--expr-comment)",
    fontStyle: "italic",
  },
  {
    tag: t.lineComment,
    color: "var(--expr-comment)",
    fontStyle: "italic",
  },
  {
    tag: t.blockComment,
    color: "var(--expr-comment)",
    fontStyle: "italic",
  },
  { tag: t.paren, color: "var(--expr-punctuation)" },
  { tag: t.squareBracket, color: "var(--expr-punctuation)" },
  { tag: t.brace, color: "var(--expr-punctuation)" },
  { tag: t.punctuation, color: "var(--expr-punctuation)" },
]);

function createCompletions(customVariables: VariableDefinitionInput[] = []) {
  return [
    ...SHIPMENT_VARIABLES.map((v) => ({
      label: v.name,
      type: "variable" as const,
      detail: v.type,
      info: v.description,
      boost: 2,
    })),
    ...AVAILABLE_FUNCTIONS.map((f) => ({
      label: f.name,
      type: "function" as const,
      detail: f.signature,
      info: f.description,
      apply: `${f.name}()`,
      boost: 1,
    })),
    ...customVariables.map((v) => ({
      label: v.name,
      type: "variable" as const,
      detail: `${v.type} (custom)`,
      info: v.description || "Custom variable",
      boost: 3,
    })),
    {
      label: "true",
      type: "keyword" as const,
      detail: "Boolean true",
    },
    {
      label: "false",
      type: "keyword" as const,
      detail: "Boolean false",
    },
    { label: "nil", type: "keyword" as const, detail: "Null value" },
  ];
}

export function exprLanguageSupport(
  customVariables: VariableDefinitionInput[] = [],
) {
  const options = createCompletions(customVariables);

  const completionSource = (context: CompletionContext) => {
    const word = context.matchBefore(/[a-zA-Z_][\w.]*/);
    if (!word || (word.from === word.to && !context.explicit)) {
      return null;
    }
    return { from: word.from, options, validFor: /^[\w.]*$/ };
  };

  return new LanguageSupport(exprLanguage, [
    syntaxHighlighting(exprHighlightStyle),
    exprLanguage.data.of({ autocomplete: completionSource }),
  ]);
}

export { exprLanguage };
