import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  useCreateReportDefinition,
  useReportPreview,
  useUpdateReportDefinition,
} from "@/hooks/use-reports";
import { useDebounce } from "@/hooks/use-debounce";
import { graphQLErrorMessage } from "@/lib/graphql";
import type { ReportCatalog, ReportDefinition } from "@/lib/graphql/reports";
import { cn } from "@/lib/utils";
import {
  aggregationsForField,
  parseReportIR,
  type ReportIR,
  type ReportParameterDef,
} from "@/types/report";
import { ArrowLeftIcon, PlayIcon, SaveIcon, TriangleAlertIcon } from "lucide-react";
import { AnimatePresence, m } from "motion/react";
import { useMemo, useState } from "react";
import { Link, useNavigate } from "react-router";
import { toast } from "sonner";
import { RunReportDialog } from "../../_components/run-report-dialog";
import {
  buildCatalogIndex,
  defaultPreviewParams,
  emptyIR,
  irToInput,
  previewReady,
  uniqueColumnId,
  withColumns,
  withParameters,
} from "./builder-state";
import { CatalogFieldTree, type FieldSelection } from "./catalog-field-tree";
import { ColumnsPanel } from "./columns-panel";
import { EntityPicker } from "./entity-picker";
import { FiltersPanel } from "./filters-panel";
import { HavingPanel } from "./having-panel";
import { ParametersPanel } from "./parameters-panel";
import { PivotPanel } from "./pivot-panel";
import { PreviewGrid } from "./preview-grid";
import { DEFAULT_REPORT_META, SaveReportDialog, type ReportMeta } from "./save-report-dialog";
import { SortLimitPanel } from "./sort-limit-panel";

const PREVIEW_DEBOUNCE_MS = 600;

type ReportBuilderProps = {
  catalog: ReportCatalog;
  definition?: ReportDefinition;
};

type InspectorTab = "columns" | "filters" | "options" | "params";

const INSPECTOR_TABS: { key: InspectorTab; label: string }[] = [
  { key: "columns", label: "Columns" },
  { key: "filters", label: "Filters" },
  { key: "options", label: "Options" },
  { key: "params", label: "Params" },
];

function countFilters(group: ReportIR["filters"]): number {
  if (!group) return 0;
  return (
    (group.filters?.length ?? 0) +
    (group.groups ?? []).reduce((total, nested) => total + countFilters(nested), 0)
  );
}

function definitionToMeta(definition: ReportDefinition): ReportMeta {
  return {
    name: definition.name,
    description: definition.description,
    category: definition.category || "custom",
    tags: definition.tags,
    visibility: definition.visibility,
    status: definition.status === "needs_attention" ? "active" : definition.status,
    defaultFormat: definition.defaultFormat,
  };
}

function SectionLabel({ children }: { children: string }) {
  return (
    <p className="px-0.5 pt-1 pb-2 text-2xs font-medium tracking-wide text-muted-foreground uppercase">
      {children}
    </p>
  );
}

