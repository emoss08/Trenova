import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Input } from "@trenova/shared/components/ui/input";
import { Label } from "@trenova/shared/components/ui/label";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { useDeleteReportDashboard, useUpdateReportDashboard } from "@/hooks/use-reports";
import { graphQLErrorMessage } from "@trenova/shared/lib/graphql";
import { cn } from "@trenova/shared/lib/utils";
import type { ReportDashboard } from "@/lib/graphql/reports";
import {
  DASHBOARD_GRID_COLUMNS,
  MAX_DASHBOARD_TILES,
  MAX_TILE_HEIGHT,
  parseDashboardLayout,
  type ReportDashboardFilter,
  type ReportDashboardLayout,
  type ReportDashboardTile,
} from "@/types/report";
import {
  closestCenter,
  DndContext,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
  type DragEndEvent,
} from "@dnd-kit/core";
import { arrayMove, rectSortingStrategy, SortableContext, useSortable } from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import {
  ArrowLeftIcon,
  ChevronsLeftRightIcon,
  ChevronsUpDownIcon,
  GripVerticalIcon,
  DownloadIcon,
  LayoutDashboardIcon,
  MoreHorizontalIcon,
  PencilIcon,
  PlusIcon,
  SettingsIcon,
  Trash2Icon,
  XIcon,
} from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { Link, useNavigate } from "react-router";
import { toast } from "sonner";
import { downloadBlob } from "@trenova/shared/lib/utils";
import { exportDashboard } from "@/lib/graphql/reports";
import { ParameterField, defaultParamValues } from "../../_components/report-parameter-field";
import { FilterValueEditor } from "../../builder/_components/filter-value-editor";
import {
  buildCatalogIndex,
  refLabel,
  refTargetEntity,
  resolveField,
  type CatalogIndex,
} from "../../builder/_components/builder-state";
import { useReportCatalog } from "@/hooks/use-reports";
import { DashboardTileBody, TileFooterLink } from "./dashboard-tile";
import { nextTileId, packTiles } from "./dashboard-state";
import { DashboardSettingsDialog } from "./dashboard-settings-dialog";
import { TileEditorDialog } from "./tile-editor-dialog";
import { useDashboardFilterCandidates, useTileReport } from "./use-tile-report";
import {
  crossFilterDefinitions,
  crossFilterFor,
  crossFilterValues,
  removeCrossFilter,
  selectedCategory,
  toggleCrossFilter,
  type CrossFilterState,
} from "./cross-filter";
import { TileFilterPopover, useTileFilterFields } from "./tile-filter-popover";

const EMPTY_VALUES: Record<string, unknown> = {};

type DashboardViewProps = {
  dashboard: ReportDashboard;
  canEdit: boolean;
};

function SizeStepper({
  icon: Icon,
  label,
  value,
  min,
  max,
  onChange,
}: {
  icon: typeof ChevronsLeftRightIcon;
  label: string;
  value: number;
  min: number;
  max: number;
  onChange: (next: number) => void;
}) {
  return (
    <div className="flex items-center gap-2">
      <Icon className="size-3.5 shrink-0 text-muted-foreground" />
      <span className="flex-1 text-xs text-muted-foreground">{label}</span>
      <Button
        variant="outline"
        size="icon"
        className="size-6"
        aria-label={`Decrease ${label.toLowerCase()}`}
        disabled={value <= min}
        onClick={() => onChange(value - 1)}
      >
        −
      </Button>
      <span className="w-5 text-center text-xs tabular-nums">{value}</span>
      <Button
        variant="outline"
        size="icon"
        className="size-6"
        aria-label={`Increase ${label.toLowerCase()}`}
        disabled={value >= max}
        onClick={() => onChange(value + 1)}
      >
        +
      </Button>
    </div>
  );
}

/**
 * A narrow tile cannot fit a row of inline controls, and the tile clips its
 * own overflow — which used to strand a shrunken tile with its "wider" button
 * cut off. One trigger always fits, and the popover stays open so the size can
 * be nudged repeatedly.
 */
