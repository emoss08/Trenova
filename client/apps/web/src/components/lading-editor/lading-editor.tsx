import { useResolvedTheme } from "@/hooks/use-resolved-theme";
import { useEditorPreferenceStore } from "@/stores/editor-preference-store";
import { forceLinting, lintGutter } from "@codemirror/lint";
import { Compartment, Prec, type EditorState, type Extension } from "@codemirror/state";
import { EditorView, keymap, tooltips, type Command } from "@codemirror/view";
import type {
  LadingContext,
  LadingEdit,
  LadingLocation,
  LadingRange,
  LadingProjectInput,
  LadingRunValue,
} from "@trenova/shared/edi/lading";
import { useEdi } from "@trenova/shared/hooks/use-edi";
import { useLatestCallback } from "@trenova/shared/hooks/use-latest-callback";
import { cn } from "@trenova/shared/lib/utils";
import CodeMirror, { type ReactCodeMirrorRef } from "@uiw/react-codemirror";
import { useEffect, useMemo, useRef, useState } from "react";
import { darkTheme, languageServiceTheme, lightTheme } from "../formula-editor/editor-theme";
import {
  requestEditorSave,
  tariffEditorCommands,
  tariffLanguageData,
} from "../formula-editor/tariff-commands";
import { showLineDiagnostics } from "../formula-editor/tariff-lint";
import { useTariffVim } from "../formula-editor/use-tariff-vim";
import {
  formatLadingDocument,
  goToLadingDefinition,
  ladingAssist,
  revealLadingRange,
} from "./lading-assist";
import { ladingAutocomplete } from "./lading-complete";
import { ladingHighlight } from "./lading-highlight";
import { ladingHoverCard, showLadingHoverAtCursor } from "./lading-hover";
import { ladingLinter } from "./lading-lint";
import {
  findLadingReferences,
  ladingTools,
  refreshLadingInlays,
  setLadingCoverage,
  showLadingActions,
  startLadingRename,
} from "./lading-tools";
import type { LadingLanguageService } from "./lading-service";
import {
  INITIAL_LADING_STATUS,
  LadingStatusBar,
  ladingEditorStatus,
  sameLadingStatus,
  type LadingEditorStatus,
} from "./lading-status-bar";

const NO_VALUES: readonly LadingRunValue[] = [];
const NO_RANGES: readonly LadingRange[] = [];

export type LadingEditorProps = {
  /** The file's path in the project. */
  path: string;
  value: string;
  onChange: (source: string) => void;
  /** The whole project as the studio holds it, this file's current text included. */
  project: LadingProjectInput;
  context: LadingContext;
  /** The last run's values: hover shows what each field came to. */
  values?: readonly LadingRunValue[];
  /** Opens a location in another file of the project (go to definition, references). */
  onOpenFile: (location: LadingLocation) => void;
  /** Applies a rename's or a refactoring's edits to the project's other files. */
  onEditFiles?: (edits: readonly LadingEdit[]) => void;
  /** Told why formatting was refused, or null when it formatted. */
  onFormat?: (reason: string | null) => void;
  /** Shows every finding in the project: Mod-Shift-M, Vim's `<Space>lD` and the status bar's counts. */
  onShowProblems?: () => void;
  /** A range to move to once, such as a finding picked in the problems list. */
  reveal?: { from: number; to: number } | null;
  /** What the last test run did not reach in this file; those lines are marked. */
  uncovered?: readonly LadingRange[];
  readOnly?: boolean;
  className?: string;
};

/**
 * One file of a Lading project. Everything it shows about the text comes from the EDI module:
 * colour from `lading.tokens`, findings and their fixes from `lading.check` of the whole project,
 * folding from the check's outline, and completion, hover, signature help, go to definition,
 * references, rename, fixes and refactorings and formatting from the editor operations, so the
 * browser holds no grammar of its own. Its keys are VS Code's (F12, Shift-F12, F2, F8, Mod-.,
 * Shift-Alt-F, Mod-S saves the draft), and Vim's with the Tariff editor's mappings when the
 * person has turned Vim keys on. It suspends until the module has loaded, so it sits under a
 * Suspense boundary with the studio's skeleton.
 */
