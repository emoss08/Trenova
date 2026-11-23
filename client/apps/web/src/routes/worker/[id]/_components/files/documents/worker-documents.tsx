import { ConfirmDialog } from "@/components/confirm-dialog";
import {
  CaptureButton,
  CaptureRequestsPanel,
  CaptureStacksPanel,
  useCaptureKind,
} from "@/components/documents/capture/capture-panel";
import { DocumentBulkActionDock } from "@/components/documents/document-bulk-action-dock";
import { DocumentFileTypeIcon } from "@/components/documents/document-file-type-icon";
import { DocumentIntelligenceDialog } from "@/components/documents/document-intelligence-dialog";
import { DocumentReviewBadge } from "@/components/documents/document-review-badge";
import { DocumentSignatureBadge } from "@/components/documents/document-signature-badge";
import { DocumentSignatureDialog } from "@/components/documents/document-signature-dialog";
import { DocumentStorageAlert } from "@/components/documents/document-storage-alert";
import { formatFileSize } from "@/components/documents/document-upload-zone";
import { DocumentVersionDialog } from "@/components/documents/document-version-dialog";
import { ExtractionNeedsProviderAlert } from "@/components/documents/extraction-needs-provider-alert";
import { PacketCompletenessPanel } from "@/components/documents/packet-completeness-panel";
import { RejectDocumentDialog } from "@/components/documents/reject-document-dialog";
import { HoverGlideGroup } from "@/components/hover-glide";
import { RowActionsMenu, type RowAction } from "@/components/row-actions-menu";
import { SectionPanel } from "@/components/section-panel";
import { useWorkPane } from "@/components/work-pane";
import { usePermission } from "@/hooks/use-permission";
import { canApproveDocument, canRejectDocument } from "@/lib/document-review";
import { documentNeedsSignature } from "@/lib/document-signature";
import { PlanFeature } from "@/lib/plan-feature";
import { user as userQueries } from "@/lib/queries/user";
import type { WorkerTabProps } from "@/routes/worker/use-worker-state";
import { useQueries } from "@tanstack/react-query";
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
  SearchLgIcon,
  SlashCircle01Icon,
  Trash01Icon,
  Upload01Icon,
  UploadCloud01Icon,
  XCloseIcon,
} from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { Checkbox } from "@trenova/shared/components/ui/checkbox";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@trenova/shared/components/ui/dropdown-menu";
import { EmptySheet, GhostLine } from "@trenova/shared/components/ui/empty-sheet";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
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
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import type { Document } from "@trenova/shared/types/document";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useEffect, useRef, useState } from "react";
import { FilesSkeleton } from "../files-skeleton";
import { DocumentPane } from "./document-pane";
import {
  DOCUMENT_KIND_LABELS,
  DOCUMENT_KINDS,
  canPreviewDocument,
  documentTypeName,
  filterDocuments,
  kindCounts,
  sortDocuments,
  type DocumentKind,
} from "./documents-model";
import { UploadDocumentPane } from "./upload-document-pane";
import { WORKER_DOCUMENT_RESOURCE, useWorkerDocuments } from "./use-worker-documents";

