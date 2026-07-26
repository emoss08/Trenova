import { Button } from "@trenova/shared/components/ui/button";
import { Input } from "@trenova/shared/components/ui/input";
import { Label } from "@trenova/shared/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import { formatReportValue, resolveReportDisplay } from "@trenova/shared/lib/report-format";
import {
  MAX_BAND_EDGES,
  bandIsUsable,
  bandMode,
  type ReportBandMode,
  type ReportBandSpec,
  type ReportColumnSpec,
} from "@/types/report";
import { PlusIcon, XIcon } from "lucide-react";
import { useMemo } from "react";

type BandEditorProps = {
  column: ReportColumnSpec;
  valueType: string;
  formatHint: string | undefined;
  onUpdate: (column: ReportColumnSpec) => void;
};

const MODE_CHOICES: { value: ReportBandMode | "none"; label: string }[] = [
  { value: "none", label: "Exact value" },
  { value: "width", label: "Even ranges" },
  { value: "edges", label: "Custom ranges" },
];

/** Enough rows to show the shape of the ranges without filling the panel. */
const PREVIEW_LIMIT = 4;

/**
 * Groups a numeric dimension into ranges. A charge amount or a dwell time has
 * far too many distinct values to group by directly; the ranges are what turn
 * it into a distribution someone can read.
 */
export function BandEditor({ column, valueType, formatHint, onUpdate }: BandEditorProps) {
  const band = column.band;
  const mode: ReportBandMode | "none" = band ? bandMode(band) : "none";
  const edges = band?.edges ?? [];

  const setBand = (next: ReportBandSpec | undefined) => {
    // A band already reshapes the value, so it cannot sit on top of a transform.
    onUpdate({ ...column, band: next, transform: next ? undefined : column.transform });
  };

  const setMode = (next: ReportBandMode | "none") => {
    if (next === mode) return;
    switch (next) {
      case "none":
        setBand(undefined);
        break;
      case "width":
        setBand({ width: band?.width && band.width > 0 ? band.width : 1000 });
        break;
      case "edges":
        setBand({ edges: edges.length >= 2 ? edges : [0, 1000] });
        break;
    }
  };

  const updateEdge = (position: number, value: number) =>
    setBand({ edges: edges.map((edge, i) => (i === position ? value : edge)) });

  const removeEdge = (position: number) =>
    setBand({ edges: edges.filter((_, i) => i !== position) });

  const addEdge = () => {
    const last = edges.at(-1) ?? 0;
    const previous = edges.at(-2) ?? last - 1000;
    setBand({ edges: [...edges, last + Math.max(last - previous, 1)] });
  };

  const ascending = edges.every((edge, i) => i === 0 || edge > edges[i - 1]);

  return (
    <div className="col-span-2 flex flex-col gap-2">
      <div className="flex flex-col gap-1">
        <Label className="text-xs text-muted-foreground">Group into</Label>
        <Select
          value={mode}
          onValueChange={(next) => {
            if (next) setMode(next as ReportBandMode | "none");
          }}
          items={MODE_CHOICES}
        >
          <SelectTrigger className="h-7">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {MODE_CHOICES.map((choice) => (
              <SelectItem key={choice.value} value={choice.value}>
                {choice.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      {mode === "width" && (
        <div className="flex flex-col gap-1">
          <Label className="text-xs text-muted-foreground">Range size</Label>
          <Input
            className="h-7"
            type="number"
            min={0}
            step="any"
            value={band?.width ?? ""}
            onChange={(event) => setBand({ width: Number(event.target.value) })}
          />
        </div>
      )}

      {mode === "edges" && (
        <div className="flex flex-col gap-1.5">
          <div className="flex items-center gap-2">
            <Label className="text-xs text-muted-foreground">Range starts at</Label>
            <div className="flex-1" />
            <Button
              variant="ghost"
              size="sm"
              className="h-6"
              disabled={edges.length >= MAX_BAND_EDGES}
              onClick={addEdge}
            >
              <PlusIcon className="size-3.5" />
              Range
            </Button>
          </div>
          {edges.map((edge, position) => (
            <div key={position} className="flex items-center gap-1.5">
              <Input
                className="h-7"
                type="number"
                step="any"
                value={edge}
                onChange={(event) => updateEdge(position, Number(event.target.value))}
              />
              <Button
                variant="ghost"
                size="icon"
                className="size-6 shrink-0"
                disabled={edges.length <= 2}
                onClick={() => removeEdge(position)}
                aria-label="Remove range"
              >
                <XIcon className="size-3.5" />
              </Button>
            </div>
          ))}
          {!ascending && (
            <p className="text-2xs text-destructive">
              Each range has to start above the one before it.
            </p>
          )}
        </div>
      )}

      {bandIsUsable(band) && ascending && (
        <BandPreview
          band={band as ReportBandSpec}
          column={column}
          valueType={valueType}
          formatHint={formatHint}
        />
      )}
    </div>
  );
}

/**
 * Shows the rows the report will actually produce. Both ends render through the
 * column's own display, which is the same path the grid and the exports take.
 */
function BandPreview({
  band,
  column,
  valueType,
  formatHint,
}: {
  band: ReportBandSpec;
  column: ReportColumnSpec;
  valueType: string;
  formatHint: string | undefined;
}) {
  const labels = useMemo(() => {
    const display = resolveReportDisplay({
      type: valueType,
      hint: formatHint,
      band: { width: band.width ?? 0, edges: band.edges ?? [] },
      overrides: column.display,
    });

    const lowerBounds =
      band.edges && band.edges.length > 0
        ? band.edges
        : Array.from({ length: PREVIEW_LIMIT }, (_, i) => i * (band.width ?? 0));

    return lowerBounds.slice(0, PREVIEW_LIMIT).map((bound) => formatReportValue(bound, display));
  }, [band, column.display, formatHint, valueType]);

  const truncated = (band.edges?.length ?? PREVIEW_LIMIT + 1) > PREVIEW_LIMIT;

  return (
    <div className="flex flex-wrap items-center gap-1">
      {labels.map((label, i) => (
        <span
          key={i}
          className="rounded-sm bg-muted px-1.5 py-px font-mono text-2xs text-foreground/80 tabular-nums"
        >
          {label}
        </span>
      ))}
      {truncated && <span className="text-2xs text-muted-foreground">…</span>}
    </div>
  );
}
