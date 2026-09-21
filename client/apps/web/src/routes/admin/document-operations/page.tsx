import { useT } from "@trenova/shared/i18n/use-t";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@trenova/shared/components/ui/alert-dialog";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Card, CardContent } from "@trenova/shared/components/ui/card";
import { DescriptionItem, DescriptionList } from "@trenova/shared/components/ui/description-list";
import { Input } from "@trenova/shared/components/ui/input";
import { Separator } from "@trenova/shared/components/ui/separator";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import { apiService } from "@/services/api";
import type { Document, DocumentUploadSession } from "@trenova/shared/types/document";
import type { DocumentOperationsDiagnostics, WorkflowReference } from "@/types/document-operations";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  AlertTriangleIcon,
  CheckIcon,
  ClockIcon,
  CopyIcon,
  FileSearchIcon,
  GitBranchIcon,
  ImageIcon,
  LayersIcon,
  RefreshCwIcon,
  SearchIcon,
  UploadIcon,
  WorkflowIcon,
  XCircleIcon,
} from "lucide-react";
import { type FormEvent, useState } from "react";
import { toast } from "sonner";
import { formatUnixDateTimeOrDash } from "@trenova/shared/lib/date";

function formatTimestamp(ts: number): string {
  return formatUnixDateTimeOrDash(ts);
}

function relativeTime(ts: number): string {
  if (!ts) return "";
  const diff = Math.floor(Date.now() / 1000 - ts);
  if (diff < 60) return `${diff}s ago`;
  if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
  if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
  return `${Math.floor(diff / 86400)}d ago`;
}

function statusVariant(status: string): "neutral" | "success" | "danger" | "warning" | "info" {
  switch (status) {
    case "Active":
    case "Completed":
    case "Available":
    case "Extracted":
    case "Indexed":
    case "Ready":
      return "success";
    case "Failed":
    case "Rejected":
    case "Canceled":
    case "Expired":
    case "Quarantined":
      return "warning";
    case "Pending":
    case "Extracting":
    case "Uploading":
    case "Verifying":
    case "Finalizing":
    case "Completing":
      return "info";
    case "Draft":
    case "Initiated":
    case "Paused":
    case "Unavailable":
      return "neutral";
    case "Unsupported":
      return "warning";
    default:
      return "neutral";
  }
}

function statusDotColor(status: string): string {
  switch (status) {
    case "Active":
    case "Completed":
    case "Available":
    case "Extracted":
    case "Indexed":
    case "Ready":
      return "bg-success";
    case "Failed":
    case "Rejected":
    case "Canceled":
    case "Expired":
    case "Quarantined":
      return "bg-danger";
    case "Pending":
    case "Extracting":
    case "Uploading":
    case "Verifying":
    case "Finalizing":
    case "Completing":
      return "bg-info";
    default:
      return "bg-muted-foreground";
  }
}

function CopyableId({ value, truncate = true }: { value: string; truncate?: boolean }) {
  const { copy, isCopied } = useCopyToClipboard();

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <button
            type="button"
            className="group bg-muted/50 hover:bg-muted inline-flex cursor-pointer items-center gap-1.5 rounded-md px-2 py-0.5 font-mono text-xs transition-colors"
            onClick={() => void copy(value, { timeout: 2000, withToast: true })}
          />
        }
      >
        <span className="truncate">
          {truncate && value.length > 20 ? `${value.slice(0, 10)}...${value.slice(-6)}` : value}
        </span>
        {isCopied ? (
          <CheckIcon className="size-3 shrink-0 text-success-foreground" />
        ) : (
          <CopyIcon className="size-3 shrink-0 opacity-0 transition-opacity group-hover:opacity-60" />
        )}
      </TooltipTrigger>
      <TooltipContent className="font-mono text-xs">{value}</TooltipContent>
    </Tooltip>
  );
}