/** How long a row that just landed keeps its wash. */
const FLASH_MS = 1600;

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
  const [dropped, setDropped] = useState<File[]>([]);
  const [flashId, setFlashId] = useState<string | null>(null);
  const dragDepth = useRef(0);
  const scrollRef = useRef<HTMLDivElement>(null);

  const documents = useWorkerDocuments(workerId, versionOf);
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

  useEffect(() => {
    if (flashId === null) return undefined;
    const timer = window.setTimeout(() => setFlashId(null), FLASH_MS);
    return () => window.clearTimeout(timer);
  }, [flashId]);

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
  const closeUpload = () => onOpen({ tab: "files", view: "documents" });
  const read = (document: Document) =>
    closeThen(() => {
      if (uploadOpen) closeUpload();
      setReading(document.id);
    });
  const clearFilters = () => {
    setSearch("");
    setKind("all");
  };
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
        onSelect: () => setInspecting(document),
      });
    }
    actions.push({
      id: "versions",
      label: t("Versions"),
      icon: ClockRewindIcon,
      onSelect: () => setVersionOf(document),
    });
    if (canUpload) {
      actions.push({
        id: "new-version",
        label: t("Upload a new version"),
        icon: Upload01Icon,
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
        onSelect: () => setRejecting(document),
      });
    }
    if (canSign && documentNeedsSignature(document)) {
      actions.push({
        id: "signature",
        label: t("Record the signature"),
        icon: Edit05Icon,
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
        <CaptureRequestsPanel kind={captureKind} recordId={workerId} />
      ) : null}
      {captureKind !== null ? <CaptureStacksPanel kind={captureKind} recordId={workerId} /> : null}

      <SectionPanel
        title={t("Documents")}
        count={all.length}
        hint={
          filtered && all.length > 0
            ? t("{0} of {1}", formatNumber(shown.length), formatNumber(all.length))
            : undefined
        }
        action={
          <div className="flex flex-wrap items-center justify-end gap-2">
            <label className="flex h-7 w-60 items-center gap-1.5 rounded-md border ui-container-focus-ring border-input bg-card px-2 text-muted-foreground">
              <SearchLgIcon className="size-3.5 shrink-0" />
              <input
                value={search}
                onChange={(event) => setSearch(event.target.value)}
                onKeyDown={(event) => {
                  if (event.key === "Escape" && search !== "") {
                    event.stopPropagation();
                    setSearch("");
                  }
                }}
                placeholder={t("Search name, type or person")}
                aria-label={t("Search documents")}
                className="min-w-0 flex-1 bg-transparent text-sm text-foreground outline-none placeholder:text-foreground-subtle"
              />
              {search !== "" ? (
                <Button
                  type="button"
                  size="icon-xs"
                  variant="ghost"
                  aria-label={t("Clear the search")}
                  onClick={() => setSearch("")}
                >
                  <XCloseIcon className="size-3" />
                </Button>
              ) : null}
            </label>
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
              <CaptureButton kind={captureKind} recordId={workerId} disabled={!canUpload} />
            ) : null}
            {canUpload ? (
              <Button size="sm" onClick={() => openUpload()}>
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
              sketch={
                <div className="flex flex-col gap-2 rounded-md border bg-card p-4">
                  <GhostLine className="w-2/3" />
                  <GhostLine className="w-1/2" />
                  <GhostLine className="w-3/4" />
                </div>
              }
              action={
                canUpload ? (
                  <Button size="sm" onClick={() => openUpload()}>
                    <Upload01Icon className="size-3.5" />
                    {t("Upload")}
                  </Button>
                ) : undefined
              }
            />
          ) : (
            <ScrollArea
              viewportClassName="max-h-[32rem]"
              viewportRef={scrollRef}
              maskVariant="card"
            >
              <HoverGlideGroup>
                <table className="w-full min-w-[46.5rem] table-fixed text-sm">
                  <thead className="sticky top-0 z-10 bg-card text-left text-xs text-muted-foreground">
                    <tr className="border-b border-border-subtle">
                      {selecting ? (
                        <th className="w-10 py-2 pl-3">
                          <Checkbox
                            checked={shown.length > 0 && selected.size === shown.length}
                            onCheckedChange={() =>
                              setSelected(
                                selected.size === shown.length
                                  ? new Set()
                                  : new Set(shown.map((document) => document.id)),
                              )
                            }
                            aria-label={t("Select every document shown")}
                          />
                        </th>
                      ) : null}
                      <th className="px-3 py-2 font-medium">{t("Name")}</th>
                      <th className="w-28 px-3 py-2 font-medium">{t("Type")}</th>
                      <th className="w-40 px-3 py-2 font-medium">{t("Added by")}</th>
                      <th className="w-28 px-3 py-2 font-medium">{t("Added")}</th>
                      <th className="w-20 px-3 py-2 text-right font-medium">{t("Size")}</th>
                      <th className="w-12 px-2 py-2">
                        <span className="sr-only">{t("Actions")}</span>
                      </th>
                    </tr>
                  </thead>
                  <tbody>
                    {shown.map((document) => (
                      <tr
                        key={document.id}
                        data-glide-target
                        data-testid={`document-${document.id}`}
                        tabIndex={0}
                        aria-label={document.originalName}
                        onClick={() => read(document)}
                        onKeyDown={(event) => {
                          if (event.target !== event.currentTarget) return;
                          if (event.key === "Enter" || event.key === " ") {
                            event.preventDefault();
                            read(document);
                          }
                        }}
                        className={cn(
                          "cursor-pointer border-b border-border-subtle outline-none last:border-0 focus-visible:bg-surface-hover",
                          flashId === document.id && "motion-safe:animate-row-changed-a",
                        )}
                      >
                        {selecting ? (
                          <td className="py-2 pl-3" onClick={(event) => event.stopPropagation()}>
                            <Checkbox
                              checked={selected.has(document.id)}
                              onCheckedChange={() => toggle(document.id)}
                              aria-label={t("Select {0}", document.originalName)}
                            />
                          </td>
                        ) : null}
                        <td className="px-3 py-2">
                          <span className="flex min-w-0 items-center gap-2">
                            <DocumentFileTypeIcon
                              fileType={document.fileType}
                              fileName={document.originalName}
                              size="sm"
                              className="size-6 rounded-md"
                            />
                            <span
                              className="truncate text-sm font-medium"
                              title={document.originalName}
                            >
                              {document.originalName}
                            </span>
                            <DocumentReviewBadge document={document} />
                            {documentNeedsSignature(document) ? (
                              <DocumentSignatureBadge document={document} />
                            ) : null}
                          </span>
                        </td>
                        <td className="truncate px-3 py-2 text-sm">{documentTypeName(document)}</td>
                        <td className="truncate px-3 py-2 text-sm">
                          {uploaderName(document.uploadedById) ?? (
                            <span className="text-muted-foreground">{t("Not known")}</span>
                          )}
                        </td>
                        <td className="px-3 py-2 font-mono text-xs whitespace-nowrap">
                          {formatUnixDateMedium(document.createdAt)}
                        </td>
                        <td className="px-3 py-2 text-right font-mono text-xs whitespace-nowrap">
                          {formatFileSize(document.fileSize)}
                        </td>
                        <td
                          className="px-2 py-1.5 text-right"
                          onClick={(event) => event.stopPropagation()}
                          onKeyDown={(event) => event.stopPropagation()}
                        >
                          <RowActionsMenu
                            label={t("Actions for {0}", document.originalName)}
                            actions={rowActions(document)}
                          />
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </HoverGlideGroup>
              {shown.length === 0 ? (
                <div className="flex flex-col items-center gap-2 px-4 py-7">
                  <p className="text-xs text-muted-foreground">{t("No documents match.")}</p>
                  <Button size="xs" variant="outline" onClick={clearFilters}>
                    {t("Clear the filters")}
                  </Button>
                </div>
              ) : null}
            </ScrollArea>
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

      {selecting ? (
        <DocumentBulkActionDock
          selectedCount={selected.size}
          totalCount={shown.length}
          onDelete={() => setBulkConfirm(true)}
          onClearSelection={() => setSelected(new Set())}
          onSelectAll={() =>
            setSelected(
              selected.size === shown.length
                ? new Set()
                : new Set(shown.map((document) => document.id)),
            )
          }
          isDeleting={documents.removeMany.isPending}
        />
      ) : null}

      <UploadDocumentPane
        open={uploadOpen && canUpload}
        onOpenChange={(next) => !next && closeUpload()}
        workerId={workerId}
        workerName={worker.firstName}
        lineageId={lineageId}
        canUpload={canUpload}
        dropped={dropped}
        onDroppedTaken={() => setDropped([])}
        onUploaded={(document) => {
          setFlashId(document.id);
          scrollRef.current?.scrollTo({ top: 0 });
          void documents.refresh();
        }}
        onDone={closeUpload}
      />
      <DocumentPane
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
      <DocumentIntelligenceDialog
        open={readsDocuments && inspecting !== null}
        onOpenChange={(next) => !next && setInspecting(null)}
        document={inspecting}
        resourceType={WORKER_DOCUMENT_RESOURCE}
        resourceId={workerId}
      />
      <RejectDocumentDialog
        open={rejecting !== null}
        onOpenChange={(next) => !next && setRejecting(null)}
        document={rejecting}
        onRejected={() => void documents.refresh()}
      />
      <DocumentSignatureDialog
        open={signing !== null}
        onOpenChange={(next) => !next && setSigning(null)}
        document={signing}
        onSaved={() => void documents.refresh()}
      />
      <DocumentVersionDialog
        open={versionOf !== null}
        onOpenChange={(next) => !next && setVersionOf(null)}
        document={versionOf}
        versions={documents.versions}
        isLoading={documents.versionsLoading}
        isRestoring={documents.restore.isPending}
        onRestore={(document) =>
          void documents.restore.mutateAsync(document).catch(() => undefined)
        }
        onUploadNewVersion={(document) => {
          setVersionOf(null);
          openUpload(document.lineageId);
        }}
      />
    </div>
  );
}
