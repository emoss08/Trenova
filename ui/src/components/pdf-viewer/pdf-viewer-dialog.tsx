import { TableSheetProps } from "@/types/data-table";
import "@react-pdf-viewer/core/lib/styles/index.css";
import "@react-pdf-viewer/full-screen/lib/styles/index.css";
import "@react-pdf-viewer/print/lib/styles/index.css";
import "@react-pdf-viewer/search/lib/styles/index.css";
import { lazy } from "react";
import { LazyComponent } from "../error-boundary";
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "../ui/dialog";
import { VisuallyHidden } from "../ui/visually-hidden";

const PDFViewer = lazy(() => import("./pdf-viewer"));

type PDFViewerDialogProps = {
  fileUrl: string;
} & TableSheetProps;

export function PDFViewerDialog({ fileUrl, ...props }: PDFViewerDialogProps) {
  return (
    <Dialog {...props}>
      <DialogContent className="max-h-[90vh] max-w-4xl p-0 overflow-hidden">
        <VisuallyHidden>
          <DialogHeader>
            <DialogTitle>Document Viewer</DialogTitle>
            <DialogDescription>
              View the Document file in the dialog.
            </DialogDescription>
          </DialogHeader>
        </VisuallyHidden>
        <DialogBody className="p-0">
          <LazyComponent>
            <PDFViewer fileUrl={fileUrl} />
          </LazyComponent>
        </DialogBody>
      </DialogContent>
    </Dialog>
  );
}
