import { useT } from "@trenova/shared/i18n/use-t";
import { FindingList } from "@/components/carrier-intelligence/finding-list";
import { ReviewStateBadge } from "@/components/carrier-intelligence/review-state-badge";
import { RiskLevelBadge } from "@/components/carrier-intelligence/risk-level-badge";
import { useCarrierIntelLabels } from "@/components/carrier-intelligence/use-carrier-intel-labels";
import { carrierIntelProviderLabel } from "@/lib/carrier-intelligence";
import {
  CARRIER_INTEL_HISTORY_KEY,
  fetchCarrierIntelSnapshotHistory,
  type CarrierIntelSnapshotSummary,
} from "@/lib/graphql/carrier-intelligence";
import { useQuery } from "@tanstack/react-query";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@trenova/shared/components/ui/collapsible";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatUnixDateTimeMedium } from "@trenova/shared/lib/date";
import { ChevronRightIcon, FileJsonIcon } from "lucide-react";
import { useState } from "react";
import { RawPayloadDialog } from "./raw-payload-dialog";

export type IntelligenceHistoryProps = {
  carrierId: string;
  currentSnapshotId: string | null;
  ruleLabels: Readonly<Record<string, string>>;
  canExport: boolean;
};

const HISTORY_LIMIT = 25;

function SnapshotRow({
  snapshot,
  current,
  ruleLabels,
  canExport,
  onViewRaw,
}: {
  snapshot: CarrierIntelSnapshotSummary;
  current: boolean;
  ruleLabels: Readonly<Record<string, string>>;
  canExport: boolean;
  onViewRaw: (snapshot: CarrierIntelSnapshotSummary) => void;
}) {
  const t = useT();
  const labels = useCarrierIntelLabels();
  const [open, setOpen] = useState(false);

  return (
    <li>
      <Collapsible open={open} onOpenChange={setOpen}>
        <div className="flex flex-wrap items-center gap-2 px-3 py-2">
          <CollapsibleTrigger
            render={
              <Button
                type="button"
                variant="ghost"
                size="icon-xs"
                aria-label={open ? t("Hide findings") : t("Show findings")}
              />
            }
          >
            <ChevronRightIcon
              className={open ? "rotate-90 transition-transform" : "transition-transform"}
            />
          </CollapsibleTrigger>
          <span className="text-sm font-medium tabular-nums">
            {formatUnixDateTimeMedium(snapshot.fetchedAt)}
          </span>
          {current ? (
            <Badge variant="info" className="max-h-5">
              {t("Current")}
            </Badge>
          ) : null}
          <RiskLevelBadge level={snapshot.riskLevel} />
          <ReviewStateBadge state={snapshot.reviewState} reviewedAt={snapshot.reviewedAt} />
          <span className="text-muted-foreground text-xs">
            {carrierIntelProviderLabel(snapshot.provider)} · {labels.depth[snapshot.depth]} ·{" "}
            {snapshot.source}
          </span>
          <span className="text-muted-foreground ml-auto text-xs tabular-nums">
            {t(
              "{0, plural, one {# blocker} other {# blockers}}, {1, plural, one {# advisory} other {# advisories}}",
              snapshot.blockingCodes.length,
              snapshot.advisoryCodes.length,
            )}
          </span>
          {canExport && snapshot.hasRawPayload ? (
            <Button type="button" size="xs" variant="outline" onClick={() => onViewRaw(snapshot)}>
              <FileJsonIcon />
              {t("Raw payload")}
            </Button>
          ) : null}
        </div>
        <CollapsibleContent>
          <div className="flex flex-col gap-2 px-3 pb-3">
            {snapshot.notFound ? (
              <p className="text-muted-foreground text-sm">
                {t("The provider had no record for USDOT {0} at this time.", snapshot.dotNumber)}
              </p>
            ) : null}
            {snapshot.reviewNote ? (
              <p className="text-sm">
                <span className="text-muted-foreground">{t("Review note:")}</span>{" "}
                {snapshot.reviewNote}
              </p>
            ) : null}
            <FindingList findings={snapshot.findings} ruleLabels={ruleLabels} />
          </div>
        </CollapsibleContent>
      </Collapsible>
    </li>
  );
}

export function IntelligenceHistory({
  carrierId,
  currentSnapshotId,
  ruleLabels,
  canExport,
}: IntelligenceHistoryProps) {
  const t = useT();
  const [rawSnapshot, setRawSnapshot] = useState<CarrierIntelSnapshotSummary | null>(null);

  const historyQuery = useQuery({
    queryKey: [CARRIER_INTEL_HISTORY_KEY, carrierId, HISTORY_LIMIT],
    queryFn: ({ signal }) => fetchCarrierIntelSnapshotHistory(carrierId, HISTORY_LIMIT, { signal }),
  });

  if (historyQuery.isPending) {
    return (
      <div className="flex flex-col gap-2">
        <Skeleton className="h-10 w-full" />
        <Skeleton className="h-10 w-full" />
        <Skeleton className="h-10 w-full" />
      </div>
    );
  }

  if (historyQuery.isError) {
    return (
      <p className="text-destructive rounded-lg border border-dashed p-3 text-sm">
        {t("Snapshot history could not be loaded. {0}", historyQuery.error.message)}
      </p>
    );
  }

  if (historyQuery.data.length === 0) {
    return (
      <p className="text-muted-foreground rounded-lg border border-dashed p-3 text-sm">
        {t("No snapshots recorded yet.")}
      </p>
    );
  }

  return (
    <>
      <ul className="bg-card divide-y rounded-lg border">
        {historyQuery.data.map((snapshot) => (
          <SnapshotRow
            key={snapshot.id}
            snapshot={snapshot}
            current={snapshot.id === currentSnapshotId}
            ruleLabels={ruleLabels}
            canExport={canExport}
            onViewRaw={setRawSnapshot}
          />
        ))}
      </ul>
      {canExport ? (
        <RawPayloadDialog
          carrierId={carrierId}
          snapshot={rawSnapshot}
          open={rawSnapshot !== null}
          onOpenChange={(open) => {
            if (!open) {
              setRawSnapshot(null);
            }
          }}
        />
      ) : null}
    </>
  );
}
