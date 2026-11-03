import { DocumentTypeAutocompleteField } from "@/components/autocomplete-fields";
import { DocumentSignatureDialog } from "@/components/documents/document-signature-dialog";
import { UploadPanel } from "@/components/documents/upload-panel";
import { useDocumentUpload } from "@/hooks/use-document-upload";
import { usePermission } from "@/hooks/use-permission";
import { documentNeedsSignature } from "@/lib/document-signature";
import { sortUploadFiles } from "@/lib/document-upload-files";
import { billingQueue } from "@/lib/queries/billing-queue";
import { shipment } from "@/lib/queries/shipment";
import { apiService } from "@/services/api";
import type { UploadState } from "@/types/upload";
import { SelectionDock } from "@/components/selection-dock";
import {
  ArrowDownIcon,
  ArrowUpIcon,
  CheckIcon,
  File06Icon,
  PlusIcon,
  Trash01Icon,
} from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@trenova/shared/components/ui/dropdown-menu";
import { toast } from "@trenova/shared/components/ui/toast";
import { useLatestCallback } from "@trenova/shared/hooks/use-latest-callback";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { DockAction } from "@trenova/shared/types/data-table";
import type { Document } from "@trenova/shared/types/document";
import { Operation, Resource } from "@trenova/shared/types/permission";
import type { ShipmentBillingRequirement } from "@trenova/shared/types/shipment";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useMemo, useRef, useState } from "react";
import { useForm, useWatch } from "react-hook-form";
import { shipmentDocumentsQuery } from "../billing-queue-queries";
import { BillingQueueDocumentBoard, type BoardPacket } from "./billing-queue-document-board";
import { BillingQueueDocumentBoardSkeleton } from "./billing-queue-document-board-skeleton";
import { DocumentDetail } from "./billing-queue-document-detail";
import { DocumentsDeleteConfirm } from "./billing-queue-documents-delete-confirm";
import { BillingQueueDocumentsEmpty } from "./billing-queue-documents-empty";
import { packetAfterPlace, packetAfterToggle, packetDocumentIds } from "./billing-queue-packet";
import { BillingQueuePacketPreview } from "./billing-queue-packet-preview";

/** The documents an invoice packet carries; until the biller picks, the required documents. */
export type PacketControl = {
  documentIds: string[];
  /** The biller has picked the packet, so it is exactly documentIds, even when empty. */
  picked: boolean;
  editable: boolean;
  onChange: (documentIds: string[]) => Promise<unknown>;
  /** The queue item whose packet can be previewed as the one PDF it goes out as. */
  itemId?: string;
};

type Translate = ReturnType<typeof useT>;

/** The metadata key marking an upload started from a missing document's slot. */
const SLOT_KEY = "packetSlot";

const SETTLED_UPLOADS = new Set<UploadState["status"]>(["success", "error", "quarantined"]);

/** What a document needs from the biller before it can go out, if anything. */
function documentAttention(doc: Document, t: Translate): string | undefined {
  if (documentNeedsSignature(doc)) {
    if (doc.signatureStatus === "Unsigned") return t("Unsigned");
    if (doc.signatureStatus !== "Signed") return t("Signature not read");
  }
  if (doc.status === "Rejected") return t("Rejected");
  return undefined;
}

/**
 * The shipment's paperwork as the invoice will use it, laid out as a packet board:
 * which documents the customer requires and which of them are still missing, the
 * packet in the order it follows the invoice, what stays behind, and the selected
 * document's line with what can be done with it. Selecting several brings up the
 * dock, whose actions apply to all of them.
 */
