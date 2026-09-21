import { FindingList, type FindingKind } from "@/components/carrier-intelligence/finding-list";
import { MarkReviewedDialog } from "@/components/carrier-intelligence/mark-reviewed-dialog";
import { RelativeTime } from "@/components/carrier-intelligence/relative-time";
import { RiskLabel, StatusDot, riskTone } from "@/components/carrier-intelligence/status-dot";
import { useCarrierIntelRuleLabels } from "@/components/carrier-intelligence/use-carrier-intel-rule-labels";
import { carrierIntelProviderLabel } from "@/lib/carrier-intelligence";
import { carrierPanelPath } from "@/lib/carrier-links";
import {
  CARRIER_INTELLIGENCE_KEY,
  CARRIER_INTEL_REVIEW_QUEUE_KEY,
  fetchCarrierIntelReviewQueue,
  type CarrierIntelReviewQueueItem,
} from "@/lib/graphql/carrier-intelligence";
import { queries } from "@/lib/queries";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { EmptySheet, GhostLine } from "@trenova/shared/components/ui/empty-sheet";
import { Input } from "@trenova/shared/components/ui/input";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@trenova/shared/components/ui/sheet";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTimeMedium } from "@trenova/shared/lib/date";
import { graphQLErrorMessage } from "@trenova/shared/lib/graphql";
import { ClipboardCheckIcon, ExternalLinkIcon, RefreshCwIcon, SearchIcon } from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { Link } from "react-router";

export const REVIEW_QUEUE_LIMIT = 100;
const COMPACT_FINDING_LIMIT = 3;
const COMPACT_FINDING_KINDS: readonly FindingKind[] = ["blocker", "advisory"];

export type ReviewQueueProps = {
  canUpdate: boolean;
};

export function reviewCarrierName(item: CarrierIntelReviewQueueItem, t: TranslateFn): string {
  const identity = item.profile.identity;
  return identity?.legalName || identity?.dbaName || t("USDOT {0}", item.dotNumber);
}

export function filterReviewQueue(
  items: readonly CarrierIntelReviewQueueItem[],
  search: string,
  t: TranslateFn,
): CarrierIntelReviewQueueItem[] {
  const needle = search.trim().toLowerCase();
  if (needle === "") {
    return [...items];
  }
  return items.filter((item) =>
    [reviewCarrierName(item, t), item.dotNumber, item.docketNumber ?? ""].some((value) =>
      value.toLowerCase().includes(needle),
    ),
  );
}

function ReviewSketch() {
  return (
    <div className="flex flex-col rounded-lg border">
      {["w-2/3", "w-1/2", "w-3/5"].map((width) => (
        <div key={width} className="flex items-start gap-3 border-b px-3 py-3 last:border-b-0">
          <span className="bg-muted-foreground/20 mt-1 size-2 rounded-full" />
          <div className="flex flex-1 flex-col gap-1.5">
            <GhostLine className={width} />
            <GhostLine className="w-1/3" />
          </div>
        </div>
      ))}
    </div>
  );
}

type ReviewRowProps = {
  item: CarrierIntelReviewQueueItem;
  ruleLabels: Readonly<Record<string, string>>;
  canUpdate: boolean;
  onOpen: (item: CarrierIntelReviewQueueItem) => void;
  onMarkReviewed: (item: CarrierIntelReviewQueueItem) => void;
};

function ReviewRow({ item, ruleLabels, canUpdate, onOpen, onMarkReviewed }: ReviewRowProps) {
  const t = useT();
  const name = reviewCarrierName(item, t);

  return (
    <li
      className="group flex cursor-pointer items-start gap-3 border-b border-border/60 px-3 py-3 transition-colors last:border-b-0 hover:bg-muted/50"
      onClick={() => onOpen(item)}
      data-testid="review-queue-item"
    >
      <span className="flex h-5 items-center">
        <StatusDot tone={riskTone(item.riskLevel)} />
      </span>
      <div className="flex min-w-0 flex-1 flex-col gap-2">
        <div className="flex min-w-0 flex-col gap-0.5">
          <div className="flex min-w-0 items-center gap-2">
            <span className="truncate text-sm font-medium">{name}</span>
            <RiskLabel level={item.riskLevel} showDot={false} className="shrink-0" />
          </div>
          <div className="text-muted-foreground flex min-w-0 flex-wrap items-center gap-x-1.5 text-xs">
            <span className="tabular-nums">{t("USDOT {0}", item.dotNumber)}</span>
            {item.docketNumber ? (
              <>
                <span aria-hidden>·</span>
                <span className="tabular-nums">{t("MC {0}", item.docketNumber)}</span>
              </>
            ) : null}
            <span aria-hidden>·</span>
            <span>{carrierIntelProviderLabel(item.provider)}</span>
            <span aria-hidden>·</span>
            <span className="flex items-center gap-1">
              {t("Fetched")} <RelativeTime timestamp={item.fetchedAt} />
            </span>
          </div>
        </div>
        <FindingList
          findings={item.findings}
          ruleLabels={ruleLabels}
          kinds={COMPACT_FINDING_KINDS}
          compact
          limit={COMPACT_FINDING_LIMIT}
          emptyMessage={t("No blocking or advisory findings")}
        />
      </div>
      {canUpdate && item.carrierId ? (
        <Button
          type="button"
          variant="outline"
          className="h-8 shrink-0 text-xs"
          onClick={(event) => {
            event.stopPropagation();
            onMarkReviewed(item);
          }}
        >
          <ClipboardCheckIcon className="size-3.5" />
          {t("Mark reviewed")}
        </Button>
      ) : null}
    </li>
  );
}

