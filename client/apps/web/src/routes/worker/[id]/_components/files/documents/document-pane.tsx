import { DocumentReviewBadge } from "@/components/documents/document-review-badge";
import { DocumentSignatureBadge } from "@/components/documents/document-signature-badge";
import { DocumentPanePreview } from "@/components/documents/document-pane-preview";
import { WorkPane, WorkPaneBody, WorkPaneFooter } from "@/components/work-pane";
import { documentNeedsSignature } from "@/lib/document-signature";
import { Download01Icon, EyeIcon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { DescriptionItem, DescriptionList } from "@trenova/shared/components/ui/description-list";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import type { Document } from "@trenova/shared/types/document";
import { canPreviewDocument, documentTypeName } from "./documents-model";

export type DocumentPaneProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** Null when the document left the list (deleted elsewhere). */
  document: Document | null;
  uploaderName: string | null;
  canDelete: boolean;
  onOpenFile: (document: Document) => Promise<void>;
  onDownload: (document: Document) => Promise<void>;
  onDelete: (document: Document) => void;
};

/** One document read beside the table: its preview, its facts, and opening or downloading it. */
export function DocumentPane({
  open,
  onOpenChange,
  document,
  uploaderName,
  canDelete,
  onOpenFile,
  onDownload,
  onDelete,
}: DocumentPaneProps) {
  const t = useT();

  if (!document) {
    return (
      <WorkPane open={open} onOpenChange={onOpenChange} title={t("Document")}>
        <WorkPaneBody className="text-sm">
          <p className="text-sm text-muted-foreground">
            {t("This document is no longer on the file.")}
          </p>
        </WorkPaneBody>
      </WorkPane>
    );
  }

  return (
    <WorkPane
      open={open}
      onOpenChange={onOpenChange}
      title={document.originalName}
      contentKey={document.id}
    >
      <WorkPaneBody className="text-sm">
        <DocumentPanePreview document={document} openLink={false}>
          <div className="flex flex-wrap justify-center gap-1">
            <DocumentReviewBadge document={document} />
            {documentNeedsSignature(document) ? (
              <DocumentSignatureBadge document={document} />
            ) : null}
          </div>
        </DocumentPanePreview>
        <DescriptionList layout="ruled" columns={2}>
          <DescriptionItem label={t("Type")}>{documentTypeName(document)}</DescriptionItem>
          <DescriptionItem label={t("Added by")}>{uploaderName ?? t("Not known")}</DescriptionItem>
          <DescriptionItem label={t("Added")} numeric>
            {formatUnixDateMedium(document.createdAt)}
          </DescriptionItem>
          <DescriptionItem label={t("Version")} numeric>
            {t("Version {0}", document.versionNumber)}
          </DescriptionItem>
          <DescriptionItem label={t("Expires")} numeric>
            {document.expirationDate ? formatUnixDateMedium(document.expirationDate) : t("Never")}
          </DescriptionItem>
          {document.rejectionReason ? (
            <DescriptionItem label={t("Rejected because")} span={2}>
              {document.rejectionReason}
            </DescriptionItem>
          ) : null}
        </DescriptionList>
      </WorkPaneBody>
      <WorkPaneFooter
        leading={
          canDelete ? (
            <Button
              size="sm"
              variant="outline"
              className="text-danger-foreground"
              onClick={() => onDelete(document)}
            >
              {t("Delete")}
            </Button>
          ) : null
        }
      >
        {canPreviewDocument(document) ? (
          <Button size="sm" variant="outline" onClick={() => void onOpenFile(document)}>
            <EyeIcon className="size-3.5" />
            {t("Open")}
          </Button>
        ) : null}
        <Button size="sm" onClick={() => void onDownload(document)}>
          <Download01Icon className="size-3.5" />
          {t("Download")}
        </Button>
      </WorkPaneFooter>
    </WorkPane>
  );
}
