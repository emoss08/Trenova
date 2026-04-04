import { DocumentShipmentDraftReviewDialog } from "@/components/documents/document-shipment-draft-review-dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { ScrollArea } from "@/components/ui/scroll-area";
import { apiService } from "@/services/api";
import type {
  Document,
  DocumentContent,
  DocumentIntelligenceAIDiagnostics,
  DocumentIntelligenceConflict,
  DocumentIntelligence,
  DocumentIntelligenceField,
  DocumentIntelligenceStop,
  DocumentShipmentDraft,
} from "@/types/document";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  AlertCircleIcon,
  LoaderCircleIcon,
  RefreshCcwIcon,
  SparklesIcon,
} from "lucide-react";
import { useState } from "react";
import { Link } from "react-router";
import { toast } from "sonner";

interface DocumentIntelligenceDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  document: Document | null;
  resourceType: string;
  resourceId: string;
}

function statusBadgeVariant(status: Document["contentStatus"]) {
  switch (status) {
    case "Indexed":
      return "active";
    case "Extracting":
      return "warning";
    case "Failed":
      return "outline";
    default:
      return "secondary";
  }
}

function formatValue(value: unknown): string {
  if (value == null) return "Not available";
  if (typeof value === "string") {
    return value.trim() === "" ? "Not available" : value;
  }
  if (typeof value === "number" || typeof value === "boolean") {
    return String(value);
  }
  if (Array.isArray(value)) {
    return value.length === 0
      ? "Not available"
      : value.map(formatValue).join(", ");
  }
  return JSON.stringify(value, null, 2);
}

function formatConfidence(confidence?: number | null) {
  if (confidence == null || Number.isNaN(confidence)) return "Not scored";
  return `${Math.round(confidence * 100)}%`;
}

function formatUnixTimestamp(value?: number | null) {
  if (!value) return "Not recorded";
  return new Date(value * 1000).toLocaleString();
}

function normalizeDraftFields(
  draft: DocumentShipmentDraft | null,
): Array<{ key: string; field: DocumentIntelligenceField }> {
  if (!draft) return [];

  const structuredFields = Object.entries(draft.draftData?.fields ?? {});
  if (structuredFields.length > 0) {
    return structuredFields.map(([key, field]) => ({ key, field }));
  }

  return Object.entries(draft.draftData ?? {})
    .filter(
      ([key]) =>
        ![
          "overallConfidence",
          "reviewStatus",
          "missingFields",
          "signals",
          "conflicts",
          "fields",
          "stops",
          "rawExcerpt",
        ].includes(key),
    )
    .map(([key, value]) => ({
      key,
      field: {
        label: key,
        value,
      },
    }));
}

function confidenceVariant(confidence?: number) {
  if (confidence == null) return "secondary";
  if (confidence >= 0.85) return "active";
  if (confidence >= 0.7) return "warning";
  return "outline";
}

function formatStopSummary(stop: DocumentIntelligenceStop) {
  const addressParts = [
    stop.addressLine1,
    stop.addressLine2,
    [stop.city, stop.state, stop.postalCode].filter(Boolean).join(" "),
  ].filter(Boolean);
  return addressParts.length > 0
    ? addressParts.join(", ")
    : "Address not extracted";
}

function formatAIAcceptanceStatus(status?: string) {
  switch (status) {
    case "accepted":
      return "Accepted";
    case "rejected":
      return "Rejected";
    case "not_attempted":
      return "Not Attempted";
    default:
      return "Unknown";
  }
}

function aiAcceptanceVariant(status?: string) {
  switch (status) {
    case "accepted":
      return "active";
    case "rejected":
      return "outline";
    case "not_attempted":
      return "secondary";
    default:
      return "secondary";
  }
}

function formatDiagnosticReason(reason?: string) {
  if (!reason) return "None";
  return reason
    .split("_")
    .filter(Boolean)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(" ");
}