function SectionHeader({
  icon: Icon,
  title,
  count,
}: {
  icon: React.ComponentType<{ className?: string }>;
  title: string;
  count?: number;
}) {
  return (
    <div className="flex items-center gap-2">
      <Icon className="text-muted-foreground size-3.5 shrink-0" />
      <h3 className="text-sm font-semibold">{title}</h3>
      {count != null && (
        <span className="bg-muted text-muted-foreground rounded-full px-1.5 py-0.5 text-2xs font-medium tabular-nums">
          {count}
        </span>
      )}
    </div>
  );
}

function DocumentSearch({
  onSearch,
  isLoading,
}: {
  onSearch: (id: string) => void;
  isLoading: boolean;
}) {
  const t = useT();

  const [input, setInput] = useState("");

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    const trimmed = input.trim();
    if (trimmed) {
      onSearch(trimmed);
    }
  }

  return (
    <form onSubmit={handleSubmit} className="flex items-center gap-2">
      <Input
        value={input}
        onChange={(e) => setInput(e.target.value)}
        placeholder={t("Paste a document ID to inspect...")}
        className="truncate pr-18 font-mono placeholder:font-sans"
        leftElement={<SearchIcon className="text-muted-foreground size-3.5" />}
        rightElement={
          <Button
            type="submit"
            size="xxxs"
            disabled={!input.trim() || isLoading}
            isLoading={isLoading}
          >
            {t("Inspect")}
          </Button>
        }
      />
    </form>
  );
}

