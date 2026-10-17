import { PdfFrame } from "@/components/elements/pdf-frame";
import { documentPage } from "@/lib/queries/document-page";
import { documentContentEndpoint } from "@/services/document";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { useObjectUrl } from "@trenova/shared/hooks/use-object-url";
import { useT } from "@trenova/shared/i18n/use-t";

type FileDialogProps = {
  documentId: string;
  fileName: string;
  image: boolean;
  /** The page the card was showing, which the viewer opens on. */
  page: number;
  pageCount: number;
  open: boolean;
  onOpenChange: (open: boolean) => void;
};

/**
 * The whole file in the regular viewer, for search, print and pages the card does not draw: a
 * PDF in the browser's own viewer on the page the card was showing, an image at its size.
 */
export function ExtractionFileDialog({
  documentId,
  fileName,
  image,
  page,
  pageCount,
  open,
  onOpenChange,
}: FileDialogProps) {
  const t = useT();
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="2xl" className="dk-ex-file">
        <DialogHeader>
          <DialogTitle className="truncate">{fileName || t("Document")}</DialogTitle>
          <DialogDescription>
            {t("{0, plural, one {# page} other {# pages}}", pageCount)}
          </DialogDescription>
        </DialogHeader>
        {open &&
          (image ? (
            <OriginalImage documentId={documentId} fileName={fileName} />
          ) : (
            <PdfFrame
              endpoint={documentContentEndpoint(documentId, "view")}
              title={fileName || t("Document")}
              page={page}
              className="dk-ex-file-frame"
            />
          ))}
      </DialogContent>
    </Dialog>
  );
}

function OriginalImage({ documentId, fileName }: { documentId: string; fileName: string }) {
  const t = useT();
  const query = useQuery({ ...documentPage.original(documentId), staleTime: 5 * 60 * 1000 });
  const url = useObjectUrl(query.data);

  if (query.isError) {
    return (
      <div className="dk-ex-pfail" role="status">
        <p>{t("This file could not be shown.")}</p>
        <div className="dk-ex-pfail-acts">
          <Button variant="outline" size="sm" onClick={() => void query.refetch()}>
            {t("Try again")}
          </Button>
        </div>
      </div>
    );
  }
  if (url === null) {
    return (
      <div
        className="dk-ex-file-frame ui-shimmer"
        aria-busy="true"
        aria-label={t("Loading the file")}
      />
    );
  }
  return (
    <div className="dk-ex-file-frame dk-ex-file-image">
      <img src={url} alt={fileName || t("Document")} />
    </div>
  );
}