function ConflictSection({
  conflicts,
}: {
  conflicts: DocumentIntelligenceConflict[];
}) {
  if (conflicts.length === 0) return null;

  return (
    <div className="rounded-lg border border-dashed p-3">
      <div className="mb-2 flex items-center gap-2 text-sm font-medium text-foreground">
        <AlertCircleIcon className="size-4" />
        Conflicts detected
      </div>
      <div className="grid gap-2">
        {conflicts.map((conflict, index) => (
          <div
            key={`${conflict.key || conflict.label || "conflict"}-${index}`}
            className="rounded-md border p-3"
          >
            <div className="flex flex-wrap items-center gap-2">
              <span className="text-sm font-medium">
                {conflict.label || conflict.key || "Conflict"}
              </span>
              {conflict.pageNumbers.length > 0 ? (
                <Badge variant="outline">
                  Pages {conflict.pageNumbers.join(", ")}
                </Badge>
              ) : null}
            </div>
            {conflict.values.length > 0 ? (
              <div className="mt-2 flex flex-wrap gap-2">
                {conflict.values.map((value) => (
                  <Badge key={value} variant="secondary">
                    {value}
                  </Badge>
                ))}
              </div>
            ) : null}
            {conflict.evidenceExcerpt ? (
              <div className="mt-2 rounded-md bg-muted/40 px-2 py-1 font-mono text-[11px] text-muted-foreground">
                {conflict.evidenceExcerpt}
              </div>
            ) : null}
          </div>
        ))}
      </div>
    </div>
  );
}

function StopsSection({ stops }: { stops: DocumentIntelligenceStop[] }) {
  if (stops.length === 0) {
    return (
      <div className="rounded-lg border border-dashed p-3 text-sm text-muted-foreground">
        No shipment stops were extracted from this document.
      </div>
    );
  }

  return (
    <div className="grid gap-3">
      {stops.map((stop) => (
        <div
          key={`${stop.role}-${stop.sequence}-${stop.pageNumber ?? 0}`}
          className="rounded-lg border p-3"
        >
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div>
              <div className="text-xs font-medium tracking-wide text-muted-foreground uppercase">
                {stop.role} stop #{stop.sequence}
              </div>
              <div className="mt-1 text-sm font-medium">
                {stop.name ||
                  `${stop.role === "pickup" ? "Pickup" : "Delivery"} location`}
              </div>
            </div>
            <div className="flex flex-wrap items-center gap-2">
              {stop.confidence != null ? (
                <Badge variant={confidenceVariant(stop.confidence)}>
                  {formatConfidence(stop.confidence)}
                </Badge>
              ) : null}
              {stop.reviewRequired ? (
                <Badge variant="outline">Review</Badge>
              ) : null}
              {stop.pageNumber ? (
                <Badge variant="secondary">Page {stop.pageNumber}</Badge>
              ) : null}
            </div>
          </div>
          <div className="mt-3 grid gap-2 md:grid-cols-2">
            <div className="rounded-md bg-muted/20 p-2">
              <div className="text-[11px] font-medium tracking-wide text-muted-foreground uppercase">
                Address
              </div>
              <div className="mt-1 text-sm">{formatStopSummary(stop)}</div>
            </div>
            <div className="rounded-md bg-muted/20 p-2">
              <div className="text-[11px] font-medium tracking-wide text-muted-foreground uppercase">
                Timing
              </div>
              <div className="mt-1 text-sm">
                {[stop.date, stop.timeWindow].filter(Boolean).join(" · ") ||
                  "Not extracted"}
              </div>
            </div>
          </div>
          {stop.evidenceExcerpt ? (
            <div className="mt-2 rounded-md bg-muted/40 px-2 py-1 font-mono text-[11px] whitespace-pre-wrap text-muted-foreground">
              {stop.evidenceExcerpt}
            </div>
          ) : null}
        </div>
      ))}
    </div>
  );
}

