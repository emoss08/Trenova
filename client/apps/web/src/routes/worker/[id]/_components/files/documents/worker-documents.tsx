import { ConfirmDialog } from "@/components/confirm-dialog";
import { useCaptureKind } from "@/components/documents/capture/use-capture-kind";
import { DocumentStorageAlert } from "@/components/documents/document-storage-alert";
import { ExtractionNeedsProviderAlert } from "@/components/documents/extraction-needs-provider-alert";
import { PacketCompletenessPanel } from "@/components/documents/packet-completeness-panel";
import type { RowAction } from "@/components/row-actions-menu";
import { SearchInput } from "@/components/search-input";
import { SectionPanel } from "@/components/section-panel";
import { lazyWorkPane, preloadOn, useWorkPane } from "@/components/work-pane";
import { useDocumentUpload } from "@/hooks/use-document-upload";
import { useOpenedOnce } from "@/hooks/use-opened-once";
import { usePermission } from "@/hooks/use-permission";
import { canApproveDocument, canRejectDocument } from "@/lib/document-review";
import { documentNeedsSignature } from "@/lib/document-signature";
import { lazyComponent } from "@/lib/lazy-component";
import { PlanFeature } from "@/lib/plan-feature";
import { user as userQueries } from "@/lib/queries/user";
import type { WorkerTabProps } from "@/routes/worker/use-worker-state";
import { useQueries } from "@tanstack/react-query";
import { LazyComponent } from "@trenova/shared/components/error-boundary";
import { ErrorState } from "@trenova/shared/components/errors/error-state";
import {
  BrainCircuitIcon,
  CheckCircleIcon,
  CheckSquareIcon,
  ClockRewindIcon,
  DotsHorizontalIcon,
  Download01Icon,
  Edit05Icon,
  EyeIcon,
  SlashCircle01Icon,
  Trash01Icon,
  Upload01Icon,
  UploadCloud01Icon,
} from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@trenova/shared/components/ui/dropdown-menu";
import { EmptySheet } from "@trenova/shared/components/ui/empty-sheet";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import { usePlanFeature } from "@trenova/shared/hooks/use-plan-feature";
import { formatNumber } from "@trenova/shared/i18n/format";
import { useRichT } from "@trenova/shared/i18n/rich";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { Document } from "@trenova/shared/types/document";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useRef, useState } from "react";
import type { BoundedTableHandle } from "../../bounded-table";
import { LineSketch } from "../../list-sketch";
import { useRowFlash } from "../../use-row-flash";
import { useWorkerFormGate } from "../../use-worker-form-gate";
import { WORKER_FORM_PANES } from "../../worker-form-panes";
import { FilesSkeleton } from "../files-skeleton";
import { DOCUMENT_DIALOG_PRELOADS, DocumentDialogs } from "./document-dialogs";
import { DocumentTable } from "./document-table";
import {
  DOCUMENT_KIND_LABELS,
  DOCUMENT_KINDS,
  WORKER_DOCUMENT_RESOURCE,
  canPreviewDocument,
  filterDocuments,
  kindCounts,
  sortDocuments,
  type DocumentKind,
} from "./documents-model";
import { useWorkerDocuments } from "./use-worker-documents";

const UploadPane = lazyWorkPane(
  WORKER_FORM_PANES["upload-document"],
  (module) => module.UploadDocumentPane,
  {
    title: (props, t) => (props.lineageId ? t("Upload a new version") : t("Upload documents")),
  },
);
const DocumentPane = lazyWorkPane(
  () => import("./document-pane"),
  (module) => module.DocumentPane,
  {
    title: (props, t) => props.document?.originalName ?? t("Document"),
    openKey: (props) => props.document?.id ?? null,
    fields: 3,
  },
);
/*
 * Capture (scan, print into the record, cover sheets) is read only where the person may capture
 * into this record: its pieces and their dialogs stay out of the documents chunk.
 */
const loadCapture = () => import("@/components/documents/capture/capture-panel");
const CaptureButton = lazyComponent(loadCapture, (module) => module.CaptureButton);
const CaptureRequestsPanel = lazyComponent(loadCapture, (module) => module.CaptureRequestsPanel);
const CaptureStacksPanel = lazyComponent(loadCapture, (module) => module.CaptureStacksPanel);
const BulkActionDock = lazyComponent(
  () => import("@/components/documents/document-bulk-action-dock"),
  (module) => module.DocumentBulkActionDock,
);

