import { ChevronLeftDoubleIcon, ChevronRightDoubleIcon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { formatNumber } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";

/** Folds the inspector away so the board takes the width; the rail brings it back. */
export function CollapseBriefButton({ onCollapse }: { onCollapse: () => void }) {
  const t = useT();
  return (
    <Button
      type="button"
      size="icon-xs"
      variant="ghost"
      aria-label={t("Hide the brief")}
      title={t("Hide the brief")}
      onClick={onCollapse}
      className="text-foreground-subtle"
    >
      <ChevronRightDoubleIcon className="size-3.5" />
    </Button>
  );
}

/**
 * What is left of the inspector while it is folded: a way back, its name, and how many moves
 * are waiting, so a folded brief still says when it has something to act on.
 */
export function BriefRail({ moves, onExpand }: { moves: number; onExpand: () => void }) {
  const t = useT();
  return (
    <div className="absolute inset-0 flex animate-in items-center gap-2 px-3.5 duration-300 fade-in-0 xl:flex-col xl:px-0 xl:pt-2.5">
      <Button
        type="button"
        size="icon-xs"
        variant="ghost"
        aria-label={t("Show the brief")}
        title={t("Show the brief")}
        onClick={onExpand}
        className="text-foreground-subtle"
      >
        <ChevronLeftDoubleIcon className="size-3.5 max-xl:rotate-90" />
      </Button>
      <button
        type="button"
        onClick={onExpand}
        className="flex cursor-pointer items-center gap-2 rounded-sm ui-focus-ring xl:flex-col xl:gap-2.5"
      >
        <span className="text-sm font-semibold xl:[writing-mode:vertical-rl]">{t("Brief")}</span>
        {moves > 0 ? (
          <span className="grid h-4.5 min-w-4.5 place-items-center rounded-badge bg-success-subtle px-1 font-mono text-2xs font-medium text-success-foreground tabular-nums">
            {formatNumber(moves)}
          </span>
        ) : null}
      </button>
    </div>
  );
}
