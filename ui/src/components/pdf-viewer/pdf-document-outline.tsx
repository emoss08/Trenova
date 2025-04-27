import { faXmark } from "@fortawesome/pro-solid-svg-icons";
import "react-pdf/dist/esm/Page/AnnotationLayer.css";
import "react-pdf/dist/esm/Page/TextLayer.css";
import { Button } from "../ui/button";
import { Icon } from "../ui/icons";

export function PDFDocumentOutline({
  setShowOutline,
  hasOutline,
}: {
  setShowOutline: (showOutline: boolean) => void;
  hasOutline: boolean;
}) {
  return (
    <div className="w-full md:w-64 bg-background border-r border-input overflow-auto">
      <div className="flex justify-between items-center p-4 border-b border-input">
        <h3 className="text-lg font-medium">Document Outline</h3>
        <Button
          onClick={() => setShowOutline(false)}
          variant="outline"
          className="text-muted-foreground hover:text-muted-foreground/80"
          aria-label="Close outline"
        >
          <Icon icon={faXmark} className="size-5" />
        </Button>
      </div>
      {!hasOutline && (
        <div className="p-4 text-center text-gray-500">
          <p>No outline available for this document.</p>
        </div>
      )}
    </div>
  );
}
