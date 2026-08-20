import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Input } from "@trenova/shared/components/ui/input";
import { Label } from "@trenova/shared/components/ui/label";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectLabel,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import { Textarea } from "@trenova/shared/components/ui/textarea";
import { useCannedReports, useReportCatalog, useReportDefinitionList } from "@/hooks/use-reports";
import { buildCatalogIndex, outputColumnChoices } from "../../builder/_components/builder-state";
import {
  REPORT_TILE_KIND_CHOICES,
  reportOutputColumnIds,
  tileNeedsReport,
  type ReportDashboardTile,
  type ReportDashboardTileKind,
  type ReportParameterDef,
} from "@/types/report";
import { useEffect, useMemo, useState } from "react";
import { useTileReport } from "./use-tile-report";

type TileEditorDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  tile: ReportDashboardTile;
  /** The dashboard's parameters, so the report's can be linked to them. */
  dashboardParams: ReportParameterDef[];
  onSave: (tile: ReportDashboardTile) => void;
};

const UNLINKED = "__unlinked__";

const SAVED_PREFIX = "def:";
const CANNED_PREFIX = "canned:";

function sourceValue(tile: ReportDashboardTile): string {
  if (tile.definitionId) return `${SAVED_PREFIX}${tile.definitionId}`;
  if (tile.cannedKey) return `${CANNED_PREFIX}${tile.cannedKey}`;
  return "";
}