function TileControls({
  tile,
  onEdit,
  onRemove,
  onResize,
}: {
  tile: ReportDashboardTile;
  onEdit: () => void;
  onRemove: () => void;
  onResize: (patch: Partial<ReportDashboardTile>) => void;
}) {
  return (
    <Popover>
      <PopoverTrigger
        render={
          <Button variant="ghost" size="icon" className="size-6 shrink-0" aria-label="Tile options">
            <MoreHorizontalIcon className="size-3.5" />
          </Button>
        }
      />
      <PopoverContent className="w-60 p-2" align="end">
        <div className="flex flex-col gap-2">
          <SizeStepper
            icon={ChevronsLeftRightIcon}
            label="Width"
            value={tile.w}
            min={1}
            max={DASHBOARD_GRID_COLUMNS}
            onChange={(w) => onResize({ w })}
          />
          <SizeStepper
            icon={ChevronsUpDownIcon}
            label="Height"
            value={tile.h}
            min={1}
            max={MAX_TILE_HEIGHT}
            onChange={(h) => onResize({ h })}
          />
          <div className="flex flex-col gap-1 border-t border-border pt-2">
            <Button variant="ghost" size="sm" className="h-7 justify-start" onClick={onEdit}>
              <PencilIcon className="size-3.5" />
              Edit tile
            </Button>
            <Button
              variant="ghost"
              size="sm"
              className="h-7 justify-start text-destructive"
              onClick={onRemove}
            >
              <Trash2Icon className="size-3.5" />
              Remove tile
            </Button>
          </div>
        </div>
      </PopoverContent>
    </Popover>
  );
}

function TileFrame({
  tile,
  params,
  filters,
  filterValues,
  index,
  tileFilterValues,
  onTileFilterChange,
  onTileFilterFields,
  crossFilters,
  onCrossFilter,
  editing,
  onEdit,
  onRemove,
  onResize,
}: {
  tile: ReportDashboardTile;
  params: Record<string, unknown>;
  filters: ReportDashboardFilter[];
  filterValues: Record<string, unknown>;
  index: CatalogIndex | null;
  tileFilterValues: Record<string, unknown>;
  onTileFilterChange: (values: Record<string, unknown>) => void;
  onTileFilterFields: (tileId: string, fields: ReportDashboardFilter[]) => void;
  crossFilters: CrossFilterState;
  onCrossFilter: (filter: ReportDashboardFilter, value: unknown, label: string) => void;
  editing: boolean;
  onEdit: () => void;
  onRemove: () => void;
  onResize: (patch: Partial<ReportDashboardTile>) => void;
}) {
  const sortable = useSortable({ id: tile.id, disabled: !editing });
  const report = useTileReport(tile.kind === "text" ? null : tile);
  const title = tile.title || report.name || "Tile";
  const tileFilters = useTileFilterFields(index, report.ir);

  // The export runs server-side and needs the field definitions behind a
  // tile's ad-hoc filter values, which only this tile knows — it is the one
  // that resolved its own report.
  useEffect(() => {
    onTileFilterFields(tile.id, tileFilters);
  }, [tile.id, tileFilters, onTileFilterFields]);

  // Only a chart tile has a category axis to click, and only while viewing —
  // during editing a click belongs to the layout, not the data.
  const chart =
    report.ir?.charts?.find((entry) => entry.id === tile.chartId) ?? report.ir?.charts?.[0];
  const crossFilter = useMemo(
    () =>
      !editing && tile.kind === "chart" && report.ir
        ? crossFilterFor(report.ir, chart?.xColumnId)
        : null,
    [editing, tile.kind, report.ir, chart?.xColumnId],
  );
  const categorySelect = crossFilter
    ? {
        selected: selectedCategory(crossFilters, crossFilter),
        onSelect: (raw: unknown, label: string) => onCrossFilter(crossFilter, raw, label),
      }
    : undefined;

  return (
    <div
      ref={sortable.setNodeRef}
      style={{
        gridColumn: `span ${tile.w} / span ${tile.w}`,
        gridRow: `span ${tile.h} / span ${tile.h}`,
        transform: CSS.Transform.toString(sortable.transform),
        transition: sortable.transition,
      }}
      className={cn(
        "group/tile flex min-h-0 flex-col overflow-hidden rounded-lg border border-border bg-background",
        sortable.isDragging && "z-10 opacity-80 shadow-lg",
      )}
    >
      <div className="flex h-8 shrink-0 items-center gap-1 border-b border-border px-2">
        {editing && (
          <button
            type="button"
            className="shrink-0 cursor-grab text-muted-foreground hover:text-foreground"
            aria-label="Move tile"
            {...sortable.attributes}
            {...sortable.listeners}
          >
            <GripVerticalIcon className="size-3.5" />
          </button>
        )}
        <span className="min-w-0 flex-1 truncate text-xs font-medium">{title}</span>
        {editing ? (
          <TileControls tile={tile} onEdit={onEdit} onRemove={onRemove} onResize={onResize} />
        ) : (
          <>
            {index && report.ir && (
              <TileFilterPopover
                index={index}
                ir={report.ir}
                values={tileFilterValues}
                onChange={onTileFilterChange}
              />
            )}
            <TileFooterLink tile={tile} />
          </>
        )}
      </div>
      <div className="flex min-h-0 flex-1 flex-col">
        <DashboardTileBody
          tile={tile}
          params={params}
          filters={filters}
          filterValues={filterValues}
          tileFilters={tileFilters}
          tileFilterValues={tileFilterValues}
          categorySelect={categorySelect}
        />
      </div>
    </div>
  );
}

