import { translate } from "@trenova/shared/i18n/runtime";
import { type CompletionContext, type Completion } from "@codemirror/autocomplete";
import {
  HighlightStyle,
  LanguageSupport,
  StreamLanguage,
  syntaxHighlighting,
} from "@codemirror/language";
import { Tag, tags as t } from "@lezer/highlight";
import { translateMessage } from "@trenova/shared/i18n/message";
import {
  functionInsertion,
  isKeyword,
  isKnownFunction,
  isOperatorWord,
  isKnownVariable,
  type KnownIdentifiers,
} from "./known-identifiers";

/*
 * The syntax highlighter: a CodeMirror StreamLanguage, so it stays
 * synchronous and incremental line by line. It colours, nothing more; what a
 * formula means is the rules module's (`formula.check` underlines it), and the
 * words it colours as keywords, operators and functions come from the
 * module's catalog through `KnownIdentifiers`.
 */

export const unknownVariableTag = Tag.define();

type ExprState = {
  inBlockComment: boolean;
  /** Names bound with `let` so far; they read as the expression's own from then on. */
  bindings: Set<string>;
  /** The last word was `let`, so the next one is a name being bound. */
  binding: boolean;
};

export function createExprLanguage(known: KnownIdentifiers) {
  return StreamLanguage.define<ExprState>({
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

      if (stream.match(/\d+\.?\d*([eE][+-]?\d+)?/) || stream.match(/\.\d+([eE][+-]?\d+)?/)) {
        return "number";
      }

      if (stream.match(/[+\-*/%^]=?|[<>!=]=?|&&|\|\||[?:]/)) {
        return "operator";
      }

      if (stream.match(/[()[\]{},;.]/)) {
        return "punctuation";
      }

      if (stream.match(/[a-zA-Z_#][a-zA-Z0-9_]*/)) {
        const word = stream.current();

        if (state.binding) {
          state.binding = false;
          state.bindings.add(word);
          return "localVariable";
        }

        if (isKeyword(known, word) || word === "#" || isOperatorWord(known, word)) {
          state.binding = word === "let";
          return "keyword";
        }

        if (state.bindings.has(word)) {
          return "localVariable";
        }

        if (isKnownFunction(known, word) && stream.peek() === "(") {
          return "exprFunction";
        }

        if (isKnownVariable(known, word)) {
          return "knownVariable";
        }

        return "unknownVariable";
      }

      stream.next();
      return null;
    },

    startState() {
      return { inBlockComment: false, bindings: new Set<string>(), binding: false };
    },

    copyState(state: ExprState): ExprState {
      return {
        inBlockComment: state.inBlockComment,
        bindings: new Set(state.bindings),
        binding: state.binding,
      };
    },

    tokenTable: {
      exprFunction: t.function(t.variableName),
      knownVariable: t.special(t.variableName),
      localVariable: t.local(t.variableName),
      unknownVariable: unknownVariableTag,
    },

    languageData: {
      commentTokens: { line: "//", block: { open: "/*", close: "*/" } },
      closeBrackets: { brackets: ["(", "[", "{", '"', "'", "`"] },
    },
  });
}

export const exprHighlightStyle = HighlightStyle.define([
  { tag: t.keyword, color: "var(--expr-keyword)", fontWeight: "500" },
  { tag: t.operator, color: "var(--expr-operator)" },
  { tag: t.number, color: "var(--expr-number)" },
  { tag: t.string, color: "var(--expr-string)" },
  { tag: t.variableName, color: "var(--expr-variable)" },
  {
    tag: t.special(t.variableName),
    color: "var(--expr-variable)",
    backgroundColor: "color-mix(in oklab, var(--expr-variable) 14%, transparent)",
    borderRadius: "4px",
    padding: "1px 2px",
  },
  { tag: t.function(t.variableName), color: "var(--expr-function)" },
  {
    tag: unknownVariableTag,
    color: "var(--expr-unknown)",
    textDecoration: "underline wavy var(--expr-unknown)",
    textUnderlineOffset: "3px",
  },
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

function completionInfo(title: string, describe: () => string, exemplify?: () => string) {
  return () => {
    const description = describe();
    const example = exemplify?.();
    const node = document.createElement("div");
    node.className = "cm-expr-completion-info";

    const heading = document.createElement("div");
    heading.className = "cm-expr-completion-info-title";
    heading.textContent = title;
    node.appendChild(heading);

    if (description) {
      const body = document.createElement("div");
      body.textContent = description;
      node.appendChild(body);
    }

    if (example) {
      const code = document.createElement("code");
      code.className = "cm-expr-completion-info-example";
      code.textContent = example;
      node.appendChild(code);
    }

    return node;
  };
}

function createCompletions(known: KnownIdentifiers): Completion[] {
  const options: Completion[] = [];

  for (const variable of known.variables) {
    options.push({
      label: variable.name,
      type: "variable",
      detail: variable.type,
      info: completionInfo(variable.name, () => variable.description),
      boost: variable.custom ? 3 : 2,
    });
  }

  for (const fn of known.functions) {
    const insertion = functionInsertion(fn);
    options.push({
      label: fn.name,
      type: fn.operator ? "keyword" : "function",
      detail: fn.signature,
      info: completionInfo(
        fn.signature,
        () => translateMessage(fn.description),
        () => translateMessage(fn.example),
      ),
      apply: (view, _completion, from, to) => {
        view.dispatch({
          changes: { from, to, insert: insertion.text },
          selection: { anchor: from + insertion.cursor },
        });
      },
      boost: 1,
    });
  }

  options.push(
    { label: "true", type: "keyword", detail: translate("Boolean true") },
    { label: "false", type: "keyword", detail: translate("Boolean false") },
    { label: "nil", type: "keyword", detail: translate("Null value") },
  );

  return options;
}

export function exprLanguageSupport(known: KnownIdentifiers) {
  const language = createExprLanguage(known);
  const options = createCompletions(known);

  const completionSource = (context: CompletionContext) => {
    const word = context.matchBefore(/[a-zA-Z_][\w.]*/);
    if (!word || (word.from === word.to && !context.explicit)) {
      return null;
    }
    return { from: word.from, options, validFor: /^[\w.]*$/ };
  };

  return new LanguageSupport(language, [
    syntaxHighlighting(exprHighlightStyle),
    language.data.of({ autocomplete: completionSource }),
  ]);
}