export function ReportBuilder({ catalog, definition }: ReportBuilderProps) {
  const navigate = useNavigate();
  const index = useMemo(() => buildCatalogIndex(catalog), [catalog]);

  const [ir, setIR] = useState<ReportIR>(() => {
    if (definition) {
      const parsed = parseReportIR(definition.definition);
      if (parsed) return parsed;
    }
    return emptyIR("");
  });
  const [meta, setMeta] = useState<ReportMeta>(() =>
    definition ? definitionToMeta(definition) : DEFAULT_REPORT_META,
  );
  const [inspectorTab, setInspectorTab] = useState<InspectorTab>("columns");
  const [saveOpen, setSaveOpen] = useState(false);
  const [runOpen, setRunOpen] = useState(false);

  const createDefinition = useCreateReportDefinition();
  const updateDefinition = useUpdateReportDefinition();

  const debouncedIR = useDebounce(ir, PREVIEW_DEBOUNCE_MS);
  const ready = previewReady(debouncedIR);
  const previewInput = useMemo(() => (ready ? irToInput(debouncedIR) : null), [ready, debouncedIR]);
  const preview = useReportPreview(previewInput, defaultPreviewParams(debouncedIR));

  const entity = index.entities.get(ir.entity);
  const parameters: ReportParameterDef[] = ir.parameters ?? [];

  const tabCounts: Record<InspectorTab, number> = {
    columns: ir.columns.length,
    filters: countFilters(ir.filters) + (ir.having?.filters?.length ?? 0),
    options: (ir.sort?.length ?? 0) + (ir.pivot ? 1 : 0),
    params: parameters.length,
  };

  const handleAddColumn = (selection: FieldSelection) => {
    const aggregations = aggregationsForField(selection.field);
    const kind = selection.crossesToMany ? "measure" : "dimension";
    if (kind === "measure" && aggregations.length === 0) {
      toast.error("This field crosses a to-many relationship and has no legal aggregations");
      return;
    }
    setIR((prev) => ({
      ...prev,
      columns: [
        ...prev.columns,
        {
          id: uniqueColumnId(prev, selection.ref.field),
          ref: selection.ref,
          kind,
          agg: kind === "measure" ? aggregations[0] : undefined,
        },
      ],
    }));
    setInspectorTab("columns");
  };

  const handleSave = () => {
    const input = {
      name: meta.name.trim(),
      description: meta.description || undefined,
      category: meta.category || undefined,
      tags: meta.tags,
      visibility: meta.visibility,
      status: meta.status,
      defaultFormat: meta.defaultFormat,
      definition: irToInput(ir),
    };

    if (definition) {
      updateDefinition.mutate(
        { ...input, id: definition.id, version: definition.version },
        {
          onSuccess: () => {
            setSaveOpen(false);
            toast.success("Report saved");
          },
          onError: (error) => toast.error(graphQLErrorMessage(error, "Failed to save the report")),
        },
      );
    } else {
      createDefinition.mutate(input, {
        onSuccess: (created) => {
          setSaveOpen(false);
          toast.success("Report created");
          void navigate(`/reports/builder/${created.id}`, { replace: true });
        },
        onError: (error) => toast.error(graphQLErrorMessage(error, "Failed to create the report")),
      });
    }
  };

  return (
    <div className="flex h-full min-h-0 flex-col">
      <header className="flex h-12 shrink-0 items-center gap-2 border-b border-border px-3">
        <Button
          variant="ghost"
          size="icon"
          className="size-7"
          render={<Link to="/reports" />}
          aria-label="Back to reports"
        >
          <ArrowLeftIcon className="size-4" />
        </Button>
        <div className="flex min-w-0 flex-1 items-center gap-2">
          <input
            value={meta.name}
            onChange={(event) => setMeta((prev) => ({ ...prev, name: event.target.value }))}
            placeholder="Untitled report"
            className="max-w-80 min-w-0 flex-1 truncate rounded-md bg-transparent px-1.5 py-1 text-sm font-medium transition-colors outline-none placeholder:text-muted-foreground/60 hover:bg-muted/60 focus:bg-muted/60"
          />
          {entity && (
            <Badge variant="outline" className="shrink-0">
              {entity.label}
            </Badge>
          )}
          {definition?.kind === "canned_fork" && (
            <Badge variant="indigo" className="shrink-0">
              Customized
            </Badge>
          )}
          {definition?.status === "needs_attention" && (
            <Badge variant="warning" className="shrink-0">
              Needs Attention
            </Badge>
          )}
        </div>
        {definition && (
          <Button
            variant="outline"
            size="sm"
            className="h-7"
            onClick={() => setRunOpen(true)}
            disabled={definition.status !== "active"}
          >
            <PlayIcon className="size-3.5" />
            Run
          </Button>
        )}
        <Button
          size="sm"
          className="h-7"
          onClick={() => setSaveOpen(true)}
          disabled={ir.columns.length === 0}
        >
          <SaveIcon className="size-3.5" />
          Save
        </Button>
      </header>

      {definition && definition.diagnostics.length > 0 && (
        <div className="flex items-start gap-2 border-b border-border bg-amber-500/5 px-4 py-2 text-xs">
          <TriangleAlertIcon className="mt-0.5 size-3.5 shrink-0 text-amber-600 dark:text-amber-400" />
          <div className="flex flex-col gap-0.5">
            <p className="font-medium">This report needs attention</p>
            {definition.diagnostics.map((diagnostic, i) => (
              <p key={i} className="text-muted-foreground">
                {diagnostic}
              </p>
            ))}
          </div>
        </div>
      )}

      {!ir.entity ? (
        <EntityPicker catalog={catalog} onSelect={(entityKey) => setIR(emptyIR(entityKey))} />
      ) : (
        <div className="grid min-h-0 flex-1 grid-cols-[260px_minmax(0,1fr)_360px]">
          <aside className="flex min-h-0 flex-col border-r border-border">
            <div className="flex h-8 shrink-0 items-center border-b border-border px-3">
              <span className="text-2xs font-medium tracking-wide text-muted-foreground uppercase">
                {entity?.label ?? ir.entity} Fields
              </span>
            </div>
            <div className="min-h-0 flex-1 p-2.5">
              <CatalogFieldTree
                index={index}
                entityKey={ir.entity}
                className="h-full"
                onSelectField={handleAddColumn}
              />
            </div>
          </aside>

          <main className="min-h-0 bg-muted/20">
            <PreviewGrid
              preview={preview.data}
              loading={preview.isFetching}
              error={
                preview.isError
                  ? graphQLErrorMessage(preview.error, "The preview could not be compiled")
                  : null
              }
              ready={ready}
            />
          </main>

          <aside className="flex min-h-0 flex-col border-l border-border">
            <div className="flex h-8 shrink-0 items-center gap-0.5 border-b border-border px-1.5">
              {INSPECTOR_TABS.map((tab) => (
                <button
                  key={tab.key}
                  type="button"
                  onClick={() => setInspectorTab(tab.key)}
                  className={cn(
                    "relative flex h-full items-center gap-1 px-2 text-xs transition-colors",
                    inspectorTab === tab.key
                      ? "font-medium text-foreground"
                      : "text-muted-foreground hover:text-foreground",
                  )}
                >
                  {tab.label}
                  {tabCounts[tab.key] > 0 && (
                    <span className="rounded-sm bg-muted px-1 text-2xs text-muted-foreground tabular-nums">
                      {tabCounts[tab.key]}
                    </span>
                  )}
                  {inspectorTab === tab.key && (
                    <m.span
                      layoutId="inspector-tab-indicator"
                      className="absolute inset-x-1.5 bottom-0 h-0.5 rounded-full bg-primary"
                    />
                  )}
                </button>
              ))}
            </div>
            <div className="min-h-0 flex-1 overflow-y-auto">
              <AnimatePresence mode="wait" initial={false}>
                <m.div
                  key={inspectorTab}
                  initial={{ opacity: 0, y: 4 }}
                  animate={{ opacity: 1, y: 0 }}
                  exit={{ opacity: 0, y: -4 }}
                  transition={{ duration: 0.15, ease: "easeOut" }}
                  className="p-3"
                >
                  {inspectorTab === "columns" && (
                    <ColumnsPanel
                      index={index}
                      ir={ir}
                      onChange={(columns) => setIR((prev) => withColumns(prev, columns))}
                    />
                  )}
                  {inspectorTab === "filters" && (
                    <div className="flex flex-col gap-4">
                      <div>
                        <SectionLabel>Row Filters</SectionLabel>
                        <FiltersPanel
                          index={index}
                          ir={ir}
                          group={ir.filters}
                          onChange={(filters) => setIR((prev) => ({ ...prev, filters }))}
                        />
                      </div>
                      <div>
                        <SectionLabel>Measure Filters</SectionLabel>
                        <HavingPanel
                          index={index}
                          ir={ir}
                          onChange={(having) => setIR((prev) => ({ ...prev, having }))}
                        />
                      </div>
                    </div>
                  )}
                  {inspectorTab === "options" && (
                    <div className="flex flex-col gap-4">
                      <div>
                        <SectionLabel>Sort & Limit</SectionLabel>
                        <SortLimitPanel
                          index={index}
                          ir={ir}
                          onSortChange={(sort) => setIR((prev) => ({ ...prev, sort }))}
                          onLimitChange={(limit) => setIR((prev) => ({ ...prev, limit }))}
                        />
                      </div>
                      <div>
                        <SectionLabel>Pivot</SectionLabel>
                        <PivotPanel
                          index={index}
                          ir={ir}
                          onChange={(pivot) => setIR((prev) => ({ ...prev, pivot }))}
                        />
                      </div>
                    </div>
                  )}
                  {inspectorTab === "params" && (
                    <ParametersPanel
                      ir={ir}
                      onChange={(nextParameters, rename) =>
                        setIR((prev) => withParameters(prev, nextParameters, rename))
                      }
                    />
                  )}
                </m.div>
              </AnimatePresence>
            </div>
          </aside>
        </div>
      )}

      <SaveReportDialog
        open={saveOpen}
        onOpenChange={setSaveOpen}
        meta={meta}
        onMetaChange={setMeta}
        onSave={handleSave}
        saving={createDefinition.isPending || updateDefinition.isPending}
        isNew={!definition}
      />
      {definition && (
        <RunReportDialog
          open={runOpen}
          onOpenChange={setRunOpen}
          target={{ definitionId: definition.id }}
          reportName={meta.name || definition.name}
          defaultFormat={meta.defaultFormat}
          parameters={parameters}
        />
      )}
    </div>
  );
}