export function BillingQueueDocumentsTab({
  shipmentId,
  onDocumentSelect,
  isEditable,
  packet,
  onRequestFromDriver,
  context = "billing-queue",
}: {
  shipmentId: string;
  onDocumentSelect: (docId: string, fileName: string) => void;
  isEditable?: boolean;
  /** Lets the biller choose which documents go out with the invoice, and in what order. */
  packet?: PacketControl;
  /** Asks the driver for the missing paperwork. */
  onRequestFromDriver?: () => void;
  context?: "billing-queue" | "invoice";
}) {
  const t = useT();
  const [uploadOpen, setUploadOpen] = useState(false);
  const [replacingLineageId, setReplacingLineageId] = useState<string | null>(null);
  const [signatureDocument, setSignatureDocument] = useState<Document | null>(null);
  const [packetOpen, setPacketOpen] = useState(false);
  const [busy, setBusy] = useState<ReadonlySet<string>>(() => new Set());
  const [selectionIds, setSelectionIds] = useState<ReadonlySet<string>>(() => new Set());
  const [deleting, setDeleting] = useState<readonly string[] | null>(null);
  const slotInput = useRef<HTMLInputElement>(null);
  const slotRequirement = useRef<ShipmentBillingRequirement | null>(null);
  const { allowed: canRecordSignature } = usePermission(Resource.Document, Operation.Update);
  const queryClient = useQueryClient();
  const docTypeForm = useForm<{ documentTypeId: string }>({
    defaultValues: { documentTypeId: "" },
  });
  const selectedDocTypeId = useWatch({ control: docTypeForm.control, name: "documentTypeId" });

  const documentsQuery = shipmentDocumentsQuery(shipmentId);
  const queryKey = documentsQuery.queryKey;
  const billingReadinessQuery = shipment.billingReadiness(shipmentId);
  const { data: billingReadiness } = useQuery({ ...billingReadinessQuery, enabled: !!shipmentId });
  const { data: documents = [], isLoading } = useQuery({
    ...documentsQuery,
    enabled: !!shipmentId,
  });

  const requirements = billingReadiness?.requirements ?? [];
  const selection: ReadonlySet<string> = new Set(
    documents.filter((doc) => selectionIds.has(doc.id)).map((doc) => doc.id),
  );
  const missing = requirements.filter((requirement) => !requirement.satisfied);
  const currentPacket = packet
    ? packetDocumentIds(packet.documentIds, packet.picked, requirements)
    : [];

  const uploadMetadata = useMemo((): Record<string, string> => {
    const metadata: Record<string, string> = {};
    if (selectedDocTypeId) metadata.documentTypeId = selectedDocTypeId;
    if (replacingLineageId) metadata.lineageId = replacingLineageId;
    return metadata;
  }, [selectedDocTypeId, replacingLineageId]);

  // Holds each card the request touches until it settles, and hands the request
  // back so a control that started it can hold too.
  const hold = (documentIds: readonly string[], work: () => Promise<unknown>) => {
    setBusy((current) => new Set([...current, ...documentIds]));
    return work()
      .then(() => undefined)
      .finally(() =>
        setBusy((current) => {
          const next = new Set(current);
          for (const id of documentIds) next.delete(id);
          return next;
        }),
      );
  };

  const savePacket = (documentIds: readonly string[], next: string[]) =>
    packet ? hold(documentIds, () => packet.onChange(next)) : Promise.resolve();

  const quietly = (request: Promise<void>) => {
    void request.catch(() => {
      // The request has already reported why it failed.
    });
  };

  // A file uploaded into a missing document's slot belongs in the packet. A packet
  // still on its default takes it with the requirement it satisfies; one the biller
  // has picked takes it last.
  const joinPacket = useLatestCallback((doc: Document, upload: UploadState) => {
    if (!upload.metadata?.[SLOT_KEY] || !packet?.editable || !packet.picked) {
      return;
    }
    quietly(savePacket([doc.id], packetAfterToggle(doc.id, true, currentPacket)));
  });

  const { mutateAsync: deleteDocuments } = useMutation({
    mutationFn: (documentIds: string[]) => apiService.documentService.bulkDelete(documentIds),
    onSuccess: ({ errorCount }) => {
      if (errorCount > 0) {
        toast.error(
          t("{0, plural, one {# document} other {# documents}} could not be deleted.", errorCount),
        );
      }
    },
    onError: () => toast.error(t("Failed to delete documents")),
    // A deleted document also leaves the packet that carried it, on the server.
    onSettled: () => {
      void queryClient.invalidateQueries({ queryKey });
      void queryClient.invalidateQueries({ queryKey: billingReadinessQuery.queryKey });
      if (packet?.itemId) {
        void queryClient.invalidateQueries({ queryKey: billingQueue.get(packet.itemId).queryKey });
      }
    },
  });

  const { uploads, uploadFiles, cancelUpload, retryUpload, removeUpload, clearCompleted } =
    useDocumentUpload({
      resourceId: shipmentId,
      resourceType: "shipment",
      uploadMetadata,
      invalidateQueryKey: queryKey,
      onSuccess: (doc, upload) => {
        docTypeForm.setValue("documentTypeId", "");
        setReplacingLineageId(null);
        void queryClient.invalidateQueries({ queryKey: billingReadinessQuery.queryKey });
        joinPacket(doc, upload);
      },
      onError: () => setUploadOpen(true),
    });

  const uploadable = (files: File[]) => {
    const { accepted, rejected } = sortUploadFiles(files);
    if (rejected.length > 0) {
      toast.error(
        t(
          "{0, plural, one {# file was} other {# files were}} not uploaded: only PDFs, images and office documents up to 50 MB can be.",
          rejected.length,
        ),
      );
    }
    return accepted;
  };

  const uploadInto = (requirement: ShipmentBillingRequirement, files: File[]) => {
    const accepted = uploadable(files);
    if (accepted.length === 0) return;
    uploadFiles(accepted, {
      documentTypeId: requirement.documentTypeId,
      [SLOT_KEY]: requirement.documentTypeId,
    });
  };

  const uploadDropped = (files: File[]) => {
    const accepted = uploadable(files);
    if (accepted.length === 0) return;
    setUploadOpen(true);
    uploadFiles(accepted, selectedDocTypeId ? { documentTypeId: selectedDocTypeId } : {});
  };

  const progressFor = (requirement: ShipmentBillingRequirement) => {
    const running = uploads.filter(
      (upload) =>
        upload.metadata?.[SLOT_KEY] === requirement.documentTypeId &&
        !SETTLED_UPLOADS.has(upload.status),
    );
    if (running.length === 0) return undefined;
    return Math.round(running.reduce((sum, upload) => sum + upload.progress, 0) / running.length);
  };

  const editablePacket: BoardPacket | undefined = packet
    ? {
        ids: currentPacket,
        editable: packet.editable,
        onPlace: (documentId, index) =>
          quietly(savePacket([documentId], packetAfterPlace(documentId, index, currentPacket))),
        onRemove: (documentId) =>
          quietly(savePacket([documentId], packetAfterToggle(documentId, false, currentPacket))),
        onArrange: (next, moved) => savePacket(moved, next),
      }
    : undefined;

  const selectedDocuments = documents.filter((doc) => selection.has(doc.id));
  const inPacket = new Set(currentPacket);
  const selectedIn = selectedDocuments.filter((doc) => inPacket.has(doc.id));
  const selectedOut = selectedDocuments.filter((doc) => !inPacket.has(doc.id));
  const packetEditable = Boolean(packet?.editable);

  const bulkActions: DockAction<Document>[] = [
    ...(packetEditable && selectedOut.length > 0
      ? [
          {
            id: "send",
            label: "Send with invoice",
            icon: ArrowUpIcon,
            clearSelectionOnSuccess: true,
            onClick: () =>
              savePacket(
                selectedOut.map((doc) => doc.id),
                [...currentPacket, ...selectedOut.map((doc) => doc.id)],
              ),
          },
        ]
      : []),
    ...(packetEditable && selectedIn.length > 0
      ? [
          {
            id: "leave-out",
            label: "Leave out",
            icon: ArrowDownIcon,
            clearSelectionOnSuccess: true,
            onClick: () =>
              savePacket(
                selectedIn.map((doc) => doc.id),
                currentPacket.filter((id) => !selection.has(id)),
              ),
          },
        ]
      : []),
    ...(isEditable
      ? [
          {
            id: "delete",
            label: "Delete",
            icon: Trash01Icon,
            variant: "destructive" as const,
            onClick: () => setDeleting(selectedDocuments.map((doc) => doc.id)),
          },
        ]
      : []),
  ];
  const selected = selectedDocuments.length === 1 ? selectedDocuments[0] : null;
  const selectedInPacket = selected ? inPacket.has(selected.id) : false;
  const isInvoice = context === "invoice";
  const canRequest = Boolean(onRequestFromDriver) && missing.length > 0;

  return (
    <section aria-label={t("Documents")} className="flex flex-col gap-4 text-sm">
      <input
        ref={slotInput}
        type="file"
        multiple
        hidden
        onChange={(event) => {
          const requirement = slotRequirement.current;
          const files = [...(event.currentTarget.files ?? [])];
          event.currentTarget.value = "";
          if (requirement && files.length > 0) uploadInto(requirement, files);
        }}
      />

      {isLoading ? (
        <BillingQueueDocumentBoardSkeleton />
      ) : documents.length === 0 && missing.length === 0 ? (
        <BillingQueueDocumentsEmpty
          title={isInvoice ? t("No supporting documents") : t("No documents yet")}
          description={
            isEditable
              ? t(
                  "Upload the proof of delivery, bill of lading and anything else this shipment is billed on. Pick a type first and the file is classified as it goes up.",
                )
              : isInvoice
                ? t(
                    "Documents attached to the shipment while it was in the billing queue travel with the invoice and show up here.",
                  )
                : t(
                    "Documents are added while the item is in review. Anything attached to the shipment from its own record shows up here too.",
                  )
          }
          onUpload={isEditable ? () => setUploadOpen(true) : undefined}
        />
      ) : (
        <>
          <div className="flex flex-wrap items-center gap-x-3 gap-y-2">
            <p className="text-foreground-muted flex min-w-48 flex-1 flex-wrap items-center gap-x-3.5 gap-y-1 text-xs">
              {requirements.length > 0 ? (
                <>
                  <span>{t("Requires")}</span>
                  {requirements.map((requirement) => (
                    <span
                      key={requirement.documentTypeId}
                      className={cn(
                        "inline-flex items-center gap-1.5",
                        !requirement.satisfied && "text-foreground",
                      )}
                    >
                      {requirement.satisfied ? (
                        <CheckIcon aria-hidden className="size-3" />
                      ) : (
                        <span aria-hidden className="bg-warning size-1.5 rounded-full" />
                      )}
                      {requirement.documentTypeName}
                      <span className="sr-only">
                        {requirement.satisfied ? t("received") : t("missing")}
                      </span>
                    </span>
                  ))}
                </>
              ) : null}
            </p>
            {packet?.itemId ? (
              <Button size="xs" variant="outline" onClick={() => setPacketOpen(true)}>
                <File06Icon className="size-3.5" />
                {t("Preview packet")}
              </Button>
            ) : null}
            {isEditable && canRequest ? (
              <DropdownMenu>
                <DropdownMenuTrigger
                  render={
                    <Button size="icon-xs" variant="outline" aria-label={t("Add documents")}>
                      <PlusIcon className="size-3.5" />
                    </Button>
                  }
                />
                <DropdownMenuContent align="end" className="w-48">
                  <DropdownMenuItem title={t("Upload files")} onClick={() => setUploadOpen(true)} />
                  <DropdownMenuItem
                    title={t("Request from driver")}
                    onClick={onRequestFromDriver}
                  />
                </DropdownMenuContent>
              </DropdownMenu>
            ) : isEditable ? (
              <Button
                size="icon-xs"
                variant="outline"
                aria-label={t("Upload files")}
                onClick={() => setUploadOpen(true)}
              >
                <PlusIcon className="size-3.5" />
              </Button>
            ) : null}
          </div>

          <BillingQueueDocumentBoard
            documents={documents}
            missing={missing}
            packet={editablePacket}
            selection={selection}
            onSelectionChange={setSelectionIds}
            multiSelect={packetEditable || Boolean(isEditable)}
            onOpen={(doc) => onDocumentSelect(doc.id, doc.originalName)}
            busy={busy}
            attentionOf={(doc) => documentAttention(doc, t)}
            progressFor={progressFor}
            onUploadFor={
              isEditable
                ? (requirement) => {
                    slotRequirement.current = requirement;
                    slotInput.current?.click();
                  }
                : undefined
            }
            onDropFor={isEditable ? uploadInto : undefined}
            onAddFiles={isEditable ? () => setUploadOpen(true) : undefined}
            onDropFiles={isEditable ? uploadDropped : undefined}
          />

          {selected ? (
            <DocumentDetail
              doc={selected}
              attention={documentAttention(selected, t)}
              busy={busy.has(selected.id)}
              inPacket={packet ? selectedInPacket : undefined}
              onView={() => onDocumentSelect(selected.id, selected.originalName)}
              onRecordSignature={
                canRecordSignature &&
                documentNeedsSignature(selected) &&
                selected.signatureStatus !== "Signed"
                  ? () => setSignatureDocument(selected)
                  : undefined
              }
              onMove={
                editablePacket?.editable
                  ? () =>
                      selectedInPacket
                        ? editablePacket.onRemove(selected.id)
                        : editablePacket.onPlace(selected.id, currentPacket.length)
                  : undefined
              }
              onReplace={
                isEditable
                  ? () => {
                      setReplacingLineageId(selected.lineageId);
                      setUploadOpen(true);
                    }
                  : undefined
              }
              onDelete={isEditable ? () => setDeleting([selected.id]) : undefined}
            />
          ) : selection.size === 0 && documents.length > 0 ? (
            <p className="border-border text-foreground-muted border-t pt-3.5 text-xs">
              {t(
                "Select a document to see its file and what you can do with it, or tick several to act on them together.",
              )}
            </p>
          ) : null}
        </>
      )}

      {packet?.itemId ? (
        <BillingQueuePacketPreview
          itemId={packet.itemId}
          open={packetOpen}
          onOpenChange={setPacketOpen}
        />
      ) : null}
      <SelectionDock
        selectedCount={selection.size > 1 ? selection.size : 0}
        actions={bulkActions}
        getSelected={() => selectedDocuments}
        onClearSelection={() => setSelectionIds(new Set())}
      />
      <DocumentsDeleteConfirm
        count={deleting?.length ?? 0}
        open={deleting !== null}
        onOpenChange={(open) => {
          if (!open) setDeleting(null);
        }}
        onConfirm={async () => {
          if (!deleting) return;
          const ids = [...deleting];
          await hold(ids, () => deleteDocuments(ids));
          setSelectionIds((current) => new Set([...current].filter((id) => !ids.includes(id))));
        }}
      />
      <DocumentSignatureDialog
        document={signatureDocument}
        open={signatureDocument !== null}
        onOpenChange={(open) => {
          if (!open) setSignatureDocument(null);
        }}
      />
      <UploadPanel
        isOpen={uploadOpen}
        onClose={() => {
          setUploadOpen(false);
          setReplacingLineageId(null);
        }}
        uploads={uploads}
        onFilesSelected={uploadFiles}
        onCancel={cancelUpload}
        onRetry={retryUpload}
        onRemove={removeUpload}
        onClearCompleted={clearCompleted}
        title={replacingLineageId ? t("Replace document") : t("Upload documents")}
        description={
          replacingLineageId
            ? t("Upload a new version to replace the existing document.")
            : selectedDocTypeId
              ? t("Uploads will be classified with the selected document type.")
              : t("Select a document type to classify uploads (optional).")
        }
        options={
          replacingLineageId ? null : (
            <DocumentTypeAutocompleteField
              control={docTypeForm.control}
              name="documentTypeId"
              placeholder={t("Document type (optional)")}
              clearable
            />
          )
        }
      />
    </section>
  );
}
