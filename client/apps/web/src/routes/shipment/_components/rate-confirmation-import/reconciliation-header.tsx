import { useT } from "@trenova/shared/i18n/use-t";
import { Button } from "@trenova/shared/components/ui/button";
import { ZapIcon } from "lucide-react";
import type { ReconciliationCounts } from "./types";

type ReconciliationHeaderProps = {
  overallConfidence: number;
  counts: ReconciliationCounts;
  issueCount: number;
  onAcceptAllConfident: () => void;
  onToggleFilter: () => void;
  showIssuesOnly: boolean;
};

export function ReconciliationHeader({
  overallConfidence,
  counts,
  issueCount,
  onAcceptAllConfident,
  onToggleFilter,
  showIssuesOnly,
}: ReconciliationHeaderProps) {
  const t = useT();

  const pct = Math.round(overallConfidence * 100);

  return (
    <div className="shrink-0 border-b px-4 py-2.5">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="flex flex-wrap items-center gap-3 text-xs">
          <span className="font-medium tabular-nums">{t("{0}% confidence", pct)}</span>
          <span className="text-muted-foreground/30">|</span>
          <div className="flex items-center gap-1.5">
            <div className="size-1.5 rounded-full bg-emerald-500" />
            <span className="text-muted-foreground">
              {t("{0} accepted", counts.accepted + counts.edited)}
            </span>
          </div>
          {counts.needsReview > 0 && (
            <div className="flex items-center gap-1.5">
              <div className="size-1.5 rounded-full bg-amber-500" />
              <span className="text-muted-foreground">{t("{0} review", counts.needsReview)}</span>
            </div>
          )}
          {counts.missing > 0 && (
            <div className="flex items-center gap-1.5">
              <div className="bg-muted-foreground/30 size-1.5 rounded-full" />
              <span className="text-muted-foreground">{t("{0} missing", counts.missing)}</span>
            </div>
          )}
          {counts.conflicting > 0 && (
            <div className="flex items-center gap-1.5">
              <div className="size-1.5 rounded-full bg-amber-500" />
              <span className="text-muted-foreground">{t("{0} conflicting", counts.conflicting)}</span>
            </div>
          )}
        </div>

        <div className="flex items-center gap-1.5">
          {issueCount > 0 && (
            <Button variant="ghost" size="sm" onClick={onToggleFilter} className="h-7 text-xs">
              {showIssuesOnly ? t("Show all") : t("Issues only")}
            </Button>
          )}
          {counts.needsReview > 0 && (
            <Button
              variant="outline"
              size="sm"
              onClick={onAcceptAllConfident}
              className="h-7 text-xs"
            >
              <ZapIcon className="size-3" />
              {t("Accept confident")}
            </Button>
          )}
        </div>
      </div>
    </div>
  );
}