function IntelligenceSummary({
  intelligence,
  fallbackKind,
  fallbackConfidence,
}: {
  intelligence: DocumentIntelligence | null | undefined;
  fallbackKind?: string | null;
  fallbackConfidence?: number | null;
}) {
  return (
    <div className="grid gap-3 md:grid-cols-6">
      <div className="rounded-lg border p-3">
        <div className="text-xs font-medium tracking-wide text-muted-foreground uppercase">
          Classification
        </div>
        <div className="mt-1 text-sm">
          {intelligence?.kind || fallbackKind || "Other"}
        </div>
      </div>
      <div className="rounded-lg border p-3">
        <div className="text-xs font-medium tracking-wide text-muted-foreground uppercase">
          Confidence
        </div>
        <div className="mt-1 text-sm">
          {formatConfidence(
            intelligence?.overallConfidence ?? fallbackConfidence,
          )}
        </div>
      </div>
      <div className="rounded-lg border p-3">
        <div className="text-xs font-medium tracking-wide text-muted-foreground uppercase">
          Review Status
        </div>
        <div className="mt-1 text-sm">
          {intelligence?.reviewStatus || "NeedsReview"}
        </div>
      </div>
      <div className="rounded-lg border p-3">
        <div className="text-xs font-medium tracking-wide text-muted-foreground uppercase">
          Classifier Source
        </div>
        <div className="mt-1 text-sm">
          {intelligence?.classifierSource || "deterministic"}
        </div>
      </div>
      <div className="rounded-lg border p-3">
        <div className="text-xs font-medium tracking-wide text-muted-foreground uppercase">
          Provider Fingerprint
        </div>
        <div className="mt-1 text-sm">
          {intelligence?.providerFingerprint || "None"}
        </div>
      </div>
      <div className="rounded-lg border p-3">
        <div className="text-xs font-medium tracking-wide text-muted-foreground uppercase">
          Missing Critical Fields
        </div>
        <div className="mt-1 text-sm">
          {intelligence?.missingFields?.length === 0
            ? "None"
            : (intelligence?.missingFields?.length ?? "Not scored")}
        </div>
      </div>
      {intelligence?.classificationReason ? (
        <div className="rounded-lg border p-3 md:col-span-6">
          <div className="text-xs font-medium tracking-wide text-muted-foreground uppercase">
            Classification Reason
          </div>
          <div className="mt-1 text-sm">
            {intelligence.classificationReason}
          </div>
        </div>
      ) : null}
    </div>
  );
}

function AnalysisSnapshotCard({
  title,
  analysis,
}: {
  title: string;
  analysis: DocumentIntelligence | null | undefined;
}) {
  if (!analysis) {
    return (
      <div className="rounded-lg border border-dashed p-3 text-sm text-muted-foreground">
        No analysis captured.
      </div>
    );
  }

  const fieldCount = Object.keys(analysis.fields ?? {}).length;

  return (
    <div className="rounded-lg border p-3">
      <div className="mb-3 flex items-start justify-between gap-3">
        <div>
          <div className="text-xs font-medium tracking-wide text-muted-foreground uppercase">
            {title}
          </div>
          <div className="mt-1 text-sm font-medium">
            {analysis.kind || "Other"}
          </div>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <Badge variant={confidenceVariant(analysis.overallConfidence)}>
            {formatConfidence(analysis.overallConfidence)}
          </Badge>
          {analysis.reviewStatus ? (
            <Badge variant={analysis.reviewStatus === "Ready" ? "active" : "outline"}>
              {analysis.reviewStatus}
            </Badge>
          ) : null}
        </div>
      </div>
      <div className="grid gap-2 md:grid-cols-3">
        <div className="rounded-md bg-muted/20 p-2">
          <div className="text-[11px] font-medium tracking-wide text-muted-foreground uppercase">
            Fields
          </div>
          <div className="mt-1 text-sm">{fieldCount}</div>
        </div>
        <div className="rounded-md bg-muted/20 p-2">
          <div className="text-[11px] font-medium tracking-wide text-muted-foreground uppercase">
            Stops
          </div>
          <div className="mt-1 text-sm">{analysis.stops?.length ?? 0}</div>
        </div>
        <div className="rounded-md bg-muted/20 p-2">
          <div className="text-[11px] font-medium tracking-wide text-muted-foreground uppercase">
            Source
          </div>
          <div className="mt-1 text-sm">{analysis.classifierSource || "Unknown"}</div>
        </div>
      </div>
      {analysis.missingFields?.length ? (
        <div className="mt-3">
          <div className="mb-1 text-[11px] font-medium tracking-wide text-muted-foreground uppercase">
            Missing Fields
          </div>
          <div className="flex flex-wrap gap-2">
            {analysis.missingFields.map((field) => (
              <Badge key={field} variant="secondary">
                {field}
              </Badge>
            ))}
          </div>
        </div>
      ) : null}
      {analysis.stops?.length ? (
        <div className="mt-3">
          <div className="mb-1 text-[11px] font-medium tracking-wide text-muted-foreground uppercase">
            Stops
          </div>
          <StopsSection stops={analysis.stops.slice(0, 3)} />
        </div>
      ) : null}
    </div>
  );
}

