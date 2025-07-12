import { cn } from "@/lib/utils";
import { useMemo } from "react";

// Formula language tokens and keywords
const FORMULA_TOKENS = {
  // Keywords
  keywords: ["if", "true", "false", "null"],

  // Built-in functions
  functions: [
    "abs",
    "min",
    "max",
    "pow",
    "sqrt",
    "log",
    "exp",
    "sin",
    "cos",
    "tan",
    "round",
    "floor",
    "ceil",
    "number",
    "string",
    "bool",
    "len",
    "sum",
    "avg",
    "slice",
    "concat",
    "contains",
    "indexOf",
    "coalesce",
  ],

  // Common shipment variables
  variables: [
    "proNumber",
    "status",
    "weight",
    "pieces",
    "temperatureMin",
    "temperatureMax",
    "freightChargeAmount",
    "ratingMethod",
    "ratingUnit",
    "hasHazmat",
    "temperatureDifferential",
    "requiresTemperatureControl",
    "totalStops",
    "customer",
    "tractorType",
    "isExpedited",
    "rate",
    "totalDistance",
    "base",
    "isSameDay",
    "isNextDay",
    "trailerType",
    "commodities",
  ],

  // Operators
  operators: [
    "+",
    "-",
    "*",
    "/",
    "%",
    "==",
    "!=",
    "<=",
    ">=",
    "<",
    ">",
    "&&",
    "||",
    "!",
  ],
};

interface Token {
  type:
    | "keyword"
    | "function"
    | "variable"
    | "operator"
    | "number"
    | "string"
    | "punctuation"
    | "text";
  value: string;
}

function tokenizeExpression(expression: string): Token[] {
  const tokens: Token[] = [];
  let i = 0;

  while (i < expression.length) {
    const char = expression[i];

    // Skip whitespace
    if (/\s/.test(char)) {
      const start = i;
      while (i < expression.length && /\s/.test(expression[i])) {
        i++;
      }
      tokens.push({ type: "text", value: expression.slice(start, i) });
      continue;
    }

    // Handle strings
    if (char === '"' || char === "'") {
      const quote = char;
      const start = i;
      i++; // Skip opening quote
      while (i < expression.length && expression[i] !== quote) {
        if (expression[i] === "\\") i++; // Skip escaped characters
        i++;
      }
      if (i < expression.length) i++; // Skip closing quote
      tokens.push({ type: "string", value: expression.slice(start, i) });
      continue;
    }

    // Handle numbers
    if (/\d/.test(char)) {
      const start = i;
      while (i < expression.length && /[\d.]/.test(expression[i])) {
        i++;
      }
      tokens.push({ type: "number", value: expression.slice(start, i) });
      continue;
    }

    // Handle operators
    const twoCharOp = expression.slice(i, i + 2);
    const oneCharOp = char;

    if (FORMULA_TOKENS.operators.includes(twoCharOp)) {
      tokens.push({ type: "operator", value: twoCharOp });
      i += 2;
      continue;
    }

    if (FORMULA_TOKENS.operators.includes(oneCharOp)) {
      tokens.push({ type: "operator", value: oneCharOp });
      i++;
      continue;
    }

    // Handle punctuation
    if (/[(),[\];]/.test(char)) {
      tokens.push({ type: "punctuation", value: char });
      i++;
      continue;
    }

    // Handle identifiers (keywords, functions, variables)
    if (/[a-zA-Z_$]/.test(char)) {
      const start = i;
      while (i < expression.length && /[a-zA-Z0-9_$]/.test(expression[i])) {
        i++;
      }
      const identifier = expression.slice(start, i);

      // Check if it's a keyword
      if (FORMULA_TOKENS.keywords.includes(identifier)) {
        tokens.push({ type: "keyword", value: identifier });
      }
      // Check if it's a function (followed by parenthesis)
      else if (
        FORMULA_TOKENS.functions.includes(identifier) &&
        i < expression.length &&
        expression[i] === "("
      ) {
        tokens.push({ type: "function", value: identifier });
      }
      // Check if it's a known variable
      else if (FORMULA_TOKENS.variables.includes(identifier)) {
        tokens.push({ type: "variable", value: identifier });
      }
      // Unknown identifier
      else {
        tokens.push({ type: "text", value: identifier });
      }
      continue;
    }

    // Default: treat as text
    tokens.push({ type: "text", value: char });
    i++;
  }

  return tokens;
}

function getTokenClassName(token: Token): string {
  const baseClasses = "inline";

  switch (token.type) {
    case "keyword":
      return `${baseClasses} text-red-500 dark:text-red-600`;
    case "function":
      return `${baseClasses} text-lime-300 dark:text-lime-600`;
    case "variable":
      return `${baseClasses} text-background`;
    case "operator":
      return `${baseClasses} text-orange-400 dark:text-orange-600`;
    case "number":
      return `${baseClasses} text-background`;
    case "string":
      return `${baseClasses} text-yellow-300 dark:text-yellow-600`;
    case "punctuation":
      return `${baseClasses} text-red-500 dark:text-red-600`;
    default:
      return `${baseClasses} text-orange-400 dark:text-orange-600`;
  }
}

export interface ExpressionHighlightProps {
  expression: string;
  className?: string;
}

export function ExpressionHighlight({
  expression,
  className = "",
}: ExpressionHighlightProps) {
  const tokens = useMemo(() => tokenizeExpression(expression), [expression]);

  return (
    <div
      className={cn(
        "font-mono text-sm leading-relaxed overflow-x-auto",
        className,
      )}
    >
      <code>
        {tokens.map((token, index) => (
          <span key={index} className={getTokenClassName(token)}>
            {token.value}
          </span>
        ))}
      </code>
    </div>
  );
}
