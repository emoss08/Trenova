import { useT } from "@trenova/shared/i18n/use-t";
import { ReportSourcePicker, type ReportSource } from "@/components/reports/report-source-picker";
import { useReportCatalog, useReportDashboards } from "@/hooks/use-reports";
import type { HomeMetricOption, HomeWidget, HomeWidgetOption } from "@/lib/graphql/home-layout";
import {
  buildCatalogIndex,
  outputColumnChoices,
} from "@/routes/reports/builder/_components/builder-state";
import { useTileReport } from "@/routes/reports/dashboards/_components/use-tile-report";
import type { ReportDashboardTile } from "@/types/report";
import { Button } from "@trenova/shared/components/ui/button";
import { Checkbox } from "@trenova/shared/components/ui/checkbox";
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
  NumberField as NumberFieldRoot,
  NumberFieldDecrement,
  NumberFieldGroup,
  NumberFieldIncrement,
  NumberFieldInput,
} from "@trenova/shared/components/ui/number-field";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import { Textarea } from "@trenova/shared/components/ui/textarea";
import { cn } from "@trenova/shared/lib/utils";
import { SearchIcon } from "lucide-react";
import { useMemo, useState } from "react";
import { widgetVisualFor } from "./widget-gallery-visuals";

const MAX_METRIC_ROW = 6;
const MAX_ANNOUNCEMENT = 2000;

export type WidgetConfigDialogProps = {
  widget: HomeWidget;
  option: HomeWidgetOption | undefined;
  metrics: HomeMetricOption[];
  onSave: (widget: HomeWidget) => void;
  onCancel: () => void;
};

/**
 * Edits the one thing a widget needs to know. Which control appears is decided
 * by the widget's configKind, which the server publishes alongside the widget —
 * the client never has to keep its own list of which widget takes what.
 */
