import { BillingWorkspaceLayout } from "@/components/billing/billing-workspace-layout";
import { LazyComponent } from "@/components/error-boundary";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { useQueryStates } from "nuqs";
import { lazy, useCallback, useState } from "react";
import { InvoiceSidebar } from "./_components/invoice-sidebar";
import { invoiceSearchParamsParser } from "./use-invoice-state";

const InvoiceDetailPane = lazy(
  () => import("./_components/invoice-detail-pane"),
);
const BillingQueueDocumentPreview = lazy(
  () => import("../billing-queue/_components/billing-queue-document-preview"),
);

export function InvoicesPage() {
  const [searchParams, setSearchParams] = useQueryStates(
    invoiceSearchParamsParser,
  );
  const selectedInvoiceId = searchParams.item;
  const [selectedDocumentId, setSelectedDocumentId] = useState<string | null>(
    null,
  );
  const [selectedDocumentName, setSelectedDocumentName] = useState<
    string | null
  >(null);

  const handleSelectInvoice = useCallback(
    (id: string) => {
      void setSearchParams({ item: id });
      setSelectedDocumentId(null);
      setSelectedDocumentName(null);
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

  return (
    <>
      <BillingWorkspaceLayout
        pageHeaderProps={{
          title: "Invoices",
          description:
            "Review draft invoices, confirm billing details, and post completed charges",
        }}
        sidebar={
          <InvoiceSidebar
            selectedInvoiceId={selectedInvoiceId}
            onSelectInvoice={handleSelectInvoice}
          />
        }
        detail={
          <LazyComponent>
            <InvoiceDetailPane
              selectedInvoiceId={selectedInvoiceId}
              selectedDocumentId={selectedDocumentId}
              onDocumentSelect={handleDocumentSelect}
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
              Review the supporting shipment document attached to this invoice.
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
