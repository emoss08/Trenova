import { Badge } from "@/components/ui/badge";
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
        <div className="sticky top-0 min-h-10.25 justify-center border-b bg-sidebar px-3 py-2.5 text-left text-sm font-semibold">
          Versions
        </div>
        {versions.map((version) => (
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
        ))}
      </div>
      <div className="min-h-0 overflow-auto">
        <div className="sticky top-0 border-b bg-background px-3 py-2 text-xs font-semibold">
          Outline
        </div>
        {segments.map((segment) => {
          const segmentDiagnostics = diagnosticsForSegment(diagnostics, segment);
          return (
            <button
              key={segment.id}
              type="button"
              onClick={() => onSegmentSelect(segment)}
              className={cn(
                "flex w-full items-center justify-between gap-2 border-b px-3 py-2 text-left hover:bg-muted",
                selectedSegmentId === segment.id && "bg-muted",
              )}
            >
              <span className="min-w-0">
                <span className="block font-mono text-sm font-medium">{segment.segmentId}</span>
                <span className="block truncate text-xs text-muted-foreground">{segment.name}</span>
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
        })}
      </div>
    </div>
  );
}
