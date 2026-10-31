import { useT } from "@trenova/shared/i18n/use-t";
import { DocumentTypeAutocompleteField } from "@/components/autocomplete-fields";
import { UploadPanel } from "@/components/documents/upload-panel";
import { Button } from "@trenova/shared/components/ui/button";
import { Checkbox } from "@trenova/shared/components/ui/checkbox";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useDocumentUpload } from "@/hooks/use-document-upload";
import { shipment } from "@/lib/queries/shipment";
import { cn, formatFileSize } from "@trenova/shared/lib/utils";
import { useNow } from "@trenova/shared/hooks/use-now";
import { formatCompactAge } from "@trenova/shared/lib/date";
import { apiService } from "@/services/api";
import { documentContentUrl } from "@/services/document";
import { LazyImage } from "@/components/image";
import type { ShipmentBillingRequirement } from "@trenova/shared/types/shipment";
import type { Document } from "@trenova/shared/types/document";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { RefreshCw02Icon, Trash01Icon, UploadCloud02Icon } from "@trenova/shared/components/icons";
import { useMemo, useState, type ReactNode } from "react";
import { useForm, useWatch } from "react-hook-form";
import { toast } from "@trenova/shared/components/ui/toast";
import { UnmetRequirementNote } from "@/components/documents/unmet-requirement-note";
import { DocumentSignatureDialog } from "@/components/documents/document-signature-dialog";
import { documentNeedsSignature } from "@/lib/document-signature";
import { usePermission } from "@/hooks/use-permission";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { shipmentDocumentsQuery } from "../billing-queue-queries";
import { BillingQueueDocumentsEmpty } from "./billing-queue-documents-empty";
import { BillingQueueLaneSkeleton } from "./billing-queue-list-skeleton";
import { isInPacket, packetAfterToggle } from "./billing-queue-packet";
import { BillingQueuePacketPreview } from "./billing-queue-packet-preview";

/** The documents an invoice packet carries; empty means the required documents. */
export type PacketControl = {
  documentIds: string[];
  editable: boolean;
  onChange: (documentIds: string[]) => void;
  /** The queue item whose packet can be previewed as the one PDF it goes out as. */
  itemId?: string;
};

type DocState = { label: string; tone: "success" | "warning" | "neutral" };

function documentState(doc: Document, t: ReturnType<typeof useT>): DocState {
  if (documentNeedsSignature(doc)) {
    if (doc.signatureStatus === "Signed") return { label: t("Signed"), tone: "success" };
    if (doc.signatureStatus === "Unsigned") return { label: t("Unsigned"), tone: "warning" };
    return { label: t("Signature not read"), tone: "neutral" };
  }
  if (doc.status === "Rejected") return { label: t("Rejected"), tone: "warning" };
  return { label: t("Received"), tone: "success" };
}

/**
 * The shipment's paperwork as the invoice will use it: every document with whether
 * the customer requires it and what state it is in, each required type that is still
 * missing, which of them go out with the invoice, and a place to add more.
 */
