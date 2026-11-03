import { DocumentFileTypeIcon } from "@/components/documents/document-file-type-icon";
import { getFileCategory } from "@/components/documents/document-utils";
import { documentReviewState } from "@/lib/document-review";
import { documentPage, PAGE_IMAGE_CACHE } from "@/lib/queries/document-page";
import { hasPendingDocumentWork } from "@/lib/resource-documents";
import { documentReviewLabel } from "@/routes/desk/_components/artifacts/record-card/record-documents";
import { documentContentUrl } from "@/services/document";
import {
  ArrowUpRightIcon,
  ChevronLeftIcon,
  ChevronRightIcon,
} from "@trenova/shared/components/icons";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useNow } from "@trenova/shared/hooks/use-now";
import { useObjectUrl } from "@trenova/shared/hooks/use-object-url";
import { defineLabels } from "@trenova/shared/i18n/labels";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium, formatUnixDateTimeShort } from "@trenova/shared/lib/date";
import { formatFileSize } from "@trenova/shared/lib/utils";
import type { Document, DocumentShipmentDraft } from "@trenova/shared/types/document";
import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { PALETTE_ENTITY_ICONS } from "../../palette-entities";
import { documentKind } from "../../palette-fields";
import type { StatusState } from "../../palette-parts";
import {
  PreviewError,
  PreviewFields,
  PreviewHead,
  PreviewSection,
  PreviewSkeleton,
  PreviewSource,
  RelatedRow,
  StatePill,
  type PreviewFieldRow,
} from "./preview-frame";
import { documentExtractionQuery, documentPreviewQuery, hasPages } from "./preview-queries";
import type { RecordPreviewProps } from "./record-preview-props";

const EXTRACTED_FIELDS_SHOWN = 6;
const DAY_SECONDS = 86_400;
/** A document expiring inside this many days is worth a look, as its record's checks count it. */
const EXPIRY_NOTICE_DAYS = 30;

const INTAKE_LABELS = defineLabels({
  none: "Uploaded by hand",
  rate_confirmation_import: "Rate confirmation import",
  assistant_attachment: "A conversation with Nova",
  inbound_attachment: "A monitored mailbox",
  capture: "Scanning and printing",
  factoring_remittance: "A factor's remittance",
});

const CATEGORY_LABELS = defineLabels({
  pdf: "PDF",
  image: "Image",
  spreadsheet: "Spreadsheet",
  document: "Document",
  data: "Data",
});

const REVIEW_STATE: Record<"approved" | "rejected" | "awaiting", StatusState> = {
  approved: "ok",
  rejected: "bad",
  awaiting: "warn",
};

/** Keeps focus in the search box: nothing in the preview takes it on a click. */
function keepFocus(event: React.MouseEvent) {
  event.preventDefault();
}

function readable(value: unknown): string | null {
  if (typeof value === "string") {
    return value.trim() === "" ? null : value.trim();
  }
  if (typeof value === "number" && Number.isFinite(value)) {
    return value.toLocaleString();
  }
  if (typeof value === "boolean") {
    return value ? "✓" : "—";
  }
  return null;
}

/** The fields Nova read, as rows: the label it gave each, what it read, and a dot where it asked for review. */
function extractedRows(draft: DocumentShipmentDraft): PreviewFieldRow[] {
  return Object.entries(draft.draftData.fields)
    .flatMap(([key, field]): PreviewFieldRow[] => {
      const value = readable(field.value);
      if (value === null) {
        return [];
      }
      return [
        {
          label: field.label?.trim() || key,
          value,
          state: field.conflict ? "bad" : field.reviewRequired ? "warn" : undefined,
        },
      ];
    })
    .slice(0, EXTRACTED_FIELDS_SHOWN);
}

function confidenceOf(draft: DocumentShipmentDraft): number | null {
  const confidence = draft.draftData.overallConfidence ?? draft.confidence;
  return Number.isFinite(confidence) && confidence > 0 ? Math.round(confidence * 100) : null;
}

/** When the document lapses, read against now: lapsed, close, or still good. */
export function expiryRow(
  expiresAt: number | null | undefined,
  now: number,
  t: ReturnType<typeof useT>,
): PreviewFieldRow | null {
  if (!expiresAt || expiresAt <= 0) {
    return null;
  }
  const date = formatUnixDateMedium(expiresAt);
  if (expiresAt <= now) {
    return { label: t("Expires"), value: t("Expired {0}", date), state: "bad" };
  }
  const days = Math.ceil((expiresAt - now) / DAY_SECONDS);
  return {
    label: t("Expires"),
    value: date,
    state: days <= EXPIRY_NOTICE_DAYS ? "warn" : undefined,
  };
}

