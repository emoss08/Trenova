/*
 * Formula Expression Linter
 *
 * Provides real-time validation and error highlighting
 */

import { type Diagnostic } from "@codemirror/lint";
import { type EditorView } from "@codemirror/view";
import { formulaTemplateService } from "@/services/formula-template-service";

// Debounce validation to avoid excessive API calls
let validationTimeout: NodeJS.Timeout | null = null;
const VALIDATION_DELAY = 500; // ms

/**
 * Validate a formula expression and return diagnostics
 */
export async function validateFormula(view: EditorView): Promise<Diagnostic[]> {
  const doc = view.state.doc;
  const expression = doc.toString();

  // Don't validate empty expressions
  if (!expression.trim()) {
    return [];
  }

  try {
    const result = await formulaTemplateService.validateExpression(expression);

    if (result.valid) {
      return [];
    }

    // Create diagnostic for the error
    const diagnostics: Diagnostic[] = [];

    // If we have line/column information, use it
    if (result.line !== undefined && result.column !== undefined) {
      // Convert line/column to document position
      const line = Math.max(0, result.line - 1); // Convert to 0-based
      const lineStart = doc.line(line + 1).from;
      const pos = lineStart + Math.max(0, result.column);

      diagnostics.push({
        from: pos,
        to: pos + 1,
        severity: "error",
        message: result.message || result.error || "Syntax error",
      });
    } else {
      // No position info, highlight the entire expression
      diagnostics.push({
        from: 0,
        to: doc.length,
        severity: "error",
        message: result.message || result.error || "Syntax error",
      });
    }

    return diagnostics;
  } catch (error) {
    console.error("Validation error:", error);
    return [];
  }
}

/**
 * Create a linter function with debouncing
 */
export function createFormulaLinter() {
  return (view: EditorView): Promise<Diagnostic[]> => {
    return new Promise((resolve) => {
      if (validationTimeout) {
        clearTimeout(validationTimeout);
      }

      validationTimeout = setTimeout(async () => {
        const diagnostics = await validateFormula(view);
        resolve(diagnostics);
      }, VALIDATION_DELAY);
    });
  };
}
