import { DocumentFileTypeIcon } from "@/components/documents/document-file-type-icon";
import { documentContentUrl } from "@/services/document";
import { defineLabels } from "@trenova/shared/i18n/labels";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTimeShort } from "@trenova/shared/lib/date";
import type { Document, DocumentShipmentDraft } from "@trenova/shared/types/document";
import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { PALETTE_ENTITY_ICONS } from "../../palette-entities";
import { documentKind } from "../../palette-fields";
import {
  PreviewError,
  PreviewFields,
  PreviewHead,
  PreviewSection,
  PreviewSkeleton,
  PreviewSource,
  RelatedRow,
  type PreviewFieldRow,
} from "./preview-frame";
import { documentExtractionQuery, documentPreviewQuery } from "./preview-queries";
import type { RecordPreviewProps } from "./record-preview-props";

const EXTRACTED_FIELDS_SHOWN = 6;

const INTAKE_LABELS = defineLabels({
  none: "Uploaded by hand",
  rate_confirmation_import: "Rate confirmation import",
  assistant_attachment: "A conversation with Nova",
  inbound_attachment: "A monitored mailbox",
  capture: "Scanning and printing",
  factoring_remittance: "A factor's remittance",
});

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

function FirstPage({ document, name }: { document: Document; name: string }) {
  const [failed, setFailed] = useState(false);
  const ready = document.previewStatus === "Ready" && !failed;
  return (
    <div className="bg-sunken rounded-surface ring-border-subtle flex h-45 items-center justify-center overflow-hidden ring-1">
      {ready ? (
        <img
          src={documentContentUrl(document.id, "preview")}
          alt={name}
          loading="lazy"
          onError={() => setFailed(true)}
          className="max-h-full max-w-full object-contain"
        />
      ) : (
        <DocumentFileTypeIcon fileType={document.fileType} fileName={name} size="lg" />
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
      <FirstPage key={document.id} document={document} name={name} />
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
      <PreviewFields rows={[intake ? { label: t("Came in through"), value: intake } : null]} />
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
