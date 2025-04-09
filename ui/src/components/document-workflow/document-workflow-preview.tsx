import { type Document } from "@/types/document";
import { LazyImage } from "../ui/image";
import { DocumentActions } from "./document-workflow-actions";

export function DocumentCard({ document }: { document: Document }) {
  return (
    <div className="border border-border rounded-md overflow-hidden bg-card relative min-h-[200px] cursor-pointer">
      <div className="size-full flex items-center justify-center p-2 bg-muted">
        <div className="absolute top-2 right-2">
          <DocumentActions document={document} />
        </div>

        <DocumentPreview document={document} />
      </div>
    </div>
  );
}

function DocumentPreview({ document }: { document: Document }) {
  return (
    <div className="relative size-[200px] shadow-md">
      <LazyImage
        src={document.previewUrl || ""}
        alt={document.fileName}
        layout="constrained"
        width={200}
        height={200}
        className="w-full h-full object-cover"
      />
    </div>
  );
}
