import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Tabs, TabsContent, TabsList, TabsTab } from "@/components/ui/tabs";
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import { cn } from "@/lib/utils";
import type { EDIX12EnvelopeSettings } from "@/types/edi";
import { CopyIcon } from "lucide-react";
import { useMemo } from "react";
import {
  parseX12Document,
  x12DisplayText,
  type X12ControlNumbers,
  type X12Segment,
} from "../designer/inspector/utils/x12-parser";

export function EDIRawX12Viewer({
  content,
  envelope,
}: {
  content: string;
  envelope?: Partial<EDIX12EnvelopeSettings> | null;
}) {
  const { copy } = useCopyToClipboard();
  const document = useMemo(() => parseX12Document(content, envelope), [content, envelope]);
  const rawText = useMemo(() => x12DisplayText(document), [document]);

  const { controlNumbers, seSegmentCount } = document.metadata;
  const controlSummary = summarizeControlNumbers(controlNumbers);

  return (
    <Tabs defaultValue="formatted" className="gap-2">
      <div className="flex items-center justify-between gap-2">
        <TabsList variant="underline">
          <TabsTab value="formatted">Formatted</TabsTab>
          <TabsTab value="raw">Raw X12</TabsTab>
        </TabsList>
        <Button
          type="button"
          size="xs"
          variant="outline"
          onClick={() => void copy(rawText, { withToast: true })}
        >
          <CopyIcon className="size-3.5" />
          Copy
        </Button>
      </div>

      {(controlSummary.length > 0 || document.segments.length > 0) && (
        <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-muted-foreground">
          <span>
            {document.segments.length} segment{document.segments.length === 1 ? "" : "s"}
          </span>
          {controlSummary.map((item) => (
            <span key={item.label}>
              <span className="text-muted-foreground/70">{item.label}</span>{" "}
              <span className="font-mono text-foreground">{item.value}</span>
            </span>
          ))}
          {seSegmentCount && !seSegmentCount.matches && (
            <Badge variant="warning">
              SE count {seSegmentCount.actual}/{seSegmentCount.expected}
            </Badge>
          )}
        </div>
      )}

      <TabsContent value="formatted">
        <div className="max-h-96 min-h-0 divide-y overflow-auto rounded-md border">
          {document.segments.map((segment) => (
            <SegmentRow key={`${segment.index}-${segment.raw}`} segment={segment} />
          ))}
        </div>
      </TabsContent>
      <TabsContent value="raw">
        <pre className="max-h-96 overflow-auto rounded-md border bg-muted/30 p-3 font-mono text-xs whitespace-pre">
          {rawText}
        </pre>
      </TabsContent>
    </Tabs>
  );
}

function SegmentRow({ segment }: { segment: X12Segment }) {
  return (
    <div className="px-3 py-2">
      <div className="flex flex-wrap items-center gap-2">
        <span className="w-6 shrink-0 font-mono text-xs text-muted-foreground">
          {segment.index}
        </span>
        <span className="font-mono text-sm font-semibold">{segment.segmentId}</span>
        <span className="text-sm text-muted-foreground">{segment.label}</span>
        {segment.control && <Badge variant="outline">Control</Badge>}
        {segment.malformed && <Badge variant="inactive">Malformed</Badge>}
      </div>
      {segment.elements.length > 0 && (
        <div className="mt-1.5 grid gap-1 pl-8">
          {segment.elements.map((element) => (
            <div
              key={element.position}
              className="grid grid-cols-[4.5rem_minmax(140px,200px)_minmax(0,1fr)] gap-2 text-xs"
            >
              <span className="font-mono text-muted-foreground">
                {segment.segmentId}
                {String(element.position).padStart(2, "0")}
              </span>
              <span className="truncate text-muted-foreground">{element.label}</span>
              <span className={cn("font-mono wrap-break-word", element.empty && "text-muted-foreground")}>
                {element.empty ? "[empty]" : element.value}
                {element.components.length > 1 && (
                  <span className="ml-2 text-muted-foreground">
                    {element.components
                      .map((component) => component.value || "[empty]")
                      .join(" › ")}
                  </span>
                )}
              </span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function summarizeControlNumbers(controlNumbers: X12ControlNumbers) {
  const entries: { label: string; value: string }[] = [];
  if (controlNumbers.isa) entries.push({ label: "ISA", value: controlNumbers.isa });
  if (controlNumbers.gs) entries.push({ label: "GS", value: controlNumbers.gs });
  if (controlNumbers.st) entries.push({ label: "ST", value: controlNumbers.st });
  return entries;
}