function ActionButton({
  label,
  description,
  detail,
  icon: Icon,
  documentId,
  mutationFn,
  onSuccess,
}: {
  label: string;
  description: string;
  detail: string;
  icon: React.ComponentType<{ className?: string }>;
  documentId: string;
  mutationFn: (id: string) => Promise<void>;
  onSuccess: () => void;
}) {
  const t = useT();

  const mutation = useMutation({
    mutationFn,
    onSuccess: () => {
      toast.success(`${label} initiated`);
      onSuccess();
    },
    onError: (error) => {
      toast.error(error instanceof Error ? error.message : `${label} failed`);
    },
  });

  return (
    <AlertDialog>
      <AlertDialogTrigger
        render={
          <button
            type="button"
            disabled={mutation.isPending}
            className="group border-border/80 hover:border-border hover:bg-muted/30 flex min-w-0 flex-1 cursor-pointer flex-col items-start gap-2 rounded-lg border p-3 text-left transition-all disabled:pointer-events-none disabled:opacity-50"
          />
        }
      >
        <div className="flex w-full items-center justify-between">
          <span className="bg-muted text-muted-foreground group-hover:bg-background group-hover:text-foreground inline-flex size-7 items-center justify-center rounded-md transition-colors">
            <Icon className="size-3.5" />
          </span>
          {mutation.isPending && (
            <span className="relative flex size-2">
              <span className="absolute inline-flex size-full animate-ping rounded-full bg-info opacity-75" />
              <span className="relative inline-flex size-2 rounded-full bg-info" />
            </span>
          )}
        </div>
        <div>
          <div className="text-sm font-medium">{label}</div>
          <div className="text-muted-foreground mt-0.5 text-xs">{detail}</div>
        </div>
      </AlertDialogTrigger>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{label}?</AlertDialogTitle>
          <AlertDialogDescription>{description}</AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>{t("Cancel")}</AlertDialogCancel>
          <AlertDialogAction onClick={() => mutation.mutate(documentId)}>
            {t("Confirm")}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}

function StatusPipeline({ doc }: { doc: Document }) {
  const t = useT();

  const stages = [
    { label: t("Upload"), status: "Active" as const },
    { label: t("Preview"), status: doc.previewStatus },
    { label: t("Extraction"), status: doc.contentStatus },
    { label: t("Draft"), status: doc.shipmentDraftStatus },
  ];

  return (
    <div className="flex items-center gap-1">
      {stages.map((stage, i) => (
        <div key={stage.label} className="flex items-center gap-1">
          {i > 0 && <div className="bg-border h-px w-4" />}
          <Tooltip>
            <TooltipTrigger
              render={<div className="flex items-center gap-1.5 rounded-full border px-2 py-1" />}
            >
              <span className={`size-1.5 rounded-full ${statusDotColor(stage.status)}`} />
              <span className="text-xs font-medium">{t(stage.label)}</span>
            </TooltipTrigger>
            <TooltipContent>
              {t(stage.label)}: {stage.status}
            </TooltipContent>
          </Tooltip>
        </div>
      ))}
    </div>
  );
}

function DocumentOverviewSection({ doc }: { doc: Document }) {
  const t = useT();

  return (
    <section className="grid gap-3">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="flex items-center gap-3">
          <div className="min-w-0">
            <div className="flex items-center gap-2">
              <h2 className="truncate text-base font-semibold">{doc.originalName}</h2>
              <Badge variant={statusVariant(doc.status)}>{doc.status}</Badge>
              {doc.detectedKind && doc.detectedKind !== "Other" && (
                <Badge variant="info">{doc.detectedKind}</Badge>
              )}
            </div>
            <div className="text-muted-foreground mt-0.5 flex items-center gap-2 text-xs">
              <CopyableId value={doc.id} />
              <span>&middot;</span>
              <span>{t("{0} KB", (doc.fileSize / 1024).toFixed(1))}</span>
              <span>&middot;</span>
              <span>{doc.fileType}</span>
            </div>
          </div>
        </div>
        <StatusPipeline doc={doc} />
      </div>

      <DescriptionList columns={4}>
        <DescriptionItem label={t("Resource")}>
          <div className="flex items-center gap-1.5">
            <Badge variant="neutral" appearance="outline">
              {doc.resourceType}
            </Badge>
            <CopyableId value={doc.resourceId} />
          </div>
        </DescriptionItem>
        <DescriptionItem label={t("Version")}>
          <span className="font-mono">{t("v{0}", doc.versionNumber)}</span>
          {doc.isCurrentVersion && (
            <Badge variant="success" className="ml-1.5">
              {t("Current")}
            </Badge>
          )}
        </DescriptionItem>
        <DescriptionItem label={t("Created")}>
          <span>{formatTimestamp(doc.createdAt)}</span>
          <span className="text-muted-foreground ml-1">{relativeTime(doc.createdAt)}</span>
        </DescriptionItem>
        <DescriptionItem label={t("Updated")}>
          <span>{formatTimestamp(doc.updatedAt)}</span>
          <span className="text-muted-foreground ml-1">{relativeTime(doc.updatedAt)}</span>
        </DescriptionItem>
        <DescriptionItem label={t("Preview status")}>
          <Badge variant={statusVariant(doc.previewStatus)}>{doc.previewStatus}</Badge>
        </DescriptionItem>
        <DescriptionItem label={t("Content status")}>
          <Badge variant={statusVariant(doc.contentStatus)}>{doc.contentStatus}</Badge>
        </DescriptionItem>
        <DescriptionItem label={t("Draft status")}>
          <Badge variant={statusVariant(doc.shipmentDraftStatus)}>{doc.shipmentDraftStatus}</Badge>
        </DescriptionItem>
        <DescriptionItem label={t("Detected kind")}>
          {doc.detectedKind ? (
            <Badge variant="neutral">{doc.detectedKind}</Badge>
          ) : (
            <span className="text-muted-foreground">{t("Not classified")}</span>
          )}
        </DescriptionItem>
      </DescriptionList>
    </section>
  );
}

function ActionsSection({ documentId, onSuccess }: { documentId: string; onSuccess: () => void }) {
  const t = useT();

  return (
    <section className="grid gap-3">
      <SectionHeader icon={RefreshCwIcon} title={t("Recovery actions")} />
      <div className="grid gap-2.5 sm:grid-cols-3">
        <ActionButton
          label={t("Reextract content")}
          detail={t("Re-process text and structured data")}
          description={t(
            "Re-run content extraction for this document. This will re-process the document and update extracted text and structured data.",
          )}
          icon={FileSearchIcon}
          documentId={documentId}
          mutationFn={(id) => apiService.documentOperationsService.reextract(id)}
          onSuccess={onSuccess}
        />
        <ActionButton
          label={t("Regenerate preview")}
          detail={t("Start a new thumbnail workflow")}
          description={t(
            "Regenerate the document preview thumbnail. A new Temporal workflow will be started to generate the thumbnail.",
          )}
          icon={ImageIcon}
          documentId={documentId}
          mutationFn={(id) => apiService.documentOperationsService.regeneratePreview(id)}
          onSuccess={onSuccess}
        />
        <ActionButton
          label={t("Resync search")}
          detail={t("Update the search index projection")}
          description={t(
            "Re-sync this document's search index entry. This will update the search projection with the latest document data.",
          )}
          icon={RefreshCwIcon}
          documentId={documentId}
          mutationFn={(id) => apiService.documentOperationsService.resyncSearch(id)}
          onSuccess={onSuccess}
        />
      </div>
    </section>
  );
}

function PresenceSection({ hasContent, hasDraft }: { hasContent: boolean; hasDraft: boolean }) {
  const t = useT();

  return (
    <div className="grid gap-2.5 sm:grid-cols-2">
      <div
        className={`flex items-center gap-3 rounded-lg border p-3 ${hasContent ? "border-success-border bg-success-subtle" : "border-dashed"}`}
      >
        <span
          className={`inline-flex size-8 shrink-0 items-center justify-center rounded-md ${hasContent ? "bg-success-subtle text-success-foreground" : "bg-muted text-muted-foreground"}`}
        >
          <FileSearchIcon className="size-4" />
        </span>
        <div>
          <div className="text-sm font-medium">{t("Extracted content")}</div>
          <div className="text-muted-foreground text-xs">
            {hasContent ? t("Content available") : t("Not extracted yet")}
          </div>
        </div>
      </div>
      <div
        className={`flex items-center gap-3 rounded-lg border p-3 ${hasDraft ? "border-success-border bg-success-subtle" : "border-dashed"}`}
      >
        <span
          className={`inline-flex size-8 shrink-0 items-center justify-center rounded-md ${hasDraft ? "bg-success-subtle text-success-foreground" : "bg-muted text-muted-foreground"}`}
        >
          <LayersIcon className="size-4" />
        </span>
        <div>
          <div className="text-sm font-medium">{t("Shipment draft")}</div>
          <div className="text-muted-foreground text-xs">
            {hasDraft ? t("Draft available") : t("No draft generated")}
          </div>
        </div>
      </div>
    </div>
  );
}

function VersionsSection({ versions }: { versions: Document[] }) {
  const t = useT();

  if (versions.length === 0) return null;

  return (
    <section className="grid gap-3">
      <SectionHeader icon={GitBranchIcon} title={t("Version history")} count={versions.length} />
      <div className="grid gap-2">
        {versions.map((v) => (
          <div
            key={v.id}
            className={`flex items-center gap-3 rounded-lg border p-3 ${v.isCurrentVersion ? "border-brand/20 bg-brand/5" : ""}`}
          >
            <span className="bg-muted text-muted-foreground inline-flex size-8 shrink-0 items-center justify-center rounded-md font-mono text-xs font-semibold">
              {t("v{0}", v.versionNumber)}
            </span>
            <div className="min-w-0 flex-1">
              <div className="flex items-center gap-2">
                <CopyableId value={v.id} />
                {v.isCurrentVersion && <Badge variant="success">{t("Current")}</Badge>}
              </div>
              <div className="text-muted-foreground mt-0.5 flex items-center gap-1 text-xs">
                <ClockIcon className="size-3" />
                {formatTimestamp(v.createdAt)}
              </div>
            </div>
            <Badge variant={statusVariant(v.status)}>{v.status}</Badge>
          </div>
        ))}
      </div>
    </section>
  );
}

function SessionsSection({ sessions }: { sessions: DocumentUploadSession[] }) {
  const t = useT();

  if (sessions.length === 0) return null;

  return (
    <section className="grid gap-3">
      <SectionHeader icon={UploadIcon} title={t("Upload sessions")} count={sessions.length} />
      <div className="grid gap-2">
        {sessions.map((s) => {
          const hasFailure = !!(s.failureCode || s.failureMessage);
          return (
            <div
              key={s.id}
              className={`rounded-lg border p-3 ${hasFailure ? "border-danger-border" : ""}`}
            >
              <div className="flex items-center justify-between gap-3">
                <div className="flex items-center gap-2">
                  <span className={`size-2 shrink-0 rounded-full ${statusDotColor(s.status)}`} />
                  <CopyableId value={s.id} />
                  <Badge variant={statusVariant(s.status)}>{s.status}</Badge>
                </div>
                <div className="text-muted-foreground flex items-center gap-1 text-xs">
                  <ClockIcon className="size-3" />
                  {relativeTime(s.lastActivityAt)}
                </div>
              </div>

              <div className="mt-2.5 grid gap-2 md:grid-cols-3">
                <div className="text-xs">
                  <span className="text-muted-foreground">{t("Lineage")} </span>
                  {s.lineageId ? (
                    <CopyableId value={s.lineageId} />
                  ) : (
                    <span className="text-muted-foreground">-</span>
                  )}
                </div>
                <div className="text-xs">
                  <span className="text-muted-foreground">{t("Document")} </span>
                  {s.documentId ? (
                    <CopyableId value={s.documentId} />
                  ) : (
                    <span className="text-muted-foreground">-</span>
                  )}
                </div>
                <div className="text-muted-foreground text-xs">
                  {t("Created {0}", formatTimestamp(s.createdAt))}
                </div>
              </div>

              {hasFailure && (
                <div className="border-danger-border bg-danger-subtle text-destructive mt-2 rounded-md border px-2.5 py-1.5 font-mono text-xs">
                  {s.failureCode && <span className="font-semibold">{s.failureCode}: </span>}
                  {s.failureMessage}
                </div>
              )}
            </div>
          );
        })}
      </div>
    </section>
  );
}

function WorkflowsSection({ refs }: { refs: WorkflowReference[] }) {
  const t = useT();

  if (refs.length === 0) return null;

  return (
    <section className="grid gap-3">
      <SectionHeader icon={WorkflowIcon} title={t("Workflow references")} count={refs.length} />
      <div className="grid gap-2 sm:grid-cols-2">
        {refs.map((ref) => (
          <div
            key={`${ref.kind}:${ref.workflowId}`}
            className="flex items-center gap-3 rounded-lg border p-3"
          >
            <span className="bg-muted inline-flex size-7 shrink-0 items-center justify-center rounded-md">
              <WorkflowIcon className="text-muted-foreground size-3.5" />
            </span>
            <div className="min-w-0 flex-1">
              <div className="text-muted-foreground text-xs font-medium">
                {ref.kind.replace(/_/g, " ")}
              </div>
              <div className="mt-0.5">
                <CopyableId value={ref.workflowId} truncate={false} />
              </div>
            </div>
          </div>
        ))}
      </div>
    </section>
  );
}

function ErrorsBanner({ errors }: { errors: string[] }) {
  const t = useT();

  if (errors.length === 0) return null;

  return (
    <div className="border-danger-border bg-danger-subtle rounded-lg border p-3">
      <div className="text-destructive flex items-center gap-2 text-sm font-medium">
        <AlertTriangleIcon className="size-4" />
        {t("{0} {1} detected", errors.length, errors.length === 1 ? "error" : "errors")}
      </div>
      <div className="mt-2 space-y-1">
        {errors.map((err, i) => (
          <div
            key={i}
            className="bg-danger-subtle text-destructive rounded-md px-2.5 py-1.5 font-mono text-xs"
          >
            {err}
          </div>
        ))}
      </div>
    </div>
  );
}

function DiagnosticsView({ data }: { data: DocumentOperationsDiagnostics }) {
  const queryClient = useQueryClient();

  function handleActionSuccess() {
    void queryClient.invalidateQueries({
      queryKey: ["document-operations-diagnostics", data.document.id],
    });
  }

  return (
    <Card className="border-border/80 gap-0 overflow-hidden">
      <CardContent className="grid gap-6 p-5">
        <ErrorsBanner errors={data.lastErrors} />
        <DocumentOverviewSection doc={data.document} />
        <Separator />
        <PresenceSection hasContent={!!data.content} hasDraft={!!data.shipmentDraft} />
        <Separator />
        <ActionsSection documentId={data.document.id} onSuccess={handleActionSuccess} />
        {(data.versions.length > 0 || data.sessions.length > 0 || data.workflowRefs.length > 0) && (
          <>
            <Separator />
            <div className="grid gap-6 lg:grid-cols-2">
              <div className="grid gap-6">
                <VersionsSection versions={data.versions} />
                <WorkflowsSection refs={data.workflowRefs} />
              </div>
              <SessionsSection sessions={data.sessions} />
            </div>
          </>
        )}
      </CardContent>
    </Card>
  );
}

function DiagnosticsSkeleton() {
  return (
    <Card className="border-border/80 gap-0 overflow-hidden">
      <CardContent className="grid gap-6 p-5">
        <div className="flex items-center gap-3">
          <Skeleton className="size-10 rounded-lg" />
          <div className="flex-1 space-y-2">
            <Skeleton className="h-5 w-48" />
            <Skeleton className="h-3.5 w-72" />
          </div>
        </div>
        <div className="grid gap-2.5 md:grid-cols-4">
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-[72px] w-full rounded-lg" />
          ))}
        </div>
        <Separator />
        <div className="grid gap-2.5 sm:grid-cols-2">
          <Skeleton className="h-[60px] rounded-lg" />
          <Skeleton className="h-[60px] rounded-lg" />
        </div>
        <Separator />
        <div className="space-y-2">
          <Skeleton className="h-4 w-36" />
          <div className="grid gap-2.5 sm:grid-cols-3">
            <Skeleton className="h-[88px] rounded-lg" />
            <Skeleton className="h-[88px] rounded-lg" />
            <Skeleton className="h-[88px] rounded-lg" />
          </div>
        </div>
      </CardContent>
    </Card>
  );
}

