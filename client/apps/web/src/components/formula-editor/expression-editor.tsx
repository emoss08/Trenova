import { FieldWrapper } from "@/components/fields/field-components";
import { useResolvedTheme } from "@/hooks/use-resolved-theme";
import { cn } from "@trenova/shared/lib/utils";
import type { FormControlProps } from "@trenova/shared/types/fields";
import type { VariableDefinitionInput } from "@trenova/shared/types/formula-template";
import { EditorView } from "@codemirror/view";
import CodeMirror, {
  type ReactCodeMirrorProps,
  type ReactCodeMirrorRef,
} from "@uiw/react-codemirror";
import { useCallback, useEffect, useMemo, useRef, type Ref } from "react";
import { Controller, type FieldValues } from "react-hook-form";
import { useActiveEditorRegistration } from "./active-editor";
import { darkTheme, languageServiceTheme, lightTheme } from "./editor-theme";
import { formulaCatalog, type FormulaFormatResult } from "@trenova/shared/rules/formula";
import { useRules } from "@trenova/shared/hooks/use-rules";
import { inlineValuesExtension, setInlineValues, type InlineValue } from "./expr-inline-values";
import { buildKnownIdentifiers, type KnownIdentifiers } from "./known-identifiers";
import type { DeclareInput } from "./expr-lint";
import { highlightRangeExtension, setHighlightRange, type EditorRange } from "./highlight-range";
import { usePreviewValues } from "./preview-values";
import { useExprExtensions } from "./use-expr-extensions";

type ExpressionEditorProps<T extends FieldValues> = FormControlProps<T> &
  ReactCodeMirrorProps & {
    customVariables?: VariableDefinitionInput[];
    knownIdentifiers?: KnownIdentifiers;
    label?: string;
    description?: string;
    variant?: "default" | "mini";
    lint?: boolean;
    editorRef?: Ref<ReactCodeMirrorRef>;
    /** The editor reference-pane inserts fall back to when nothing else was focused. */
    primary?: boolean;
    /** Drawn inside a card that owns the frame: no border of its own. */
    embedded?: boolean;
    /** A range of the expression to light, such as the statement an explanation row is about. */
    highlight?: EditorRange | null;
    /** Told where the cursor is, one-based, as it moves. */
    onCursorChange?: (position: { line: number; column: number }) => void;
    /** Declares an unknown name as an input, from a quick fix in the editor. */
    declareInput?: DeclareInput;
    /** What each statement came to on the preview load, drawn at the end of its line. */
    inlineValues?: readonly InlineValue[] | null;
    /** Told what formatting from the keyboard did. */
    onFormat?: (result: FormulaFormatResult) => void;
  };

function assignRef<T>(ref: Ref<T> | undefined, value: T | null) {
  if (!ref) return;
  if (typeof ref === "function") {
    ref(value);
  } else {
    (ref as { current: T | null }).current = value;
  }
}

/** Moves the cursor onto a span, scrolls it into view and focuses the editor. */
export function selectEditorRange(view: EditorView, range: EditorRange) {
  const length = view.state.doc.length;
  const from = Math.max(0, Math.min(range.from, length));
  const to = Math.max(from, Math.min(range.to, length));
  view.dispatch({
    selection: { anchor: from, head: to },
    effects: EditorView.scrollIntoView(from, { y: "center" }),
  });
  view.focus();
}

export function ExpressionEditor<T extends FieldValues>({
  name,
  label,
  description,
  customVariables = [],
  knownIdentifiers,
  variant = "default",
  lint = true,
  editorRef,
  primary = false,
  embedded = false,
  highlight = null,
  onCursorChange,
  declareInput,
  inlineValues = null,
  onFormat,
  control,
  rules,
  ...props
}: ExpressionEditorProps<T>) {
  const resolvedTheme = useResolvedTheme();
  const { setActive, setPrimary } = useActiveEditorRegistration();
  const isMini = variant === "mini";

  // The editor calls the rules module synchronously (lint, hover, signature
  // help, format), so it renders only once the module has loaded.
  const runtime = useRules();
  const known = useMemo(
    () =>
      knownIdentifiers ??
      buildKnownIdentifiers(formulaCatalog(runtime), undefined, customVariables),
    [knownIdentifiers, runtime, customVariables],
  );

  const viewRef = useRef<EditorView | null>(null);
  const previewValues = usePreviewValues();
  const exprExtensions = useExprExtensions(runtime, known, lint, viewRef, {
    hoverValues: previewValues,
    declareInput,
    gutter: !isMini,
    onFormat,
  });
  const extensions = useMemo(
    () => [
      ...exprExtensions,
      highlightRangeExtension(),
      inlineValuesExtension(),
      languageServiceTheme,
    ],
    [exprExtensions],
  );

  useEffect(() => {
    viewRef.current?.dispatch({ effects: setHighlightRange.of(highlight) });
  }, [highlight]);

  useEffect(() => {
    viewRef.current?.dispatch({ effects: setInlineValues.of(inlineValues ?? []) });
  }, [inlineValues]);

  const cursorRef = useRef(onCursorChange);
  useEffect(() => {
    cursorRef.current = onCursorChange;
  }, [onCursorChange]);

  const handleRef = useCallback(
    (instance: ReactCodeMirrorRef | null) => {
      viewRef.current = instance?.view ?? null;
      assignRef(editorRef, instance);
      if (primary) {
        setPrimary?.(instance?.view ?? null);
      }
    },
    [editorRef, primary, setPrimary],
  );

  // The view is created after the ref callback fires, so the primary
  // registration is refreshed once it exists.
  useEffect(() => {
    if (primary && viewRef.current) {
      setPrimary?.(viewRef.current);
    }
  });

  return (
    <Controller<T>
      name={name}
      control={control}
      rules={rules}
      render={({ field, fieldState }) => (
        <FieldWrapper
          label={label}
          description={description}
          error={fieldState.error?.message}
          required={!!rules?.required}
        >
          <div
            onFocusCapture={() => setActive?.(viewRef.current)}
            className={cn(
              "overflow-hidden transition-all duration-200",
              embedded ? "rounded-none" : "border-input ui-container-focus-ring rounded-md border",
              fieldState.invalid &&
                !embedded &&
                "ui-container-focus-ring border-destructive [--ring:var(--ring-danger)]",
            )}
          >
            <CodeMirror
              ref={handleRef}
              value={field.value || ""}
              onChange={(value) => field.onChange(value)}
              onUpdate={(update) => {
                if (!cursorRef.current || (!update.selectionSet && !update.docChanged)) return;
                const head = update.state.selection.main.head;
                const line = update.state.doc.lineAt(head);
                cursorRef.current({ line: line.number, column: head - line.from + 1 });
              }}
              extensions={extensions}
              aria-invalid={fieldState.invalid ? "true" : "false"}
              minHeight={isMini ? "60px" : undefined}
              basicSetup={{
                lineNumbers: !isMini,
                foldGutter: false,
                dropCursor: true,
                allowMultipleSelections: false,
                indentOnInput: true,
                bracketMatching: true,
                closeBrackets: true,
                autocompletion: true,
                highlightActiveLine: !isMini,
                highlightActiveLineGutter: !isMini,
                highlightSelectionMatches: true,
                searchKeymap: false,
              }}
              theme={resolvedTheme === "dark" ? darkTheme : lightTheme}
              {...props}
            />
          </div>
        </FieldWrapper>
      )}
    />
  );
}