function AIDiagnosticsSection({
  diagnostics,
}: {
  diagnostics: DocumentIntelligenceAIDiagnostics | null | undefined;
}) {
  if (!diagnostics) {
    return null;
  }

  return (
    <div className="grid gap-3">
      <div>
        <h3 className="text-sm font-medium">AI Extraction Diagnostics</h3>
        <p className="text-xs text-muted-foreground">
          Compare the non-AI fallback result with the AI candidate and see why
          the AI output was accepted, rejected, or skipped.
        </p>
      </div>
      <div className="grid gap-3 md:grid-cols-2">
        <div className="rounded-lg border p-3">
          <div className="text-xs font-medium tracking-wide text-muted-foreground uppercase">
            AI Outcome
          </div>
          <div className="mt-2 flex flex-wrap items-center gap-2">
            <Badge variant={aiAcceptanceVariant(diagnostics.acceptanceStatus)}>
              {formatAIAcceptanceStatus(diagnostics.acceptanceStatus)}
            </Badge>
            {diagnostics.rejectionReason ? (
              <Badge variant="secondary">
                {formatDiagnosticReason(diagnostics.rejectionReason)}
              </Badge>
            ) : null}
          </div>
        </div>
        <div className="rounded-lg border p-3">
          <div className="text-xs font-medium tracking-wide text-muted-foreground uppercase">
            Rejection Reason
          </div>
          <div className="mt-1 text-sm">
            {formatDiagnosticReason(diagnostics.rejectionReason)}
          </div>
        </div>
      </div>
      <div className="grid gap-3 lg:grid-cols-2">
        <AnalysisSnapshotCard
          title="Fallback Analysis"
          analysis={diagnostics.fallbackAnalysis}
        />
        <AnalysisSnapshotCard
          title="AI Candidate"
          analysis={diagnostics.candidateAnalysis}
        />
      </div>
    </div>
  );
}

