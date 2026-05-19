import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import type { EDIInspectionDiagnostic, EDIX12Inspection, EDIX12Segment } from "@/types/edi";
import { CopyIcon } from "lucide-react";

export default function FormattedViewTab({
  inspection,
  diagnostics,
  onSelectSegment,
}: {
  inspection: EDIX12Inspection;
  diagnostics: EDIInspectionDiagnostic[];
  onSelectSegment: (segmentIndex: number) => void;
}) {
  const { copy } = useCopyToClipboard();

  return (
    <div className="grid h-full min-h-0 grid-rows-[auto_minmax(0,1fr)]">
      <div className="mb-2 flex items-center gap-2">
        <Button
          type="button"
          variant="outline"
          onClick={() => void copy(inspection.formatted, { withToast: true })}
        >
          <CopyIcon className="size-4" />
          Copy formatted
        </Button>
      </div>
      <div className="min-h-0 overflow-auto rounded-md border">
        {inspection.segments.map((segment) => {
          const segmentDiagnostics = diagnosticsForX12Segment(diagnostics, segment);
          return (
            <button
              key={`${segment.index}-${segment.raw}`}
              type="button"
              onClick={() => onSelectSegment(segment.index)}
              className="block w-full border-b px-3 py-2 text-left hover:bg-muted"
            >
              <div className="flex flex-wrap items-center gap-2">
                <span className="font-mono text-sm font-semibold">{segment.segmentId}</span>
                <span className="text-sm">{segment.name}</span>
                {isControlSegment(segment) ? <Badge variant="outline">Control</Badge> : null}
                {segment.malformed ? <Badge variant="inactive">Malformed</Badge> : null}
                {segmentDiagnostics.length > 0 ? (
                  <Badge variant="warning">{segmentDiagnostics.length}</Badge>
                ) : null}
              </div>
              <div className="mt-2 grid gap-1">
                {segment.elements.map((element) => {
                  const elementDiagnostics = diagnosticsForX12Element(
                    diagnostics,
                    segment,
                    element.position,
                  );
                  return (
                    <div
                      key={`${segment.index}-${element.position}`}
                      className="grid grid-cols-[5rem_minmax(160px,220px)_minmax(0,1fr)] gap-2 text-xs"
                    >
                      <span className="font-mono text-muted-foreground">
                        {segment.segmentId}
                        {String(element.position).padStart(2, "0")}
                      </span>
                      <span className="truncate">{element.label}</span>
                      <span className="font-mono wrap-break-word">
                        {element.empty ? (
                          <span className="text-muted-foreground">[empty]</span>
                        ) : (
                          element.value
                        )}
                        {element.components.length > 1 ? (
                          <span className="ml-2 text-muted-foreground">
                            {element.components
                              .map((component) => component.value || "[empty]")
                              .join(" > ")}
                          </span>
                        ) : null}
                        {elementDiagnostics.length > 0 ? (
                          <Badge variant="warning" className="ml-2">
                            {elementDiagnostics.length}
                          </Badge>
                        ) : null}
                      </span>
                    </div>
                  );
                })}
              </div>
            </button>
          );
        })}
      </div>
    </div>
  );
}

function isControlSegment(segment: EDIX12Segment) {
  return ["interchange", "group", "transaction"].includes(segment.type);
}

function diagnosticsForX12Segment(
  diagnostics: EDIInspectionDiagnostic[],
  segment: EDIX12Segment,
) {
  return diagnostics.filter((diagnostic) => diagnostic.segmentIndex === segment.index);
}

function diagnosticsForX12Element(
  diagnostics: EDIInspectionDiagnostic[],
  segment: EDIX12Segment,
  position: number,
) {
  return diagnosticsForX12Segment(diagnostics, segment).filter(
    (diagnostic) => diagnostic.elementPosition === position,
  );
}
