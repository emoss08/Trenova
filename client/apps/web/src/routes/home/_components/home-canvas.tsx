import { nextItemId } from "@/components/tile-grid/pack-tiles";
import { SizeStepper } from "@/components/tile-grid/size-stepper";
import { SortableTile, TileDragHandle } from "@/components/tile-grid/sortable-tile";
import { TileGrid } from "@/components/tile-grid/tile-grid";
import { Button } from "@trenova/shared/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import type { HomeWidget, HomeWidgetCatalog, HomeWidgetOption } from "@/lib/graphql/home-layout";
import {
  ChevronsLeftRightIcon,
  ChevronsUpDownIcon,
  LayoutGridIcon,
  MoreHorizontalIcon,
  PencilIcon,
  PlusIcon,
  Trash2Icon,
} from "lucide-react";
import { m } from "motion/react";
import { useMemo, useState } from "react";
import { AddWidgetDialog } from "./add-widget-dialog";
import { HomeEditDock } from "./edit-dock";
import type { HomeData } from "./use-home-data";
import { widgetComponentFor, type WidgetProps } from "./widget-registry";
import { WidgetConfigDialog } from "./widget-config-dialog";
import { WidgetEditingProvider } from "./widget-shell";

const DEFAULT_GRID_COLUMNS = 12;

export type HomeCanvasDock = {
  dirty: boolean;
  saving: boolean;
  saveLabel?: string;
  onSave: () => void;
  onDiscard: () => void;
};

export type HomeCanvasProps = {
  widgets: HomeWidget[];
  data: HomeData;
  catalog: HomeWidgetCatalog | undefined;
  catalogLoading: boolean;
  editing: boolean;
  onChange: (widgets: HomeWidget[]) => void;
  /**
   * Save and discard actions for the floating edit dock. The dock renders from
   * inside the canvas because that is where the add-widget flow lives; the
   * caller only supplies what the canvas cannot know — whether the draft is
   * dirty and how to persist it.
   */
  dock?: HomeCanvasDock;
};

type ConfigTarget = {
  widget: HomeWidget;
  /** Set when the widget was just added, so cancelling removes it again. */
  isNew: boolean;
};

/**
 * The widget grid. In read mode it is a plain grid of live tiles; in edit mode
 * the same tiles swap their header for a drag bar with controls, and their
 * bodies go inert so a drag can never fire a link underneath it.
 */
export function HomeCanvas({
  widgets,
  data,
  catalog,
  catalogLoading,
  editing,
  onChange,
  dock,
}: HomeCanvasProps) {
  const [addOpen, setAddOpen] = useState(false);
  const [configuring, setConfiguring] = useState<ConfigTarget | null>(null);

  const columns = catalog?.gridColumns ?? DEFAULT_GRID_COLUMNS;
  const maxWidgets = catalog?.maxWidgets ?? 24;
  const optionsByKey = useMemo(
    () => new Map((catalog?.widgets ?? []).map((option) => [option.key, option])),
    [catalog?.widgets],
  );
  const usedKeys = useMemo(() => new Set(widgets.map((widget) => widget.key)), [widgets]);

  const addWidget = (option: HomeWidgetOption) => {
    const widget: HomeWidget = {
      id: nextItemId(widgets, "widget_"),
      key: option.key,
      title: null,
      w: option.defaultW,
      h: option.defaultH,
      config: emptyConfig(),
    };

    setAddOpen(false);
    onChange([...widgets, widget]);

    // A widget that cannot draw anything until it is told what to show opens
    // its editor immediately, rather than landing as an empty card.
    if (option.configKind !== "none" && option.configKind !== "queue") {
      setConfiguring({ widget, isNew: true });
    }
  };

  const patchWidget = (id: string, patch: Partial<HomeWidget>) =>
    onChange(widgets.map((widget) => (widget.id === id ? { ...widget, ...patch } : widget)));

  const removeWidget = (id: string) => onChange(widgets.filter((widget) => widget.id !== id));

  const closeConfig = (saved: boolean) => {
    if (configuring && configuring.isNew && !saved) {
      removeWidget(configuring.widget.id);
    }
    setConfiguring(null);
  };

  if (widgets.length === 0 && !editing) {
    return (
      <div className="flex flex-col items-center justify-center gap-2 rounded-lg border border-dashed border-border py-16 text-center">
        <LayoutGridIcon className="size-5 text-muted-foreground/40" />
        <p className="text-sm font-medium">Your home screen is empty</p>
        <p className="max-w-xs text-xs text-muted-foreground">
          Choose Customize to add the queues and numbers you want to land on.
        </p>
      </div>
    );
  }

  return (
    <WidgetEditingProvider value={editing}>
      {editing && widgets.length === 0 && (
        <button
          type="button"
          onClick={() => setAddOpen(true)}
          className="flex flex-col items-center justify-center gap-2 rounded-lg border border-dashed border-border py-12 text-center transition-colors hover:border-foreground/25 hover:bg-muted/30"
        >
          <PlusIcon className="size-5 text-muted-foreground/50" />
          <span className="text-sm font-medium">Add your first widget</span>
          <span className="max-w-xs text-xs text-muted-foreground">
            Pick the queues and numbers this home screen should open on.
          </span>
        </button>
      )}

      <TileGrid
        items={widgets}
        columns={columns}
        onReorder={onChange}
        className={editing ? "rounded-lg bg-muted/20 p-2" : undefined}
      >
        {(widget, index) => (
          <SortableTile
            key={widget.id}
            id={widget.id}
            w={widget.w}
            h={widget.h}
            editing={editing}
            className={editing ? "ring-1 ring-border" : undefined}
          >
            {(drag) => (
              <m.div
                initial={{ opacity: 0, y: 8 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.3, delay: Math.min(index, 8) * 0.03, ease: "easeOut" }}
                className="flex min-h-0 flex-1 flex-col"
              >
                {editing && (
                  <div className="flex h-8 shrink-0 items-center gap-1.5 border-b border-border/60 bg-muted/40 px-2">
                    <TileDragHandle {...drag} />
                    <span className="cc-label min-w-0 flex-1 truncate text-foreground">
                      {widget.title || optionsByKey.get(widget.key)?.label || widget.key}
                    </span>
                    <span className="shrink-0 font-mono text-[9px] text-muted-foreground tabular-nums">
                      {widget.w}×{widget.h}
                    </span>
                    <WidgetControls
                      widget={widget}
                      option={optionsByKey.get(widget.key)}
                      columns={columns}
                      onEdit={() => setConfiguring({ widget, isNew: false })}
                      onRemove={() => removeWidget(widget.id)}
                      onResize={(patch) => patchWidget(widget.id, patch)}
                    />
                  </div>
                )}
                <div
                  className="flex min-h-0 flex-1 flex-col"
                  inert={editing || undefined}
                >
                  <WidgetBody widget={widget} data={data} />
                </div>
              </m.div>
            )}
          </SortableTile>
        )}
      </TileGrid>

      {editing && dock && (
        <>
          {/* Keeps the last row of tiles reachable underneath the fixed dock. */}
          <div className="h-16" aria-hidden />
          <HomeEditDock
            dirty={dock.dirty}
            saving={dock.saving}
            saveLabel={dock.saveLabel}
            addDisabled={widgets.length >= maxWidgets}
            maxWidgets={maxWidgets}
            onAdd={() => setAddOpen(true)}
            onDiscard={dock.onDiscard}
            onSave={dock.onSave}
          />
        </>
      )}

      <AddWidgetDialog
        open={addOpen}
        onOpenChange={setAddOpen}
        catalog={catalog}
        loading={catalogLoading}
        usedKeys={usedKeys}
        remaining={Math.max(maxWidgets - widgets.length, 0)}
        onAdd={addWidget}
      />

      {configuring && (
        <WidgetConfigDialog
          widget={configuring.widget}
          option={optionsByKey.get(configuring.widget.key)}
          metrics={catalog?.metrics ?? []}
          onSave={(updated) => {
            patchWidget(updated.id, updated);
            closeConfig(true);
          }}
          onCancel={() => closeConfig(false)}
        />
      )}
    </WidgetEditingProvider>
  );
}