export function WidgetConfigDialog({
  widget,
  option,
  metrics,
  onSave,
  onCancel,
}: WidgetConfigDialogProps) {
  const t = useT();

  const [draft, setDraft] = useState<HomeWidget>(widget);
  const kind = option?.configKind ?? "none";
  const { icon: Icon } = widgetVisualFor(widget.key);

  const patchConfig = (patch: Partial<HomeWidget["config"]>) =>
    setDraft((prev) => ({ ...prev, config: { ...prev.config, ...patch } }));

  const blocker = configBlocker(kind, draft);

  return (
    <Dialog open onOpenChange={(open) => !open && onCancel()}>
      <DialogContent className="flex max-h-[88vh] w-full flex-col gap-0 overflow-hidden p-0 sm:max-w-lg">
        <DialogHeader className="border-border/70 flex-row items-start gap-2.5 border-b px-4 py-3">
          <span className="bg-brand/10 text-brand mt-0.5 flex size-7 shrink-0 items-center justify-center rounded-md">
            <Icon className="size-4" />
          </span>
          <div className="flex min-w-0 flex-col gap-0.5">
            <DialogTitle>{option?.label ?? t("Widget")}</DialogTitle>
            <DialogDescription className="text-xs">{option?.description}</DialogDescription>
          </div>
        </DialogHeader>

        <div className="flex min-h-0 flex-1 flex-col gap-4 overflow-y-auto px-4 py-3.5">
          <Field
            label={t("Title")}
            htmlFor="widget-title"
            hint={t("Leave blank to use the widget’s own name.")}
          >
            <Input
              id="widget-title"
              placeholder={option?.label ?? ""}
              value={draft.title ?? ""}
              onChange={(event) =>
                setDraft((prev) => ({ ...prev, title: event.target.value || null }))
              }
            />
          </Field>

          {kind === "none" && (
            <p className="text-muted-foreground text-xs">
              {t("This widget draws itself — there is nothing else to choose.")}
            </p>
          )}

          {kind === "metric" && (
            <MetricPicker
              metrics={metrics}
              selected={draft.config.metric ? [draft.config.metric] : []}
              max={1}
              onChange={(next) => patchConfig({ metric: next[0] ?? null })}
            />
          )}

          {kind === "metricRow" && (
            <MetricPicker
              metrics={metrics}
              selected={draft.config.metrics ?? []}
              max={MAX_METRIC_ROW}
              onChange={(next) => patchConfig({ metrics: next })}
            />
          )}

          {kind === "queue" && (
            <ConfigNumberField
              id="widget-limit"
              label={t("Rows to show")}
              value={draft.config.limit ?? null}
              min={0}
              max={50}
              hint={t("Leave empty to show as many as fit.")}
              onChange={(value) => patchConfig({ limit: value })}
            />
          )}

          {kind === "trend" && (
            <ConfigNumberField
              id="widget-window"
              label={t("Window (days)")}
              value={draft.config.windowDays ?? null}
              min={0}
              max={365}
              hint={t("Leave empty to use the organization default.")}
              onChange={(value) => patchConfig({ windowDays: value })}
            />
          )}

          {kind === "report" && <ReportConfig config={draft.config} onPatch={patchConfig} />}

          {kind === "dashboard" && (
            <DashboardPicker
              dashboardId={draft.config.dashboardId ?? null}
              onChange={(dashboardId) => patchConfig({ dashboardId })}
            />
          )}

          {kind === "text" && (
            <Field
              label={t("Announcement")}
              htmlFor="widget-text"
              hint={`${(draft.config.text ?? "").length} / ${MAX_ANNOUNCEMENT}`}
            >
              <Textarea
                id="widget-text"
                rows={5}
                maxLength={MAX_ANNOUNCEMENT}
                placeholder={t("What everyone on this home screen should read first.")}
                value={draft.config.text ?? ""}
                onChange={(event) => patchConfig({ text: event.target.value || null })}
              />
            </Field>
          )}
        </div>

        <DialogFooter className="mx-0 mb-0 items-center">
          <p className="text-2xs text-muted-foreground mr-auto hidden min-w-0 flex-1 text-left sm:block">
            {blocker}
          </p>
          <Button variant="outline" onClick={onCancel}>
            {t("Cancel")}
          </Button>
          <Button disabled={blocker != null} onClick={() => onSave(draft)}>
            {t("Save")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

/**
 * What still has to be decided before the widget can be saved, phrased as the
 * next thing to do. It mirrors the server's own validation so a save is never
 * sent only to come back as a field error on a dialog that has already closed.
 */
function configBlocker(kind: string, draft: HomeWidget): string | null {
  const config = draft.config;

  switch (kind) {
    case "metric":
      return config.metric ? null : "Choose the metric this tile shows.";
    case "metricRow":
      return (config.metrics ?? []).length > 0 ? null : "Choose at least one metric.";
    case "report":
      if (!config.definitionId && !config.cannedKey) return "Choose the report this tile shows.";
      if (config.columnId === "") return "Choose the measure this tile shows.";
      return null;
    case "dashboard":
      return config.dashboardId ? null : "Choose the dashboard this tile links to.";
    case "text":
      return (config.text ?? "").trim() === "" ? "Write the announcement." : null;
    default:
      return null;
  }
}

function Field({
  label,
  htmlFor,
  hint,
  children,
}: {
  label: string;
  htmlFor?: string;
  hint?: string;
  children: React.ReactNode;
}) {
  return (
    <div className="flex flex-col gap-1.5">
      <Label htmlFor={htmlFor}>{label}</Label>
      {children}
      {hint && <p className="text-2xs text-muted-foreground">{hint}</p>}
    </div>
  );
}

export function ConfigNumberField({
  id,
  label,
  value,
  min,
  max,
  hint,
  onChange,
}: {
  id: string;
  label: string;
  value: number | null;
  min: number;
  max: number;
  hint?: string;
  onChange: (value: number | null) => void;
}) {
  return (
    <Field label={label} htmlFor={id} hint={hint}>
      <NumberFieldRoot
        id={id}
        value={value}
        min={min}
        max={max}
        step={1}
        size="sm"
        onValueChange={(next) => onChange(next)}
      >
        <NumberFieldGroup>
          <NumberFieldDecrement />
          <NumberFieldInput />
          <NumberFieldIncrement />
        </NumberFieldGroup>
      </NumberFieldRoot>
    </Field>
  );
}

function MetricPicker({
  metrics,
  selected,
  max,
  onChange,
}: {
  metrics: HomeMetricOption[];
  selected: string[];
  max: number;
  onChange: (next: string[]) => void;
}) {
  const t = useT();

  const [search, setSearch] = useState("");
  const term = search.trim().toLowerCase();

  const visible = useMemo(
    () => metrics.filter((metric) => term === "" || metric.label.toLowerCase().includes(term)),
    [metrics, term],
  );

  const toggle = (key: string) => {
    if (selected.includes(key)) {
      onChange(selected.filter((entry) => entry !== key));
      return;
    }
    // A single-metric tile swaps rather than refusing: the click is
    // unambiguous, so making the user clear the old one first is friction.
    onChange(max === 1 ? [key] : [...selected, key].slice(0, max));
  };

  if (metrics.length === 0) {
    return (
      <Field label={max === 1 ? "Metric" : "Metrics"}>
        <p className="border-border text-muted-foreground rounded-md border border-dashed px-3 py-4 text-center text-xs">
          {t("No metrics are available to you on this organization.")}
        </p>
      </Field>
    );
  }

  return (
    <div className="flex flex-col gap-1.5">
      <div className="flex items-baseline gap-2">
        <Label>{max === 1 ? t("Metric") : t("Metrics")}</Label>
        {max > 1 && (
          <span className="text-2xs text-muted-foreground ml-auto tabular-nums">
            {t("{0} of {1} chosen", selected.length, max)}
          </span>
        )}
      </div>

      <div className="border-border bg-background flex flex-col rounded-md border">
        {metrics.length > 8 && (
          <div className="border-border/70 border-b p-2">
            <Input
              aria-label={t("Search metrics")}
              placeholder={t("Search metrics…")}
              value={search}
              onChange={(event) => setSearch(event.target.value)}
              leftElement={<SearchIcon className="text-muted-foreground size-3.5" />}
            />
          </div>
        )}

        <div className="grid max-h-52 gap-1 overflow-y-auto p-1.5 sm:grid-cols-2">
          {visible.map((metric) => {
            const checked = selected.includes(metric.key);
            const disabled = !checked && max > 1 && selected.length >= max;

            return (
              <label
                key={metric.key}
                className={cn(
                  "flex cursor-pointer items-center gap-2 rounded border px-2 py-1.5 text-xs transition-colors",
                  checked ? "border-brand/50 bg-brand/10" : "hover:bg-muted/70 border-transparent",
                  disabled && "cursor-not-allowed opacity-45",
                )}
              >
                <Checkbox
                  checked={checked}
                  disabled={disabled}
                  onCheckedChange={() => toggle(metric.key)}
                />
                <span className="truncate">{t(metric.label)}</span>
              </label>
            );
          })}

          {visible.length === 0 && (
            <p className="text-muted-foreground col-span-full px-2 py-6 text-center text-xs">
              {t("No metric matches “{0}”.", search.trim())}
            </p>
          )}
        </div>
      </div>
    </div>
  );
}

type ReportShows = "table" | "chart" | "kpi";

function reportShows(config: HomeWidget["config"]): ReportShows {
  // Mirrors HomeReportTile: a tile told to show one column is a KPI even when
  // the report also defines charts.
  if (config.columnId != null) return "kpi";
  if (config.chartId != null) return "chart";
  return "table";
}

/**
 * The report a tile draws, and how it draws it. The "how" was previously
 * unreachable from Home even though the widget config has always carried
 * chartId and columnId, so a report with charts could only ever land as a
 * table.
 */
function ReportConfig({
  config,
  onPatch,
}: {
  config: HomeWidget["config"];
  onPatch: (patch: Partial<HomeWidget["config"]>) => void;
}) {
  const t = useT();

  const shows = reportShows(config);

  const tile = useMemo<ReportDashboardTile | null>(
    () =>
      config.definitionId || config.cannedKey
        ? {
            id: "home_report_config",
            kind: "table",
            definitionId: config.definitionId ?? undefined,
            cannedKey: config.cannedKey ?? undefined,
            x: 0,
            y: 0,
            w: 6,
            h: 5,
          }
        : null,
    [config.definitionId, config.cannedKey],
  );

  const report = useTileReport(tile);
  const catalog = useReportCatalog(tile != null);
  const index = useMemo(
    () => (catalog.data ? buildCatalogIndex(catalog.data) : null),
    [catalog.data],
  );

  const measures = useMemo(() => {
    if (!report.ir || !index) return [];
    return outputColumnChoices(index, report.ir).filter((output) => !output.isDim);
  }, [index, report.ir]);
  const charts = report.ir?.charts ?? [];

  const source: ReportSource = {
    definitionId: config.definitionId ?? null,
    cannedKey: config.cannedKey ?? null,
  };

  // Changing the source invalidates a chart or measure chosen from the previous
  // report: their ids mean nothing outside the report that defined them.
  const setSource = (next: ReportSource) => onPatch({ ...next, chartId: null, columnId: null });

  const setShows = (next: ReportShows) => {
    if (next === "chart") {
      onPatch({ chartId: charts[0]?.id ?? "", columnId: null });
      return;
    }
    if (next === "kpi") {
      onPatch({ chartId: null, columnId: measures[0]?.id ?? "" });
      return;
    }
    onPatch({ chartId: null, columnId: null });
  };

  return (
    <>
      <div className="flex flex-col gap-1.5">
        <Label id="widget-report-label">{t("Report")}</Label>
        <ReportSourcePicker labelledBy="widget-report-label" value={source} onChange={setSource} />
      </div>

      {tile && (
        <Field label={t("Shows as")}>
          <SegmentedControl
            aria-label={t("How this report is drawn")}
            fullWidth
            value={shows}
            onValueChange={setShows}
            items={[
              { value: "table", label: "Table" },
              { value: "chart", label: "Chart", disabled: charts.length === 0 },
              { value: "kpi", label: "Single number", disabled: measures.length === 0 },
            ]}
          />
        </Field>
      )}

      {tile && shows === "chart" && (
        <Field label={t("Chart")}>
          {charts.length === 0 ? (
            <p className="text-muted-foreground text-xs">
              {t("This report has no charts yet — add one in the report builder.")}
            </p>
          ) : (
            <Select
              value={config.chartId || charts[0].id}
              onValueChange={(chartId) => chartId && onPatch({ chartId })}
              items={charts.map((chart) => ({ value: chart.id, label: chart.title || chart.type }))}
            >
              <SelectTrigger className="w-full">
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
        </Field>
      )}

      {tile && shows === "kpi" && (
        <Field label={t("Measure")}>
          {measures.length === 0 ? (
            <p className="text-muted-foreground text-xs">
              {t("This report returns no measures to show as a single number.")}
            </p>
          ) : (
            <Select
              value={config.columnId || ""}
              onValueChange={(columnId) => columnId && onPatch({ columnId })}
              items={measures.map((output) => ({ value: output.id, label: output.label }))}
            >
              <SelectTrigger className="w-full">
                <SelectValue placeholder={t("Choose a measure")} />
              </SelectTrigger>
              <SelectContent>
                {measures.map((output) => (
                  <SelectItem key={output.id} value={output.id}>
                    {t(output.label)}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          )}
        </Field>
      )}

      {tile && shows === "table" && (
        <ConfigNumberField
          id="widget-report-limit"
          label={t("Rows to show")}
          value={config.limit ?? null}
          min={0}
          max={50}
          hint={t("Leave empty to use the report’s own limit.")}
          onChange={(limit) => onPatch({ limit })}
        />
      )}
    </>
  );
}

function DashboardPicker({
  dashboardId,
  onChange,
}: {
  dashboardId: string | null;
  onChange: (dashboardId: string | null) => void;
}) {
  const t = useT();

  const dashboards = useReportDashboards();
  const [search, setSearch] = useState("");
  const term = search.trim().toLowerCase();

  const visible = useMemo(
    () =>
      (dashboards.data ?? []).filter(
        (entry) =>
          term === "" ||
          entry.name.toLowerCase().includes(term) ||
          (entry.description ?? "").toLowerCase().includes(term),
      ),
    [dashboards.data, term],
  );

  return (
    <div className="flex flex-col gap-1.5">
      <Label id="widget-dashboard-label">{t("Dashboard")}</Label>
      <div
        aria-labelledby="widget-dashboard-label"
        className="border-border bg-background flex flex-col rounded-md border"
      >
        {(dashboards.data ?? []).length > 6 && (
          <div className="border-border/70 border-b p-2">
            <Input
              aria-label={t("Search dashboards")}
              placeholder={t("Search dashboards…")}
              value={search}
              onChange={(event) => setSearch(event.target.value)}
              leftElement={<SearchIcon className="text-muted-foreground size-3.5" />}
            />
          </div>
        )}

        <div
          role="listbox"
          aria-label={t("Dashboards")}
          className="flex max-h-52 min-h-24 flex-col gap-1 overflow-y-auto p-1.5"
        >
          {visible.map((entry) => (
            <button
              key={entry.id}
              type="button"
              role="option"
              aria-selected={dashboardId === entry.id}
              onClick={() => onChange(entry.id)}
              className={cn(
                "flex w-full flex-col gap-0.5 rounded border px-2 py-1.5 text-left transition-colors",
                dashboardId === entry.id
                  ? "border-brand/50 bg-brand/10"
                  : "hover:bg-muted/70 border-transparent",
              )}
            >
              <span className="truncate text-xs font-medium">{entry.name}</span>
              {entry.description && (
                <span className="text-muted-foreground truncate text-[11px]">
                  {t(entry.description)}
                </span>
              )}
            </button>
          ))}

          {visible.length === 0 && (
            <p className="text-muted-foreground flex flex-1 items-center justify-center px-4 py-6 text-center text-xs">
              {dashboards.isLoading
                ? t("Loading dashboards…")
                : term === ""
                  ? t("You have no saved dashboards yet.")
                  : t("No dashboard matches “{0}”.", search.trim())}
            </p>
          )}
        </div>
      </div>
    </div>
  );
}
