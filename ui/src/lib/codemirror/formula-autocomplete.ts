/*
 * Formula Expression Autocomplete Provider
 *
 * Provides intelligent autocomplete for variables and functions
 * based on the current context in the formula editor
 */

import { type CompletionContext, type CompletionResult, type Completion } from "@codemirror/autocomplete";
import { syntaxTree } from "@codemirror/language";
import {
  formulaTemplateService,
  type VariableInfo,
  type FunctionInfo,
} from "@/services/formula-template-service";

// Cache for variables and functions to avoid repeated API calls
let variablesCache: VariableInfo[] | null = null;
let functionsCache: FunctionInfo[] | null = null;
let cacheTimestamp = 0;
const CACHE_TTL = 5 * 60 * 1000; // 5 minutes

/**
 * Load variables and functions into cache
 */
async function loadAutocompleteData(): Promise<void> {
  const now = Date.now();

  // Return cached data if still valid
  if (variablesCache && functionsCache && now - cacheTimestamp < CACHE_TTL) {
    return;
  }

  try {
    const [variablesResponse, functionsResponse] = await Promise.all([
      formulaTemplateService.getVariables(),
      formulaTemplateService.getFunctions(),
    ]);

    variablesCache = variablesResponse.variables;
    functionsCache = functionsResponse.functions;
    cacheTimestamp = now;
  } catch (error) {
    console.error("Failed to load autocomplete data:", error);
    // Use empty arrays if fetch fails
    variablesCache = [];
    functionsCache = [];
  }
}

/**
 * Convert variable info to CodeMirror completion
 */
function variableToCompletion(variable: VariableInfo): Completion {
  return {
    label: variable.name,
    type: "variable",
    detail: variable.type.toLowerCase(),
    info: variable.description,
    boost: 10, // Boost variables higher than keywords
  };
}

/**
 * Convert function info to CodeMirror completion
 */
function functionToCompletion(func: FunctionInfo): Completion {
  return {
    label: func.name,
    type: "function",
    detail: func.signature,
    info: `${func.description}\n\nExample: ${func.example}`,
    apply: `${func.name}()`,
    boost: 9, // Slightly lower than variables
  };
}

/**
 * Get word before cursor for filtering
 */
function getWordBefore(context: CompletionContext): { from: number; text: string } {
  const word = context.matchBefore(/\w*/);
  return word ? { from: word.from, text: word.text } : { from: context.pos, text: "" };
}

/**
 * Determine if we're inside a function call
 */
function isInFunctionCall(context: CompletionContext): boolean {
  const tree = syntaxTree(context.state);
  const node = tree.resolveInner(context.pos, -1);

  // Walk up the tree to find if we're inside parentheses
  let current = node;
  while (current) {
    if (current.name === "CallExpression" || current.name === "Arguments") {
      return true;
    }
    current = current.parent!;
  }

  return false;
}

/**
 * Main autocomplete function
 */
export async function formulaAutocomplete(
  context: CompletionContext
): Promise<CompletionResult | null> {
  // Load autocomplete data (uses cache if available)
  await loadAutocompleteData();

  if (!variablesCache || !functionsCache) {
    return null;
  }

  const word = getWordBefore(context);

  // Don't show completions if we're not at a word boundary
  if (word.text === "" && context.explicit === false) {
    return null;
  }

  const completions: Completion[] = [];
  const isInFunc = isInFunctionCall(context);

  // Add variables (always available)
  completions.push(...variablesCache.map(variableToCompletion));

  // Add functions (only if not already in a function call)
  if (!isInFunc) {
    completions.push(...functionsCache.map(functionToCompletion));
  }

  // Add operators and keywords
  const operators = [
    { label: "if", type: "keyword", detail: "conditional", info: "Conditional expression: if(condition, trueValue, falseValue)" },
    { label: "true", type: "keyword", detail: "boolean" },
    { label: "false", type: "keyword", detail: "boolean" },
    { label: "null", type: "keyword", detail: "null value" },
  ];

  completions.push(...operators);

  return {
    from: word.from,
    options: completions,
    filter: true, // Enable client-side filtering
  };
}

/**
 * Clear the autocomplete cache (useful when data changes)
 */
export function clearAutocompleteCache(): void {
  variablesCache = null;
  functionsCache = null;
  cacheTimestamp = 0;
}

/**
 * Preload autocomplete data (call this when the editor mounts)
 */
export function preloadAutocompleteData(): void {
  loadAutocompleteData().catch((error) => {
    console.error("Failed to preload autocomplete data:", error);
  });
}