function PagerButton({
  label,
  disabled,
  onClick,
  children,
}: {
  label: string;
  disabled: boolean;
  onClick: () => void;
  children: React.ReactNode;
}) {
  return (
    <button
      type="button"
      tabIndex={-1}
      aria-label={label}
      title={label}
      disabled={disabled}
      onMouseDown={keepFocus}
      onClick={onClick}
      className="text-foreground-muted hover:bg-surface-hover hover:text-foreground rounded-control flex size-6 items-center justify-center transition-colors disabled:pointer-events-none disabled:opacity-40"
    >
      {children}
    </button>
  );
}

/**
 * The document's pages as the server draws them, one at a time, the way the
 * Desk's extraction card reads them: each page fetched with the session's
 * credentials and cached, the next read ahead so the pager knows whether
 * there is one, and the stored preview standing in for page 1 while it is
 * drawn. Pressing the page opens the whole file in a new tab. A file with no
 * pages to draw, or one the server cannot draw, shows its file's mark.
 */
function DocumentPages({ document, name }: { document: Document; name: string }) {
  const t = useT();
  const [page, setPage] = useState(1);
  const [standInFailed, setStandInFailed] = useState(false);
  const pages = hasPages(document);
  const pdf = getFileCategory(document.fileType, document.originalName) === "pdf";
  const current = useQuery({
    ...documentPage.image(document.id, page),
    ...PAGE_IMAGE_CACHE,
    enabled: pages,
  });
  const ahead = useQuery({
    ...documentPage.image(document.id, page + 1),
    ...PAGE_IMAGE_CACHE,
    enabled: pdf && current.isSuccess,
  });
  const url = useObjectUrl(current.data);
  const hasNext = ahead.isSuccess;
  const standIn =
    page === 1 && document.previewStatus === "Ready" && !standInFailed
      ? documentContentUrl(document.id, "preview")
      : null;

  const review = documentReviewState(document);
  const reviewMark =
    review === "approved" || review === "rejected" || review === "awaiting"
      ? { label: documentReviewLabel(review, t), state: REVIEW_STATE[review] }
      : null;
  const pending = hasPendingDocumentWork(document);
  const code = document.documentType?.code;

  let sheet: React.ReactNode;
  if (pages && url) {
    sheet = (
      <img
        src={url}
        alt={t("Page {0} of {1}", page, name)}
        draggable={false}
        className="ring-border-subtle max-h-full max-w-full object-contain ring-1"
      />
    );
  } else if (pages && current.isPending && standIn) {
    sheet = (
      <img
        src={standIn}
        alt=""
        draggable={false}
        onError={() => setStandInFailed(true)}
        className="ring-border-subtle max-h-full max-w-full object-contain opacity-70 ring-1"
      />
    );
  } else if (pages && current.isPending) {
    sheet = <Skeleton className="h-full w-3/5" />;
  } else {
    sheet = (
      <span className="flex flex-col items-center gap-2">
        <DocumentFileTypeIcon fileType={document.fileType} fileName={name} size="lg" />
        {pages && current.isError && (
          <span className="text-foreground-subtle text-sm">{t("No preview of this page")}</span>
        )}
      </span>
    );
  }

  return (
    <div className="flex flex-col gap-2">
      <a
        href={documentContentUrl(document.id, "view")}
        target="_blank"
        rel="noopener noreferrer"
        tabIndex={-1}
        onMouseDown={keepFocus}
        title={t("Open {0}", name)}
        aria-busy={pages && current.isPending}
        className="group bg-sunken rounded-surface ring-border-subtle relative flex h-80 items-center justify-center overflow-hidden p-3 ring-1"
      >
        {sheet}
        {(reviewMark || code) && (
          <span className="absolute top-2 left-2 flex items-center gap-1">
            {code && (
              <span className="bg-card text-foreground ring-border-subtle rounded-badge px-1.5 py-px font-mono text-xs ring-1">
                {code}
              </span>
            )}
            {reviewMark && <StatePill label={reviewMark.label} state={reviewMark.state} />}
          </span>
        )}
        {pending && (
          <span className="bg-card text-foreground-muted ring-border-subtle rounded-badge absolute top-2 right-2 px-2 py-px text-xs ring-1">
            {t("Preparing…")}
          </span>
        )}
        <span className="bg-card text-foreground ring-border-subtle rounded-badge absolute right-2 bottom-2 flex items-center gap-1 px-2 py-0.5 text-xs opacity-0 ring-1 transition-opacity group-hover:opacity-100">
          {t("Open file")}
          <ArrowUpRightIcon className="size-3" />
        </span>
      </a>
      {(page > 1 || hasNext) && (
        <div className="flex items-center justify-center gap-2">
          <PagerButton
            label={t("Previous page")}
            disabled={page <= 1}
            onClick={() => setPage((value) => Math.max(1, value - 1))}
          >
            <ChevronLeftIcon className="size-3.5" />
          </PagerButton>
          <span className="text-foreground-subtle min-w-16 text-center text-sm tabular-nums">
            {t("Page {0}", page)}
          </span>
          <PagerButton
            label={t("Next page")}
            disabled={!hasNext}
            onClick={() => setPage((value) => value + 1)}
          >
            <ChevronRightIcon className="size-3.5" />
          </PagerButton>
        </div>
      )}
    </div>
  );
}

