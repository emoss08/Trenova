/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

export function PDFFooter({
  numPages,
  pageNumber,
  scale,
  rotation,
  searchText,
}: {
  numPages: number;
  pageNumber: number;
  scale: number;
  rotation: number;
  searchText: string;
}) {
  return (
    <div className="bg-background border-t border-input px-4 py-2 text-xs text-muted-foreground flex justify-between items-center sticky bottom-0 left-0 right-0">
      <div>
        {numPages > 0 && (
          <span>
            Page {pageNumber} of {numPages}
          </span>
        )}
      </div>
      <div className="flex items-center gap-4">
        <div>Zoom: {Math.round(scale * 100)}%</div>
        <div>Rotation: {rotation}°</div>
        {searchText && <div>Search: &ldquo;{searchText}&rdquo;</div>}
      </div>
    </div>
  );
}
