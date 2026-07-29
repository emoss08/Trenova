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
import { Textarea } from "@trenova/shared/components/ui/textarea";
import {
  useCannedReports,
  useReportDashboards,
  useReportDefinitionList,
} from "@/hooks/use-reports";
import type { HomeMetricOption, HomeWidget, HomeWidgetOption } from "@/lib/graphql/home-layout";
import { useState } from "react";

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
  const [draft, setDraft] = useState<HomeWidget>(widget);
  const kind = option?.configKind ?? "none";

  const patchConfig = (patch: Partial<HomeWidget["config"]>) =>
    setDraft((prev) => ({ ...prev, config: { ...prev.config, ...patch } }));

  return (
    <Dialog open onOpenChange={(open) => !open && onCancel()}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>{option?.label ?? "Widget"}</DialogTitle>
          <DialogDescription>{option?.description}</DialogDescription>
        </DialogHeader>

        <div className="flex flex-col gap-4">
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="widget-title">Title</Label>
            <Input
              id="widget-title"
              placeholder={option?.label ?? ""}
              value={draft.title ?? ""}
              onChange={(event) =>
                setDraft((prev) => ({ ...prev, title: event.target.value || null }))
              }
            />
            <p className="text-2xs text-muted-foreground">
              Leave blank to use the widget&rsquo;s own name.
            </p>
          </div>

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
              label="Rows to show"
              value={draft.config.limit ?? null}
              min={0}
              max={50}
              hint="Leave empty to show as many as fit."
              onChange={(value) => patchConfig({ limit: value })}
            />
          )}

          {kind === "trend" && (
            <ConfigNumberField
              id="widget-window"
              label="Window (days)"
              value={draft.config.windowDays ?? null}
              min={0}
              max={365}
              hint="Leave empty to use the organization default."
              onChange={(value) => patchConfig({ windowDays: value })}
            />
          )}

          {kind === "report" && (
            <ReportPicker
              definitionId={draft.config.definitionId ?? null}
              cannedKey={draft.config.cannedKey ?? null}
              onChange={(next) => patchConfig(next)}
            />
          )}

          {kind === "dashboard" && (
            <DashboardPicker
              dashboardId={draft.config.dashboardId ?? null}
              onChange={(dashboardId) => patchConfig({ dashboardId })}
            />
          )}

          {kind === "text" && (
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="widget-text">Announcement</Label>
              <Textarea
                id="widget-text"
                rows={5}
                maxLength={MAX_ANNOUNCEMENT}
                value={draft.config.text ?? ""}
                onChange={(event) => patchConfig({ text: event.target.value || null })}
              />
              <p className="text-2xs text-muted-foreground">
                {(draft.config.text ?? "").length} / {MAX_ANNOUNCEMENT}
              </p>
            </div>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={onCancel}>
            Cancel
          </Button>
          <Button onClick={() => onSave(draft)}>Save</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
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
    <div className="flex flex-col gap-1.5">
      <Label htmlFor={id}>{label}</Label>
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
      {hint && <p className="text-2xs text-muted-foreground">{hint}</p>}
    </div>
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
  const toggle = (key: string) => {
    if (selected.includes(key)) {
      onChange(selected.filter((entry) => entry !== key));
      return;
    }
    // A single-metric tile swaps rather than refusing: the click is
    // unambiguous, so making the user clear the old one first is friction.
    onChange(max === 1 ? [key] : [...selected, key].slice(0, max));
  };

  return (
    <div className="flex flex-col gap-1.5">
      <Label>{max === 1 ? "Metric" : `Metrics (up to ${max})`}</Label>
      <div className="grid max-h-60 gap-1 overflow-y-auto sm:grid-cols-2">
        {metrics.map((metric) => {
          const checked = selected.includes(metric.key);
          const disabled = !checked && max > 1 && selected.length >= max;

          return (
            <label
              key={metric.key}
              className="flex items-center gap-2 rounded border border-border px-2 py-1.5 text-xs has-[:disabled]:opacity-50"
            >
              <Checkbox
                checked={checked}
                disabled={disabled}
                onCheckedChange={() => toggle(metric.key)}
              />
              <span className="truncate">{metric.label}</span>
            </label>
          );
        })}
      </div>
      {metrics.length === 0 && (
        <p className="text-2xs text-muted-foreground">
          No metrics are available to you on this organization.
        </p>
      )}
    </div>
  );
}

function ReportPicker({
  definitionId,
  cannedKey,
  onChange,
}: {
  definitionId: string | null;
  cannedKey: string | null;
  onChange: (next: { definitionId: string | null; cannedKey: string | null }) => void;
}) {
  const definitions = useReportDefinitionList("");
  const canned = useCannedReports();

  return (
    <div className="flex flex-col gap-1.5">
      <Label>Report</Label>
      <div className="flex max-h-60 flex-col gap-1 overflow-y-auto">
        {(canned.data ?? []).map((entry) => (
          <button
            key={entry.key}
            type="button"
            onClick={() => onChange({ definitionId: null, cannedKey: entry.key })}
            className={pickerRowClass(cannedKey === entry.key)}
          >
            <span className="truncate">{entry.name}</span>
            <span className="shrink-0 text-[9px] text-muted-foreground">{entry.category}</span>
          </button>
        ))}
        {(definitions.data ?? []).map((entry) => (
          <button
            key={entry.id}
            type="button"
            onClick={() => onChange({ definitionId: entry.id, cannedKey: null })}
            className={pickerRowClass(definitionId === entry.id)}
          >
            <span className="truncate">{entry.name}</span>
            <span className="shrink-0 text-[9px] text-muted-foreground">Saved</span>
          </button>
        ))}
      </div>
    </div>
  );
}

function DashboardPicker({
  dashboardId,
  onChange,
}: {
  dashboardId: string | null;
  onChange: (dashboardId: string | null) => void;
}) {
  const dashboards = useReportDashboards();

  return (
    <div className="flex flex-col gap-1.5">
      <Label>Dashboard</Label>
      <div className="flex max-h-60 flex-col gap-1 overflow-y-auto">
        {(dashboards.data ?? []).map((entry) => (
          <button
            key={entry.id}
            type="button"
            onClick={() => onChange(entry.id)}
            className={pickerRowClass(dashboardId === entry.id)}
          >
            <span className="truncate">{entry.name}</span>
          </button>
        ))}
        {(dashboards.data ?? []).length === 0 && (
          <p className="text-2xs text-muted-foreground">You have no saved dashboards yet.</p>
        )}
      </div>
    </div>
  );
}

function pickerRowClass(selected: boolean): string {
  return selected
    ? "flex items-center justify-between gap-2 rounded border border-brand bg-brand/10 px-2 py-1.5 text-left text-xs"
    : "flex items-center justify-between gap-2 rounded border border-border px-2 py-1.5 text-left text-xs transition-colors hover:bg-muted/60";
}
