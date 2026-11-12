import { useT } from "@trenova/shared/i18n/use-t";
import { BillingWorkspaceLayout } from "@/components/billing/billing-workspace-layout";
import { billingQueue } from "@/lib/queries/billing-queue";
import type { RoutePrefetch, RoutePrefetchEntry } from "@/lib/route-prefetch";
import { useQuery } from "@tanstack/react-query";
import { LazyComponent } from "@trenova/shared/components/error-boundary";
import type { OpenStatement } from "@trenova/shared/types/statement";
import { createLoader, useQueryStates } from "nuqs";
import { lazy, useCallback, useEffect, useRef, useState } from "react";
import {
  BILLING_QUEUE_OPEN_LANES,
  type BillingQueueLane,
} from "@trenova/shared/types/billing-queue";
import {
  BillingQueueDetailSkeleton,
  BillingQueueDetailUnselected,
} from "./_components/billing-queue-detail-skeleton";
import { BillingQueueDocumentReview } from "./_components/billing-queue-document-review";
import { BillingQueueOverview } from "./_components/billing-queue-overview";
import { useOpenFirstItem } from "./use-open-first-item";
import { LANES_FOLDED_BY_DEFAULT } from "./_components/billing-queue-lanes";
import { BillingQueueCaughtUp, isCaughtUp } from "./_components/billing-queue-caught-up";
import { activeQueueFilterCount } from "./_components/billing-queue-filters";
import { BillingQueueSidebar } from "./_components/billing-queue-sidebar";
import {
  billingQueueFilterPresetsQuery,
  billingQueueLaneQuery,
  billingQueueStatsQuery,
} from "./billing-queue-queries";
import { statementListQuery } from "./statement-queries";
import {
  queueSearchParamsParser,
  queueSelectionSearchParamsParser,
  queueSidebarSearchParamsParser,
  queueViewSearchParamsParser,
  statementSelectionSearchParamsParser,
} from "./use-billing-queue-state";
import { useTransferDialog } from "./use-transfer-dialog";
import { BillingQueueViewSwitch } from "./_components/billing-queue-view-switch";
import { BulkBillingTransferAction } from "./_components/bulk-transfer/bulk-billing-transfer-action";
import { BillingQueueRedetectSignaturesAction } from "./_components/billing-queue-redetect-signatures-action";
import { StatementKPIStrip } from "./_components/statements/statement-kpi-strip";
import {
  StatementDetailSkeleton,
  StatementDetailUnselected,
} from "./_components/statements/statement-detail-skeleton";
import { StatementSidebar } from "./_components/statements/statement-sidebar";

const BillingQueueDetailPane = lazy(() => import("./_components/billing-queue-detail-pane"));
const StatementDetail = lazy(() =>
  import("./_components/statements/statement-detail").then((module) => ({
    default: module.StatementDetail,
  })),
);

const loadQueueSearch = createLoader(queueSearchParamsParser);

const EMPTY_STATEMENTS: OpenStatement[] = [];

// The overview, the suggestions, the presets and the first page of every open lane all
// fire on mount with nothing but the URL to go on, so the loader can reproduce their
// keys exactly. The sidebar's search box is deferred, which on first render is the URL
// value too.
export const prefetch: RoutePrefetch = ({ request }) => {
  const {
    view,
    item,
    status,
    query,
    billType,
    billers,
    payer,
    includeClosed,
    lanes,
    minAmount,
    maxAmount,
    createdBefore,
    shipmentCustomer,
    delivered,
    pickedUp,
    queued,
  } = loadQueueSearch(request);

  // The switch shows the statement count in both views, so the list is warmed
  // either way; only the statements view pays for the rest.
  const list: RoutePrefetchEntry[] = [statementListQuery()];

  if (view === "statements") {
    return list;
  }

  const filters = {
    status,
    billers,
    billType,
    payer,
    search: query,
    includeClosed,
    minAmount,
    maxAmount,
    createdBefore,
    shipmentCustomer,
    delivered,
    pickedUp,
    queued,
  };
  const openLanes = BILLING_QUEUE_OPEN_LANES.filter((lane) =>
    lanes.length > 0 ? lanes.includes(lane) : !LANES_FOLDED_BY_DEFAULT.includes(lane),
  );
  list.push(
    billingQueueStatsQuery(),
    billingQueue.assist(),
    billingQueue.recommendations(),
    billingQueueFilterPresetsQuery(),
    ...openLanes.map((lane) => billingQueueLaneQuery(lane, filters)),
  );
  if (item) {
    list.push(billingQueue.get(item));
  }
  return list;
};

/**
 * One clock for the whole view.
 *
 * Every countdown and progress bar on the page has to agree, and "now" recomputed
 * per component would let two cards on the same screen disagree about whether a
 * period has closed. A minute is fine: the shortest thing shown is hours.
 */
function useNowSeconds(): number {
  const [now, setNow] = useState(() => Math.floor(Date.now() / 1000));
  useEffect(() => {
    const timer = setInterval(() => setNow(Math.floor(Date.now() / 1000)), 60_000);
    return () => clearInterval(timer);
  }, []);
  return now;
}

