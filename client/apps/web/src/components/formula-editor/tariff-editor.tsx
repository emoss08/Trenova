import { FieldWrapper } from "@/components/fields/field-components";
import { useResolvedTheme } from "@/hooks/use-resolved-theme";
import type { EditorView } from "@codemirror/view";
import { cn } from "@trenova/shared/lib/utils";
import type { FormControlProps } from "@trenova/shared/types/fields";
import type {
  TariffCheckResult,
  TariffContext,
  TariffFileKind,
  TariffRange,
} from "@trenova/shared/types/tariff";
import CodeMirror, { type ReactCodeMirrorRef } from "@uiw/react-codemirror";
import { useEffect, useRef } from "react";
import { Controller, type FieldValues } from "react-hook-form";
import { useActiveEditorRegistration } from "./active-editor";
import { darkTheme, languageServiceTheme, lightTheme } from "./editor-theme";
import { highlightRangeExtension, setHighlightRange } from "./highlight-range";

export type TariffEditorProps = {
  /** The kind of file: a whole tariff, a library, or an accessorial's apply-when condition. */
  kind: TariffFileKind;
  value: string;
  onChange: (source: string) => void;
  /**
   * The organisation's libraries, rate tables, charge codes and settings the file is checked
   * against (`useTariffCheckContext()`); null while it loads, which checks with no tables and
   * leaves charge codes unchecked.
   */
  context: TariffContext | null;
  /** Told the module's check of each text the editor shows, the same one its linter draws. */
  onCheck?: (result: TariffCheckResult) => void;
  /** A range of the source to light, such as the line a bill row or a failing test is about. */
  highlight?: TariffRange | null;
  /** Given the editor's view once it exists, and null when it goes. */
  onView?: (view: EditorView | null) => void;
  /** `mini` is a short editor without line numbers, for a condition in a form row. */
  variant?: "default" | "mini";
  /** Drawn inside a surface that owns the frame: no border of its own. */
  embedded?: boolean;
  /** The reference pane inserts into this editor when no other was focused. */
  primary?: boolean;
  readOnly?: boolean;
  invalid?: boolean;
  /** The editor's accessible name, when no visible label names it. */
  ariaLabel?: string;
  minHeight?: string;
  maxHeight?: string;
  className?: string;
};

/**
 * The Tariff source editor. Everything it shows about the text comes from the rules module:
 * highlighting from `tariff.tokens`, findings and their fixes from `tariff.check`, and
 * completion, hover, signature help, go to definition, rename and inlay hints from the editor
 * operations, so the browser has no grammar of its own.
 */
export function TariffEditor({
  kind,
  value,
  onChange,
  highlight = null,
  onView,
  variant = "default",
  embedded = false,
  primary = false,
  readOnly = false,
  invalid = false,
  ariaLabel,
  minHeight,
  maxHeight,
  className,
}: TariffEditorProps) {
  const resolvedTheme = useResolvedTheme();
  const { setActive, setPrimary } = useActiveEditorRegistration();
  const viewRef = useRef<EditorView | null>(null);
  const mini = variant === "mini";

  useEffect(() => {
    viewRef.current?.dispatch({ effects: setHighlightRange.of(highlight) });
  }, [highlight]);

  const handleRef = (instance: ReactCodeMirrorRef | null) => {
    const view = instance?.view ?? null;
    viewRef.current = view;
    onView?.(view);
    if (primary) setPrimary?.(view);
  };

  return (
    <div
      data-tariff-kind={kind}
      onFocusCapture={() => setActive?.(viewRef.current)}
      className={cn(
        "overflow-hidden",
        embedded ? "rounded-none" : "border-input ui-container-focus-ring rounded-md border",
        invalid && !embedded && "border-destructive [--ring:var(--ring-danger)]",
        className,
      )}
    >
      <CodeMirror
        ref={handleRef}
        value={value}
        onChange={onChange}
        readOnly={readOnly}
        aria-label={ariaLabel}
        aria-invalid={invalid ? "true" : "false"}
        minHeight={minHeight ?? (mini ? "60px" : undefined)}
        maxHeight={maxHeight}
        extensions={[highlightRangeExtension(), languageServiceTheme]}
        basicSetup={{
          lineNumbers: !mini,
          foldGutter: !mini,
          dropCursor: true,
          allowMultipleSelections: false,
          indentOnInput: true,
          bracketMatching: true,
          closeBrackets: true,
          autocompletion: false,
          highlightActiveLine: !mini,
          highlightActiveLineGutter: !mini,
          highlightSelectionMatches: true,
          searchKeymap: true,
        }}
        theme={resolvedTheme === "dark" ? darkTheme : lightTheme}
      />
    </div>
  );
}

export type TariffEditorFieldProps<T extends FieldValues> = FormControlProps<T> &
  Omit<TariffEditorProps, "value" | "onChange" | "invalid"> & {
    label?: string;
    description?: string;
  };

/** `TariffEditor` bound to a react-hook-form field, with its label and error. */
export function TariffEditorField<T extends FieldValues>({
  name,
  control,
  rules,
  label,
  description,
  ...editor
}: TariffEditorFieldProps<T>) {
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
          <TariffEditor
            {...editor}
            value={typeof field.value === "string" ? field.value : ""}
            onChange={field.onChange}
            invalid={fieldState.invalid}
          />
        </FieldWrapper>
      )}
    />
  );
}