function DraftSection({ draft }: { draft: DocumentShipmentDraft | null }) {
  if (!draft || draft.status === "Unavailable") {
    return (
      <div className="rounded-lg border border-dashed p-3 text-sm text-muted-foreground">
        No shipment draft is available for this document.
      </div>
    );
  }

  if (draft.status === "Failed") {
    return (
      <div className="rounded-lg border border-dashed p-3 text-sm text-muted-foreground">
        <div className="font-medium text-foreground">
          Shipment draft extraction failed.
        </div>
        <div className="mt-1">
          {[draft.failureCode, draft.failureMessage]
            .filter(Boolean)
            .join(" · ") || "No failure details were recorded."}
        </div>
      </div>
    );
  }

  const entries = normalizeDraftFields(draft);
  const missingFields = draft.draftData?.missingFields ?? [];
  const signals = draft.draftData?.signals ?? [];
  const conflicts = draft.draftData?.conflicts ?? [];
  const stops = draft.draftData?.stops ?? [];

  if (entries.length === 0) {
    return (
      <div className="rounded-lg border border-dashed p-3 text-sm text-muted-foreground">
        The system classified this as a rate confirmation, but no structured
        shipment fields were extracted.
      </div>
    );
  }

  return (
    <div className="grid gap-2">
      <div className="grid gap-3 md:grid-cols-3">
        <div className="rounded-lg border p-3">
          <div className="text-xs font-medium tracking-wide text-muted-foreground uppercase">
            Draft Confidence
          </div>
          <div className="mt-1 text-sm font-medium">
            {formatConfidence(
              draft.draftData?.overallConfidence ?? draft.confidence,
            )}
          </div>
        </div>
        <div className="rounded-lg border p-3">
          <div className="text-xs font-medium tracking-wide text-muted-foreground uppercase">
            Review Status
          </div>
          <div className="mt-1 text-sm">
            {draft.draftData?.reviewStatus || "NeedsReview"}
          </div>
        </div>
        <div className="rounded-lg border p-3">
          <div className="text-xs font-medium tracking-wide text-muted-foreground uppercase">
            Missing Critical Fields
          </div>
          <div className="mt-1 text-sm">
            {missingFields.length === 0 ? "None" : missingFields.length}
          </div>
        </div>
      </div>

      {draft.attachedShipmentId ? (
        <div className="rounded-lg border border-emerald-200 bg-emerald-50/70 p-3 text-sm text-emerald-950">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div>
              <div className="font-medium">
                This document is already attached to a shipment.
              </div>
              <div className="mt-1 text-emerald-900/80">
                Shipment {draft.attachedShipmentId} attached{" "}
                {formatUnixTimestamp(draft.attachedAt)}.
              </div>
            </div>
            <Button
              variant="outline"
              size="sm"
              render={<Link to="/shipment-management/shipments" />}
            >
              Open Shipments
            </Button>
          </div>
        </div>
      ) : null}

      {signals.length > 0 ? (
        <div className="rounded-lg border p-3">
          <div className="mb-2 flex items-center gap-2 text-xs font-medium tracking-wide text-muted-foreground uppercase">
            <SparklesIcon className="size-3.5" />
            Classification Signals
          </div>
          <div className="flex flex-wrap gap-2">
            {signals.map((signal) => (
              <Badge key={signal} variant="secondary">
                {signal}
              </Badge>
            ))}
          </div>
        </div>
      ) : null}

      {missingFields.length > 0 ? (
        <div className="rounded-lg border border-dashed p-3 text-sm text-muted-foreground">
          <div className="mb-2 flex items-center gap-2 font-medium text-foreground">
            <AlertCircleIcon className="size-4" />
            Review needed before using this draft
          </div>
          <div className="flex flex-wrap gap-2">
            {missingFields.map((field) => (
              <Badge key={field} variant="outline">
                {field}
              </Badge>
            ))}
          </div>
        </div>
      ) : null}

      <ConflictSection conflicts={conflicts} />

      <div className="grid gap-2">
        <div className="text-xs font-medium tracking-wide text-muted-foreground uppercase">
          Extracted Stops
        </div>
        <StopsSection stops={stops} />
      </div>

      {entries.map(({ key, field }) => (
        <div key={key} className="rounded-lg border p-3">
          <div className="flex items-start justify-between gap-3">
            <div className="text-xs font-medium tracking-wide text-muted-foreground uppercase">
              {field.label || key}
            </div>
            <div className="flex flex-wrap items-center gap-2">
              {field.confidence != null ? (
                <Badge variant={confidenceVariant(field.confidence)}>
                  {formatConfidence(field.confidence)}
                </Badge>
              ) : null}
              {field.reviewRequired ? (
                <Badge variant="outline">Review</Badge>
              ) : null}
              {field.conflict ? (
                <Badge variant="outline">Conflict</Badge>
              ) : null}
            </div>
          </div>
          <div className="mt-2 text-sm whitespace-pre-wrap">
            {formatValue(field.value)}
          </div>
          {field.excerpt ? (
            <div className="mt-2 rounded-md bg-muted/40 px-2 py-1 font-mono text-[11px] text-muted-foreground">
              {field.pageNumber ? (
                <div className="mb-1 font-sans text-[10px] uppercase">
                  Page {field.pageNumber}
                </div>
              ) : null}
              {field.excerpt}
            </div>
          ) : null}
        </div>
      ))}
    </div>
  );
}