type ReviewDetailSheetProps = {
  item: CarrierIntelReviewQueueItem | null;
  ruleLabels: Readonly<Record<string, string>>;
  canUpdate: boolean;
  onOpenChange: (open: boolean) => void;
  onMarkReviewed: (item: CarrierIntelReviewQueueItem) => void;
};

function ReviewDetailSheet({
  item,
  ruleLabels,
  canUpdate,
  onOpenChange,
  onMarkReviewed,
}: ReviewDetailSheetProps) {
  const t = useT();

  return (
    <Sheet open={item !== null} onOpenChange={onOpenChange}>
      <SheetContent className="w-full gap-0 overflow-y-auto p-0 sm:max-w-lg">
        {item ? (
          <>
            <SheetHeader className="border-border gap-3 border-b px-4 py-3">
              <RiskLabel level={item.riskLevel} />
              <SheetTitle className="pr-8">{reviewCarrierName(item, t)}</SheetTitle>
              <SheetDescription className="text-xs tabular-nums">
                {item.docketNumber
                  ? t("USDOT {0} · MC {1}", item.dotNumber, item.docketNumber)
                  : t("USDOT {0}", item.dotNumber)}
              </SheetDescription>
              <div className="flex flex-wrap items-center gap-2">
                {canUpdate && item.carrierId ? (
                  <Button
                    type="button"
                    variant="outline"
                    className="h-8 text-xs"
                    onClick={() => onMarkReviewed(item)}
                  >
                    <ClipboardCheckIcon className="size-3.5" />
                    {t("Mark reviewed")}
                  </Button>
                ) : null}
                {item.carrierId ? (
                  <Button
                    variant="ghost"
                    className="h-8 text-xs"
                    nativeButton={false}
                    render={<Link to={carrierPanelPath(item.carrierId, "intelligence")} />}
                  >
                    <ExternalLinkIcon className="size-3.5" />
                    {t("Open carrier")}
                  </Button>
                ) : null}
              </div>
            </SheetHeader>
            <FindingList
              findings={item.findings}
              ruleLabels={ruleLabels}
              grouped
              emptyMessage={t("No findings. The review was requested by a change on the carrier.")}
              className="px-5 py-4"
            />
            <section className="flex flex-col gap-3 border-t border-border/60 px-5 py-4">
              <h3 className="text-muted-foreground text-xs font-medium">{t("Snapshot")}</h3>
              <dl className="grid grid-cols-[7.5rem_minmax(0,1fr)] gap-x-4 gap-y-2 text-sm">
                <dt className="text-muted-foreground text-xs">{t("Provider")}</dt>
                <dd>{carrierIntelProviderLabel(item.provider)}</dd>
                <dt className="text-muted-foreground text-xs">{t("Fetched")}</dt>
                <dd>{formatUnixDateTimeMedium(item.fetchedAt)}</dd>
                <dt className="text-muted-foreground text-xs">{t("Data as of")}</dt>
                <dd>{formatUnixDateTimeMedium(item.effectiveAsOf)}</dd>
                {item.confirmedAt ? (
                  <>
                    <dt className="text-muted-foreground text-xs">{t("Confirmed")}</dt>
                    <dd>{formatUnixDateTimeMedium(item.confirmedAt)}</dd>
                  </>
                ) : null}
              </dl>
            </section>
          </>
        ) : null}
      </SheetContent>
    </Sheet>
  );
}

