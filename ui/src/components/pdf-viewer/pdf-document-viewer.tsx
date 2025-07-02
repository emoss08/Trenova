import { cn } from "@/lib/utils";
import { faLoader } from "@fortawesome/pro-solid-svg-icons";
import type { PDFDocumentProxy } from "pdfjs-dist";
import { Document, Outline, Page } from "react-pdf";
import "react-pdf/dist/Page/AnnotationLayer.css";
import "react-pdf/dist/Page/TextLayer.css";
import type {
  CustomTextRenderer,
  OnItemClickArgs,
} from "react-pdf/dist/shared/types.js";
import { Icon } from "../ui/icons";
import type { PDFFile } from "./types";

export function PDFDocumentViewer({
  viewerRef,
  showOutline,
  hasOutline,
  isLoading,
  numPages,
  onDocumentLoadSuccess,
  onDocumentLoadError,
  onItemClick,
  onOutlineLoadSuccess,
  onOutlineLoadError,
  outlineRef,
  pageNumber,
  scale,
  rotation,
  containerWidth,
  fileUrl,
  options,
  searchText,
  textRenderer,
}: {
  viewerRef: React.RefObject<HTMLDivElement | null>;
  showOutline: boolean;
  numPages: number;
  hasOutline: boolean;
  isLoading: boolean;
  onDocumentLoadSuccess: (pdf: PDFDocumentProxy) => void;
  onDocumentLoadError: () => void;
  onItemClick: (props: OnItemClickArgs) => void;
  onOutlineLoadSuccess: (outline: unknown) => void;
  onOutlineLoadError: (error: Error) => void;
  outlineRef: React.RefObject<HTMLDivElement | null>;
  pageNumber: number;
  scale: number;
  rotation: number;
  fileUrl: PDFFile;
  options: any;
  searchText: string;
  textRenderer: CustomTextRenderer;
  containerWidth?: number;
}) {
  return (
    <div
      ref={viewerRef}
      className={cn(
        "flex-1 overflow-auto bg-muted p-4 flex flex-col items-center",
        showOutline ? "md:ml-0" : "",
      )}
    >
      {isLoading && (
        <div className="w-full h-64 flex items-center justify-center">
          <div className="flex flex-col items-center">
            <Icon
              icon={faLoader}
              className="animate-spin size-8 text-blue-500"
            />
            <p className="mt-2 text-sm text-muted-foreground">
              Loading PDF document...
            </p>
          </div>
        </div>
      )}

      <Document
        file={fileUrl}
        onLoadSuccess={onDocumentLoadSuccess}
        onLoadError={onDocumentLoadError}
        options={options}
        className="flex flex-col items-center max-h-[calc(100vh-300px)]"
        error={
          <div className="w-full p-8 text-center">
            <div className="inline-flex items-center justify-center w-12 h-12 rounded-full bg-red-100 mb-4">
              <svg
                className="size-6 text-red-600"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
                xmlns="http://www.w3.org/2000/svg"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth="2"
                  d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
                ></path>
              </svg>
            </div>
            <h3 className="text-lg font-medium text-gray-900">
              Failed to load PDF file
            </h3>
            <p className="mt-2 text-sm text-gray-500">
              The document could not be loaded. Please check if the file is
              valid.
            </p>
          </div>
        }
      >
        {/* Outline component inside the Document */}
        {showOutline && hasOutline && (
          <div className="hidden">
            <Outline
              onItemClick={onItemClick}
              onLoadSuccess={onOutlineLoadSuccess}
              onLoadError={onOutlineLoadError}
              inputRef={outlineRef}
              className="text-sm"
            />
          </div>
        )}

        {numPages > 0 && (
          <Page
            key={`page_${pageNumber}`}
            pageNumber={pageNumber}
            scale={scale}
            rotate={rotation}
            width={containerWidth ? Math.min(containerWidth - 32, 800) : 800}
            renderTextLayer={true}
            renderAnnotationLayer={true}
            renderForms={true}
            canvasBackground="#ffffff"
            customTextRenderer={searchText ? textRenderer : undefined}
            loading={
              <div className="animate-pulse flex flex-col items-center mb-4">
                <div className="h-96 w-64 bg-foreground rounded"></div>
              </div>
            }
            onRenderSuccess={() => {
              // Scroll to top of page when rendering a new page
              if (viewerRef.current) {
                viewerRef.current.scrollTop = 0;
              }
            }}
          />
        )}
      </Document>
    </div>
  );
}
