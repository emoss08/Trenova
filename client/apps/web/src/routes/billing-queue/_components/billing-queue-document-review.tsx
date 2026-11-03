import { useT } from "@trenova/shared/i18n/use-t";
import { billingQueue } from "@/lib/queries/billing-queue";
import { billingQueueItemLabel } from "@/lib/billing-queue-label";
import { documentNeedsSignature } from "@/lib/document-signature";
import { storedDocumentPages } from "@/lib/queries/document-page";
import { documentReading } from "@/lib/queries/document-reading";
import { shipment as shipmentQueries } from "@/lib/queries/shipment";
import type { Extraction } from "@/lib/document-reading/extraction-payload";
import { ArrowLeftIcon } from "@trenova/shared/components/icons";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { cn } from "@trenova/shared/lib/utils";
import type { BillingQueueItem } from "@trenova/shared/types/billing-queue";
import type { Document } from "@trenova/shared/types/document";
import type { ShipmentBillingRequirement } from "@trenova/shared/types/shipment";
import { useNow } from "@trenova/shared/hooks/use-now";
import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { shipmentDocumentsQuery } from "../billing-queue-queries";
import { BillingQueueDocumentDetails, type ReadingState } from "./billing-queue-document-details";
import { BillingQueueDocumentPages } from "./billing-queue-document-pages";
import { packetAfterToggle, packetDocumentIds } from "./billing-queue-packet";
import { useInvoiceDetails } from "./use-invoice-details";

/** How often a document Nova is still reading is looked at again. */
const READING_POLL_MS = 4000;
/** How long a requested read may stay queued before the panel stops waiting for it. */
const READ_WAIT_MS = 90_000;
/** How long after an upload Nova's own read of the file is still expected. */
const NOVA_READ_WAIT_S = 120;

/** Whether Nova is still reading a just-uploaded piece of paperwork for its facts. */
function factsPending(doc: Document | null | undefined, nowSeconds: number): boolean {
  return (
    doc?.processingProfile === "shipment_paperwork" &&
    !doc.deliveryFacts &&
    nowSeconds - doc.createdAt < NOVA_READ_WAIT_S
  );
}

/**
 * Whether Nova is reading the document now. Pending alone does not say so: a
 * document nobody asked to read stays Pending, so it counts only once a read was
 * asked for here.
 */
function stillReading(
  doc: Document | null | undefined,
  requested: ReadonlyMap<string, number>,
  nowMs: number,
): boolean {
  if (!doc) return false;
  if (doc.contentStatus === "Extracting") return true;
  const askedAt = requested.get(doc.id);
  // A read that never starts (intelligence turned off for the organization) stops
  // being waited on, so the panel offers to read again rather than spinning.
  return doc.contentStatus === "Pending" && askedAt !== undefined && nowMs - askedAt < READ_WAIT_MS;
}

/**
 * A shipment's documents reviewed one at a time over the whole screen: the page as
 * the server drew it with a box over every value Nova read, and beside it what the
 * document says and what to do about it. The tabs across the top move between the
 * shipment's documents; the PRO number goes back to the queue.
 */
