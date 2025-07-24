/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { TableSheetProps } from "@/types/data-table";
import { pdfjs } from "react-pdf";
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
import PDFViewer from "./pdf-viewer";

pdfjs.GlobalWorkerOptions.workerSrc = new URL(
  "pdfjs-dist/build/pdf.worker.min.mjs",
  import.meta.url,
).toString();

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