export function LadingEditor({
  path,
  value,
  onChange,
  project,
  context,
  values = NO_VALUES,
  onOpenFile,
  onEditFiles,
  onFormat,
  onShowProblems,
  reveal = null,
  uncovered = NO_RANGES,
  readOnly = false,
  className,
}: LadingEditorProps) {
  const edi = useEdi();
  const resolvedTheme = useResolvedTheme();
  const viewRef = useRef<EditorView | null>(null);

  const getEdi = useLatestCallback(() => edi);
  const getProject = useLatestCallback(() => project);
  const getContext = useLatestCallback(() => context);
  const getPath = useLatestCallback(() => path);
  const getValues = useLatestCallback(() => values);
  const openLocation = useLatestCallback((location: LadingLocation) => {
    if (location.path === path && viewRef.current) {
      revealLadingRange(viewRef.current, location.range);
      return;
    }
    onOpenFile(location);
  });
  const reportFormat = useLatestCallback((reason: string | null) => onFormat?.(reason));
  const showProblems = useLatestCallback((): boolean => {
    if (!onShowProblems) return false;
    onShowProblems();
    return true;
  });

  const [status, setStatus] = useState<LadingEditorStatus>(INITIAL_LADING_STATUS);
  const reportStatus = useLatestCallback((state: EditorState) => {
    const next = ladingEditorStatus(state);
    setStatus((current) => (sameLadingStatus(current, next) ? current : next));
  });

  const vimKeys = useEditorPreferenceStore((state) => state.vimKeys) && !readOnly;
  const vimExtension = useTariffVim(vimKeys);
  const [vimSlot] = useState(() => new Compartment());
  const applyVim = useLatestCallback((view: EditorView) =>
    view.dispatch({ effects: vimSlot.reconfigure(vimExtension ?? []) }),
  );
  const applyEdits = useLatestCallback((edits: readonly LadingEdit[]) => {
    const view = viewRef.current;
    const here = edits.filter((edit) => edit.path === path);
    const elsewhere = edits.filter((edit) => edit.path !== path);
    if (view && here.length > 0) {
      const length = view.state.doc.length;
      view.dispatch({
        changes: here.map((edit) => ({
          from: Math.min(edit.range.from, length),
          to: Math.min(edit.range.to, length),
          insert: edit.insert,
        })),
        userEvent: "refactor",
        scrollIntoView: true,
      });
    }
    if (elsewhere.length > 0) onEditFiles?.(elsewhere);
  });

  const extensions = useMemo((): Extension[] => {
    const service: LadingLanguageService = {
      edi: getEdi,
      project: getProject,
      context: getContext,
      path: getPath,
      values: getValues,
      openLocation,
      applyEdits,
    };
    const format: Command = (view) => {
      if (view.state.readOnly) return false;
      reportFormat(formatLadingDocument(view, service).reason);
      return true;
    };
    return [
      vimSlot.of([]),
      tariffEditorCommands.of({
        format,
        definition: goToLadingDefinition,
        hover: showLadingHoverAtCursor,
        lineProblems: showLineDiagnostics,
        rename: startLadingRename,
        references: findLadingReferences,
        actions: showLadingActions,
        problems: showProblems,
      }),
      tariffLanguageData,
      ladingHighlight(getEdi),
      ladingLinter(service),
      ladingAssist(service),
      ladingHoverCard(service),
      ladingAutocomplete(service),
      ladingTools(service),
      lintGutter(),
      EditorView.lineWrapping,
      languageServiceTheme,
      Prec.high(
        keymap.of([
          { key: "Mod-s", run: requestEditorSave, preventDefault: true },
          { key: "Mod-Shift-m", run: showProblems, preventDefault: true },
        ]),
      ),
      // Completion, hover and the lint card float over the studio's panels below the editor,
      // which the editor's own stacking context (`isolate`) would put them under.
      tooltips({ parent: document.body }),
      EditorView.updateListener.of((update) => {
        if (update.transactions.length > 0) reportStatus(update.state);
      }),
    ];
  }, [
    getEdi,
    getProject,
    getContext,
    getPath,
    getValues,
    openLocation,
    applyEdits,
    reportFormat,
    reportStatus,
    showProblems,
    vimSlot,
  ]);

  useEffect(() => {
    viewRef.current?.dispatch({ effects: refreshLadingInlays.of(null) });
  }, [values, project]);

  useEffect(() => {
    viewRef.current?.dispatch({ effects: setLadingCoverage.of(uncovered) });
  }, [uncovered]);

  useEffect(() => {
    if (viewRef.current) forceLinting(viewRef.current);
  }, [context]);

  useEffect(() => {
    if (reveal && viewRef.current) revealLadingRange(viewRef.current, reveal);
  }, [reveal]);

  useEffect(() => {
    if (viewRef.current) applyVim(viewRef.current);
  }, [vimExtension, applyVim]);

  const handleRef = (instance: ReactCodeMirrorRef | null) => {
    const view = instance?.view ?? null;
    if (view === viewRef.current) return;
    viewRef.current = view;
    if (view && vimExtension) applyVim(view);
    if (view) reportStatus(view.state);
  };

  const formatNow = () => {
    const view = viewRef.current;
    if (!view) return;
    view.state.facet(tariffEditorCommands)?.format(view);
    view.focus();
  };

  return (
    <div
      data-lading-path={path}
      className={cn(
        // CodeMirror stacks its gutters, panels (Vim's mode line) and cards at z-index 200 to
        // 500; isolating the editor keeps those inside it, under the app's tooltips and menus.
        "isolate flex h-full min-h-0 flex-col overflow-hidden",
        className,
      )}
    >
      <div className="min-h-0 flex-1">
        <CodeMirror
          ref={handleRef}
          value={value}
          onChange={onChange}
          readOnly={readOnly}
          editable={!readOnly}
          aria-label={path}
          height="100%"
          className="h-full"
          extensions={extensions}
          basicSetup={{
            lineNumbers: true,
            foldGutter: true,
            dropCursor: true,
            allowMultipleSelections: true,
            indentOnInput: true,
            bracketMatching: true,
            closeBrackets: true,
            autocompletion: false,
            highlightActiveLine: true,
            highlightActiveLineGutter: true,
            highlightSelectionMatches: true,
            rectangularSelection: true,
            crosshairCursor: true,
            searchKeymap: true,
            syntaxHighlighting: false,
          }}
          theme={resolvedTheme === "dark" ? darkTheme : lightTheme}
        />
      </div>
      <LadingStatusBar
        status={status}
        onProblems={showProblems}
        onFormat={formatNow}
        formatDisabled={readOnly}
      />
    </div>
  );
}