export function BillingQueuePage() {
  const t = useT();

  const [viewParams, setViewParams] = useQueryStates(queueViewSearchParamsParser);
  const [selectionParams, setSelectionParams] = useQueryStates(queueSelectionSearchParamsParser);
  const [statementParams, setStatementParams] = useQueryStates(
    statementSelectionSearchParamsParser,
  );
  const [sidebarParams, setSidebarParams] = useQueryStates(queueSidebarSearchParamsParser);
  const transfer = useTransferDialog();
  const { view } = viewParams;
  const { item: selectedItemId } = selectionParams;
  const { customer: selectedCustomerId } = statementParams;

  const nowSeconds = useNowSeconds();
  const { data: queueStats } = useQuery({
    ...billingQueueStatsQuery(),
    enabled: view === "shipments",
  });
  const queueFiltered =
    activeQueueFilterCount(sidebarParams) > 0 || sidebarParams.query.trim() !== "";
  const caughtUp = !queueFiltered && isCaughtUp(queueStats);
  const { data: statementData, isLoading: statementsLoading } = useQuery(statementListQuery());
  const statements = statementData?.results ?? EMPTY_STATEMENTS;

  const headerActions = (
    <div className="flex items-center gap-2">
      <BillingQueueRedetectSignaturesAction />
      <BulkBillingTransferAction
        open={transfer.open}
        onOpenChange={transfer.onOpenChange}
        runId={transfer.runId}
        onRunIdChange={transfer.onRunIdChange}
      />
      <BillingQueueViewSwitch
        view={view}
        statementCount={statementData ? statements.filter((s) => s.shipmentCount > 0).length : null}
        onChange={(next) => void setViewParams({ view: next })}
      />
    </div>
  );

  const [selectedDocumentId, setSelectedDocumentId] = useState<string | null>(null);

  const handleSelectItem = useCallback(
    (id: string) => {
      void setSelectionParams({ item: id });
      setSelectedDocumentId(null);
    },
    [setSelectionParams],
  );

  const toggleLane = useCallback(
    (lane: BillingQueueLane) => {
      const current = sidebarParams.lanes;
      const open = lane as (typeof BILLING_QUEUE_OPEN_LANES)[number];
      void setSidebarParams({
        lanes: current.length === 1 && current[0] === open ? [] : [open],
      });
    },
    [sidebarParams.lanes, setSidebarParams],
  );

  const handleDocumentSelect = useCallback((docId: string) => {
    setSelectedDocumentId(docId);
  }, []);

  // The sidebar reports the ids it shows, in order; after an approval the pane moves
  // to the next one below, read before the list refetches and the item leaves its lane.
  const order = useRef<string[]>([]);
  const { firstItemId, reportFirstItem } = useOpenFirstItem(selectedItemId, handleSelectItem);
  const handleOrderChange = useCallback(
    (ids: string[]) => {
      order.current = ids;
      reportFirstItem(ids[0] ?? null);
    },
    [reportFirstItem],
  );

  const handleAutoAdvance = useCallback(() => {
    const ids = order.current;
    const index = selectedItemId ? ids.indexOf(selectedItemId) : -1;
    const next = ids[index + 1] ?? (index > 0 ? ids[index - 1] : undefined);
    if (next && next !== selectedItemId) {
      handleSelectItem(next);
    }
  }, [selectedItemId, handleSelectItem]);

  if (view === "statements") {
    return (
      <BillingWorkspaceLayout
        pageHeaderProps={{
          title: t("Billing queue"),
          description: t(
            "What each statement customer has accrued this period, and the invoices it becomes",
          ),
          actions: headerActions,
        }}
        toolbar={<StatementKPIStrip statements={statements} nowSeconds={nowSeconds} />}
        sidebar={
          <StatementSidebar
            statements={statements}
            loading={statementsLoading}
            nowSeconds={nowSeconds}
            selectedCustomerId={selectedCustomerId}
            onSelect={(customerId) => void setStatementParams({ customer: customerId })}
          />
        }
        detail={
          <LazyComponent
            fallback={
              selectedCustomerId ? <StatementDetailSkeleton /> : <StatementDetailUnselected />
            }
          >
            <StatementDetail
              key={selectedCustomerId ?? "none"}
              customerId={selectedCustomerId}
              nowSeconds={nowSeconds}
            />
          </LazyComponent>
        }
      />
    );
  }

  return (
    <>
      <BillingWorkspaceLayout
        pageHeaderProps={{
          title: t("Billing queue"),
          description: t("Review and approve shipments before invoicing"),
          actions: headerActions,
        }}
        toolbar={
          <BillingQueueOverview activeLanes={sidebarParams.lanes} onLaneSelect={toggleLane} />
        }
        sidebarWidth="wide"
        empty={
          caughtUp && queueStats ? (
            <BillingQueueCaughtUp
              stats={queueStats}
              onShowApproved={() => void setSidebarParams({ lanes: ["Approved"] })}
            />
          ) : undefined
        }
        sidebar={
          <BillingQueueSidebar
            selectedItemId={selectedItemId}
            onSelectItem={handleSelectItem}
            onOrderChange={handleOrderChange}
          />
        }
        detail={
          // Until the list answers, or while its first item is being opened, the pane
          // is the item it is about to show.
          selectedItemId || firstItemId === null ? (
            <LazyComponent
              fallback={
                selectedItemId ? <BillingQueueDetailSkeleton /> : <BillingQueueDetailUnselected />
              }
            >
              <BillingQueueDetailPane
                selectedItemId={selectedItemId}
                onDocumentSelect={handleDocumentSelect}
                onAutoAdvance={handleAutoAdvance}
              />
            </LazyComponent>
          ) : (
            <BillingQueueDetailSkeleton />
          )
        }
      />

      <BillingQueueDocumentReview
        itemId={selectedItemId}
        documentId={selectedDocumentId}
        onDocumentChange={setSelectedDocumentId}
      />
    </>
  );
}