export function DocumentOperationsPage() {
  const t = useT();

  const [documentId, setDocumentId] = useState<string | null>(null);

  const diagnosticsQuery = useQuery({
    queryKey: ["document-operations-diagnostics", documentId],
    queryFn: () => apiService.documentOperationsService.getDiagnostics(documentId!),
    enabled: !!documentId,
    retry: false,
  });

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Document operations"),
        description: t("Inspect document lifecycle state and trigger recovery actions"),
      }}
    >
      <DocumentSearch
        onSearch={(id) => setDocumentId(id)}
        isLoading={diagnosticsQuery.isFetching}
      />
      {!documentId && (
        <div className="flex flex-col items-center justify-center py-20">
          <div className="bg-muted flex size-14 items-center justify-center rounded-full">
            <FileSearchIcon className="text-muted-foreground size-7" />
          </div>
          <h3 className="mt-4 text-sm font-medium">{t("No document selected")}</h3>
          <p className="text-muted-foreground mt-1 max-w-[260px] text-center text-xs">
            {t(
              "Paste a document ID above to view its lifecycle state and available recovery actions",
            )}
          </p>
        </div>
      )}
      {documentId && diagnosticsQuery.isLoading && <DiagnosticsSkeleton />}
      {documentId && diagnosticsQuery.isError && (
        <Card className="border-border/80 gap-0 overflow-hidden">
          <CardContent className="flex flex-col items-center justify-center py-16">
            <div className="bg-danger-subtle flex size-12 items-center justify-center rounded-full">
              <XCircleIcon className="text-destructive size-6" />
            </div>
            <h3 className="mt-3 text-sm font-medium">{t("Failed to load diagnostics")}</h3>
            <p className="text-muted-foreground mt-1 max-w-[300px] text-center text-xs">
              {diagnosticsQuery.error instanceof Error
                ? diagnosticsQuery.error.message
                : t("Document not found or an unexpected error occurred")}
            </p>
            <Button
              variant="outline"
              size="xs"
              className="mt-4"
              onClick={() => diagnosticsQuery.refetch()}
            >
              <RefreshCwIcon className="size-3" />
              {t("Retry")}
            </Button>
          </CardContent>
        </Card>
      )}

      {diagnosticsQuery.data && <DiagnosticsView data={diagnosticsQuery.data} />}
    </PageLayout>
  );
}
