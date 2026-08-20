import { Badge } from "@trenova/shared/components/ui/badge";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import {
  useSelectedTemplateDesignerData,
  useSelectedTemplateDesignerIds,
  useTemplateDesignerUrlActions,
} from "@/hooks/use-template-designer-state";
import { cn } from "@trenova/shared/lib/utils";
import { useTemplateDesignerStore } from "@/stores/template-designer-store";
import { VersionStatusBadge } from "../components/designer-shared";
import { diagnosticsForSegment } from "../utils/edi-designer-utils";

export default function VersionAndSegmentRail() {
  const { versions } = useSelectedTemplateDesignerData();
  const { selectedVersionId, selectedSegmentId, selectedElementPosition } =
    useSelectedTemplateDesignerIds();
  const segments = useTemplateDesignerStore((state) => state.segmentsDraft);
  const diagnostics = useTemplateDesignerStore((state) => state.diagnostics);
  const resetDraftState = useTemplateDesignerStore((state) => state.resetDraftState);
  const { patchTemplateUrlState } = useTemplateDesignerUrlActions();

  return (
    <div className="grid min-h-0 grid-rows-[180px_minmax(0,1fr)] overflow-hidden border-r">
      <ScrollArea className="min-h-0 border-b" viewportClassName="min-h-0">
        <div className="bg-sidebar sticky top-0 z-10 border-b px-3 py-2">
          <div className="text-sm font-semibold">Versions</div>
          <div className="text-muted-foreground text-xs">{versions.length} available</div>
        </div>
        {versions.length === 0 ? (
          <div className="text-muted-foreground p-3 text-sm">No versions.</div>
        ) : (
          versions.map((version) => (
            <button
              key={version.id}
              type="button"
              onClick={() => {
                if (selectedVersionId === version.id) return;
                resetDraftState();
                patchTemplateUrlState({
                  versionId: version.id,
                  segmentId: "",
                  elementPosition: 0,
                });
              }}
              className={cn(
                "hover:bg-muted flex w-full items-center justify-between gap-2 border-b px-3 py-2 text-left",
                selectedVersionId === version.id && "bg-muted",
              )}
            >
              <span className="font-mono text-xs">v{version.versionNumber}</span>
              <VersionStatusBadge version={version} />
            </button>
          ))
        )}
      </ScrollArea>
      <div className="grid min-h-0 grid-rows-[auto_minmax(0,1fr)] overflow-hidden">
        <div className="bg-background sticky top-0 z-10 border-b px-3 py-2">
          <div className="text-xs font-semibold">Segment Outline</div>
          <div className="text-muted-foreground text-xs">{segments.length} segments</div>
        </div>
        <ScrollArea className="min-h-0" viewportClassName="min-h-0">
          {segments.length === 0 ? (
            <div className="text-muted-foreground p-3 text-sm">No segments in this version.</div>
          ) : (
            segments.map((segment) => {
              const segmentDiagnostics = diagnosticsForSegment(diagnostics, segment);
              return (
                <button
                  key={segment.id}
                  type="button"
                  onClick={() => {
                    const firstElementPosition = segment.elements[0]?.position ?? 0;
                    if (
                      selectedSegmentId === segment.id &&
                      selectedElementPosition === firstElementPosition
                    ) {
                      return;
                    }
                    patchTemplateUrlState({
                      segmentId: segment.id,
                      elementPosition: firstElementPosition,
                    });
                  }}
                  className={cn(
                    "hover:bg-muted flex w-full cursor-pointer items-center justify-between gap-2 border-b px-3 py-2 text-left",
                    selectedSegmentId === segment.id && "bg-muted ring-border ring-1 ring-inset",
                  )}
                >
                  <span className="min-w-0">
                    <span className="flex items-center gap-2">
                      <span className="font-mono text-sm font-medium">{segment.segmentId}</span>
                      <Badge variant={segment.required ? "active" : "outline"}>
                        {segment.required ? "Req" : "Opt"}
                      </Badge>
                    </span>
                    <span className="text-muted-foreground block truncate text-xs">
                      {segment.name}
                    </span>
                  </span>
                  {segmentDiagnostics.length > 0 ? (
                    <Badge
                      variant={
                        segmentDiagnostics.some((item) => item.severity === "Error")
                          ? "inactive"
                          : "warning"
                      }
                    >
                      {segmentDiagnostics.length}
                    </Badge>
                  ) : null}
                </button>
              );
            })
          )}
        </ScrollArea>
      </div>
    </div>
  );
}
