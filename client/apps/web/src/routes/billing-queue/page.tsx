import { BillingWorkspaceLayout } from "@/components/billing/billing-workspace-layout";
import { LazyComponent } from "@trenova/shared/components/error-boundary";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@trenova/shared/components/ui/sheet";
import { useHotkey } from "@tanstack/react-hotkeys";
import { useQueryClient } from "@tanstack/react-query";
import { useQueryStates } from "nuqs";
import { lazy, useCallback, useState } from "react";
import { BillingQueueKPIStrip } from "./_components/billing-queue-kpi-strip";
import { BillingQueueSidebar } from "./_components/billing-queue-sidebar";
import {
  queueSelectionSearchParamsParser,
  queueToolbarSearchParamsParser,
} from "./use-billing-queue-state";

const BillingQueueDetailPane = lazy(() => import("./_components/billing-queue-detail-pane"));
const BillingQueueDocumentPreview = lazy(
  () => import("./_components/billing-queue-document-preview"),
);

export function BillingQueuePage() {
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
        <SheetContent
          side="right"
          className="w-[min(92vw,1100px)] p-0 sm:max-w-none"
        >
          <SheetHeader className="border-b border-border pr-12">
            <SheetTitle>
              {selectedDocumentName || "Document Preview"}
            </SheetTitle>
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
