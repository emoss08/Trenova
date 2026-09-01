import { linter, type Diagnostic } from "@codemirror/lint";
import type { Extension } from "@codemirror/state";
import {
  EXPR_KEYWORDS,
  isKnownFunction,
  isKnownVariable,
  type KnownIdentifiers,
} from "./known-identifiers";

export type ExprDiagnostic = {
  from: number;
  to: number;
  severity: "error" | "warning";
  message: string;
};

export type IdentifierToken = {
  name: string;
  from: number;
  to: number;
  isCall: boolean;
};

const IDENTIFIER_PATTERN = /[a-zA-Z_][a-zA-Z0-9_]*(?:\.[a-zA-Z_][a-zA-Z0-9_]*)*/y;

/**
 * Walks the expression collecting identifier tokens while skipping strings and
 * comments, mirroring the tokenizer the highlighter uses so the two never
 * disagree about what an identifier is.
 */
export function collectIdentifierTokens(expression: string): IdentifierToken[] {
  const tokens: IdentifierToken[] = [];
  let index = 0;

  while (index < expression.length) {
    const char = expression[index];

    if (char === "/" && expression[index + 1] === "/") {
      const newline = expression.indexOf("\n", index);
      index = newline === -1 ? expression.length : newline + 1;
      continue;
    }

    if (char === "/" && expression[index + 1] === "*") {
      const close = expression.indexOf("*/", index + 2);
      index = close === -1 ? expression.length : close + 2;
      continue;
    }

    if (char === '"' || char === "'" || char === "`") {
      index += 1;
      while (index < expression.length && expression[index] !== char) {
        index += expression[index] === "\\" ? 2 : 1;
      }
      index += 1;
      continue;
    }

    if (/[a-zA-Z_]/.test(char)) {
      IDENTIFIER_PATTERN.lastIndex = index;
      const match = IDENTIFIER_PATTERN.exec(expression);
      if (match) {
        const name = match[0];
        const end = index + name.length;
        let cursor = end;
        while (cursor < expression.length && /\s/.test(expression[cursor])) {
          cursor += 1;
        }

        tokens.push({
          name,
          from: index,
          to: end,
          isCall: expression[cursor] === "(",
        });

        index = end;
        continue;
      }
    }

    index += 1;
  }

  return tokens;
}

function levenshtein(a: string, b: string): number {
  if (Math.abs(a.length - b.length) > 3) return Number.MAX_SAFE_INTEGER;

  const rows = a.length + 1;
  const cols = b.length + 1;
  const distances = Array.from({ length: rows * cols }, () => 0);

  for (let i = 0; i < rows; i++) distances[i * cols] = i;
  for (let j = 0; j < cols; j++) distances[j] = j;

  for (let i = 1; i < rows; i++) {
    for (let j = 1; j < cols; j++) {
      const substitutionCost = a[i - 1] === b[j - 1] ? 0 : 1;
      distances[i * cols + j] = Math.min(
        distances[(i - 1) * cols + j] + 1,
        distances[i * cols + j - 1] + 1,
        distances[(i - 1) * cols + j - 1] + substitutionCost,
      );
    }
  }

  return distances[rows * cols - 1];
}

function nearestKnown(name: string, candidates: Iterable<string>): string | null {
  let best: string | null = null;
  let bestDistance = 3;

  for (const candidate of candidates) {
    const distance = levenshtein(name.toLowerCase(), candidate.toLowerCase());
    if (distance < bestDistance || (distance === bestDistance && best === null)) {
      best = candidate;
      bestDistance = distance;
    }
  }

  return best;
}

const OPEN_FOR_CLOSE: Record<string, string> = { ")": "(", "]": "[", "}": "{" };

function lintBrackets(expression: string, diagnostics: ExprDiagnostic[]) {
  const stack: { char: string; index: number }[] = [];
  let index = 0;

  while (index < expression.length) {
    const char = expression[index];

    if (char === "/" && expression[index + 1] === "/") {
      const newline = expression.indexOf("\n", index);
      index = newline === -1 ? expression.length : newline + 1;
      continue;
    }
    if (char === "/" && expression[index + 1] === "*") {
      const close = expression.indexOf("*/", index + 2);
      index = close === -1 ? expression.length : close + 2;
      continue;
    }
    if (char === '"' || char === "'" || char === "`") {
      index += 1;
      while (index < expression.length && expression[index] !== char) {
        index += expression[index] === "\\" ? 2 : 1;
      }
      index += 1;
      continue;
    }

    if (char === "(" || char === "[" || char === "{") {
      stack.push({ char, index });
    } else if (char === ")" || char === "]" || char === "}") {
      const top = stack[stack.length - 1];
      if (top && top.char === OPEN_FOR_CLOSE[char]) {
        stack.pop();
      } else {
        diagnostics.push({
          from: index,
          to: index + 1,
          severity: "error",
          message: `Unmatched '${char}'`,
        });
      }
    }

    index += 1;
  }

  for (const unclosed of stack) {
    diagnostics.push({
      from: unclosed.index,
      to: unclosed.index + 1,
      severity: "error",
      message: `Unclosed '${unclosed.char}'`,
    });
  }
}

const TRAILING_OPERATOR = /(?:[+\-*/%^]|&&|\|\||==|!=|[<>]=?|\?|:|\b(?:and|or|not|in)\b)\s*$/;

export function lintExpression(expression: string, known: KnownIdentifiers): ExprDiagnostic[] {
  const diagnostics: ExprDiagnostic[] = [];
  const trimmed = expression.trim();

  if (trimmed === "") {
    return diagnostics;
  }

  lintBrackets(expression, diagnostics);

  if (TRAILING_OPERATOR.test(expression.trimEnd())) {
    const end = expression.trimEnd().length;
    diagnostics.push({
      from: Math.max(0, end - 2),
      to: end,
      severity: "error",
      message: "Expression ends with an operator",
    });
  }

  for (const token of collectIdentifierTokens(expression)) {
    if (EXPR_KEYWORDS.has(token.name)) continue;

    if (token.isCall) {
      const callee = token.name;
      if (!callee.includes(".") && !isKnownFunction(known, callee)) {
        const suggestion = nearestKnown(callee, known.functionNames);
        diagnostics.push({
          from: token.from,
          to: token.to,
          severity: "error",
          message: suggestion
            ? `Unknown function '${callee}'. Did you mean '${suggestion}'?`
            : `Unknown function '${callee}'`,
        });
      }
      continue;
    }

    if (!isKnownVariable(known, token.name)) {
      const suggestion = nearestKnown(token.name, known.variablePaths);
      diagnostics.push({
        from: token.from,
        to: token.to,
        severity: "warning",
        message: suggestion
          ? `Unknown variable '${token.name}'. Did you mean '${suggestion}'?`
          : `Unknown variable '${token.name}'. Declare it as a custom variable or pick one from the reference.`,
      });
    }
  }

  return diagnostics;
}

export function createExprLinter(getKnown: () => KnownIdentifiers): Extension {
  return linter(
    (view): Diagnostic[] =>
      lintExpression(view.state.doc.toString(), getKnown()).map((diagnostic) => ({
        from: diagnostic.from,
        to: diagnostic.to,
        severity: diagnostic.severity,
        message: diagnostic.message,
      })),
    { delay: 400 },
  );
}
