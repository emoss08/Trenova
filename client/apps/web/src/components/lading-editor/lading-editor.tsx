import { useResolvedTheme } from "@/hooks/use-resolved-theme";
import { lintGutter } from "@codemirror/lint";
import type { Extension } from "@codemirror/state";
import { EditorView } from "@codemirror/view";
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
import { useEffect, useMemo, useRef } from "react";
import { darkTheme, languageServiceTheme, lightTheme } from "../formula-editor/editor-theme";
import { ladingAssist, revealLadingRange } from "./lading-assist";
import { ladingHighlight } from "./lading-highlight";
import { ladingLinter } from "./lading-lint";
import { ladingTools, refreshLadingInlays, setLadingCoverage } from "./lading-tools";
import type { LadingLanguageService } from "./lading-service";

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
 * folding from the check's outline, and completion, hover, go to definition and formatting from
 * the editor operations, so the browser holds no grammar of its own. It suspends until the
 * module has loaded, so it sits under a Suspense boundary with the studio's skeleton.
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
    return [
      ladingHighlight(getEdi),
      ladingLinter(service),
      ladingAssist(service, reportFormat),
      ladingTools(service),
      lintGutter(),
      EditorView.lineWrapping,
      languageServiceTheme,
    ];
  }, [getEdi, getProject, getContext, getPath, getValues, openLocation, applyEdits, reportFormat]);

  useEffect(() => {
    viewRef.current?.dispatch({ effects: refreshLadingInlays.of(null) });
  }, [values, project]);

  useEffect(() => {
    viewRef.current?.dispatch({ effects: setLadingCoverage.of(uncovered) });
  }, [uncovered]);

  useEffect(() => {
    if (reveal && viewRef.current) revealLadingRange(viewRef.current, reveal);
  }, [reveal]);

  const handleRef = (instance: ReactCodeMirrorRef | null) => {
    viewRef.current = instance?.view ?? null;
  };

  return (
    <div
      data-lading-path={path}
      className={cn("isolate h-full min-h-0 overflow-hidden", className)}
    >
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
          allowMultipleSelections: false,
          indentOnInput: true,
          bracketMatching: true,
          closeBrackets: true,
          autocompletion: false,
          highlightActiveLine: true,
          highlightActiveLineGutter: true,
          highlightSelectionMatches: true,
          searchKeymap: true,
          syntaxHighlighting: false,
        }}
        theme={resolvedTheme === "dark" ? darkTheme : lightTheme}
      />
    </div>
  );
}
