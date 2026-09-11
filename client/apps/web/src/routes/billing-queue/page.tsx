import { BillingWorkspaceLayout } from "@/components/billing/billing-workspace-layout";
import { queries } from "@/lib/queries";
import type { RoutePrefetch, RoutePrefetchQuery } from "@/lib/route-prefetch";
import { useHotkey } from "@tanstack/react-hotkeys";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { LazyComponent } from "@trenova/shared/components/error-boundary";
import type { OpenStatement } from "@trenova/shared/types/statement";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@trenova/shared/components/ui/sheet";
import { createLoader, useQueryStates } from "nuqs";
import { lazy, useCallback, useEffect, useState } from "react";
import { BillingQueueKPIStrip } from "./_components/billing-queue-kpi-strip";
import { BillingQueueSidebar } from "./_components/billing-queue-sidebar";
import {
  BILLING_QUEUE_LIST_KEY,
  billingQueueFilterPresetsQuery,
  billingQueueListQuery,
} from "./billing-queue-queries";
import { statementListQuery } from "./statement-queries";
import {
  queueSearchParamsParser,
  queueSelectionSearchParamsParser,
  queueToolbarSearchParamsParser,
  queueViewSearchParamsParser,
  statementSelectionSearchParamsParser,
} from "./use-billing-queue-state";
import { BillingQueueViewSwitch } from "./_components/billing-queue-view-switch";
import { StatementKPIStrip } from "./_components/statements/statement-kpi-strip";
import { StatementSidebar } from "./_components/statements/statement-sidebar";

const BillingQueueDetailPane = lazy(() => import("./_components/billing-queue-detail-pane"));
const BillingQueueDocumentPreview = lazy(
  () => import("./_components/billing-queue-document-preview"),
);
const StatementDetail = lazy(() =>
  import("./_components/statements/statement-detail").then((module) => ({
    default: module.StatementDetail,
  })),
);

const loadQueueSearch = createLoader(queueSearchParamsParser);

const EMPTY_STATEMENTS: OpenStatement[] = [];

// The KPI strip, the sidebar's presets and its first page of items all fire on mount
// with nothing but the URL to go on, so the loader can reproduce their keys exactly.
// The sidebar's search box is deferred, which on first render is the URL value too.
export const prefetch: RoutePrefetch = ({ request }) => {
  const { view, item, status, query, billType, billers, includePosted } =
    loadQueueSearch(request);

  // The switch shows the statement count in both views, so the list is warmed
  // either way; only the statements view pays for the rest.
  const list: RoutePrefetchQuery[] = [statementListQuery()];

  if (view === "statements") {
    return list;
  }

  list.push(
    queries.billingQueue.stats(),
    billingQueueFilterPresetsQuery(),
    billingQueueListQuery({ status, billers, billType, search: query, includePosted }),
  );
  if (item) {
    list.push(queries.billingQueue.get(item));
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
  const [viewParams, setViewParams] = useQueryStates(queueViewSearchParamsParser);
  const [selectionParams, setSelectionParams] = useQueryStates(queueSelectionSearchParamsParser);
  const [statementParams, setStatementParams] = useQueryStates(
    statementSelectionSearchParamsParser,
  );
  const [toolbarParams, setToolbarParams] = useQueryStates(queueToolbarSearchParamsParser);
  const { view } = viewParams;
  const { item: selectedItemId } = selectionParams;
  const { customer: selectedCustomerId } = statementParams;
  const { status: statusFilter, includePosted } = toolbarParams;

  const nowSeconds = useNowSeconds();
  const { data: statementData, isLoading: statementsLoading } = useQuery(statementListQuery());
  const statements = statementData?.results ?? EMPTY_STATEMENTS;

  const viewSwitch = (
    <BillingQueueViewSwitch
      view={view}
      statementCount={statementData ? statements.filter((s) => s.shipmentCount > 0).length : null}
      onChange={(next) => void setViewParams({ view: next })}
    />
  );

  const [selectedDocumentId, setSelectedDocumentId] = useState<string | null>(null);
  const [selectedDocumentName, setSelectedDocumentName] = useState<string | null>(null);

  const handleSelectItem = useCallback(
    (id: string) => {
      void setSelectionParams({ item: id });
      setSelectedDocumentId(null);
      setSelectedDocumentName(null);
    },
    [setSelectionParams],
  );

  const setStatusFilter = useCallback(
    (status: string | null) => {
      void setToolbarParams({ status });
    },
    [setToolbarParams],
  );

  const handleDocumentSelect = useCallback((docId: string, fileName: string) => {
    setSelectedDocumentId(docId);
    setSelectedDocumentName(fileName);
  }, []);

  const queryClient = useQueryClient();

  const handleAutoAdvance = useCallback(() => {
    const cached = queryClient.getQueriesData<{
      results?: { id: string; status: string }[];
    }>({
      queryKey: [BILLING_QUEUE_LIST_KEY],
    });

    for (const [, data] of cached) {
      const items = data?.results;
      if (!items?.length) continue;

      const currentIdx = items.findIndex((i) => i.id === selectedItemId);
      const nextItem = items.find(
        (i, idx) => idx > currentIdx && i.status !== "Approved" && i.status !== "Canceled",
      );

      if (nextItem) {
        handleSelectItem(nextItem.id);
        return;
      }
    }
  }, [queryClient, selectedItemId, handleSelectItem]);

  useHotkey(
    "Escape",
    () => {
      void setSelectionParams({ item: null });
      setSelectedDocumentId(null);
      setSelectedDocumentName(null);
    },
    { ignoreInputs: true },
  );

  if (view === "statements") {
    return (
      <BillingWorkspaceLayout
        pageHeaderProps={{
          title: "Billing Queue",
          description:
            "What each statement customer has accrued this period, and the invoices it becomes",
          actions: viewSwitch,
        }}
        className="gap-y-2 p-0"
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
          <LazyComponent>
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
          title: "Billing Queue",
          description: "Review and approve shipments before invoicing",
          actions: viewSwitch,
        }}
        className="gap-y-2 p-0"
        toolbar={
          <BillingQueueKPIStrip
            statusFilter={statusFilter}
            includePosted={includePosted}
            onFilterChange={setStatusFilter}
          />
        }
        sidebar={
          <BillingQueueSidebar selectedItemId={selectedItemId} onSelectItem={handleSelectItem} />
        }
        detail={
          <LazyComponent>
            <BillingQueueDetailPane
              selectedItemId={selectedItemId}
              onDocumentSelect={handleDocumentSelect}
              onAutoAdvance={handleAutoAdvance}
            />
          </LazyComponent>
        }
      />

      <Sheet
        open={Boolean(selectedDocumentId)}
        onOpenChange={(open) => {
          if (!open) {
            setSelectedDocumentId(null);
            setSelectedDocumentName(null);
          }
        }}
      >
        <SheetContent side="right" className="w-[min(92vw,1100px)] p-0 sm:max-w-none">
          <SheetHeader className="border-border border-b pr-12">
            <SheetTitle>{selectedDocumentName || "Document Preview"}</SheetTitle>
            <SheetDescription>
              Review the supporting shipment document attached to this billing queue item.
            </SheetDescription>
          </SheetHeader>
          <div className="h-[calc(100%-73px)]">
            <LazyComponent>
              <BillingQueueDocumentPreview
                documentId={selectedDocumentId}
                fileName={selectedDocumentName}
              />
            </LazyComponent>
          </div>
        </SheetContent>
      </Sheet>
    </>
  );
}