/**
 * Renders one widget, or nothing at all when the deployed client has no
 * component for its key. A silent skip is right here: the server already
 * dropped everything the viewer may not see, so a key that survives to the
 * client and has no drawer is a version skew, not a user's problem.
 */
function WidgetBody({ widget, data }: WidgetProps) {
  const Component = widgetComponentFor(widget.key);
  if (!Component) return null;
  return <Component widget={widget} data={data} />;
}

function WidgetControls({
  widget,
  option,
  columns,
  onEdit,
  onRemove,
  onResize,
}: {
  widget: HomeWidget;
  option: HomeWidgetOption | undefined;
  columns: number;
  onEdit: () => void;
  onRemove: () => void;
  onResize: (patch: Partial<HomeWidget>) => void;
}) {
  const configurable = option != null && option.configKind !== "none";

  return (
    <Popover>
      <PopoverTrigger
        render={
          <Button
            variant="ghost"
            size="icon"
            className="size-5 shrink-0"
            aria-label="Widget options"
          >
            <MoreHorizontalIcon className="size-3" />
          </Button>
        }
      />
      <PopoverContent className="w-60 p-2" align="end">
        <div className="flex flex-col gap-2">
          <SizeStepper
            icon={ChevronsLeftRightIcon}
            label="Width"
            value={widget.w}
            min={option?.minW ?? 1}
            max={Math.min(option?.maxW ?? columns, columns)}
            onChange={(w) => onResize({ w })}
          />
          <SizeStepper
            icon={ChevronsUpDownIcon}
            label="Height"
            value={widget.h}
            min={option?.minH ?? 1}
            max={option?.maxH ?? 12}
            onChange={(h) => onResize({ h })}
          />
          <div className="flex flex-col gap-1 border-t border-border pt-2">
            {configurable && (
              <Button variant="ghost" size="sm" className="h-7 justify-start" onClick={onEdit}>
                <PencilIcon className="size-3.5" />
                Configure
              </Button>
            )}
            <Button
              variant="ghost"
              size="sm"
              className="h-7 justify-start text-destructive"
              onClick={onRemove}
            >
              <Trash2Icon className="size-3.5" />
              Remove
            </Button>
          </div>
        </div>
      </PopoverContent>
    </Popover>
  );
}

function emptyConfig(): HomeWidget["config"] {
  return {
    metric: null,
    metrics: null,
    definitionId: null,
    cannedKey: null,
    chartId: null,
    columnId: null,
    dashboardId: null,
    text: null,
    limit: null,
    windowDays: null,
  };
}