export function TileEditorDialog({
  open,
  onOpenChange,
  tile,
  dashboardParams,
  onSave,
}: TileEditorDialogProps) {
  const [draft, setDraft] = useState<ReportDashboardTile>(tile);

  useEffect(() => {
    if (open) setDraft(tile);
  }, [open, tile]);

  const definitions = useReportDefinitionList("");
  const canned = useCannedReports();
  const report = useTileReport(tileNeedsReport(draft.kind) ? draft : null);

  // Labels come from the catalog so a KPI is chosen by the header it will
  // show, not by the internal column id.
  const catalog = useReportCatalog(tileNeedsReport(draft.kind));
  const index = useMemo(
    () => (catalog.data ? buildCatalogIndex(catalog.data) : null),
    [catalog.data],
  );
  const outputs = useMemo(() => {
    if (!report.ir) return [];
    if (index) return outputColumnChoices(index, report.ir);
    return reportOutputColumnIds(report.ir).map((output) => ({
      ...output,
      label: output.id,
    }));
  }, [index, report.ir]);
  const measures = outputs.filter((output) => !output.isDim);
  const charts = report.ir?.charts ?? [];
  const reportParams = report.ir?.parameters ?? [];

  const sourceItems = useMemo(
    () => [
      ...(definitions.data ?? []).map((definition) => ({
        value: `${SAVED_PREFIX}${definition.id}`,
        label: definition.name,
      })),
      ...(canned.data ?? []).map((entry) => ({
        value: `${CANNED_PREFIX}${entry.key}`,
        label: entry.name,
      })),
    ],
    [definitions.data, canned.data],
  );

  const setKind = (kind: ReportDashboardTileKind) => {
    setDraft((prev) => ({
      ...prev,
      kind,
      chartId: kind === "chart" ? prev.chartId : undefined,
      columnId: kind === "kpi" ? prev.columnId : undefined,
      text: kind === "text" ? (prev.text ?? "") : undefined,
      definitionId: tileNeedsReport(kind) ? prev.definitionId : undefined,
      cannedKey: tileNeedsReport(kind) ? prev.cannedKey : undefined,
    }));
  };

  const setSource = (value: string) => {
    if (value.startsWith(SAVED_PREFIX)) {
      setDraft((prev) => ({
        ...prev,
        definitionId: value.slice(SAVED_PREFIX.length),
        cannedKey: undefined,
        chartId: undefined,
        columnId: undefined,
      }));
      return;
    }
    setDraft((prev) => ({
      ...prev,
      cannedKey: value.slice(CANNED_PREFIX.length),
      definitionId: undefined,
      chartId: undefined,
      columnId: undefined,
    }));
  };

  const incomplete =
    (tileNeedsReport(draft.kind) && !draft.definitionId && !draft.cannedKey) ||
    (draft.kind === "kpi" && !draft.columnId) ||
    (draft.kind === "text" && !draft.text?.trim());

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Tile</DialogTitle>
          <DialogDescription>
            Tiles read from a saved or gallery report, so the dashboard and the export always agree.
          </DialogDescription>
        </DialogHeader>

        <div className="flex flex-col gap-4">
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="tile-kind">Shows</Label>
            <Select
              value={draft.kind}
              onValueChange={(kind) => {
                if (kind) setKind(kind as ReportDashboardTileKind);
              }}
              items={REPORT_TILE_KIND_CHOICES}
            >
              <SelectTrigger className="w-full" id="tile-kind">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {REPORT_TILE_KIND_CHOICES.map((choice) => (
                  <SelectItem key={choice.value} value={choice.value}>
                    {choice.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          {tileNeedsReport(draft.kind) && (
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="tile-source">Report</Label>
              <Select
                value={sourceValue(draft)}
                onValueChange={(value) => {
                  if (value) setSource(value);
                }}
                items={sourceItems}
              >
                <SelectTrigger className="w-full" id="tile-source">
                  <SelectValue placeholder="Choose a report" />
                </SelectTrigger>
                <SelectContent>
                  <SelectGroup>
                    <SelectLabel>My reports</SelectLabel>
                    {(definitions.data ?? []).map((definition) => (
                      <SelectItem key={definition.id} value={`${SAVED_PREFIX}${definition.id}`}>
                        {definition.name}
                      </SelectItem>
                    ))}
                  </SelectGroup>
                  <SelectGroup>
                    <SelectLabel>Gallery</SelectLabel>
                    {(canned.data ?? []).map((entry) => (
                      <SelectItem key={entry.key} value={`${CANNED_PREFIX}${entry.key}`}>
                        {entry.name}
                      </SelectItem>
                    ))}
                  </SelectGroup>
                </SelectContent>
              </Select>
            </div>
          )}

          {draft.kind === "chart" && (
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="tile-chart">Chart</Label>
              {charts.length === 0 ? (
                <p className="text-muted-foreground text-xs">
                  This report has no charts yet — add one in the report builder.
                </p>
              ) : (
                <Select
                  value={draft.chartId ?? charts[0].id}
                  onValueChange={(chartId) => {
                    if (chartId) setDraft((prev) => ({ ...prev, chartId }));
                  }}
                  items={charts.map((chart) => ({
                    value: chart.id,
                    label: chart.title || chart.type,
                  }))}
                >
                  <SelectTrigger className="w-full" id="tile-chart">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {charts.map((chart) => (
                      <SelectItem key={chart.id} value={chart.id}>
                        {chart.title || chart.type}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              )}
            </div>
          )}

          {draft.kind === "kpi" && (
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="tile-column">Measure</Label>
              {measures.length === 0 ? (
                <p className="text-muted-foreground text-xs">
                  This report returns no measures to show as a KPI.
                </p>
              ) : (
                <Select
                  value={draft.columnId ?? ""}
                  onValueChange={(columnId) => {
                    if (columnId) setDraft((prev) => ({ ...prev, columnId }));
                  }}
                  items={measures.map((output) => ({ value: output.id, label: output.label }))}
                >
                  <SelectTrigger className="w-full" id="tile-column">
                    <SelectValue placeholder="Choose a measure" />
                  </SelectTrigger>
                  <SelectContent>
                    {measures.map((output) => (
                      <SelectItem key={output.id} value={output.id}>
                        {output.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              )}
            </div>
          )}

          {draft.kind === "text" && (
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="tile-text">Text</Label>
              <Textarea
                id="tile-text"
                rows={4}
                value={draft.text ?? ""}
                onChange={(event) => setDraft((prev) => ({ ...prev, text: event.target.value }))}
              />
            </div>
          )}

          <div className="flex flex-col gap-1.5">
            <Label htmlFor="tile-title">Title</Label>
            <Input
              id="tile-title"
              value={draft.title ?? ""}
              placeholder={report.name || "Optional"}
              onChange={(event) =>
                setDraft((prev) => ({ ...prev, title: event.target.value || undefined }))
              }
            />
          </div>

          {reportParams.length > 0 && dashboardParams.length > 0 && (
            <div className="flex flex-col gap-1.5">
              <Label>Dashboard parameters</Label>
              <p className="text-2xs text-muted-foreground">
                Link this report&apos;s parameters to the dashboard&apos;s. Matching names link
                themselves.
              </p>
              {reportParams.map((param) => {
                const bound = draft.paramBindings?.[param.name];
                const autoMatch = dashboardParams.some((entry) => entry.name === param.name);
                return (
                  <div key={param.name} className="flex items-center gap-2">
                    <span className="text-muted-foreground w-28 shrink-0 truncate text-xs">
                      {param.label || param.name}
                    </span>
                    <Select
                      value={bound ?? UNLINKED}
                      onValueChange={(next) => {
                        if (!next) return;
                        setDraft((prev) => {
                          const bindings = Object.fromEntries(
                            Object.entries(prev.paramBindings ?? {}).filter(
                              ([name]) => name !== param.name,
                            ),
                          );
                          if (next !== UNLINKED) bindings[param.name] = next;
                          return {
                            ...prev,
                            paramBindings: Object.keys(bindings).length > 0 ? bindings : undefined,
                          };
                        });
                      }}
                      items={[
                        { value: UNLINKED, label: autoMatch ? "Same name" : "Not linked" },
                        ...dashboardParams.map((entry) => ({
                          value: entry.name,
                          label: entry.label || entry.name,
                        })),
                      ]}
                    >
                      <SelectTrigger className="h-7 flex-1">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value={UNLINKED}>
                          {autoMatch ? "Same name" : "Not linked"}
                        </SelectItem>
                        {dashboardParams.map((entry) => (
                          <SelectItem key={entry.name} value={entry.name}>
                            {entry.label || entry.name}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>
                );
              })}
            </div>
          )}

          {draft.kind === "table" && (
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="tile-limit">Row limit</Label>
              <Input
                id="tile-limit"
                type="number"
                min={1}
                placeholder="Report default"
                value={draft.limit ? String(draft.limit) : ""}
                onChange={(event) => {
                  const parsed = Number.parseInt(event.target.value, 10);
                  setDraft((prev) => ({
                    ...prev,
                    limit: Number.isNaN(parsed) || parsed <= 0 ? undefined : parsed,
                  }));
                }}
              />
            </div>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            disabled={incomplete}
            onClick={() => {
              onSave(draft);
              onOpenChange(false);
            }}
          >
            Save Tile
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
