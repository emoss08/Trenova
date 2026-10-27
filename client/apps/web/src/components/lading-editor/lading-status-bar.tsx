import { forEachDiagnostic } from "@codemirror/lint";
import type { EditorState } from "@codemirror/state";
import { AlignLeftIcon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { Kbd } from "@trenova/shared/components/ui/kbd";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatAltShortcut } from "@trenova/shared/lib/shortcuts";
import { cn } from "@trenova/shared/lib/utils";
import { EditorKeys, VimToggle } from "../formula-editor/editor-status-keys";

/** Where the cursor is and what the linter found in the file, as the status bar shows them. */
export type LadingEditorStatus = {
  line: number;
  column: number;
  errors: number;
  warnings: number;
};

/** The cursor's place and the linter's counts, for the status bar. */
export function ladingEditorStatus(state: EditorState): LadingEditorStatus {
  const head = state.selection.main.head;
  const line = state.doc.lineAt(head);
  let errors = 0;
  let warnings = 0;
  forEachDiagnostic(state, (diagnostic) => {
    if (diagnostic.severity === "error") errors += 1;
    else if (diagnostic.severity === "warning") warnings += 1;
  });
  return { line: line.number, column: head - line.from + 1, errors, warnings };
}

export function sameLadingStatus(a: LadingEditorStatus, b: LadingEditorStatus): boolean {
  return (
    a.line === b.line && a.column === b.column && a.errors === b.errors && a.warnings === b.warnings
  );
}

export const INITIAL_LADING_STATUS: LadingEditorStatus = {
  line: 1,
  column: 1,
  errors: 0,
  warnings: 0,
};

/**
 * The bar under the Lading editor, as VS Code draws one: the file's problems (which open the
 * problems panel), the cursor's line and column, the language, Format, Vim keys and the editor's
 * keys.
 */
export function LadingStatusBar({
  status,
  onProblems,
  onFormat,
  formatDisabled,
}: {
  status: LadingEditorStatus;
  onProblems: () => void;
  onFormat: () => void;
  formatDisabled: boolean;
}) {
  const t = useT();
  const { errors, warnings } = status;
  return (
    <div className="bg-sunken text-foreground-muted flex flex-wrap items-center gap-x-3 gap-y-1 border-t px-3 py-1 text-xs">
      <Button
        type="button"
        variant="ghost"
        size="xs"
        onClick={onProblems}
        aria-label={t("Open the problems panel")}
        className="gap-2"
      >
        <span className={cn("flex items-center gap-1.5", errors > 0 && "text-danger")}>
          <span
            aria-hidden
            className={cn(
              "size-1.5 rounded-full",
              errors > 0 ? "bg-danger" : "bg-foreground-subtle",
            )}
          />
          {t("{0, plural, one {# error} other {# errors}}", errors)}
        </span>
        <span className={cn("flex items-center gap-1.5", warnings > 0 && "text-warning")}>
          <span
            aria-hidden
            className={cn(
              "size-1.5 rounded-full",
              warnings > 0 ? "bg-warning" : "bg-foreground-subtle",
            )}
          />
          {t("{0, plural, one {# warning} other {# warnings}}", warnings)}
        </span>
      </Button>
      <span className="flex-1" />
      <span className="tabular-nums">{t("Ln {0}, Col {1}", status.line, status.column)}</span>
      <span>{t("Lading")}</span>
      <Tooltip>
        <TooltipTrigger
          render={
            <Button
              type="button"
              variant="ghost"
              size="icon-xs"
              aria-label={t("Format the file")}
              disabled={formatDisabled}
              onClick={onFormat}
              className="text-foreground-muted"
            />
          }
        >
          <AlignLeftIcon className="size-3.5" />
        </TooltipTrigger>
        <TooltipContent>
          {t("Format")} <Kbd className="ml-1">{formatAltShortcut("⇧F")}</Kbd>
        </TooltipContent>
      </Tooltip>
      <VimToggle />
      <EditorKeys />
    </div>
  );
}
