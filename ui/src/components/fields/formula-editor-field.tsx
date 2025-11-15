/*
 * Formula Editor Field Component
 *
 * CodeMirror-based editor for formula expressions with:
 * - Syntax highlighting
 * - Autocomplete for variables and functions
 * - Real-time validation
 * - Error highlighting
 */

import { useCallback, useEffect, useRef } from "react";
import { Controller, FieldValues } from "react-hook-form";
import CodeMirror, { ReactCodeMirrorRef } from "@uiw/react-codemirror";
import { EditorView } from "@codemirror/view";
import { autocompletion } from "@codemirror/autocomplete";
import { linter } from "@codemirror/lint";
import { tags as t } from "@lezer/highlight";
import { createTheme } from "@uiw/codemirror-themes";

import { cn } from "@/lib/utils";
import type { TextareaFieldProps } from "@/types/fields";
import { FieldWrapper } from "./field-components";
import { useTheme } from "../theme-provider";

import {
  formulaAutocomplete,
  preloadAutocompleteData,
} from "@/lib/codemirror/formula-autocomplete";
import { createFormulaLinter } from "@/lib/codemirror/formula-linter";

interface FormulaEditorFieldProps<T extends FieldValues>
  extends Omit<TextareaFieldProps<T>, "rows" | "cols"> {
  height?: string;
  showLineNumbers?: boolean;
  enableAutocomplete?: boolean;
  enableValidation?: boolean;
}

const lightTheme = createTheme({
  theme: "light",
  settings: {
    background: "#f9fafb",
    backgroundImage: "",
    foreground: "#374151",
    caret: "#000000",
    selection: "#e5e7eb",
    selectionMatch: "#d1d5db",
    gutterBackground: "#f3f4f6",
    gutterForeground: "#6b7280",
    gutterBorder: "#e5e7eb",
    gutterActiveForeground: "#111827",
    lineHighlight: "#f3f4f6",
  },
  styles: [
    { tag: t.comment, color: "#6b7280" },
    { tag: t.variableName, color: "#7c3aed" }, // Purple for variables
    { tag: t.string, color: "#059669" }, // Green for strings
    { tag: t.number, color: "#dc2626" }, // Red for numbers
    { tag: t.bool, color: "#ea580c" }, // Orange for booleans
    { tag: t.keyword, color: "#2563eb" }, // Blue for keywords
    { tag: t.function(t.variableName), color: "#0891b2" }, // Cyan for functions
    { tag: t.operator, color: "#4b5563" }, // Gray for operators
    { tag: t.paren, color: "#6b7280" },
    { tag: t.bracket, color: "#6b7280" },
  ],
});

const darkTheme = createTheme({
  theme: "dark",
  settings: {
    background: "#0d0d0d",
    backgroundImage: "",
    foreground: "#e5e7eb",
    caret: "#ffffff",
    selection: "#374151",
    selectionMatch: "#1f2937",
    gutterBackground: "#111827",
    gutterForeground: "#9ca3af",
    gutterBorder: "#374151",
    gutterActiveForeground: "#f3f4f6",
    lineHighlight: "#1f2937",
  },
  styles: [
    { tag: t.comment, color: "#6b7280" },
    { tag: t.variableName, color: "#a78bfa" }, // Purple for variables
    { tag: t.string, color: "#34d399" }, // Green for strings
    { tag: t.number, color: "#f87171" }, // Red for numbers
    { tag: t.bool, color: "#fb923c" }, // Orange for booleans
    { tag: t.keyword, color: "#60a5fa" }, // Blue for keywords
    { tag: t.function(t.variableName), color: "#22d3ee" }, // Cyan for functions
    { tag: t.operator, color: "#9ca3af" }, // Gray for operators
    { tag: t.paren, color: "#9ca3af" },
    { tag: t.bracket, color: "#9ca3af" },
  ],
});

export function FormulaEditorField<T extends FieldValues>({
  label,
  description,
  name,
  control,
  rules,
  className,
  disabled,
  placeholder,
  height = "200px",
  showLineNumbers = true,
  enableAutocomplete = true,
  enableValidation = true,
  "aria-label": ariaLabel,
  "aria-describedby": ariaDescribedBy,
}: FormulaEditorFieldProps<T>) {
  const inputId = `formula-editor-${name}`;
  const descriptionId = `${inputId}-description`;
  const errorId = `${inputId}-error`;
  const { theme } = useTheme();
  const editorRef = useRef<ReactCodeMirrorRef>(null);

  // Preload autocomplete data when component mounts
  useEffect(() => {
    if (enableAutocomplete) {
      preloadAutocompleteData();
    }
  }, [enableAutocomplete]);

  // Build extensions
  const extensions = useCallback(() => {
    const exts = [EditorView.lineWrapping];

    // Add autocomplete
    if (enableAutocomplete) {
      exts.push(
        autocompletion({
          override: [formulaAutocomplete],
          activateOnTyping: true,
          closeOnBlur: true,
        })
      );
    }

    // Add validation linter
    if (enableValidation) {
      exts.push(linter(createFormulaLinter()));
    }

    return exts;
  }, [enableAutocomplete, enableValidation]);

  return (
    <Controller<T>
      name={name}
      control={control}
      rules={rules}
      render={({ field, fieldState }) => (
        <FieldWrapper
          label={label}
          description={description}
          required={!!rules?.required}
          error={fieldState.error?.message}
        >
          <div
            className={cn(
              "relative overflow-hidden rounded-md border border-border",
              fieldState.invalid &&
                "border-red-500 ring-2 ring-red-500/20",
              disabled && "opacity-50 cursor-not-allowed",
              className
            )}
            id={inputId}
            aria-label={ariaLabel || label}
            aria-describedby={cn(
              description && descriptionId,
              fieldState.error && errorId,
              ariaDescribedBy
            )}
          >
            <CodeMirror
              ref={editorRef}
              value={field.value || ""}
              onChange={(value) => field.onChange(value)}
              onBlur={() => field.onBlur()}
              height={height}
              extensions={extensions()}
              editable={!disabled}
              placeholder={placeholder}
              basicSetup={{
                lineNumbers: showLineNumbers,
                foldGutter: false,
                dropCursor: true,
                allowMultipleSelections: false,
                indentOnInput: true,
                bracketMatching: true,
                closeBrackets: true,
                autocompletion: enableAutocomplete,
                rectangularSelection: false,
                highlightSelectionMatches: true,
                searchKeymap: true,
                highlightActiveLine: true,
                highlightActiveLineGutter: true,
              }}
              theme={theme === "dark" ? darkTheme : lightTheme}
            />
          </div>

          {/* Helper text with keyboard shortcuts */}
          <div className="mt-2 text-xs text-muted-foreground">
            <span className="font-medium">Tips:</span> Type to see autocomplete
            suggestions. Press <kbd className="px-1 py-0.5 bg-muted rounded">Ctrl+Space</kbd> to
            manually trigger autocomplete.
          </div>
        </FieldWrapper>
      )}
    />
  );
}
