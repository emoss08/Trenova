import { useT } from "@trenova/shared/i18n/use-t";
import { BillingWorkspaceLayout } from "@/components/billing/billing-workspace-layout";
import { queries } from "@/lib/queries";
import type { RoutePrefetch, RoutePrefetchQuery } from "@/lib/route-prefetch";
import { useHotkey } from "@tanstack/react-hotkeys";
import { useQueryClient } from "@tanstack/react-query";
import { LazyComponent } from "@trenova/shared/components/error-boundary";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@trenova/shared/components/ui/sheet";
import { createLoader, useQueryStates } from "nuqs";
import { lazy, useCallback, useState } from "react";
import { BillingQueueKPIStrip } from "./_components/billing-queue-kpi-strip";
import { BillingQueueSidebar } from "./_components/billing-queue-sidebar";
import {
  BILLING_QUEUE_LIST_KEY,
  billingQueueFilterPresetsQuery,
  billingQueueListQuery,
} from "./billing-queue-queries";
import {
  queueSearchParamsParser,
  queueSelectionSearchParamsParser,
  queueToolbarSearchParamsParser,
} from "./use-billing-queue-state";

const BillingQueueDetailPane = lazy(() => import("./_components/billing-queue-detail-pane"));
const BillingQueueDocumentPreview = lazy(
  () => import("./_components/billing-queue-document-preview"),
);

const loadQueueSearch = createLoader(queueSearchParamsParser);

// The KPI strip, the sidebar's presets and its first page of items all fire on mount
// with nothing but the URL to go on, so the loader can reproduce their keys exactly.
// The sidebar's search box is deferred, which on first render is the URL value too.
export const prefetch: RoutePrefetch = ({ request }) => {
  const { item, status, query, billType, billers, includePosted } = loadQueueSearch(request);
  const list: RoutePrefetchQuery[] = [
    queries.billingQueue.stats(),
    billingQueueFilterPresetsQuery(),
    billingQueueListQuery({ status, billers, billType, search: query, includePosted }),
  ];
  if (item) {
    list.push(queries.billingQueue.get(item));
  }
  return list;
};

export function BillingQueuePage() {
  const t = useT();

  const [selectionParams, setSelectionParams] = useQueryStates(queueSelectionSearchParamsParser);
  const [toolbarParams, setToolbarParams] = useQueryStates(queueToolbarSearchParamsParser);
  const { item: selectedItemId } = selectionParams;
  const { status: statusFilter, includePosted } = toolbarParams;

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

  return (
    <>
      <BillingWorkspaceLayout
        pageHeaderProps={{
          title: "Billing Queue",
          description: "Review and approve shipments before invoicing",
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
              {t("Review the supporting shipment document attached to this billing queue item.")}
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