export function ReviewQueue({ canUpdate }: ReviewQueueProps) {
  const t = useT();
  const queryClient = useQueryClient();
  const ruleLabels = useCarrierIntelRuleLabels(true);
  const [search, setSearch] = useState("");
  const [openItem, setOpenItem] = useState<CarrierIntelReviewQueueItem | null>(null);
  const [reviewing, setReviewing] = useState<CarrierIntelReviewQueueItem | null>(null);

  const queueQuery = useQuery({
    queryKey: [CARRIER_INTEL_REVIEW_QUEUE_KEY, REVIEW_QUEUE_LIMIT],
    queryFn: ({ signal }) => fetchCarrierIntelReviewQueue(REVIEW_QUEUE_LIMIT, { signal }),
  });

  const items = useMemo(
    () => filterReviewQueue(queueQuery.data ?? [], search, t),
    [queueQuery.data, search, t],
  );

  const handleReviewed = useCallback(() => {
    setOpenItem(null);
    void queryClient.invalidateQueries({ queryKey: [CARRIER_INTEL_REVIEW_QUEUE_KEY] });
    void queryClient.invalidateQueries({ queryKey: [CARRIER_INTELLIGENCE_KEY] });
    void queryClient.invalidateQueries({ queryKey: ["carrier-list"] });
    void queryClient.invalidateQueries({
      queryKey: queries.carrierIntelSettings.monitoringStatus().queryKey,
    });
  }, [queryClient]);

  const total = queueQuery.data?.length ?? 0;

  let content;
  if (queueQuery.isPending) {
    content = (
      <ul className="flex flex-col">
        {[0, 1, 2, 3].map((index) => (
          <li key={index} className="flex items-start gap-3 border-b border-border/60 px-3 py-3">
            <Skeleton className="mt-1.5 size-2 rounded-full" />
            <div className="flex flex-1 flex-col gap-2">
              <Skeleton className="h-3.5 w-1/3" />
              <Skeleton className="h-3 w-1/2" />
              <Skeleton className="h-3 w-2/5" />
            </div>
          </li>
        ))}
      </ul>
    );
  } else if (queueQuery.isError) {
    content = (
      <div className="flex flex-col items-center gap-3 px-6 py-12 text-center">
        <p className="text-sm font-medium">{t("The review queue could not be loaded")}</p>
        <p className="text-muted-foreground text-xs">
          {graphQLErrorMessage(queueQuery.error, t("Try again in a moment."))}
        </p>
        <Button
          type="button"
          variant="outline"
          className="h-8 text-xs"
          onClick={() => void queueQuery.refetch()}
        >
          <RefreshCwIcon className="size-3.5" />
          {t("Retry")}
        </Button>
      </div>
    );
  } else if (total === 0) {
    content = (
      <EmptySheet
        title={t("Nothing awaiting review")}
        description={t(
          "Carriers land here when a vetting or a monitored change needs someone to look at it and sign off.",
        )}
        sketch={<ReviewSketch />}
      />
    );
  } else if (items.length === 0) {
    content = (
      <EmptySheet
        title={t("Nothing matches")}
        description={t("No carrier awaiting review fits this search.")}
        sketch={<ReviewSketch />}
        action={
          <Button
            type="button"
            variant="outline"
            className="h-8 text-xs"
            onClick={() => setSearch("")}
          >
            {t("Clear search")}
          </Button>
        }
      />
    );
  } else {
    content = (
      <ul aria-label={t("Carriers awaiting review")}>
        {items.map((item) => (
          <ReviewRow
            key={item.id}
            item={item}
            ruleLabels={ruleLabels}
            canUpdate={canUpdate}
            onOpen={setOpenItem}
            onMarkReviewed={setReviewing}
          />
        ))}
      </ul>
    );
  }

  return (
    <div className="flex flex-col gap-3">
      <div className="flex flex-wrap items-center gap-2">
        <Input
          type="search"
          value={search}
          onChange={(event) => setSearch(event.target.value)}
          placeholder={t("Search carrier or USDOT")}
          aria-label={t("Search carriers awaiting review")}
          leftElement={<SearchIcon className="text-muted-foreground size-3.5" />}
          inputContainerClassName="w-full sm:w-64"
          className="h-8 text-xs md:text-xs"
        />
        <div className="ml-auto flex items-center gap-2">
          {total > 0 ? (
            <span className="text-muted-foreground text-xs tabular-nums">
              {total >= REVIEW_QUEUE_LIMIT
                ? t("Showing the first {0}", REVIEW_QUEUE_LIMIT)
                : t("{0} waiting", total)}
            </span>
          ) : null}
          <Button
            type="button"
            variant="ghost"
            size="icon"
            aria-label={t("Refresh review queue")}
            isLoading={queueQuery.isRefetching}
            onClick={() => void queueQuery.refetch()}
          >
            <RefreshCwIcon className="size-3.5" />
          </Button>
        </div>
      </div>
      <div className="bg-background overflow-hidden rounded-lg border">{content}</div>
      <ReviewDetailSheet
        item={openItem}
        ruleLabels={ruleLabels}
        canUpdate={canUpdate}
        onOpenChange={(open) => {
          if (!open) setOpenItem(null);
        }}
        onMarkReviewed={setReviewing}
      />
      {reviewing?.carrierId ? (
        <MarkReviewedDialog
          carrierId={reviewing.carrierId}
          blockingCount={reviewing.blockingCodes.length}
          open
          onOpenChange={(open) => {
            if (!open) setReviewing(null);
          }}
          onReviewed={handleReviewed}
        />
      ) : null}
    </div>
  );
}
