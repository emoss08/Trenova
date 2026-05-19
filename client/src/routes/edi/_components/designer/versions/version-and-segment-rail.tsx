import { Badge } from "@/components/ui/badge";
import { ScrollArea } from "@/components/ui/scroll-area";
import { cn } from "@/lib/utils";
import type { EDIDiagnostic, EDITemplateSegment, EDITemplateVersion } from "@/types/edi";
import { VersionStatusBadge } from "../components/designer-shared";
import { diagnosticsForSegment } from "../utils/edi-designer-utils";

type VersionAndSegmentRailProps = {
  versions: EDITemplateVersion[];
  selectedVersionId: string;
  onVersionSelect: (versionId: string) => void;
  segments: EDITemplateSegment[];
  diagnostics: EDIDiagnostic[];
  selectedSegmentId: string;
  onSegmentSelect: (segment: EDITemplateSegment) => void;
};

export default function VersionAndSegmentRail({
  versions,
  selectedVersionId,
  onVersionSelect,
  segments,
  diagnostics,
  selectedSegmentId,
  onSegmentSelect,
}: VersionAndSegmentRailProps) {
  return (
    <div className="grid min-h-0 grid-rows-[180px_minmax(0,1fr)] border-r">
      <div className="min-h-0 overflow-auto border-b">
        <div className="sticky top-0 z-10 border-b bg-sidebar px-3 py-2">
          <div className="text-sm font-semibold">Versions</div>
          <div className="text-xs text-muted-foreground">{versions.length} available</div>
        </div>
        {versions.length === 0 ? (
          <div className="p-3 text-sm text-muted-foreground">No versions.</div>
        ) : (
          versions.map((version) => (
            <button
              key={version.id}
              type="button"
              onClick={() => onVersionSelect(version.id)}
              className={cn(
                "flex w-full items-center justify-between gap-2 border-b px-3 py-2 text-left hover:bg-muted",
                selectedVersionId === version.id && "bg-muted",
              )}
            >
              <span className="font-mono text-xs">v{version.versionNumber}</span>
              <VersionStatusBadge version={version} />
            </button>
          ))
        )}
      </div>
      <div className="min-h-0 overflow-auto">
        <div className="sticky top-0 z-10 border-b bg-background px-3 py-2">
          <div className="text-xs font-semibold">Segment Outline</div>
          <div className="text-xs text-muted-foreground">{segments.length} segments</div>
        </div>
        <ScrollArea className="flex h-[calc(100vh-30rem)] flex-col">
          {segments.length === 0 ? (
            <div className="p-3 text-sm text-muted-foreground">No segments in this version.</div>
          ) : (
            segments.map((segment) => {
              const segmentDiagnostics = diagnosticsForSegment(diagnostics, segment);
              return (
                <button
                  key={segment.id}
                  type="button"
                  onClick={() => onSegmentSelect(segment)}
                  className={cn(
                    "flex w-full cursor-pointer items-center justify-between gap-2 border-b px-3 py-2 text-left hover:bg-muted",
                    selectedSegmentId === segment.id && "bg-muted ring-1 ring-border ring-inset",
                  )}
                >
                  <span className="min-w-0">
                    <span className="flex items-center gap-2">
                      <span className="font-mono text-sm font-medium">{segment.segmentId}</span>
                      <Badge variant={segment.required ? "active" : "outline"}>
                        {segment.required ? "Req" : "Opt"}
                      </Badge>
                    </span>
                    <span className="block truncate text-xs text-muted-foreground">
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
