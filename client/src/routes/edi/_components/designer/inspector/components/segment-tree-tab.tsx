import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import { cn } from "@/lib/utils";
import type { EDIDiagnostic } from "@/types/edi";
import { CopyIcon } from "lucide-react";
import {
  diagnosticsForX12Element,
  diagnosticsForX12Segment,
  type ParsedX12Document,
  type X12Segment,
} from "../utils/x12-parser";

export default function SegmentTreeTab({
  document,
  diagnostics,
  selectedSegmentIndex,
  onSelectSegment,
}: {
  document: ParsedX12Document;
  diagnostics: EDIDiagnostic[];
  selectedSegmentIndex: number;
  onSelectSegment: (segmentIndex: number) => void;
}) {
  const selectedSegment =
    document.segments.find((segment) => segment.index === selectedSegmentIndex) ??
    document.segments[0];

  return (
    <div className="grid h-full min-h-0 grid-cols-[340px_minmax(0,1fr)] gap-3 max-lg:grid-cols-1">
      <div className="min-h-0 overflow-auto rounded-md border">
        {document.segments.map((segment) => {
          const segmentDiagnostics = diagnosticsForX12Segment(diagnostics, segment);
          return (
            <button
              key={`${segment.index}-${segment.raw}`}
              type="button"
              onClick={() => onSelectSegment(segment.index)}
              className={cn(
                "flex w-full items-center justify-between gap-2 border-b px-3 py-2 text-left hover:bg-muted",
                selectedSegment?.index === segment.index && "bg-muted",
              )}
            >
              <span className="min-w-0">
                <span className="flex items-center gap-2">
                  <span className="w-8 font-mono text-xs text-muted-foreground">
                    {segment.index}
                  </span>
                  <span className="font-mono text-sm font-semibold">{segment.segmentId}</span>
                  {segment.control ? <Badge variant="outline">Control</Badge> : null}
                </span>
                <span className="block truncate pl-10 text-xs text-muted-foreground">
                  {segment.label}
                </span>
              </span>
              {segmentDiagnostics.length > 0 ? (
                <Badge
                  variant={
                    segmentDiagnostics.some((diagnostic) => diagnostic.severity === "Error")
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
      {selectedSegment ? (
        <SegmentDetail segment={selectedSegment} diagnostics={diagnostics} />
      ) : (
        <div className="rounded-md border p-4 text-sm text-muted-foreground">Select a segment.</div>
      )}
    </div>
  );
}

function SegmentDetail({
  segment,
  diagnostics,
}: {
  segment: X12Segment;
  diagnostics: EDIDiagnostic[];
}) {
  const { copy } = useCopyToClipboard();
  const segmentDiagnostics = diagnosticsForX12Segment(diagnostics, segment);

  return (
    <div className="grid min-h-0 grid-rows-[auto_minmax(0,1fr)] rounded-md border">
      <div className="border-b p-3">
        <div className="flex flex-wrap items-center gap-2">
          <Badge variant={segment.control ? "active" : "outline"}>{segment.segmentId}</Badge>
          <div className="text-sm font-semibold">{segment.label}</div>
          {segment.malformed ? <Badge variant="inactive">Malformed</Badge> : null}
        </div>
        <div className="mt-2 flex flex-wrap items-center gap-2">
          <code className="rounded-sm bg-muted px-2 py-1 text-xs wrap-break-word">
            {segment.raw}
          </code>
          <Button
            type="button"
            size="xs"
            variant="outline"
            onClick={() => void copy(segment.raw, { withToast: true })}
          >
            <CopyIcon className="size-3.5" />
            Copy segment
          </Button>
        </div>
      </div>
      <div className="min-h-0 overflow-auto">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-20">Element</TableHead>
              <TableHead>Name</TableHead>
              <TableHead>Value</TableHead>
              <TableHead className="w-28">Usage</TableHead>
              <TableHead className="w-24">Issues</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {segment.elements.map((element) => {
              const elementDiagnostics = diagnosticsForX12Element(
                diagnostics,
                segment,
                element.position,
              );
              return (
                <TableRow key={`${segment.index}-${element.position}`}>
                  <TableCell className="font-mono text-xs">
                    {segment.segmentId}
                    {String(element.position).padStart(2, "0")}
                  </TableCell>
                  <TableCell>{element.label}</TableCell>
                  <TableCell className="font-mono text-xs wrap-break-word">
                    <div>{element.empty ? "[empty]" : element.value}</div>
                    {element.components.length > 1 ? (
                      <div className="mt-1 text-muted-foreground">
                        {element.components
                          .map((component) => `${component.index}: ${component.value || "[empty]"}`)
                          .join(" | ")}
                      </div>
                    ) : null}
                  </TableCell>
                  <TableCell>
                    <Badge variant={element.required ? "warning" : "outline"}>
                      {element.required ? "Required" : "Optional"}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    {elementDiagnostics.length > 0 ? (
                      <Badge
                        variant={
                          elementDiagnostics.some((diagnostic) => diagnostic.severity === "Error")
                            ? "inactive"
                            : "warning"
                        }
                      >
                        {elementDiagnostics.length}
                      </Badge>
                    ) : null}
                  </TableCell>
                </TableRow>
              );
            })}
          </TableBody>
        </Table>
        {segmentDiagnostics.length > 0 ? (
          <div className="space-y-2 border-t p-3">
            {segmentDiagnostics.map((diagnostic) => (
              <div
                key={`${diagnostic.code}-${diagnostic.elementPosition}-${diagnostic.message}`}
                className="rounded-md border p-2 text-sm"
              >
                <div className="flex items-center gap-2">
                  <Badge variant={diagnostic.severity === "Error" ? "inactive" : "warning"}>
                    {diagnostic.severity}
                  </Badge>
                  <span className="font-mono text-xs">{diagnostic.code}</span>
                </div>
                <div className="mt-1">{diagnostic.message}</div>
              </div>
            ))}
          </div>
        ) : null}
      </div>
    </div>
  );
}
