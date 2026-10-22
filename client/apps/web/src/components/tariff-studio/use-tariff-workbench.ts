import { applyTariffFix } from "@/components/formula-editor/tariff-lint";
import { revealRange } from "@/components/formula-editor/tariff-navigate";
import { checkTariffSource } from "@/components/formula-editor/tariff-service";
import type { EditorView } from "@codemirror/view";
import { useLatestCallback } from "@trenova/shared/hooks/use-latest-callback";
import type { RulesRuntime } from "@trenova/shared/rules/runtime";
import type {
  TariffCheckResult,
  TariffContext,
  TariffFileKind,
  TariffFix,
  TariffRange,
  TariffTestStatus,
} from "@trenova/shared/types/tariff";
import { useCallback, useMemo, useRef } from "react";
import { appendBlock } from "./source-edits";
import { useTariffTests, type TariffTestsState } from "./use-tariff-tests";

export type TariffWorkbench = {
  check: TariffCheckResult;
  errors: number;
  warnings: number;
  /** How many `test` blocks the outline lists. */
  testCount: number;
  tests: TariffTestsState;
  /** Each test's verdict from the last run, by name. */
  testStatus: ReadonlyMap<string, TariffTestStatus>;
};

/**
 * What a Studio knows about the file in its editor, for a tariff or a library alike: the
 * module's check against the organisation's context (no findings while the file is empty), the
 * counts every status line shows, and the file's tests, run in the browser once the source has
 * settled.
 */
export function useTariffWorkbench({
  rules,
  source,
  kind,
  context,
}: {
  rules: RulesRuntime;
  source: string;
  kind: Exclude<TariffFileKind, "condition">;
  context: TariffContext | null;
}): TariffWorkbench {
  // An empty file has nothing wrong with it yet: the Studio shows where to start instead of
  // the checker's "a file starts with its header".
  const check = useMemo(() => {
    const result = checkTariffSource(rules, source, kind, context);
    return source.trim() === "" ? { ...result, diagnostics: [] } : result;
  }, [rules, source, kind, context]);
  const errors = check.diagnostics.filter((finding) => finding.severity === "error").length;
  const warnings = check.diagnostics.filter((finding) => finding.severity === "warning").length;
  const testCount = check.outline?.items.filter((item) => item.kind === "test").length ?? 0;
  const tests = useTariffTests({ rules, source, context, errors, testCount });
  const testStatus = useMemo(
    () =>
      new Map(
        tests.status === "ran" ? tests.run.tests.map((test) => [test.name, test.status]) : [],
      ),
    [tests],
  );
  return { check, errors, warnings, testCount, tests, testStatus };
}

export type TariffEditorBinding = {
  /** Hand to `TariffEditor`'s `onView`. */
  setEditorView: (view: EditorView | null) => void;
  /** Selects a source range in the editor and brings the editor into view. */
  reveal: (range: TariffRange) => void;
  /** Applies a fix's edits through the editor, so they can be undone. */
  applyFix: (fix: TariffFix) => void;
  /** Appends a block to the source, through the editor so it can be undone. */
  appendToSource: (block: string) => void;
};

/**
 * The edits a Studio makes to its source on the person's behalf. Each goes through the editor
 * when it is mounted, so it lands in its undo history like typing would; before the editor has
 * loaded it is written to the form instead. The returned functions never change identity.
 */
export function useTariffEditorBinding({
  read,
  write,
}: {
  /** The source as the form holds it. */
  read: () => string;
  /** Writes a whole new source to the form, marking it dirty. */
  write: (next: string) => void;
}): TariffEditorBinding {
  const editorView = useRef<EditorView | null>(null);
  const readSource = useLatestCallback(read);
  const writeSource = useLatestCallback(write);

  const setEditorView = useCallback((view: EditorView | null) => {
    editorView.current = view;
  }, []);

  const reveal = useCallback((range: TariffRange) => {
    const view = editorView.current;
    if (!view) return;
    view.dom.scrollIntoView({ block: "nearest", behavior: "smooth" });
    revealRange(view, range);
  }, []);

  const applyFix = useCallback(
    (fix: TariffFix) => {
      const view = editorView.current;
      if (view) {
        applyTariffFix(view, fix);
        return;
      }
      let next = readSource();
      for (const change of [...fix.changes].sort((a, b) => b.range.from - a.range.from)) {
        next = next.slice(0, change.range.from) + change.insert + next.slice(change.range.to);
      }
      writeSource(next);
    },
    [readSource, writeSource],
  );

  const appendToSource = useCallback(
    (block: string) => {
      const view = editorView.current;
      const current = view ? view.state.doc.toString() : readSource();
      const edit = appendBlock(current, block);
      if (view) {
        view.dispatch({
          changes: edit,
          selection: { anchor: edit.from + edit.insert.length },
          userEvent: "input",
          scrollIntoView: true,
        });
        view.focus();
        return;
      }
      writeSource(current.slice(0, edit.from) + edit.insert);
    },
    [readSource, writeSource],
  );

  return useMemo(
    () => ({ setEditorView, reveal, applyFix, appendToSource }),
    [setEditorView, reveal, applyFix, appendToSource],
  );
}