type DocumentsViewProps = Pick<WorkerTabProps, "workerId" | "worker" | "open" | "onOpen">;

/**
 * The worker's documents as one bounded table: name, type, who added it, when, and its size,
 * with a search, a type filter and Upload above it. A file dragged over the table can be dropped
 * straight into the upload pane; a row opens the document beside the table, and its menu holds
 * every other thing the shared documents tab can do to it.
 */
export default function WorkerDocuments({ workerId, worker, open, onOpen }: DocumentsViewProps) {
  const t = useT();
  const rt = useRichT();
  const { closeThen } = useWorkPane();
  const captureKind = useCaptureKind(WORKER_DOCUMENT_RESOURCE);
  const readsDocuments = usePlanFeature(PlanFeature.DocumentIntelligence).available;
  const { allowed: canUpload } = usePermission(Resource.Document, Operation.Create);
  const { allowed: canDelete } = usePermission(Resource.Document, Operation.Delete);
  const { allowed: canApprove } = usePermission(Resource.Document, Operation.Approve);
  const { allowed: canReject } = usePermission(Resource.Document, Operation.Reject);
  const { allowed: canSign } = usePermission(Resource.Document, Operation.Update);

  const [search, setSearch] = useState("");
  const [kind, setKind] = useState<DocumentKind>("all");
  const [reading, setReading] = useState<string | null>(null);
  const [inspecting, setInspecting] = useState<Document | null>(null);
  const [versionOf, setVersionOf] = useState<Document | null>(null);
  const [rejecting, setRejecting] = useState<Document | null>(null);
  const [signing, setSigning] = useState<Document | null>(null);
  const [deleting, setDeleting] = useState<Document | null>(null);
  const [selecting, setSelecting] = useState(false);
  const [selected, setSelected] = useState<ReadonlySet<string>>(new Set());
  const [bulkConfirm, setBulkConfirm] = useState(false);
  const [dragging, setDragging] = useState(false);
  const [dropped, setDropped] = useState<readonly File[]>([]);
  const { flash, flashOf } = useRowFlash();
  const dragDepth = useRef(0);
  const tableRef = useRef<BoundedTableHandle>(null);

  const documents = useWorkerDocuments(workerId, versionOf);
  const uploader = useDocumentUpload({
    resourceId: workerId,
    resourceType: WORKER_DOCUMENT_RESOURCE,
    onSuccess: (document) => {
      flash(document.id);
      void documents.refresh().then(() => tableRef.current?.scrollToKey(document.id));
    },
  });
  const bulkDockOpened = useOpenedOnce(selecting);
  const all = sortDocuments(documents.documentsQuery.data ?? []);

  const uploaderIds = [...new Set(all.map((document) => document.uploadedById).filter(Boolean))];
  const uploaders = useQueries({
    queries: uploaderIds.map((id) => ({ ...userQueries.avatar(id), staleTime: 5 * 60 * 1000 })),
  });
  const names = new Map<string, string>();
  uploaders.forEach((result, index) => {
    if (result.data?.name) names.set(uploaderIds[index], result.data.name);
  });
  const uploaderName = (id: string) => names.get(id) ?? null;

  const uploadAllowed = useWorkerFormGate(open.form, { "upload-document": canUpload })(
    "upload-document",
  );

  if (documents.documentsQuery.isPending) return <FilesSkeleton view="documents" />;
  if (documents.documentsQuery.isError) {
    return (
      <ErrorState
        error={documents.documentsQuery.error}
        layout="sheet"
        title={t("The documents could not be loaded")}
        onRetry={() => void documents.documentsQuery.refetch()}
      />
    );
  }

  const counts = kindCounts(all);
  const shown = filterDocuments(all, kind, search, uploaderName);
  const filtered = search.trim() !== "" || kind !== "all";
  const readDocument = reading ? (all.find((document) => document.id === reading) ?? null) : null;
  const uploadOpen = open.form === "upload-document";
  const lineageId = uploadOpen ? open.kind : null;

  const openUpload = (lineage?: string) =>
    closeThen(() =>
      onOpen({
        tab: "files",
        view: "documents",
        form: "upload-document",
        ...(lineage ? { kind: lineage } : {}),
      }),
    );
  const closeUpload = () => {
    setDropped([]);
    onOpen({ tab: "files", view: "documents" });
  };
  const read = (document: Document) =>
    closeThen(() => {
      if (uploadOpen) closeUpload();
      setReading(document.id);
    });
  const clearFilters = () => {
    setSearch("");
    setKind("all");
  };
  const toggleAll = () =>
    setSelected(
      selected.size === shown.length ? new Set() : new Set(shown.map((document) => document.id)),
    );
  const toggle = (id: string) =>
    setSelected((current) => {
      const next = new Set(current);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });

  const rowActions = (document: Document): RowAction[] => {
    const actions: RowAction[] = [];
    if (canPreviewDocument(document)) {
      actions.push({
        id: "open",
        label: t("Open"),
        icon: EyeIcon,
        onSelect: () => void documents.open(document),
      });
    }
    actions.push({
      id: "download",
      label: t("Download"),
      icon: Download01Icon,
      onSelect: () => void documents.download(document),
    });
    if (readsDocuments) {
      actions.push({
        id: "inspect",
        label: t("What was read from it"),
        icon: BrainCircuitIcon,
        preload: DOCUMENT_DIALOG_PRELOADS.inspect,
        onSelect: () => setInspecting(document),
      });
    }
    actions.push({
      id: "versions",
      label: t("Versions"),
      icon: ClockRewindIcon,
      preload: DOCUMENT_DIALOG_PRELOADS.versions,
      onSelect: () => setVersionOf(document),
    });
    if (canUpload) {
      actions.push({
        id: "new-version",
        label: t("Upload a new version"),
        icon: Upload01Icon,
        preload: UploadPane.preload,
        onSelect: () => openUpload(document.lineageId),
      });
    }
    if (canApprove && canApproveDocument(document)) {
      actions.push({
        id: "approve",
        label: t("Approve"),
        icon: CheckCircleIcon,
        disabled: documents.approve.isPending,
        onSelect: () => void documents.approve.mutateAsync(document).catch(() => undefined),
      });
    }
    if (canReject && canRejectDocument(document)) {
      actions.push({
        id: "reject",
        label: t("Reject…"),
        icon: SlashCircle01Icon,
        preload: DOCUMENT_DIALOG_PRELOADS.reject,
        onSelect: () => setRejecting(document),
      });
    }
    if (canSign && documentNeedsSignature(document)) {
      actions.push({
        id: "signature",
        label: t("Record the signature"),
        icon: Edit05Icon,
        preload: DOCUMENT_DIALOG_PRELOADS.signature,
        onSelect: () => setSigning(document),
      });
    }
    if (canDelete) {
      actions.push({
        id: "delete",
        label: t("Delete…"),
        icon: Trash01Icon,
        destructive: true,
        disabled: documents.remove.isPending,
        onSelect: () => setDeleting(document),
      });
    }
    return actions;
  };

  const dropHandlers = canUpload
    ? {
        onDragEnter: (event: React.DragEvent) => {
          if (!event.dataTransfer.types.includes("Files")) return;
          event.preventDefault();
          UploadPane.preload().catch(() => undefined);
          dragDepth.current += 1;
          setDragging(true);
        },
        onDragOver: (event: React.DragEvent) => {
          if (event.dataTransfer.types.includes("Files")) event.preventDefault();
        },
        onDragLeave: () => {
          dragDepth.current = Math.max(0, dragDepth.current - 1);
          if (dragDepth.current === 0) setDragging(false);
        },
        onDrop: (event: React.DragEvent) => {
          event.preventDefault();
          dragDepth.current = 0;
          setDragging(false);
          const files = Array.from(event.dataTransfer.files);
          if (files.length === 0) return;
          setDropped(files);
          openUpload();
        },
      }
    : {};

  return (
    <div className="flex min-w-0 flex-col gap-4 text-sm">
      <DocumentStorageAlert />
      <ExtractionNeedsProviderAlert />
      {documents.packetSummary && documents.packetSummary.totalRules > 0 ? (
        <PacketCompletenessPanel summary={documents.packetSummary} />
      ) : null}
      {captureKind !== null ? (
        <LazyComponent fallback={null}>
          <CaptureRequestsPanel.Component kind={captureKind} recordId={workerId} />
          <CaptureStacksPanel.Component kind={captureKind} recordId={workerId} />
        </LazyComponent>
      ) : null}

      <SectionPanel
        title={t("Documents")}
        count={filtered ? undefined : all.length}
        hint={
          filtered && all.length > 0
            ? t("{0} of {1}", formatNumber(shown.length), formatNumber(all.length))
            : undefined
        }
        action={
          <div className="flex flex-wrap items-center justify-end gap-2">
            <SearchInput
              value={search}
              onValueChange={setSearch}
              placeholder={t("Search name, type or person")}
              aria-label={t("Search documents")}
              className="w-60"
            />
            <Select
              value={kind}
              onValueChange={(value) => setKind((value ?? "all") as DocumentKind)}
              items={DOCUMENT_KINDS.map((value) => ({
                value,
                label: t(DOCUMENT_KIND_LABELS[value]),
              }))}
            >
              <SelectTrigger className="min-w-32 text-sm" aria-label={t("Type")}>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {DOCUMENT_KINDS.filter((value) => value === "all" || counts[value] > 0).map(
                  (value) => (
                    <SelectItem key={value} value={value}>
                      <span className="flex w-full items-center justify-between gap-3 text-sm">
                        {t(DOCUMENT_KIND_LABELS[value])}
                        <span className="text-xs text-muted-foreground tabular-nums">
                          {formatNumber(counts[value])}
                        </span>
                      </span>
                    </SelectItem>
                  ),
                )}
              </SelectContent>
            </Select>
            {canDelete && all.length > 0 ? (
              <DropdownMenu>
                <DropdownMenuTrigger
                  render={
                    <Button
                      type="button"
                      size="icon-sm"
                      variant="ghost"
                      aria-label={t("More document actions")}
                    />
                  }
                >
                  <DotsHorizontalIcon className="size-4" />
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end" className="w-52">
                  <DropdownMenuItem
                    title={selecting ? t("Stop selecting") : t("Select documents")}
                    startContent={<CheckSquareIcon className="size-3.5" />}
                    onClick={() => {
                      setSelecting((current) => !current);
                      setSelected(new Set());
                    }}
                  />
                </DropdownMenuContent>
              </DropdownMenu>
            ) : null}
            {captureKind !== null ? (
              <LazyComponent
                fallback={<span aria-hidden className="block h-7 w-24 ui-shimmer rounded-md" />}
              >
                <CaptureButton.Component
                  kind={captureKind}
                  recordId={workerId}
                  disabled={!canUpload}
                />
              </LazyComponent>
            ) : null}
            {canUpload ? (
              <Button size="sm" {...preloadOn(UploadPane)} onClick={() => openUpload()}>
                <Upload01Icon className="size-3.5" />
                {t("Upload")}
              </Button>
            ) : null}
          </div>
        }
      >
        <div className="relative min-h-28" data-testid="documents-drop-target" {...dropHandlers}>
          {all.length === 0 ? (
            <EmptySheet
              title={t("No documents on file")}
              description={
                canUpload
                  ? t("Drop files here or upload them: licenses, medical cards, forms and reports.")
                  : t("Nothing has been added to this file yet.")
              }
              sketch={<LineSketch widths={["w-2/3", "w-1/2", "w-3/4"]} />}
              action={
                canUpload ? (
                  <Button size="sm" {...preloadOn(UploadPane)} onClick={() => openUpload()}>
                    <Upload01Icon className="size-3.5" />
                    {t("Upload")}
                  </Button>
                ) : undefined
              }
            />
          ) : shown.length === 0 ? (
            <div className="flex flex-col items-center gap-2 px-4 py-7 text-xs text-muted-foreground">
              <p>{t("No documents match.")}</p>
              <Button size="sm" variant="secondary" onClick={clearFilters}>
                {t("Clear the filters")}
              </Button>
            </div>
          ) : (
            <DocumentTable
              documents={shown}
              selecting={selecting}
              selected={selected}
              onToggle={toggle}
              onToggleAll={toggleAll}
              flashOf={flashOf}
              uploaderName={uploaderName}
              actionsFor={rowActions}
              onRead={read}
              handleRef={tableRef}
              rowIntent={preloadOn(DocumentPane)}
            />
          )}
          <div
            aria-hidden
            className={cn(
              "pointer-events-none absolute inset-1 z-20 flex flex-col items-center justify-center gap-1 rounded-md border border-dashed border-ring bg-card bg-linear-to-b from-brand-subtle to-brand-subtle text-center transition-[opacity,visibility] duration-150 ease-swift",
              dragging ? "opacity-100" : "invisible opacity-0",
            )}
          >
            <UploadCloud01Icon className="mb-1 size-5 text-brand" />
            <p className="text-xs text-muted-foreground">
              {rt(
                "<b>Drop to add it to {0}'s file</b>",
                { b: (chunk) => <b className="font-medium text-foreground">{chunk}</b> },
                worker.firstName,
              )}
            </p>
            <p className="text-xs text-muted-foreground">
              {t("PDF, images, Word, Excel and text files.")}
            </p>
          </div>
        </div>
      </SectionPanel>

      {bulkDockOpened ? (
        <LazyComponent fallback={null}>
          <BulkActionDock.Component
            selectedCount={selected.size}
            totalCount={shown.length}
            onDelete={() => setBulkConfirm(true)}
            onClearSelection={() => setSelected(new Set())}
            onSelectAll={toggleAll}
            isDeleting={documents.removeMany.isPending}
          />
        </LazyComponent>
      ) : null}

      <UploadPane.Component
        open={uploadAllowed}
        onOpenChange={(next) => !next && closeUpload()}
        workerName={worker.firstName}
        lineageId={lineageId}
        canUpload={canUpload}
        droppedFiles={dropped}
        uploader={uploader}
        onDone={closeUpload}
      />
      <DocumentPane.Component
        open={reading !== null && !uploadOpen}
        onOpenChange={(next) => !next && setReading(null)}
        document={readDocument}
        uploaderName={readDocument ? uploaderName(readDocument.uploadedById) : null}
        canDelete={canDelete}
        onOpenFile={documents.open}
        onDownload={documents.download}
        onDelete={setDeleting}
      />

      <ConfirmDialog
        open={deleting !== null}
        onOpenChange={(next) => !next && setDeleting(null)}
        title={t("Delete this document?")}
        description={
          deleting
            ? t(
                "{0} comes off {1}'s file. This cannot be undone.",
                deleting.originalName,
                worker.firstName,
              )
            : ""
        }
        confirmLabel={t("Delete document")}
        pending={documents.remove.isPending}
        onConfirm={() => {
          if (!deleting) return undefined;
          const target = deleting;
          return documents.remove.mutateAsync(target).then(() => {
            if (reading === target.id) setReading(null);
          });
        }}
      />
      <ConfirmDialog
        open={bulkConfirm}
        onOpenChange={setBulkConfirm}
        title={t(
          "{0, plural, one {Delete # document?} other {Delete # documents?}}",
          selected.size,
        )}
        description={t(
          "{0, plural, one {It comes} other {They come}} off {1}'s file. This cannot be undone.",
          selected.size,
          worker.firstName,
        )}
        confirmLabel={t(
          "{0, plural, one {Delete document} other {Delete documents}}",
          selected.size,
        )}
        pending={documents.removeMany.isPending}
        onConfirm={() =>
          documents.removeMany.mutateAsync([...selected]).then(() => {
            setSelected(new Set());
            setSelecting(false);
          })
        }
      />
      <DocumentDialogs
        workerId={workerId}
        inspecting={inspecting}
        onInspectingChange={setInspecting}
        rejecting={rejecting}
        onRejectingChange={setRejecting}
        signing={signing}
        onSigningChange={setSigning}
        versionOf={versionOf}
        onVersionOfChange={setVersionOf}
        documents={documents}
        onUploadNewVersion={(document) => {
          setVersionOf(null);
          openUpload(document.lineageId);
        }}
      />
    </div>
  );
}
