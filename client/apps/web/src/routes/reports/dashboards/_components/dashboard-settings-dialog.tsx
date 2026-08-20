import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { useReportCatalog } from "@/hooks/use-reports";
import { cn } from "@trenova/shared/lib/utils";
import type {
  ReportDashboardFilter,
  ReportDashboardLayout,
  ReportParameterDef,
} from "@/types/report";
import { useEffect, useMemo, useState } from "react";
import { buildCatalogIndex } from "../../builder/_components/builder-state";
import { ParametersPanel } from "../../builder/_components/parameters-panel";
import { DashboardFiltersEditor } from "./dashboard-filters-editor";
import type { FilterCandidate } from "./use-tile-report";

type DashboardSettingsDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  layout: ReportDashboardLayout;
  /** Entities the dashboard's tiles query, so filters can only target real data. */
  entities: string[];
  /** Fields the dashboard's reports already filter on or group by. */
  candidates: FilterCandidate[];
  onSave: (layout: ReportDashboardLayout) => void;
};

type SettingsTab = "filters" | "parameters";

const TABS: { key: SettingsTab; label: string }[] = [
  { key: "filters", label: "Filters" },
  { key: "parameters", label: "Parameters" },
];

export function DashboardSettingsDialog({
  open,
  onOpenChange,
  layout,
  entities,
  candidates,
  onSave,
}: DashboardSettingsDialogProps) {
  const [tab, setTab] = useState<SettingsTab>("filters");
  const [filters, setFilters] = useState<ReportDashboardFilter[]>(layout.filters ?? []);
  const [parameters, setParameters] = useState<ReportParameterDef[]>(layout.parameters ?? []);

  const catalog = useReportCatalog(open);
  const index = useMemo(
    () => (catalog.data ? buildCatalogIndex(catalog.data) : null),
    [catalog.data],
  );

  useEffect(() => {
    if (open) {
      setFilters(layout.filters ?? []);
      setParameters(layout.parameters ?? []);
    }
  }, [open, layout.filters, layout.parameters]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-xl">
        <DialogHeader>
          <DialogTitle>Dashboard controls</DialogTitle>
          <DialogDescription>
            Filters narrow every tile built on the same data. Parameters answer a question a report
            asks for by name, like a lookback window.
          </DialogDescription>
        </DialogHeader>

        <div className="border-border flex items-center gap-1 border-b">
          {TABS.map((entry) => (
            <button
              key={entry.key}
              type="button"
              onClick={() => setTab(entry.key)}
              className={cn(
                "relative px-2 pb-2 text-xs transition-colors",
                tab === entry.key
                  ? "text-foreground font-medium"
                  : "text-muted-foreground hover:text-foreground",
              )}
            >
              {entry.label}
              {tab === entry.key && (
                <span className="bg-primary absolute inset-x-1 -bottom-px h-0.5 rounded-full" />
              )}
            </button>
          ))}
        </div>

        <div className="max-h-96 overflow-y-auto">
          {tab === "filters" ? (
            index ? (
              <DashboardFiltersEditor
                index={index}
                filters={filters}
                candidates={candidates}
                entities={entities}
                onChange={setFilters}
              />
            ) : (
              <p className="text-muted-foreground px-2 py-4 text-center text-sm">Loading fields…</p>
            )
          ) : (
            <ParametersPanel
              parameters={parameters}
              onChange={setParameters}
              emptyMessage="Parameters feed a value into reports that declare one with the same name."
            />
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button
            onClick={() => {
              onSave({ ...layout, filters, parameters });
              onOpenChange(false);
            }}
          >
            Apply
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
