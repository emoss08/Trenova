import { AlignLeftIcon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { Kbd } from "@trenova/shared/components/ui/kbd";
import { toast } from "@trenova/shared/components/ui/toast";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { translateMessage } from "@trenova/shared/i18n/message";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatAltShortcut } from "@trenova/shared/lib/shortcuts";
import { cn } from "@trenova/shared/lib/utils";
import type { TariffFormatResult } from "@trenova/shared/rules/tariff";
import type { EditorView } from "@codemirror/view";
import type { TariffFileKind } from "@trenova/shared/types/tariff";
import { EditorKeys, VimToggle } from "../formula-editor/editor-status-keys";
import { EditorActionsMenu } from "./editor-actions-menu";
import type { TariffTestsState } from "./use-tariff-tests";

export type SourceTone = "ok" | "bad" | "warn" | "idle";

/**
 * How tall a Studio's source editor grows before it scrolls inside itself: most of the screen,
 * so a long file never pushes the status bar, the problems and the sections below out of reach.
 */
export const SOURCE_EDITOR_MAX_HEIGHT = "min(70dvh, 56rem)";

/** Says why a format was refused; a file with syntax errors is never reformatted. */
export function reportFormat(result: TariffFormatResult, t: TranslateFn) {
  if (result.formatted !== null) return;
  const first = result.diagnostics[0];
  toast.warning(t("Fix the syntax first, then format"), {
    description: first ? translateMessage(first.message, t) : undefined,
  });
}

/** The status line's verdict on a source: empty, its errors, its warnings, or clean. */
export function sourceStatus(
  { empty, errors, warnings }: { empty: boolean; errors: number; warnings: number },
  t: TranslateFn,
): { tone: SourceTone; label: string } {
  if (empty) return { tone: "idle", label: t("Empty") };
  if (errors > 0) {
    return { tone: "bad", label: t("{0, plural, one {# error} other {# errors}}", errors) };
  }
  if (warnings > 0) {
    return {
      tone: "warn",
      label: t("Checks clean · {0, plural, one {# warning} other {# warnings}}", warnings),
    };
  }
  return { tone: "ok", label: t("Checks clean") };
}

/**
 * The bar under a Studio's editor: the check's verdict, what the file holds, failing tests, the
 * format button, the editor's menu (definition, references, rename, Promote to input), Vim keys
 * and the editor's keys.
 */
export function SourceStatusBar({
  status,
  summary,
  tests,
  onFormat,
  formatDisabled,
  editor,
}: {
  status: { tone: SourceTone; label: string };
  /** What the file holds, in one line. */
  summary: string;
  tests: TariffTestsState;
  onFormat: () => void;
  formatDisabled: boolean;
  /** The editor the menu's commands act on, and the kind of file it holds. */
  editor?: { view: () => EditorView | null; kind: TariffFileKind };
}) {
  const t = useT();
  return (
    <div className="bg-sunken text-foreground-muted flex flex-wrap items-center gap-x-3 gap-y-1 border-t px-3 py-1 text-xs">
      <span
        className={cn(
          "flex items-center gap-1.5 font-medium",
          status.tone === "ok" && "text-success",
          status.tone === "warn" && "text-warning",
          status.tone === "bad" && "text-danger",
        )}
      >
        <span
          aria-hidden
          className={cn(
            "size-1.5 rounded-full",
            status.tone === "ok" && "bg-success",
            status.tone === "bad" && "bg-danger",
            status.tone === "warn" && "bg-warning",
            status.tone === "idle" && "bg-foreground-subtle",
          )}
        />
        {status.label}
      </span>
      <span>{summary}</span>
      {tests.status === "ran" && tests.failed > 0 && (
        <span className="text-danger">
          {t("{0, plural, one {# test fails} other {# tests fail}}", tests.failed)}
        </span>
      )}
      <span className="flex-1" />
      <Tooltip>
        <TooltipTrigger
          render={
            <Button
              type="button"
              variant="ghost"
              size="icon-xs"
              aria-label={t("Format the source")}
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
      {editor && <EditorActionsMenu view={editor.view} kind={editor.kind} />}
      <VimToggle />
      <EditorKeys />
    </div>
  );
}
