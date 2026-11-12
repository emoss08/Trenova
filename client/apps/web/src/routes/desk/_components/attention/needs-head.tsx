import { isTypingTarget, isWithinDialog } from "@/lib/dom";
import { useLatestCallback } from "@trenova/shared/hooks/use-latest-callback";
import { useT } from "@trenova/shared/i18n/use-t";
import { useEffect } from "react";
import { NEEDS_LENSES, lensLabel, type NeedsLens } from "./lens";

/**
 * The head of Needs you's list, the same under every lens: the page's name,
 * how much of today's work is behind the person, and the lenses with what
 * waits in each. The keys 1 to 3 pick a lens wherever the page is.
 */
export function NeedsHead({
  lens,
  lenses,
  onLens,
  counts,
  progressLabel,
  done,
  total,
  keysEnabled = true,
  onClearFilters,
}: {
  lens: NeedsLens;
  /** The lenses the person may open, in page order. */
  lenses: readonly NeedsLens[];
  onLens: (lens: NeedsLens) => void;
  counts: Readonly<Record<NeedsLens, number>>;
  progressLabel: string;
  done: number;
  total: number;
  /** Off while a dialog is open over the page. */
  keysEnabled?: boolean;
  /** Set while the list is filtered; clears the filters. */
  onClearFilters?: () => void;
}) {
  const t = useT();

  const onKey = useLatestCallback((event: KeyboardEvent) => {
    if (event.defaultPrevented || event.metaKey || event.ctrlKey || event.altKey) {
      return;
    }
    if (isTypingTarget(event.target) || isWithinDialog(event.target)) {
      return;
    }
    const picked = NEEDS_LENSES[["1", "2", "3"].indexOf(event.key)];
    if (picked && lenses.includes(picked)) {
      event.preventDefault();
      onLens(picked);
    }
  });
  useEffect(() => {
    if (!keysEnabled) {
      return;
    }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [keysEnabled, onKey]);

  return (
    <>
      <div>
        <div className="dk-w-nq-h">
          <b>{t("Needs you")}</b>
          <span className="dk-w-nq-c">{progressLabel}</span>
        </div>
        <div
          className="dk-w-nq-bar"
          role="progressbar"
          aria-label={progressLabel}
          aria-valuemin={0}
          aria-valuemax={total}
          aria-valuenow={Math.min(done, total)}
        >
          <i style={{ transform: `scaleX(${total === 0 ? 0 : Math.min(1, done / total)})` }} />
        </div>
      </div>
      <div className="dk-w-lens" role="tablist" aria-label={t("Lens")}>
        {lenses.map((value) => (
          <button
            key={value}
            type="button"
            role="tab"
            aria-selected={lens === value}
            className="dk-w-chip ui-focus-ring"
            onClick={() => onLens(value)}
            title={t("Press {0}", NEEDS_LENSES.indexOf(value) + 1)}
          >
            {lensLabel(t, value)}
            <span className="dk-w-chip-n">{counts[value]}</span>
          </button>
        ))}
        {onClearFilters && (
          <button
            type="button"
            className="dk-w-chip ui-focus-ring"
            aria-pressed
            onClick={onClearFilters}
          >
            {t("Filtered")}
            <span className="dk-w-chip-n">×</span>
          </button>
        )}
      </div>
    </>
  );
}
