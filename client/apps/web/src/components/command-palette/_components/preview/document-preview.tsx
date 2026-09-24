import { DocumentFileTypeIcon } from "@/components/documents/document-file-type-icon";
import { documentContentUrl } from "@/services/document";
import { Badge } from "@trenova/shared/components/ui/badge";
import {
  DescriptionEmpty,
  DescriptionItem,
  DescriptionList,
} from "@trenova/shared/components/ui/description-list";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { formatFileSize } from "@trenova/shared/lib/utils";
import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { PALETTE_ENTITIES } from "../../palette-entities";
import type { PaletteAction, PaletteIntent, PaletteRecord } from "../../palette-model";
import { PreviewError, PreviewFrame, PreviewSection, PreviewSkeleton } from "./preview-frame";
import { documentPreviewQuery } from "./preview-queries";

function Thumbnail({ documentId, alt }: { documentId: string; alt: string }) {
  const [failed, setFailed] = useState(false);
  if (failed) {
    return null;
  }
  return (
    <div className="rounded-surface bg-sunken ring-border-subtle flex h-44 items-center justify-center overflow-hidden ring-1">
      <img
        src={documentContentUrl(documentId, "preview")}
        alt={alt}
        loading="lazy"
        onError={() => setFailed(true)}
        className="max-h-full max-w-full object-contain"
      />
    </div>
  );
}

export function DocumentPreview({
  record,
  actions,
  onRun,
}: {
  record: PaletteRecord;
  actions: readonly PaletteAction[];
  onRun: (intent: PaletteIntent) => void;
}) {
  const t = useT();
  const { data: document, isLoading, isError } = useQuery(documentPreviewQuery(record.id));
  const entity = PALETTE_ENTITIES.document;

  if (isLoading) {
    return <PreviewSkeleton />;
  }
  if (isError || !document) {
    return <PreviewError />;
  }

  const name = document.originalName || document.fileName;

  return (
    <PreviewFrame
      icon={entity.icon}
      tileClass={entity.tileClass}
      title={name}
      subtitle={[document.documentType?.name, document.resourceType].filter(Boolean).join(" · ")}
      badge={<Badge variant="neutral">{document.status}</Badge>}
      actions={actions}
      onRun={onRun}
    >
      {document.previewStatus === "Ready" ? (
        <Thumbnail key={document.id} documentId={document.id} alt={name} />
      ) : (
        <div className="rounded-surface bg-sunken ring-border-subtle flex h-28 items-center justify-center ring-1">
          <DocumentFileTypeIcon fileType={document.fileType} fileName={name} size="lg" />
        </div>
      )}
      <PreviewSection title={t("File")}>
        <DescriptionList columns={2}>
          <DescriptionItem label={t("Type")}>
            {document.fileType || <DescriptionEmpty />}
          </DescriptionItem>
          <DescriptionItem label={t("Size")} numeric>
            {formatFileSize(document.fileSize)}
          </DescriptionItem>
          <DescriptionItem label={t("Version")} numeric>
            {document.versionNumber}
          </DescriptionItem>
          <DescriptionItem label={t("Uploaded")}>
            {formatUnixDateMedium(document.createdAt)}
          </DescriptionItem>
          {document.expirationDate ? (
            <DescriptionItem label={t("Expires")}>
              {formatUnixDateMedium(document.expirationDate)}
            </DescriptionItem>
          ) : null}
          {document.description ? (
            <DescriptionItem label={t("Description")} span="full">
              {document.description}
            </DescriptionItem>
          ) : null}
        </DescriptionList>
      </PreviewSection>
      {document.tags && document.tags.length > 0 && (
        <PreviewSection title={t("Tags")}>
          <div className="flex flex-wrap gap-1">
            {document.tags.map((tag) => (
              <Badge key={tag} variant="accent-slate">
                {tag}
              </Badge>
            ))}
          </div>
        </PreviewSection>
      )}
    </PreviewFrame>
  );
}