function DocumentBody({
  document,
  record,
  onNavigate,
}: {
  document: Document;
  record: RecordPreviewProps["record"];
  onNavigate: (href: string) => void;
}) {
  const t = useT();
  const now = useNow();
  const extracting = document.shipmentDraftStatus === "Ready";
  const extraction = useQuery({
    ...documentExtractionQuery(document.id),
    enabled: extracting,
  });
  const name = document.originalName || document.fileName;
  const kind = document.documentType?.name || documentKind(record);
  const draft = extraction.data;
  const rows = draft ? extractedRows(draft) : [];
  const confidence = draft ? confidenceOf(draft) : null;
  const attached = record.metadata.attachedLabel;
  const intake = Object.hasOwn(INTAKE_LABELS, document.processingProfile)
    ? INTAKE_LABELS[document.processingProfile as keyof typeof INTAKE_LABELS]
    : null;
  const category = getFileCategory(document.fileType, name);
  const fileLabel = Object.hasOwn(CATEGORY_LABELS, category)
    ? t(CATEGORY_LABELS[category as keyof typeof CATEGORY_LABELS])
    : null;
  const rejected = documentReviewState(document) === "rejected";

  return (
    <>
      <PreviewSource
        kind="document"
        facts={[kind, confidence !== null ? t("{0}% read", confidence) : null]}
      />
      <PreviewHead
        title={name}
        mono
        subtitle={[kind, formatUnixDateTimeShort(document.createdAt)].filter(Boolean).join(" · ")}
      />
      <DocumentPages key={document.id} document={document} name={name} />
      {rejected && document.rejectionReason?.trim() && (
        <p className="text-foreground-muted text-base">
          {t("Rejected: {0}", document.rejectionReason.trim())}
        </p>
      )}
      {rows.length > 0 && (
        <PreviewSection
          label={t("Read by Nova")}
          aside={confidence !== null ? t("{0}% confidence", confidence) : undefined}
        >
          <PreviewFields rows={rows} />
        </PreviewSection>
      )}
      <PreviewSection label={t("Attached to")}>
        <RelatedRow
          icon={
            document.resourceType === "shipment" ||
            document.resourceType === "customer" ||
            document.resourceType === "worker"
              ? PALETTE_ENTITY_ICONS[document.resourceType]
              : PALETTE_ENTITY_ICONS.document
          }
          title={attached || t("Open the record it's attached to")}
          mono={document.resourceType === "shipment"}
          sub={document.resourceType}
          onOpen={() => onNavigate(record.href)}
        />
      </PreviewSection>
      <PreviewFields
        rows={[
          {
            label: t("File"),
            value: [fileLabel, formatFileSize(document.fileSize)].filter(Boolean).join(" · "),
          },
          document.signatureStatus === "Signed"
            ? {
                label: t("Signature"),
                value: document.signedAt
                  ? t("Signed {0}", formatUnixDateMedium(document.signedAt))
                  : t("Signed"),
                state: "ok",
              }
            : document.signatureStatus === "Unsigned"
              ? { label: t("Signature"), value: t("Not signed"), state: "warn" }
              : null,
          expiryRow(document.expirationDate, now, t),
          intake ? { label: t("Came in through"), value: intake } : null,
        ]}
      />
    </>
  );
}

export function DocumentPreview({ record, onNavigate }: RecordPreviewProps) {
  const { data, isLoading, isError } = useQuery(documentPreviewQuery(record.id));

  if (isLoading) {
    return <PreviewSkeleton />;
  }
  if (isError || !data) {
    return <PreviewError />;
  }
  return <DocumentBody document={data} record={record} onNavigate={onNavigate} />;
}
