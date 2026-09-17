import { FindingList } from "@/components/carrier-intelligence/finding-list";
import { IntelInlineError } from "@/components/carrier-intelligence/intel-inline-error";
import { ReviewStateLabel } from "@/components/carrier-intelligence/review-state-label";
import { RiskLabel, StatusDot, riskTone } from "@/components/carrier-intelligence/status-dot";
import { useCarrierIntelLabels } from "@/components/carrier-intelligence/use-carrier-intel-labels";
import { carrierIntelProviderLabel } from "@/lib/carrier-intelligence";
import {
  CARRIER_INTEL_HISTORY_KEY,
  fetchCarrierIntelSnapshotHistory,
  type CarrierIntelSnapshotSummary,
} from "@/lib/graphql/carrier-intelligence";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@trenova/shared/components/ui/collapsible";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTimeMedium } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { ChevronRightIcon } from "lucide-react";
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
    <li data-snapshot-id={snapshot.id}>
      <Collapsible open={open} onOpenChange={setOpen}>
        <div className="group flex min-h-12 items-center gap-3 py-2">
          <CollapsibleTrigger
            render={
              <button
                type="button"
                className="flex min-w-0 flex-1 items-center gap-3 text-left outline-none focus-visible:underline"
                aria-label={open ? t("Hide findings") : t("Show findings")}
              />
            }
          >
            <ChevronRightIcon
              className={cn(
                "text-muted-foreground size-3.5 shrink-0 transition-transform",
                open && "rotate-90",
              )}
              aria-hidden
            />
            <StatusDot tone={riskTone(snapshot.riskLevel)} />
            <span className="flex min-w-0 flex-col gap-0.5">
              <span className="flex items-center gap-2 text-sm tabular-nums">
                {formatUnixDateTimeMedium(snapshot.fetchedAt)}
                {current ? (
                  <span className="text-muted-foreground text-xs">{t("Current")}</span>
                ) : null}
              </span>
              <span className="text-muted-foreground truncate text-xs">
                {[
                  carrierIntelProviderLabel(snapshot.provider),
                  labels.depth[snapshot.depth],
                  snapshot.source,
                ].join(" · ")}
              </span>
            </span>
          </CollapsibleTrigger>
          <span className="text-muted-foreground hidden shrink-0 items-center gap-2 text-xs sm:flex">
            <RiskLabel level={snapshot.riskLevel} showDot={false} />
            <span aria-hidden>·</span>
            <span className="tabular-nums">
              {t(
                "{0, plural, one {# blocker} other {# blockers}}, {1, plural, one {# advisory} other {# advisories}}",
                snapshot.blockingCodes.length,
                snapshot.advisoryCodes.length,
              )}
            </span>
          </span>
          {canExport && snapshot.hasRawPayload ? (
            <Button type="button" size="xs" variant="ghost" onClick={() => onViewRaw(snapshot)}>
              {t("Raw payload")}
            </Button>
          ) : null}
        </div>
        <CollapsibleContent>
          <div className="flex flex-col gap-2 pb-3 pl-6">
            <div className="text-muted-foreground flex flex-wrap items-center gap-x-2 text-xs">
              <ReviewStateLabel state={snapshot.reviewState} reviewedAt={snapshot.reviewedAt} />
              {snapshot.reviewNote ? (
                <>
                  <span aria-hidden>·</span>
                  <span>{snapshot.reviewNote}</span>
                </>
              ) : null}
            </div>
            {snapshot.notFound ? (
              <p className="text-muted-foreground text-xs">
                {t("The provider had no record for USDOT {0} at this time.", snapshot.dotNumber)}
              </p>
            ) : null}
            <FindingList
              findings={snapshot.findings}
              ruleLabels={ruleLabels}
              emptyMessage={t("Every enabled vetting rule passed on this snapshot.")}
            />
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
      <div className="divide-border flex flex-col divide-y" aria-busy>
        {[0, 1, 2].map((index) => (
          <div key={index} className="flex items-center gap-3 py-2.5">
            <Skeleton className="size-2 rounded-full" />
            <div className="flex flex-1 flex-col gap-1.5">
              <Skeleton className="h-3.5 w-40" />
              <Skeleton className="h-3 w-56" />
            </div>
          </div>
        ))}
      </div>
    );
  }

  if (historyQuery.isError) {
    return (
      <IntelInlineError
        error={historyQuery.error}
        title={t("Snapshot history could not be loaded")}
        onRetry={() => void historyQuery.refetch()}
      />
    );
  }

  if (historyQuery.data.length === 0) {
    return (
      <p className="text-muted-foreground py-6 text-center text-xs">
        {t("No snapshots recorded yet.")}
      </p>
    );
  }

  return (
    <>
      <ul className="divide-border divide-y" aria-label={t("Snapshot history")}>
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