export function BillingQueueDocumentsTab({
  shipmentId,
  selectedDocumentId,
  onDocumentSelect,
  isEditable,
  packet,
  onRequestFromDriver,
  customerName,
  context = "billing-queue",
}: {
  shipmentId: string;
  selectedDocumentId: string | null;
  onDocumentSelect: (docId: string, fileName: string) => void;
  isEditable?: boolean;
  /** Lets the biller choose which documents go out with the invoice. */
  packet?: PacketControl;
  /** Asks the driver for the missing paperwork. */
  onRequestFromDriver?: () => void;
  /** Who requires the paperwork, named in the heading. */
  customerName?: string;
  context?: "billing-queue" | "invoice";
}) {
  const t = useT();
  const now = useNow("minute");
  const [uploadOpen, setUploadOpen] = useState(false);
  const [replacingLineageId, setReplacingLineageId] = useState<string | null>(null);
  const [signatureDocument, setSignatureDocument] = useState<Document | null>(null);
  const [packetOpen, setPacketOpen] = useState(false);
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

  const uploadMetadata = useMemo((): Record<string, string> => {
    const metadata: Record<string, string> = {};
    if (selectedDocTypeId) metadata.documentTypeId = selectedDocTypeId;
    if (replacingLineageId) metadata.lineageId = replacingLineageId;
    return metadata;
  }, [selectedDocTypeId, replacingLineageId]);

  const invalidateAll = () => {
    void queryClient.invalidateQueries({ queryKey });
    void queryClient.invalidateQueries({ queryKey: billingReadinessQuery.queryKey });
  };

  const { mutate: deleteDocument } = useMutation({
    mutationFn: (documentId: string) => apiService.documentService.delete(documentId),
    onSuccess: invalidateAll,
    onError: () => toast.error(t("Failed to delete document")),
  });

  const { uploads, uploadFiles, cancelUpload, retryUpload, removeUpload, clearCompleted } =
    useDocumentUpload({
      resourceId: shipmentId,
      resourceType: "shipment",
      uploadMetadata,
      invalidateQueryKey: queryKey,
      onSuccess: () => {
        docTypeForm.setValue("documentTypeId", "");
        setReplacingLineageId(null);
        void queryClient.invalidateQueries({ queryKey: billingReadinessQuery.queryKey });
      },
    });

  const requirements = billingReadiness?.requirements ?? [];
  const requiredTypes = new Set(requirements.map((requirement) => requirement.documentTypeId));
  const missing = requirements.filter((requirement) => !requirement.satisfied);
  const inPacket = (doc: Document) =>
    packet ? isInPacket(doc, packet.documentIds, requiredTypes) : false;
  const togglePacket = (doc: Document, include: boolean) => {
    if (!packet) return;
    packet.onChange(packetAfterToggle(doc, include, documents, packet.documentIds, requiredTypes));
  };
  const packetNames = packet
    ? documents.filter(inPacket).map((doc) => doc.documentType?.name ?? doc.originalName)
    : [];

  const uploadFor = (requirement: ShipmentBillingRequirement) => {
    docTypeForm.setValue("documentTypeId", requirement.documentTypeId);
    setUploadOpen(true);
  };

  const isInvoice = context === "invoice";
  const rule =
    requirements.length > 0
      ? customerName
        ? t(
            "{0} requires {1}",
            customerName,
            requirements.map((r) => r.documentTypeName).join(", "),
          )
        : t("Requires {0}", requirements.map((r) => r.documentTypeName).join(", "))
      : null;

  return (
    <section aria-label={t("Documents")} className="flex flex-col gap-2 text-sm">
      <div className="flex flex-wrap items-center gap-2">
        <h3 className="text-foreground font-semibold">
          {isInvoice ? t("Supporting documents") : t("Documents")}
        </h3>
        {rule ? <span className="text-foreground-muted text-xs">{rule}</span> : null}
        <span className="flex-1" />
        {packet?.itemId ? (
          <Button size="xs" variant="outline" onClick={() => setPacketOpen(true)}>
            {t("Preview invoice packet")}
          </Button>
        ) : null}
      </div>

      {isLoading ? (
        <div className="border-border rounded-md border" aria-label={t("Loading documents...")}>
          <BillingQueueLaneSkeleton />
        </div>
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
        <div className="border-border rounded-md border">
          {missing.map((requirement) => (
            <DocRow
              key={`missing-${requirement.documentTypeId}`}
              title={requirement.documentTypeName}
              required
              state={{ label: t("Missing"), tone: "warning" }}
              meta={<UnmetRequirementNote requirement={requirement} />}
              fix={
                isEditable ? (
                  <Button size="xs" onClick={() => uploadFor(requirement)}>
                    {t("Upload")}
                  </Button>
                ) : null
              }
            />
          ))}
          {documents.map((doc) => {
            const state = documentState(doc, t);
            const unsigned = documentNeedsSignature(doc) && doc.signatureStatus !== "Signed";
            return (
              <DocRow
                key={doc.id}
                selected={doc.id === selectedDocumentId}
                previewUrl={
                  doc.previewStatus === "Ready" ? documentContentUrl(doc.id, "preview") : undefined
                }
                title={doc.documentType?.name ?? doc.originalName}
                required={Boolean(doc.documentTypeId && requiredTypes.has(doc.documentTypeId))}
                state={state}
                meta={t(
                  "{0} · {1} · {2} ago",
                  doc.originalName,
                  formatFileSize(doc.fileSize),
                  formatCompactAge(now - doc.createdAt),
                )}
                fix={
                  unsigned && canRecordSignature ? (
                    <Button size="xs" onClick={() => setSignatureDocument(doc)}>
                      {t("Record a signature")}
                    </Button>
                  ) : null
                }
                packet={
                  packet ? (
                    <label className="text-foreground-muted flex items-center gap-1.5 text-xs whitespace-nowrap">
                      <Checkbox
                        checked={inPacket(doc)}
                        disabled={!packet.editable}
                        onCheckedChange={(checked) => togglePacket(doc, checked === true)}
                        aria-label={t("Send {0} with the invoice", doc.originalName)}
                      />
                      {t("In packet")}
                    </label>
                  ) : null
                }
                actions={
                  <>
                    {isEditable ? (
                      <span className="flex opacity-0 transition-opacity group-focus-within:opacity-100 group-hover:opacity-100">
                        <IconAction
                          label={t("Replace")}
                          onClick={() => {
                            setReplacingLineageId(doc.lineageId);
                            setUploadOpen(true);
                          }}
                        >
                          <RefreshCw02Icon className="size-3" />
                        </IconAction>
                        <IconAction label={t("Delete")} onClick={() => deleteDocument(doc.id)}>
                          <Trash01Icon className="size-3" />
                        </IconAction>
                      </span>
                    ) : null}
                    <Button
                      size="xs"
                      variant="outline"
                      onClick={() => onDocumentSelect(doc.id, doc.originalName)}
                    >
                      {t("View")}
                    </Button>
                  </>
                }
              />
            );
          })}
          {isEditable ? (
            <div className="border-border text-foreground-muted m-2.5 flex flex-wrap items-center gap-2.5 rounded-md border border-dashed p-3.5">
              <UploadCloud02Icon className="size-4" />
              <span className="min-w-40 flex-1">
                {t("Add files and pick a type for each, or choose a type first.")}
              </span>
              <div className="w-48">
                <DocumentTypeAutocompleteField
                  control={docTypeForm.control}
                  name="documentTypeId"
                  placeholder={t("Document type (optional)")}
                  clearable
                />
              </div>
              <Button size="xs" variant="outline" onClick={() => setUploadOpen(true)}>
                {t("Choose files")}
              </Button>
              {onRequestFromDriver && missing.length > 0 ? (
                <Button size="xs" variant="ghost" onClick={onRequestFromDriver}>
                  {t("Request from driver")}
                </Button>
              ) : null}
            </div>
          ) : null}
        </div>
      )}

      {packet && packetNames.length > 0 ? (
        <p className="text-foreground-muted text-xs">
          {t("Packet goes out as one PDF: {0}", [t("Invoice"), ...packetNames].join(" · "))}
        </p>
      ) : null}

      {packet?.itemId ? (
        <BillingQueuePacketPreview
          itemId={packet.itemId}
          open={packetOpen}
          onOpenChange={setPacketOpen}
        />
      ) : null}
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
      />
    </section>
  );
}