export function BillingQueueDocumentReview({
  itemId,
  documentId,
  onDocumentChange,
}: {
  itemId: string | null;
  documentId: string | null;
  onDocumentChange: (documentId: string | null) => void;
}) {
  const t = useT();
  const open = Boolean(documentId);
  const { data: item } = useQuery({ ...billingQueue.get(itemId ?? ""), enabled: Boolean(itemId) });
  const shipmentId = item?.shipmentId ?? "";
  const [requested, setRequested] = useState<ReadonlyMap<string, number>>(() => new Map());
  const nowSeconds = useNow("second");
  const { data: documents = [] } = useQuery({
    ...shipmentDocumentsQuery(shipmentId),
    enabled: Boolean(shipmentId) && open,
    refetchInterval: (query) => {
      const current = query.state.data?.find((doc) => doc.id === documentId);
      return stillReading(current, requested, Date.now()) ||
        factsPending(current, Date.now() / 1000)
        ? READING_POLL_MS
        : false;
    },
  });
  const document = documents.find((doc) => doc.id === documentId) ?? null;
  const reading = stillReading(document, requested, nowSeconds * 1000);
  const readingQuery = useQuery({
    ...documentReading.get(documentId ?? ""),
    enabled: open,
    refetchInterval: reading ? READING_POLL_MS : false,
  });
  const { data: readiness } = useQuery({
    ...shipmentQueries.billingReadiness(shipmentId),
    enabled: Boolean(shipmentId) && open,
  });

  const [view, setView] = useState<{ documentId: string | null; page: number }>({
    documentId,
    page: 1,
  });
  const [selected, setSelected] = useState<string | null>(null);
  const [hot, setHot] = useState<string | null>(null);
  // Opening another document starts it on its first page with nothing selected.
  if (view.documentId !== documentId) {
    setView({ documentId, page: 1 });
    setSelected(null);
  }

  const extraction = readingQuery.data?.documentId === documentId ? readingQuery.data : undefined;
  const novaRead = document?.deliveryFacts?.source === "Agent";
  const readingState: ReadingState =
    novaRead || (extraction?.fields.length ?? 0) > 0
      ? "read"
      : reading || factsPending(document, nowSeconds)
        ? "reading"
        : readingQuery.isPending
          ? "loading"
          : readingQuery.isError
            ? "failed"
            : document?.contentStatus === "Extracted" || document?.contentStatus === "Indexed"
              ? "read"
              : "unread";
  // Nova's facts arrive before the conversion that places them on the page.
  const placing = readingState === "read" && reading;
  const selectField = (key: string) => {
    const field = extraction?.fields.find((entry) => entry.key === key);
    if (field && field.page > 0) setView({ documentId, page: field.page });
    setSelected(key);
  };
  const index = documents.findIndex((doc) => doc.id === documentId);
  const next =
    index >= 0 && documents.length > 1 ? documents[(index + 1) % documents.length] : null;
  const requirements = readiness?.requirements ?? [];

  return (
    <Dialog open={open} onOpenChange={(value) => !value && onDocumentChange(null)}>
      <DialogContent
        size="fullscreen"
        showCloseButton={false}
        className="bg-background flex flex-col text-sm"
      >
        <header className="border-border bg-card flex min-h-11 shrink-0 flex-wrap items-center gap-2.5 border-b px-4 py-1.5">
          <button
            type="button"
            onClick={() => onDocumentChange(null)}
            className="ui-focus-ring text-foreground-muted hover:text-foreground flex items-center gap-1.5 rounded-sm font-medium"
          >
            <ArrowLeftIcon className="size-3.5" />
            <span className="font-mono">
              {item ? (item.shipment?.proNumber ?? billingQueueItemLabel(item)) : t("Back")}
            </span>
          </button>
          <span aria-hidden className="text-foreground-subtle">
            /
          </span>
          <DialogTitle className="text-foreground text-lg font-semibold">
            {t("Documents")}
          </DialogTitle>
          <DialogDescription className="sr-only">
            {t("Each document's page beside what Nova read from it.")}
          </DialogDescription>
          <span className="flex-1" />
          <div
            role="tablist"
            aria-label={t("Documents on this shipment")}
            className="flex flex-wrap gap-0.5"
          >
            {documents.map((doc) => {
              const unsigned = documentNeedsSignature(doc) && doc.signatureStatus !== "Signed";
              const current = doc.id === documentId;
              return (
                <button
                  key={doc.id}
                  type="button"
                  role="tab"
                  aria-selected={current}
                  onClick={() => onDocumentChange(doc.id)}
                  className={cn(
                    "ui-focus-ring flex h-7 items-center gap-1.5 rounded-md px-2.5 font-medium",
                    current
                      ? "bg-muted text-foreground"
                      : "text-foreground-muted hover:bg-surface-hover",
                  )}
                >
                  <span
                    aria-hidden
                    className={cn("size-1.5 rounded-full", unsigned ? "bg-warning" : "bg-success")}
                  />
                  {doc.documentType?.name ?? doc.originalName}
                </button>
              );
            })}
          </div>
        </header>
        <main className="flex min-h-0 flex-1 flex-wrap">
          <div className="min-h-0 min-w-0 flex-[999_1_560px]">
            {documentId ? (
              <BillingQueueDocumentPages
                key={documentId}
                source={storedDocumentPages(documentId)}
                fileName={document?.originalName ?? ""}
                pageCount={extraction?.pageCount ?? 1}
                pages={extraction?.pages ?? []}
                fields={extraction?.fields ?? []}
                page={view.page}
                onPage={(page) => setView({ documentId, page })}
                selected={selected}
                hot={hot}
                onPoint={setHot}
                onSelect={selectField}
              />
            ) : null}
          </div>
          <div className="border-border bg-card max-w-full min-w-0 flex-[1_1_380px] border-l">
            {document && item ? (
              <DetailsWithPacket
                key={`${document.id}-${document.version}`}
                document={document}
                requirements={requirements}
                item={item}
                extraction={extraction}
                readingState={readingState}
                placing={placing}
                selected={selected}
                hot={hot}
                onPoint={setHot}
                onSelectField={selectField}
                onReadStarted={() =>
                  setRequested((current) => new Map(current).set(document.id, Date.now()))
                }
                onNext={next ? () => onDocumentChange(next.id) : undefined}
                onDeleted={() => onDocumentChange(next?.id ?? null)}
              />
            ) : null}
          </div>
        </main>
      </DialogContent>
    </Dialog>
  );
}

/** The details panel with its packet checkbox saved to the item's invoice details. */
function DetailsWithPacket({
  document,
  requirements,
  item,
  extraction,
  readingState,
  placing,
  selected,
  hot,
  onPoint,
  onSelectField,
  onReadStarted,
  onNext,
  onDeleted,
}: {
  document: Document;
  requirements: readonly ShipmentBillingRequirement[];
  item: BillingQueueItem;
  extraction: Extraction | undefined;
  readingState: ReadingState;
  placing: boolean;
  selected: string | null;
  hot: string | null;
  onPoint: (key: string | null) => void;
  onSelectField: (key: string) => void;
  onReadStarted: () => void;
  onNext?: () => void;
  onDeleted: () => void;
}) {
  const { save, editable, isSaving } = useInvoiceDetails(item);
  const current = packetDocumentIds(item.packetDocumentIds, item.packetPicked, requirements);
  const packet =
    item.billType === "Invoice"
      ? {
          included: current.includes(document.id),
          editable,
          saving: isSaving,
          onToggle: (include: boolean) =>
            void save({
              packetDocumentIds: packetAfterToggle(document.id, include, current),
              packetPicked: true,
            }).catch(() => {
              // The save has already reported why it failed and reloaded the item.
            }),
        }
      : undefined;

  return (
    <BillingQueueDocumentDetails
      document={document}
      item={item}
      reading={extraction}
      readingState={readingState}
      placing={placing}
      selected={selected}
      hot={hot}
      onPoint={onPoint}
      onSelectField={onSelectField}
      packet={packet}
      onReadStarted={onReadStarted}
      onNext={onNext}
      onDeleted={onDeleted}
    />
  );
}
