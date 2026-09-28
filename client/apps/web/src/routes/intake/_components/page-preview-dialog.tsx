import { PdfViewer } from "@/components/elements/pdf-viewer";
import type { CapturePage } from "@/lib/graphql/capture";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { useT } from "@trenova/shared/i18n/use-t";
import { apiUrl } from "@trenova/shared/lib/api-url";

/**
 * One page at full size. The page is shown as stored, before any rotation the
 * person has not yet saved, and the dialog says so rather than letting a
 * page turned in the strip look unturned by mistake.
 */
export function PagePreviewDialog({
  page,
  number,
  rotation,
  onOpenChange,
}: {
  page: CapturePage | null;
  number: number;
  rotation: number;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useT();

  return (
    <Dialog open={page !== null} onOpenChange={onOpenChange}>
      <DialogContent size="xl" className="max-h-[90vh] grid-rows-[auto_minmax(0,1fr)]">
        <DialogHeader>
          <DialogTitle>{t("Page {0}", number)}</DialogTitle>
          <DialogDescription>
            {rotation === 0
              ? t("As it was captured")
              : t("As it was captured. It will be filed turned {0}° clockwise.", rotation)}
          </DialogDescription>
        </DialogHeader>
        {page !== null && <PdfViewer file={apiUrl(page.contentPath)} className="min-h-[60vh]" />}
      </DialogContent>
    </Dialog>
  );
}
