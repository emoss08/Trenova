import { Button } from "@trenova/shared/components/ui/button";
import { Checkbox } from "@trenova/shared/components/ui/checkbox";
import { Input } from "@trenova/shared/components/ui/input";
import { Label } from "@trenova/shared/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import { Switch } from "@trenova/shared/components/ui/switch";
import {
  MAX_CHARTS,
  MAX_CHART_SERIES,
  REPORT_CHART_TYPE_CHOICES,
  chartNeedsCoordinates,
  chartNeedsDimension,
  chartSingleSeries,
  chartSupportsStacking,
  type ReportChartSpec,
  type ReportChartType,
  type ReportIR,
} from "@/types/report";
import { ChartColumnIcon, PlusIcon, XIcon } from "lucide-react";
import { outputColumnChoices, type CatalogIndex } from "./builder-state";

type ChartsPanelProps = {
  index: CatalogIndex;
  ir: ReportIR;
  onChange: (charts: ReportChartSpec[]) => void;
};

const NONE = "__none__";

type CoordinateChoice = { id: string; label: string };

/**
 * Seeds a coordinate when a chart is switched to a map. Matching on the label
 * is a convenience, not a contract — the author can always pick, and an
 * unmatched report simply starts empty rather than pointing somewhere wrong.
 */
function guessCoordinate(choices: CoordinateChoice[], axis: "lat" | "lng"): string | undefined {
  const needles = axis === "lat" ? ["latitude", "lat"] : ["longitude", "lng", "long"];
  const match = choices.find((choice) => {
    const label = choice.label.toLowerCase();
    return needles.some((needle) => label === needle || label.endsWith(` ${needle}`));
  });
  return match?.id;
}

