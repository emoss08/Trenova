import { Button } from "@/components/ui/button";
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
  const pct = Math.round(overallConfidence * 100);

  return (
    <div className="shrink-0 border-b px-4 py-2.5">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="flex flex-wrap items-center gap-3 text-xs">
          <span className="font-medium tabular-nums">{pct}% confidence</span>
          <span className="text-muted-foreground/30">|</span>
          <div className="flex items-center gap-1.5">
            <div className="size-1.5 rounded-full bg-emerald-500" />
            <span className="text-muted-foreground">{counts.accepted + counts.edited} accepted</span>
          </div>
          {counts.needsReview > 0 && (
            <div className="flex items-center gap-1.5">
              <div className="size-1.5 rounded-full bg-amber-500" />
              <span className="text-muted-foreground">{counts.needsReview} review</span>
            </div>
          )}
          {counts.missing > 0 && (
            <div className="flex items-center gap-1.5">
              <div className="size-1.5 rounded-full bg-muted-foreground/30" />
              <span className="text-muted-foreground">{counts.missing} missing</span>
            </div>
          )}
          {counts.conflicting > 0 && (
            <div className="flex items-center gap-1.5">
              <div className="size-1.5 rounded-full bg-amber-500" />
              <span className="text-muted-foreground">{counts.conflicting} conflicting</span>
            </div>
          )}
        </div>

        <div className="flex items-center gap-1.5">
          {issueCount > 0 && (
            <Button variant="ghost" size="sm" onClick={onToggleFilter} className="h-7 text-xs">
              {showIssuesOnly ? "Show all" : "Issues only"}
            </Button>
          )}
          {counts.needsReview > 0 && (
            <Button variant="outline" size="sm" onClick={onAcceptAllConfident} className="h-7 text-xs">
              <ZapIcon className="size-3" />
              Accept confident
            </Button>
          )}
        </div>
      </div>
    </div>
  );
}
