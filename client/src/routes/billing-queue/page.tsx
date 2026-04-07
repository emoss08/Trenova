import { LazyComponent } from "@/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { useHotkey } from "@tanstack/react-hotkeys";
import { useQueryClient } from "@tanstack/react-query";
import { useQueryStates } from "nuqs";
import { lazy, useCallback, useState } from "react";
import { BillingQueueKPIStrip } from "./_components/billing-queue-kpi-strip";
import { BillingQueueSidebar } from "./_components/billing-queue-sidebar";
import { queueSearchParamsParser } from "./use-billing-queue-state";

const BillingQueueDetailPane = lazy(
  () => import("./_components/billing-queue-detail-pane"),
);
const BillingQueueDocumentPreview = lazy(
  () => import("./_components/billing-queue-document-preview"),
);

export function BillingQueuePage() {
  const [searchParams, setSearchParams] = useQueryStates(queueSearchParamsParser);
  const { item: selectedItemId, status: statusFilter } = searchParams;

  const [selectedDocumentId, setSelectedDocumentId] = useState<string | null>(
    null,
  );
  const [selectedDocumentName, setSelectedDocumentName] = useState<
    string | null
  >(null);

  const handleSelectItem = useCallback(
    (id: string) => {
      void setSearchParams({ item: id });
      setSelectedDocumentId(null);
      setSelectedDocumentName(null);
    },
    [setSearchParams],
  );

  const setStatusFilter = useCallback(
    (status: string | null) => {
      void setSearchParams({ status });
    },
    [setSearchParams],
  );

  const handleDocumentSelect = useCallback(
    (docId: string, fileName: string) => {
      setSelectedDocumentId(docId);
      setSelectedDocumentName(fileName);
    },
    [],
  );

  const queryClient = useQueryClient();

  const handleAutoAdvance = useCallback(() => {
    const cached = queryClient.getQueriesData<{ results?: { id: string; status: string }[] }>({
      queryKey: ["billing-queue-list"],
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
      void setSearchParams({ item: null });
      setSelectedDocumentId(null);
      setSelectedDocumentName(null);
    },
    { ignoreInputs: true },
  );

  return (
    <PageLayout
      pageHeaderProps={{
        title: "Billing Queue",
        description: "Review and approve shipments before invoicing",
      }}
    >
      <BillingQueueKPIStrip statusFilter={statusFilter} onFilterChange={setStatusFilter} />
      <div className="grid grid-cols-[300px_1fr_1fr] h-[calc(100vh-220px)] gap-0 mx-4 mb-4 mt-3 rounded-lg border overflow-hidden">
        <div className="border-r overflow-hidden">
          <BillingQueueSidebar
            selectedItemId={selectedItemId}
            onSelectItem={handleSelectItem}
          />
        </div>
        <div className="border-r overflow-hidden">
          <LazyComponent>
            <BillingQueueDetailPane
              selectedItemId={selectedItemId}
              onDocumentSelect={handleDocumentSelect}
              onAutoAdvance={handleAutoAdvance}
            />
          </LazyComponent>
        </div>
        <div className="overflow-hidden">
          <LazyComponent>
            <BillingQueueDocumentPreview
              documentId={selectedDocumentId}
              fileName={selectedDocumentName}
            />
          </LazyComponent>
        </div>
      </div>
    </PageLayout>
  );
}