function CoordinateField({
  label,
  value,
  choices,
  optional,
  onChange,
}: {
  label: string;
  value: string | undefined;
  choices: CoordinateChoice[];
  optional?: boolean;
  onChange: (id: string | undefined) => void;
}) {
  const items = optional ? [{ id: NONE, label: "None" }, ...choices] : choices;

  return (
    <div className="flex flex-col gap-1">
      <Label className="text-xs text-muted-foreground">{label}</Label>
      <Select
        value={value ?? (optional ? NONE : "")}
        onValueChange={(next) => {
          if (!next) return;
          onChange(next === NONE ? undefined : next);
        }}
        items={items.map((choice) => ({ value: choice.id, label: choice.label }))}
      >
        <SelectTrigger className="h-7">
          <SelectValue placeholder="Choose column" />
        </SelectTrigger>
        <SelectContent>
          {items.map((choice) => (
            <SelectItem key={choice.id} value={choice.id}>
              {choice.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  );
}

function nextChartId(charts: ReportChartSpec[]): string {
  let suffix = charts.length + 1;
  while (charts.some((chart) => chart.id === `chart_${suffix}`)) suffix += 1;
  return `chart_${suffix}`;
}

function ChartEditor({
  ir,
  index,
  chart,
  onUpdate,
  onRemove,
}: {
  ir: ReportIR;
  index: CatalogIndex;
  chart: ReportChartSpec;
  onUpdate: (chart: ReportChartSpec) => void;
  onRemove: () => void;
}) {
  const outputs = outputColumnChoices(index, ir);
  const dimensions = outputs.filter((output) => output.isDim);
  const measures = outputs.filter((output) => !output.isDim);
  const seriesIds = chart.seriesIds ?? [];
  const singleSeries = chartSingleSeries(chart.type);
  const needsDimension = chartNeedsDimension(chart.type);

  const setType = (type: ReportChartType) => {
    onUpdate({
      ...chart,
      type,
      // Switching encoding can invalidate the axes, so anything the new type
      // cannot express is dropped rather than sent for the server to reject.
      xColumnId: chartNeedsDimension(type)
        ? (dimensions.find((d) => d.id === chart.xColumnId)?.id ?? dimensions[0]?.id)
        : type === "scatter"
          ? (measures.find((m) => m.id === chart.xColumnId)?.id ?? measures[0]?.id)
          : undefined,
      seriesIds: chartSingleSeries(type) ? seriesIds.slice(0, 1) : seriesIds,
      stacked: chartSupportsStacking(type) ? chart.stacked : false,
      compareId: type === "kpi" ? chart.compareId : undefined,
      // Coordinates only mean something on a map, and a map without them
      // cannot render — so switching in seeds the first plausible pair.
      latColumnId: chartNeedsCoordinates(type)
        ? (chart.latColumnId ?? guessCoordinate(outputs, "lat"))
        : undefined,
      lngColumnId: chartNeedsCoordinates(type)
        ? (chart.lngColumnId ?? guessCoordinate(outputs, "lng"))
        : undefined,
      labelColumnId: chartNeedsCoordinates(type) ? chart.labelColumnId : undefined,
    });
  };

  const toggleSeries = (id: string, checked: boolean) => {
    if (singleSeries) {
      onUpdate({ ...chart, seriesIds: checked ? [id] : [] });
      return;
    }
    const next = checked
      ? [...seriesIds, id].slice(0, MAX_CHART_SERIES)
      : seriesIds.filter((seriesId) => seriesId !== id);
    onUpdate({ ...chart, seriesIds: next });
  };

  const axisChoices = needsDimension ? dimensions : measures;

  return (
    <div className="flex flex-col gap-2 rounded-md border border-border p-2">
      <div className="flex items-center gap-1.5">
        <ChartColumnIcon className="size-3.5 text-muted-foreground" />
        <Input
          className="h-7 flex-1"
          value={chart.title ?? ""}
          placeholder="Chart title"
          onChange={(event) => onUpdate({ ...chart, title: event.target.value || undefined })}
        />
        <Button
          variant="ghost"
          size="icon"
          className="size-6"
          onClick={onRemove}
          aria-label="Remove chart"
        >
          <XIcon className="size-3.5" />
        </Button>
      </div>

      <div className="grid grid-cols-2 gap-2">
        <div className="flex flex-col gap-1">
          <Label className="text-xs text-muted-foreground">Type</Label>
          <Select
            value={chart.type}
            onValueChange={(type) => {
              if (type) setType(type as ReportChartType);
            }}
            items={REPORT_CHART_TYPE_CHOICES}
          >
            <SelectTrigger className="h-7">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {REPORT_CHART_TYPE_CHOICES.map((choice) => (
                <SelectItem key={choice.value} value={choice.value}>
                  {choice.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        {chartNeedsCoordinates(chart.type) && (
          <>
            <CoordinateField
              label="Latitude"
              value={chart.latColumnId}
              choices={outputs}
              onChange={(latColumnId) => onUpdate({ ...chart, latColumnId })}
            />
            <CoordinateField
              label="Longitude"
              value={chart.lngColumnId}
              choices={outputs}
              onChange={(lngColumnId) => onUpdate({ ...chart, lngColumnId })}
            />
            <CoordinateField
              label="Pin label"
              value={chart.labelColumnId}
              choices={outputs}
              optional
              onChange={(labelColumnId) => onUpdate({ ...chart, labelColumnId })}
            />
          </>
        )}

        {chart.type !== "kpi" && !chartNeedsCoordinates(chart.type) && (
          <div className="flex flex-col gap-1">
            <Label className="text-xs text-muted-foreground">
              {needsDimension ? "Group by" : "Horizontal axis"}
            </Label>
            <Select
              value={chart.xColumnId ?? ""}
              onValueChange={(xColumnId) => {
                if (xColumnId) onUpdate({ ...chart, xColumnId });
              }}
              items={axisChoices.map((choice) => ({ value: choice.id, label: choice.label }))}
            >
              <SelectTrigger className="h-7">
                <SelectValue placeholder="Choose column" />
              </SelectTrigger>
              <SelectContent>
                {axisChoices.map((choice) => (
                  <SelectItem key={choice.id} value={choice.id}>
                    {choice.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        )}

        {chart.type === "kpi" && (
          <div className="flex flex-col gap-1">
            <Label className="text-xs text-muted-foreground">Compare against</Label>
            <Select
              value={chart.compareId ?? NONE}
              onValueChange={(compareId) => {
                if (!compareId) return;
                onUpdate({
                  ...chart,
                  compareId: compareId === NONE ? undefined : compareId,
                });
              }}
              items={[
                { value: NONE, label: "No comparison" },
                ...measures.map((choice) => ({ value: choice.id, label: choice.label })),
              ]}
            >
              <SelectTrigger className="h-7">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value={NONE}>No comparison</SelectItem>
                {measures.map((choice) => (
                  <SelectItem key={choice.id} value={choice.id}>
                    {choice.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        )}
      </div>

      <div className="flex flex-col gap-1">
        <Label className="text-xs text-muted-foreground">
          {singleSeries ? "Measure" : "Measures"}
        </Label>
        {measures.length === 0 ? (
          <p className="text-2xs text-muted-foreground">
            Add a measure column before plotting this report.
          </p>
        ) : (
          <div className="flex flex-col gap-1">
            {measures.map((choice) => (
              <label key={choice.id} className="flex items-center gap-2 text-xs">
                <Checkbox
                  checked={seriesIds.includes(choice.id)}
                  onCheckedChange={(checked) => toggleSeries(choice.id, Boolean(checked))}
                />
                <span className="truncate">{choice.label}</span>
              </label>
            ))}
          </div>
        )}
      </div>

      <div className="grid grid-cols-2 gap-2">
        {chartSupportsStacking(chart.type) && seriesIds.length > 1 && (
          <div className="flex items-center justify-between gap-2 rounded-md border border-border px-2 py-1">
            <Label className="text-xs text-muted-foreground">Stacked</Label>
            <Switch
              checked={chart.stacked ?? false}
              onCheckedChange={(stacked) => onUpdate({ ...chart, stacked })}
            />
          </div>
        )}
        {(chart.type === "line" || chart.type === "area") && (
          <div className="flex items-center justify-between gap-2 rounded-md border border-border px-2 py-1">
            <Label className="text-xs text-muted-foreground">Smooth</Label>
            <Switch
              checked={chart.curved ?? false}
              onCheckedChange={(curved) => onUpdate({ ...chart, curved })}
            />
          </div>
        )}
        {(chart.type === "bar" || chart.type === "hbar") && (
          <div className="flex items-center justify-between gap-2 rounded-md border border-border px-2 py-1">
            <Label className="text-xs text-muted-foreground">Show values</Label>
            <Switch
              checked={chart.showValues ?? false}
              onCheckedChange={(showValues) => onUpdate({ ...chart, showValues })}
            />
          </div>
        )}
        {seriesIds.length > 1 && chart.type !== "kpi" && (
          <div className="flex items-center justify-between gap-2 rounded-md border border-border px-2 py-1">
            <Label className="text-xs text-muted-foreground">Legend</Label>
            <Switch
              checked={!chart.hideLegend}
              onCheckedChange={(shown) => onUpdate({ ...chart, hideLegend: !shown })}
            />
          </div>
        )}
        {chart.type !== "kpi" && (
          <div className="flex flex-col gap-1">
            <Label className="text-xs text-muted-foreground">Top N categories</Label>
            <Input
              className="h-7"
              type="number"
              min={0}
              placeholder="All"
              value={chart.limit ? String(chart.limit) : ""}
              onChange={(event) => {
                const parsed = Number.parseInt(event.target.value, 10);
                onUpdate({
                  ...chart,
                  limit: Number.isNaN(parsed) || parsed <= 0 ? undefined : parsed,
                });
              }}
            />
          </div>
        )}
      </div>

      <div className="grid grid-cols-2 gap-2 border-t border-border/60 pt-2">
        <div className="flex flex-col gap-1">
          <Label className="text-xs text-muted-foreground">Target value</Label>
          <Input
            className="h-7"
            type="number"
            step="any"
            placeholder="None"
            disabled={Boolean(chart.goal?.columnId)}
            value={chart.goal?.value ?? ""}
            onChange={(event) => {
              const parsed = Number.parseFloat(event.target.value);
              onUpdate({
                ...chart,
                goal: Number.isNaN(parsed)
                  ? undefined
                  : { ...chart.goal, value: parsed, columnId: undefined },
              });
            }}
          />
        </div>
        <div className="flex flex-col gap-1">
          <Label className="text-xs text-muted-foreground">Target column</Label>
          <Select
            value={chart.goal?.columnId ?? NONE}
            onValueChange={(columnId) => {
              if (!columnId) return;
              onUpdate({
                ...chart,
                goal:
                  columnId === NONE
                    ? chart.goal?.value !== undefined
                      ? { ...chart.goal, columnId: undefined }
                      : undefined
                    : { ...chart.goal, columnId, value: undefined },
              });
            }}
            items={[
              { value: NONE, label: "None" },
              ...measures.map((choice) => ({ value: choice.id, label: choice.label })),
            ]}
          >
            <SelectTrigger className="h-7">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value={NONE}>None</SelectItem>
              {measures.map((choice) => (
                <SelectItem key={choice.id} value={choice.id}>
                  {choice.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        {(chart.goal?.value !== undefined || chart.goal?.columnId) && (
          <div className="col-span-2 flex flex-col gap-1">
            <Label className="text-xs text-muted-foreground">Target label</Label>
            <Input
              className="h-7"
              placeholder="Target"
              value={chart.goal?.label ?? ""}
              onChange={(event) =>
                onUpdate({
                  ...chart,
                  goal: { ...chart.goal, label: event.target.value || undefined },
                })
              }
            />
          </div>
        )}
      </div>
    </div>
  );
}

export function ChartsPanel({ index, ir, onChange }: ChartsPanelProps) {
  const charts = ir.charts ?? [];
  const outputs = outputColumnChoices(index, ir);
  const measures = outputs.filter((output) => !output.isDim);
  const dimensions = outputs.filter((output) => output.isDim);

  // A map is positioned by coordinates, so it is the one encoding that needs
  // no measure at all — offering it here is what makes a plain list of places
  // plottable.
  const canMap = !!guessCoordinate(outputs, "lat") && !!guessCoordinate(outputs, "lng");

  if (measures.length === 0 && !canMap) {
    return (
      <p className="px-2 py-4 text-center text-sm text-muted-foreground">
        Charts plot measures — add one to the report first.
      </p>
    );
  }

  const addChart = () => {
    if (measures.length === 0) {
      onChange([
        ...charts,
        {
          id: nextChartId(charts),
          type: "map",
          latColumnId: guessCoordinate(outputs, "lat"),
          lngColumnId: guessCoordinate(outputs, "lng"),
        },
      ]);
      return;
    }
    onChange([
      ...charts,
      {
        id: nextChartId(charts),
        type: dimensions.length > 0 ? "bar" : "kpi",
        xColumnId: dimensions.length > 0 ? dimensions[0].id : undefined,
        seriesIds: [measures[0].id],
      },
    ]);
  };

  return (
    <div className="flex flex-col gap-2">
      {charts.map((chart, chartIndex) => (
        <ChartEditor
          key={chart.id}
          index={index}
          ir={ir}
          chart={chart}
          onUpdate={(updated) => onChange(charts.map((c, i) => (i === chartIndex ? updated : c)))}
          onRemove={() => onChange(charts.filter((_, i) => i !== chartIndex))}
        />
      ))}
      <Button
        variant="outline"
        size="sm"
        className="h-7 self-start"
        disabled={charts.length >= MAX_CHARTS}
        onClick={addChart}
      >
        <PlusIcon className="size-3.5" />
        Chart
      </Button>
    </div>
  );
}