export function DashboardView({ dashboard, canEdit }: DashboardViewProps) {
  const navigate = useNavigate();
  const updateDashboard = useUpdateReportDashboard();
  const deleteDashboard = useDeleteReportDashboard();

  const savedLayout = useMemo(() => parseDashboardLayout(dashboard.layout), [dashboard.layout]);

  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState<ReportDashboardLayout>(savedLayout);
  const [name, setName] = useState(dashboard.name);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [editingTile, setEditingTile] = useState<ReportDashboardTile | null>(null);
  const [params, setParams] = useState<Record<string, unknown>>(() =>
    defaultParamValues(savedLayout.parameters ?? []),
  );

  const layout = editing ? draft : savedLayout;
  const parameters = layout.parameters ?? [];
  const configuredFilters = useMemo(() => layout.filters ?? [], [layout.filters]);
  const sensors = useSensors(useSensor(PointerSensor), useSensor(KeyboardSensor));

  const { entities, candidates } = useDashboardFilterCandidates(layout.tiles);
  const catalog = useReportCatalog();
  const index = useMemo(
    () => (catalog.data ? buildCatalogIndex(catalog.data) : null),
    [catalog.data],
  );

  // Filter values live outside the layout: the layout says what can be
  // filtered, this says what the viewer currently picked.
  const [filterValues, setFilterValues] = useState<Record<string, unknown>>({});
  // Per-tile narrowing, kept beside the shared values: definitions are saved
  // with the dashboard, the values chosen while viewing are not.
  const [tileFilterValues, setTileFilterValues] = useState<Record<string, Record<string, unknown>>>(
    {},
  );
  // Categories picked by clicking a chart. Same lifetime as the other values:
  // transient, and always visible as a chip so the narrowing is never a mystery.
  const [crossFilters, setCrossFilters] = useState<CrossFilterState>({});

  const handleCrossFilter = useCallback(
    (filter: ReportDashboardFilter, value: unknown, label: string) =>
      setCrossFilters((prev) => toggleCrossFilter(prev, filter, value, label)),
    [],
  );

  // Configured and clicked filters are applied through one path, so entity
  // matching and override-not-stack behaviour are identical for both.
  const fieldFilters = useMemo(
    () => [...configuredFilters, ...crossFilterDefinitions(crossFilters)],
    [configuredFilters, crossFilters],
  );
  const appliedFilterValues = useMemo(
    () => ({ ...filterValues, ...crossFilterValues(crossFilters) }),
    [filterValues, crossFilters],
  );

  // A filter added or repointed during editing should not keep a value that
  // belonged to the old field.
  useEffect(() => {
    setFilterValues((prev) => {
      const live = new Set(configuredFilters.map((filter) => filter.id));
      const next = Object.fromEntries(Object.entries(prev).filter(([id]) => live.has(id)));
      return Object.keys(next).length === Object.keys(prev).length ? prev : next;
    });
  }, [configuredFilters]);

  const [exporting, setExporting] = useState(false);
  const [tileFilterFields, setTileFilterFields] = useState<
    Record<string, ReportDashboardFilter[]>
  >({});

  const registerTileFilterFields = useCallback(
    (tileId: string, fields: ReportDashboardFilter[]) =>
      setTileFilterFields((prev) => (prev[tileId] === fields ? prev : { ...prev, [tileId]: fields })),
    [],
  );

  // The export re-runs every tile server-side with the viewer's own
  // permissions, so it carries the same filter and parameter choices the
  // screen is showing rather than the rendered numbers.
  const handleExport = useCallback(async () => {
    setExporting(true);
    try {
      const { blob, fileName } = await exportDashboard(dashboard.id, {
        filterValues: appliedFilterValues,
        paramValues: params,
        tileFilters: Object.fromEntries(
          Object.entries(tileFilterValues)
            .filter(([, values]) => Object.keys(values).length > 0)
            .map(([tileId, values]) => [
              tileId,
              { filters: tileFilterFields[tileId] ?? [], values },
            ]),
        ),
      });
      downloadBlob(blob, fileName);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "The dashboard could not be exported");
    } finally {
      setExporting(false);
    }
  }, [dashboard.id, appliedFilterValues, params, tileFilterValues, tileFilterFields]);

  const startEditing = () => {
    setDraft(savedLayout);
    setName(dashboard.name);
    setEditing(true);
  };

  const patchTile = useCallback((tileId: string, patch: Partial<ReportDashboardTile>) => {
    setDraft((prev) => ({
      ...prev,
      tiles: packTiles(
        prev.tiles.map((tile) => (tile.id === tileId ? { ...tile, ...patch } : tile)),
      ),
    }));
  }, []);

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event;
    if (!over || active.id === over.id) return;
    setDraft((prev) => {
      const from = prev.tiles.findIndex((tile) => tile.id === active.id);
      const to = prev.tiles.findIndex((tile) => tile.id === over.id);
      if (from < 0 || to < 0) return prev;
      return { ...prev, tiles: packTiles(arrayMove(prev.tiles, from, to)) };
    });
  };

  const addTile = () => {
    const tile: ReportDashboardTile = {
      id: nextTileId(draft.tiles),
      kind: "chart",
      x: 0,
      y: 0,
      w: 6,
      h: 4,
    };
    setEditingTile(tile);
  };

  const saveTile = (tile: ReportDashboardTile) => {
    setDraft((prev) => {
      const exists = prev.tiles.some((existing) => existing.id === tile.id);
      const tiles = exists
        ? prev.tiles.map((existing) => (existing.id === tile.id ? tile : existing))
        : [...prev.tiles, tile];
      return { ...prev, tiles: packTiles(tiles) };
    });
  };

  const handleSave = () => {
    updateDashboard.mutate(
      {
        id: dashboard.id,
        version: dashboard.version,
        name: name.trim() || dashboard.name,
        description: dashboard.description || undefined,
        category: dashboard.category || undefined,
        tags: dashboard.tags,
        visibility: dashboard.visibility,
        layout: {
          tiles: packTiles(draft.tiles),
          parameters: draft.parameters,
          filters: draft.filters,
        },
      },
      {
        onSuccess: () => {
          setEditing(false);
          toast.success("Dashboard saved");
        },
        onError: (error) => toast.error(graphQLErrorMessage(error, "Failed to save the dashboard")),
      },
    );
  };

  const handleDelete = () => {
    deleteDashboard.mutate(dashboard.id, {
      onSuccess: () => {
        toast.success("Dashboard deleted");
        void navigate("/reports?tab=dashboards");
      },
      onError: (error) => toast.error(graphQLErrorMessage(error, "Failed to delete the dashboard")),
    });
  };

  return (
    <div className="flex h-full min-h-0 flex-col">
      <header className="flex h-12 shrink-0 items-center gap-2 border-b border-border px-3">
        <Button
          variant="ghost"
          size="icon"
          className="size-7"
          render={<Link to="/reports?tab=dashboards" />}
          aria-label="Back to dashboards"
        >
          <ArrowLeftIcon className="size-4" />
        </Button>
        {editing ? (
          <Input
            className="h-8 max-w-80"
            value={name}
            onChange={(event) => setName(event.target.value)}
            placeholder="Dashboard name"
          />
        ) : (
          <span className="truncate text-sm font-medium">{dashboard.name}</span>
        )}
        <div className="flex-1" />
        {editing ? (
          <>
            <Button variant="outline" size="sm" className="h-7" onClick={addTile}>
              <PlusIcon className="size-3.5" />
              Tile
            </Button>
            <Button
              variant="outline"
              size="sm"
              className="h-7"
              onClick={() => setSettingsOpen(true)}
            >
              <SettingsIcon className="size-3.5" />
              Filters
            </Button>
            <Button
              variant="ghost"
              size="sm"
              className="h-7 text-destructive"
              onClick={handleDelete}
              disabled={deleteDashboard.isPending}
            >
              <Trash2Icon className="size-3.5" />
              Delete
            </Button>
            <Button variant="ghost" size="sm" className="h-7" onClick={() => setEditing(false)}>
              Cancel
            </Button>
            <Button
              size="sm"
              className="h-7"
              onClick={handleSave}
              disabled={updateDashboard.isPending}
            >
              {updateDashboard.isPending ? "Saving..." : "Save"}
            </Button>
          </>
        ) : (
          <>
            <Button
              variant="outline"
              size="sm"
              className="h-7"
              disabled={exporting}
              onClick={handleExport}
            >
              <DownloadIcon className="size-3.5" />
              {exporting ? "Exporting..." : "Export"}
            </Button>
            {canEdit && (
              <Button variant="outline" size="sm" className="h-7" onClick={startEditing}>
                <PencilIcon className="size-3.5" />
                Edit
              </Button>
            )}
          </>
        )}
      </header>

      {(parameters.length > 0 ||
        configuredFilters.length > 0 ||
        Object.keys(crossFilters).length > 0) && (
        <div className="flex flex-wrap items-end gap-3 border-b border-border bg-muted/20 px-3 py-2">
          {configuredFilters.map((filter) => {
            const field = index ? resolveField(index, filter.entity, filter.ref) : undefined;
            return (
              <div key={filter.id} className="flex flex-col gap-1">
                <Label className="text-xs text-muted-foreground">
                  {filter.label || field?.label || filter.ref.field}
                </Label>
                <FilterValueEditor
                  field={field}
                  operator={filter.operator}
                  value={filterValues[filter.id]}
                  refEntityKey={
                    index ? refTargetEntity(index, filter.entity, filter.ref) : undefined
                  }
                  onChange={(value) => setFilterValues((prev) => ({ ...prev, [filter.id]: value }))}
                />
              </div>
            );
          })}
          {parameters.map((param) => (
            <div key={param.name} className="w-48">
              <ParameterField
                param={param}
                value={params[param.name]}
                compact
                onChange={(value) => setParams((prev) => ({ ...prev, [param.name]: value }))}
              />
            </div>
          ))}
          {Object.entries(crossFilters).map(([id, entry]) => (
            <Badge key={id} variant="secondary" className="h-7 gap-1 px-2">
              <span className="max-w-48 truncate">
                {(index
                  ? refLabel(index, entry.filter.entity, entry.filter.ref)
                  : entry.filter.ref.field) +
                  ": " +
                  entry.label}
              </span>
              <button
                type="button"
                aria-label="Remove filter"
                onClick={() => setCrossFilters((prev) => removeCrossFilter(prev, id))}
              >
                <XIcon className="size-3" />
              </button>
            </Badge>
          ))}
          {(Object.values(filterValues).some((value) => value !== undefined) ||
            Object.keys(crossFilters).length > 0) && (
            <Button
              variant="ghost"
              size="sm"
              className="h-7"
              onClick={() => {
                setFilterValues({});
                setCrossFilters({});
              }}
            >
              Clear
            </Button>
          )}
        </div>
      )}

      <div className="min-h-0 flex-1 overflow-auto bg-muted/20 p-3">
        {layout.tiles.length === 0 ? (
          <div className="flex h-full flex-col items-center justify-center gap-3">
            <div className="flex size-10 items-center justify-center rounded-lg bg-muted">
              <LayoutDashboardIcon className="size-5 text-muted-foreground" strokeWidth={1.75} />
            </div>
            <div className="text-center">
              <p className="text-sm font-medium">Nothing here yet</p>
              <p className="max-w-xs text-xs text-muted-foreground">
                {editing
                  ? "Add a tile to point at a report."
                  : "Edit this dashboard to add your first tile."}
              </p>
            </div>
          </div>
        ) : (
          <DndContext
            sensors={sensors}
            collisionDetection={closestCenter}
            onDragEnd={handleDragEnd}
          >
            <SortableContext
              items={layout.tiles.map((tile) => tile.id)}
              strategy={rectSortingStrategy}
            >
              <div
                className="grid gap-3"
                style={{
                  gridTemplateColumns: `repeat(${DASHBOARD_GRID_COLUMNS}, minmax(0, 1fr))`,
                  gridAutoRows: "64px",
                }}
              >
                {layout.tiles.map((tile) => (
                  <TileFrame
                    key={tile.id}
                    tile={tile}
                    params={params}
                    filters={fieldFilters}
                    filterValues={appliedFilterValues}
                    index={index}
                    onTileFilterFields={registerTileFilterFields}
                    tileFilterValues={tileFilterValues[tile.id] ?? EMPTY_VALUES}
                    onTileFilterChange={(values) =>
                      setTileFilterValues((prev) => ({ ...prev, [tile.id]: values }))
                    }
                    crossFilters={crossFilters}
                    onCrossFilter={handleCrossFilter}
                    editing={editing}
                    onEdit={() => setEditingTile(tile)}
                    onRemove={() =>
                      setDraft((prev) => ({
                        ...prev,
                        tiles: packTiles(prev.tiles.filter((t) => t.id !== tile.id)),
                      }))
                    }
                    onResize={(patch) => patchTile(tile.id, patch)}
                  />
                ))}
              </div>
            </SortableContext>
          </DndContext>
        )}
        {editing && draft.tiles.length >= MAX_DASHBOARD_TILES && (
          <p className="pt-3 text-center text-2xs text-muted-foreground">
            This dashboard has reached the {MAX_DASHBOARD_TILES}-tile limit.
          </p>
        )}
      </div>

      {editingTile && (
        <TileEditorDialog
          open
          onOpenChange={(open) => {
            if (!open) setEditingTile(null);
          }}
          tile={editingTile}
          dashboardParams={draft.parameters ?? []}
          onSave={saveTile}
        />
      )}

      <DashboardSettingsDialog
        open={settingsOpen}
        onOpenChange={setSettingsOpen}
        layout={draft}
        entities={entities}
        candidates={candidates}
        onSave={(next) => setDraft(next)}
      />
    </div>
  );
}