function ContentSection({
  content,
  fallbackStatus,
  fallbackError,
}: {
  content: DocumentContent | null;
  fallbackStatus: Document["contentStatus"];
  fallbackError?: string | null;
}) {
  const intelligence = content?.structuredData?.intelligence;
  const aiDiagnostics = content?.structuredData?.aiDiagnostics;

  if (content?.contentText?.trim()) {
    return (
      <div className="grid gap-3">
        {intelligence?.signals?.length ? (
          <div className="rounded-lg border p-3">
            <div className="mb-2 text-xs font-medium tracking-wide text-muted-foreground uppercase">
              Classification Confidence
            </div>
            <div className="mb-3 flex flex-wrap items-center gap-2">
              <Badge
                variant={confidenceVariant(intelligence.overallConfidence)}
              >
                {formatConfidence(intelligence.overallConfidence)}
              </Badge>
              {intelligence.reviewStatus !== "Ready" ? (
                <Badge variant="outline">Review</Badge>
              ) : null}
            </div>
            <div className="flex flex-wrap gap-2">
              {intelligence.signals.map((signal) => (
                <Badge key={signal} variant="secondary">
                  {signal}
                </Badge>
              ))}
            </div>
          </div>
        ) : null}

        <ConflictSection conflicts={intelligence?.conflicts ?? []} />

        <AIDiagnosticsSection diagnostics={aiDiagnostics} />

        <div className="grid gap-2">
          <div className="text-xs font-medium tracking-wide text-muted-foreground uppercase">
            Canonical Stops
          </div>
          <StopsSection stops={intelligence?.stops ?? []} />
        </div>

        <ScrollArea
          className="h-80 rounded-lg border bg-muted/20 p-3"
          viewportClassName="p-3"
        >
          <pre className="font-mono text-xs whitespace-pre-wrap text-foreground">
            {content.contentText}
          </pre>
        </ScrollArea>
        {content.pages.length > 0 ? (
          <div className="grid gap-2 md:grid-cols-2">
            {content.pages.slice(0, 6).map((page) => (
              <div
                key={page.id}
                className="rounded-lg border p-3 text-xs text-muted-foreground"
              >
                <div className="mb-1 flex items-center justify-between gap-2">
                  <span className="font-medium text-foreground">
                    Page {page.pageNumber}
                  </span>
                  <div className="flex items-center gap-2">
                    <Badge variant="secondary">{page.sourceKind}</Badge>
                    {page.preprocessingApplied ? (
                      <Badge variant="outline">Preprocessed</Badge>
                    ) : null}
                  </div>
                </div>
                {page.ocrConfidence > 0 ? (
                  <div className="mb-1">
                    OCR confidence: {formatConfidence(page.ocrConfidence)}
                  </div>
                ) : null}
                <div className="line-clamp-4 font-mono text-[11px] whitespace-pre-wrap">
                  {page.extractedText?.trim() || "No extracted text"}
                </div>
              </div>
            ))}
          </div>
        ) : null}
      </div>
    );
  }

  if (fallbackStatus === "Extracting" || fallbackStatus === "Pending") {
    return (
      <div className="rounded-lg border border-dashed p-3 text-sm text-muted-foreground">
        Extraction is still in progress.
      </div>
    );
  }

  return (
    <div className="rounded-lg border border-dashed p-3 text-sm text-muted-foreground">
      {fallbackError ||
        content?.failureMessage ||
        "No extracted text is available for this document."}
    </div>
  );
}

