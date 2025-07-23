/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import { type Document } from "@/types/document";
import { LazyImage } from "../ui/image";
import { DocumentActions } from "./document-workflow-actions";

export function DocumentCard({ document }: { document: Document }) {
  return (
    <DocumentCardOuter>
      <DocumentCardInner>
        <DocumentActions document={document} />
        <DocumentPreview document={document} />
      </DocumentCardInner>
    </DocumentCardOuter>
  );
}

function DocumentPreview({ document }: { document: Document }) {
  return (
    <div className="relative size-[200px] shadow-md">
      <LazyImage
        src={document.previewUrl || ""}
        alt={document.fileName}
        width={200}
        height={200}
        className="w-full h-full object-cover"
      />
    </div>
  );
}

function DocumentCardOuter({ children }: { children: React.ReactNode }) {
  return (
    <div className="border border-border rounded-md overflow-hidden bg-card relative min-h-[200px] cursor-pointer">
      {children}
    </div>
  );
}

function DocumentCardInner({ children }: { children: React.ReactNode }) {
  return (
    <div className="size-full flex items-center justify-center p-2 bg-muted">
      {children}
    </div>
  );
}