/**
 * A document's first page in the row, as the server drew it, so the biller sees which
 * paper it is before opening it. Until the preview exists, or for a file it cannot draw,
 * the row shows a blank page.
 */
function DocPage({ previewUrl }: { previewUrl?: string }) {
  const [failed, setFailed] = useState<string | null>(null);

  if (previewUrl && failed !== previewUrl) {
    return (
      <LazyImage
        src={previewUrl}
        alt=""
        onError={() => setFailed(previewUrl)}
        wrapperClassName="flex shrink-0"
        className="border-border bg-card h-9.5 w-7.5 rounded-xs border object-cover object-top"
      />
    );
  }

  return (
    <span
      aria-hidden
      className="border-border bg-card flex h-9.5 w-7.5 shrink-0 flex-col gap-0.75 rounded-xs border px-1.25 py-1.5"
    >
      <span className="bg-border h-0.5" />
      <span className="bg-border h-0.5 w-7/10" />
      <span className="bg-border h-0.5" />
      <span className="bg-border h-0.5 w-1/2" />
    </span>
  );
}

function DocRow({
  title,
  required,
  state,
  meta,
  fix,
  packet,
  actions,
  previewUrl,
  selected = false,
}: {
  /** The document's first page, drawn by the server; absent for a document still missing. */
  previewUrl?: string;
  title: string;
  required: boolean;
  state: DocState;
  meta: ReactNode;
  fix?: ReactNode;
  packet?: ReactNode;
  actions?: ReactNode;
  selected?: boolean;
}) {
  const t = useT();

  return (
    <div
      className={cn(
        "group border-border flex items-center gap-3 border-b px-2.5 py-2.5",
        selected && "bg-surface-selected",
      )}
    >
      <DocPage previewUrl={previewUrl} />
      <span className="flex min-w-0 flex-1 flex-col gap-0.5">
        <span className="flex flex-wrap items-center gap-1.5">
          <span className="text-foreground font-medium">{title}</span>
          {required ? (
            <span className="bg-muted text-foreground-muted rounded-xs px-1.25 text-2xs font-medium">
              {t("Required")}
            </span>
          ) : null}
          <span
            className={cn(
              "rounded-xs px-1.25 text-2xs font-medium",
              state.tone === "success" && "bg-success-subtle text-success-foreground",
              state.tone === "warning" && "bg-warning-subtle text-warning-foreground",
              state.tone === "neutral" && "bg-muted text-foreground-muted",
            )}
          >
            {state.label}
          </span>
        </span>
        <span className="text-foreground-muted truncate text-xs">{meta}</span>
      </span>
      {fix}
      {packet}
      {actions}
    </div>
  );
}

function IconAction({
  label,
  onClick,
  children,
}: {
  label: string;
  onClick: () => void;
  children: ReactNode;
}) {
  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <Button size="icon-xs" variant="ghost" aria-label={label} onClick={onClick}>
            {children}
          </Button>
        }
      />
      <TooltipContent side="top" sideOffset={10}>
        {label}
      </TooltipContent>
    </Tooltip>
  );
}