export function DocumentIntelligenceDialog({
  open,
  onOpenChange,
  document,
  resourceType,
  resourceId,
}: DocumentIntelligenceDialogProps) {
  const queryClient = useQueryClient();
  const [reviewShipmentOpen, setReviewShipmentOpen] = useState(false);
  const documentID = document?.id;
  const shipmentDraftStatus = document?.shipmentDraftStatus;

  const { data: content, isLoading: isContentLoading } = useQuery({
    queryKey: ["document-content", documentID],
    queryFn: async () => {
      if (!documentID) return null;
      try {
        return await apiService.documentService.getContent(documentID);
      } catch {
        return null;
      }
    },
    enabled: open && !!documentID,
  });

  const { data: shipmentDraft, isLoading: isDraftLoading } = useQuery({
    queryKey: ["document-shipment-draft", documentID, shipmentDraftStatus],
    queryFn: async () => {
      if (!documentID) return null;
      try {
        return await apiService.documentService.getShipmentDraft(documentID);
      } catch {
        return null;
      }
    },
    enabled: open && !!documentID && shipmentDraftStatus !== "Unavailable",
  });

  const { mutate: reextract, isPending: isReextracting } = useMutation({
    mutationFn: async () => {
      if (!document) return;
      await apiService.documentService.reextract(document.id);
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ["documents", resourceType, resourceId],
      });
      if (document) {
        void queryClient.invalidateQueries({
          queryKey: ["document-content", document.id],
        });
        void queryClient.invalidateQueries({
          queryKey: ["document-shipment-draft", document.id],
        });
      }
      toast.success("Document re-extraction started");
    },
    onError: (error) => {
      toast.error(`Failed to re-extract document: ${error.message}`);
    },
  });

  const canReviewShipmentDraft =
    !!document &&
    !!shipmentDraft &&
    shipmentDraft.status !== "Unavailable" &&
    shipmentDraft.status !== "Failed" &&
    !shipmentDraft.attachedShipmentId;
  const supportsTargetedReextract =
    document?.processingProfile === "rate_confirmation_import";

  return (
    <>
      <Dialog
        open={open}
        onOpenChange={(nextOpen) => {
          onOpenChange(nextOpen);
          if (!nextOpen) {
            setReviewShipmentOpen(false);
          }
        }}
      >
        <DialogContent
          className="gap-0 overflow-hidden p-0 sm:max-w-4xl"
          showCloseButton
        >
          {document ? (
            <>
              <DialogHeader className="border-b px-6 pt-6 pb-4">
                <div className="flex flex-wrap items-center gap-2">
                  <DialogTitle>{document.originalName}</DialogTitle>
                  <Badge variant={statusBadgeVariant(document.contentStatus)}>
                    {document.contentStatus}
                  </Badge>
                  {document.detectedKind &&
                  document.detectedKind !== "Other" ? (
                    <Badge variant="info">{document.detectedKind}</Badge>
                  ) : null}
                  {document.shipmentDraftStatus === "Ready" ? (
                    <Badge variant="teal">Shipment draft ready</Badge>
                  ) : null}
                  {shipmentDraft?.attachedShipmentId ? (
                    <Badge variant="active">Attached to shipment</Badge>
                  ) : null}
                </div>
                <DialogDescription>
                  Review extracted text, document classification, and any
                  structured shipment draft fields.
                </DialogDescription>
              </DialogHeader>

              <ScrollArea className="max-h-[calc(90vh-160px)]">
                <div className="grid gap-6 p-4">
                  <section className="grid gap-3">
                    <div>
                      <h3 className="text-sm font-medium">
                        Document Intelligence
                      </h3>
                      <p className="text-xs text-muted-foreground">
                        Extraction status, classification, and structured
                        output.
                      </p>
                    </div>
                    <div className="grid gap-3">
                      {!supportsTargetedReextract ? (
                        <div className="rounded-lg border border-dashed p-3 text-sm text-muted-foreground">
                          Extraction was not requested for this document. OCR and AI extraction now
                          run only through the targeted rate confirmation import flow.
                        </div>
                      ) : null}
                      <IntelligenceSummary
                        intelligence={content?.structuredData?.intelligence}
                        fallbackKind={
                          content?.detectedDocumentKind || document.detectedKind
                        }
                        fallbackConfidence={content?.classificationConfidence}
                      />
                      <div className="grid gap-3 md:grid-cols-2">
                        <div className="rounded-lg border p-3">
                          <div className="text-xs font-medium tracking-wide text-muted-foreground uppercase">
                            Extraction Source
                          </div>
                          <div className="mt-1 text-sm">
                            {content?.sourceKind || "Not available"}
                          </div>
                        </div>
                        <div className="rounded-lg border p-3">
                          <div className="text-xs font-medium tracking-wide text-muted-foreground uppercase">
                            Pages
                          </div>
                          <div className="mt-1 text-sm">
                            {content?.pageCount ?? "Not available"}
                          </div>
                        </div>
                      </div>
                    </div>
                  </section>

                  <section className="grid gap-3">
                    <div>
                      <h3 className="text-sm font-medium">Shipment Draft</h3>
                      <p className="text-xs text-muted-foreground">
                        Review structured fields extracted from supported
                        shipment documents.
                      </p>
                    </div>
                    {isDraftLoading ? (
                      <div className="flex items-center gap-2 rounded-lg border p-3 text-sm text-muted-foreground">
                        <LoaderCircleIcon className="size-4 animate-spin" />
                        Loading shipment draft...
                      </div>
                    ) : (
                      <>
                        <DraftSection draft={shipmentDraft ?? null} />
                        {(!shipmentDraft ||
                          shipmentDraft.status === "Unavailable") &&
                        document.detectedKind &&
                        document.detectedKind !== "RateConfirmation" ? (
                          <div className="rounded-lg border border-dashed p-3 text-sm text-muted-foreground">
                            This document is available for classification,
                            search, and extracted-text review, but it does not
                            produce a shipment draft.
                          </div>
                        ) : null}
                      </>
                    )}
                  </section>

                  <section className="grid gap-3">
                    <div>
                      <h3 className="text-sm font-medium">Extracted Text</h3>
                      <p className="text-xs text-muted-foreground">
                        Full extracted text used for search and document
                        classification.
                      </p>
                    </div>
                    {isContentLoading ? (
                      <div className="flex items-center gap-2 rounded-lg border p-3 text-sm text-muted-foreground">
                        <LoaderCircleIcon className="size-4 animate-spin" />
                        Loading extracted content...
                      </div>
                    ) : (
                      <ContentSection
                        content={content ?? null}
                        fallbackStatus={document.contentStatus}
                        fallbackError={document.contentError}
                      />
                    )}
                  </section>
                </div>
              </ScrollArea>

              <DialogFooter className="m-0" showCloseButton>
                {canReviewShipmentDraft ? (
                  <Button
                    variant="secondary"
                    onClick={() => setReviewShipmentOpen(true)}
                  >
                    Create Shipment
                  </Button>
                ) : null}
                {shipmentDraft?.attachedShipmentId ? (
                  <Button
                    variant="secondary"
                    render={<Link to="/shipment-management/shipments" />}
                  >
                    Open Shipments
                  </Button>
                ) : null}
                {supportsTargetedReextract ? (
                  <Button
                    variant="outline"
                    onClick={() => reextract()}
                    disabled={isReextracting}
                  >
                    {isReextracting ? (
                      <LoaderCircleIcon className="size-4 animate-spin" />
                    ) : (
                      <RefreshCcwIcon className="size-4" />
                    )}
                    Re-extract
                  </Button>
                ) : null}
              </DialogFooter>
            </>
          ) : null}
        </DialogContent>
      </Dialog>
      <DocumentShipmentDraftReviewDialog
        open={reviewShipmentOpen}
        onOpenChange={setReviewShipmentOpen}
        document={document}
        draft={shipmentDraft ?? null}
        sourceResourceType={resourceType}
        sourceResourceId={resourceId}
      />
    </>
  );
}
