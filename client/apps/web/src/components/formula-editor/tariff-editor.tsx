import { FieldWrapper } from "@/components/fields/field-components";
import { useResolvedTheme } from "@/hooks/use-resolved-theme";
import { useEditorPreferenceStore } from "@/stores/editor-preference-store";
import { lintGutter } from "@codemirror/lint";
import { Compartment, Prec, type Extension } from "@codemirror/state";
import { EditorView, keymap, type Command } from "@codemirror/view";
import { useLatestCallback } from "@trenova/shared/hooks/use-latest-callback";
import { useRules } from "@trenova/shared/hooks/use-rules";
import { cn } from "@trenova/shared/lib/utils";
import type { TariffFormatResult } from "@trenova/shared/rules/tariff";
import type { FormControlProps } from "@trenova/shared/types/fields";
import {
  EMPTY_TARIFF_CONTEXT,
  type TariffCheckResult,
  type TariffContext,
  type TariffFileKind,
  type TariffRange,
  type TariffTraceEvent,
} from "@trenova/shared/types/tariff";
import CodeMirror, { type ReactCodeMirrorRef } from "@uiw/react-codemirror";
import { useEffect, useMemo, useRef, useState } from "react";
import { Controller, type FieldValues } from "react-hook-form";
import { useActiveEditorRegistration } from "./active-editor";
import { darkTheme, languageServiceTheme, lightTheme } from "./editor-theme";
import { highlightRangeExtension, setHighlightRange } from "./highlight-range";
import { tariffEditorCommands, tariffLanguageData } from "./tariff-commands";
import { tariffAutocomplete } from "./tariff-complete";
import { tariffFolding } from "./tariff-fold";
import { FORMAT_KEY, formatTariffDocument } from "./tariff-format";
import { tariffHighlight } from "./tariff-highlight";
import { tariffHoverCard } from "./tariff-hover";
import { refreshTariffInlays, tariffInlayHints } from "./tariff-inlays";
import { tariffLinter } from "./tariff-lint";
import { tariffNavigation } from "./tariff-navigate";
import {
  checkTariffSource,
  type TariffDefinitionTarget,
  type TariffLanguageService,
} from "./tariff-service";
import { tariffSignatureHelp } from "./tariff-signature";
import { useTariffVim } from "./use-tariff-vim";

const NO_TRACE: readonly TariffTraceEvent[] = [];

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
  /** The last preview's trace: hover cards and inlay hints show what each binding came to. */
  trace?: readonly TariffTraceEvent[] | null;
  /** Opens a declaration that lives in an imported library (go to definition). */
  onOpenDefinition?: (target: TariffDefinitionTarget) => void;
  /** Told what formatting from the keyboard did, so the page can say why it was refused. */
  onFormat?: (result: TariffFormatResult) => void;
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
  placeholder?: string;
  minHeight?: string;
  maxHeight?: string;
  className?: string;
};

/**
 * The Tariff source editor. Everything it shows about the text comes from the rules module:
 * highlighting from `tariff.tokens`, findings and their fixes from `tariff.check`, folding from
 * the check's outline, and completion, hover, signature help, go to definition, rename and inlay
 * hints from the editor operations, so the browser holds no grammar of its own. It suspends
 * until the module has loaded, so it sits under a Suspense boundary with its screen's skeleton.
 * A full editor that can be edited takes Vim keys when the person has turned them on.
 */
export function TariffEditor({
  kind,
  value,
  onChange,
  context,
  onCheck,
  trace = null,
  onOpenDefinition,
  onFormat,
  highlight = null,
  onView,
  variant = "default",
  embedded = false,
  primary = false,
  readOnly = false,
  invalid = false,
  ariaLabel,
  placeholder,
  minHeight,
  maxHeight,
  className,
}: TariffEditorProps) {
  const rules = useRules();
  const resolvedTheme = useResolvedTheme();
  const { setActive, setPrimary } = useActiveEditorRegistration();
  const viewRef = useRef<EditorView | null>(null);
  const mini = variant === "mini";
  const knowledge = context ?? EMPTY_TARIFF_CONTEXT;

  const getRules = useLatestCallback(() => rules);
  const getKind = useLatestCallback(() => kind);
  const getContext = useLatestCallback(() => knowledge);
  const getTrace = useLatestCallback(() => trace ?? NO_TRACE);
  const openDefinition = useLatestCallback((target: TariffDefinitionTarget) =>
    onOpenDefinition?.(target),
  );
  const reportFormat = useLatestCallback((result: TariffFormatResult) => onFormat?.(result));

  const vimKeys = useEditorPreferenceStore((state) => state.vimKeys) && !mini && !readOnly;
  const vimExtension = useTariffVim(vimKeys);
  const [vimSlot] = useState(() => new Compartment());
  const applyVim = useLatestCallback((view: EditorView) =>
    view.dispatch({ effects: vimSlot.reconfigure(vimExtension ?? []) }),
  );

  const extensions = useMemo((): Extension[] => {
    const service: TariffLanguageService = {
      rules: getRules,
      kind: getKind,
      context: getContext,
      trace: getTrace,
      openDefinition,
    };
    const format: Command = (view) => {
      if (view.state.readOnly) return false;
      reportFormat(formatTariffDocument(view, getRules()));
      return true;
    };
    const list: Extension[] = [
      vimSlot.of([]),
      tariffEditorCommands.of({ format }),
      tariffLanguageData,
      tariffHighlight(getRules),
      tariffLinter(service),
      tariffFolding(service),
      tariffHoverCard(service),
      tariffSignatureHelp(service),
      tariffInlayHints(service),
      tariffNavigation(service),
      tariffAutocomplete(service),
      EditorView.lineWrapping,
      highlightRangeExtension(),
      languageServiceTheme,
      Prec.high(keymap.of([{ key: FORMAT_KEY, run: format }])),
    ];
    if (!mini) list.push(lintGutter());
    return list;
  }, [getRules, getKind, getContext, getTrace, openDefinition, reportFormat, mini, vimSlot]);

  const check = useMemo(
    () => (onCheck ? checkTariffSource(rules, value, kind, context) : null),
    [onCheck, rules, value, kind, context],
  );
  const reportCheck = useLatestCallback((result: TariffCheckResult) => onCheck?.(result));
  useEffect(() => {
    if (check) reportCheck(check);
  }, [check, reportCheck]);

  useEffect(() => {
    viewRef.current?.dispatch({ effects: setHighlightRange.of(highlight) });
  }, [highlight]);

  useEffect(() => {
    viewRef.current?.dispatch({ effects: refreshTariffInlays.of(null) });
  }, [trace]);

  useEffect(() => {
    if (viewRef.current) applyVim(viewRef.current);
  }, [vimExtension, applyVim]);

  const handleRef = (instance: ReactCodeMirrorRef | null) => {
    const view = instance?.view ?? null;
    if (view === viewRef.current) return;
    viewRef.current = view;
    if (view && vimExtension) applyVim(view);
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
        editable={!readOnly}
        placeholder={placeholder}
        aria-label={ariaLabel}
        aria-invalid={invalid ? "true" : "false"}
        minHeight={minHeight ?? (mini ? "60px" : undefined)}
        maxHeight={maxHeight}
        extensions={extensions}
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
          syntaxHighlighting: false,
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
