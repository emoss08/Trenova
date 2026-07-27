import { Metadata } from "@/components/metadata";
import {
  useHomeLayout,
  useHomeWidgetCatalog,
  useResetHomeLayout,
  useUpdateHomeLayout,
} from "@/hooks/use-home-layout";
import { DEFAULT_HOME_LAYOUT, type HomeWidget } from "@/lib/graphql/home-layout";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { graphQLErrorMessage } from "@trenova/shared/lib/graphql";
import { useEffect, useState } from "react";
import { toast } from "sonner";
import { BriefingBar } from "./_components/briefing-bar";
import { HomeCanvas } from "./_components/home-canvas";
import { useHomeData } from "./_components/use-home-data";

export function Home() {
  const layoutQuery = useHomeLayout();
  const layout = layoutQuery.data ?? DEFAULT_HOME_LAYOUT;

  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState<HomeWidget[] | null>(null);

  const catalog = useHomeWidgetCatalog(editing);
  const updateLayout = useUpdateHomeLayout();
  const resetLayout = useResetHomeLayout();

  const widgets = draft ?? layout.widgets;

  // A preset the administrator locked while this tab was open must not leave
  // the user editing something they can no longer save.
  useEffect(() => {
    if (!layout.canCustomize && editing) {
      setEditing(false);
      setDraft(null);
    }
  }, [layout.canCustomize, editing]);

  // The briefing bar reads the same payload the canvas does, so both show the
  // same instant's numbers rather than two requests a second apart.
  const data = useHomeData(widgets);

  const startEditing = () => {
    setDraft(layout.widgets);
    setEditing(true);
  };

  const save = async () => {
    try {
      await updateLayout.mutateAsync({
        version: layout.version,
        customized: true,
        density: layout.density,
        widgets: widgets.map(toInput),
      });
      setDraft(null);
      setEditing(false);
      toast.success("Home screen saved");
    } catch (error) {
      toast.error(graphQLErrorMessage(error, "Could not save your home screen"));
    }
  };

  const discard = () => {
    setDraft(null);
    setEditing(false);
  };

  const reset = async () => {
    try {
      await resetLayout.mutateAsync(undefined);
      setDraft(null);
      setEditing(false);
      toast.success(
        layout.presetName ? `Restored ${layout.presetName}` : "Restored the default home screen",
      );
    } catch (error) {
      toast.error(graphQLErrorMessage(error, "Could not restore your home screen"));
    }
  };

  return (
    <>
      <Metadata title="Home" description="Your work, your numbers, and where to go next." />

      <BriefingBar
        attention={data.attention}
        analytics={data.shipmentAnalytics}
        loading={layoutQuery.isLoading || data.attentionLoading}
        canCustomize={layout.canCustomize}
        locked={layout.locked}
        editing={editing}
        onCustomize={() => (editing ? void save() : startEditing())}
        timezone={Intl.DateTimeFormat().resolvedOptions().timeZone}
      />

      <div className="flex flex-col gap-3 p-4">
        {editing && (
          <div className="flex flex-wrap items-center justify-between gap-2 rounded-md border border-brand/30 bg-brand/5 px-3 py-2">
            <p className="text-xs text-muted-foreground">
              Drag to rearrange, resize from a widget&rsquo;s menu, then save.
            </p>
            <div className="flex items-center gap-2">
              <Button variant="ghost" size="sm" onClick={discard}>
                Discard
              </Button>
              <Button size="sm" onClick={() => void save()} disabled={updateLayout.isPending}>
                Save
              </Button>
            </div>
          </div>
        )}

        {!editing && layout.source === "USER" && (
          <DivergenceChip
            presetName={layout.presetName}
            pending={resetLayout.isPending}
            onReset={() => void reset()}
          />
        )}

        {layoutQuery.isLoading ? (
          <HomeSkeleton />
        ) : (
          <HomeCanvas
            widgets={widgets}
            catalog={catalog.data}
            catalogLoading={catalog.isLoading}
            editing={editing}
            onChange={setDraft}
          />
        )}
      </div>
    </>
  );
}

/**
 * Says what a user diverged from and offers the way back. Without it, a home
 * screen an administrator later improves would silently stop reaching the
 * people who once moved a tile on it.
 */
function DivergenceChip({
  presetName,
  pending,
  onReset,
}: {
  presetName: string | null | undefined;
  pending: boolean;
  onReset: () => void;
}) {
  return (
    <div className="flex flex-wrap items-center gap-2 text-2xs text-muted-foreground">
      <span>Customized{presetName ? ` from ${presetName}` : ""}</span>
      <Button variant="ghost" size="xxs" onClick={onReset} disabled={pending}>
        Reset
      </Button>
    </div>
  );
}

function HomeSkeleton() {
  return (
    <div
      className="grid gap-3"
      style={{ gridTemplateColumns: "repeat(12, minmax(0, 1fr))", gridAutoRows: "64px" }}
    >
      <Skeleton className="col-span-12 row-span-2 rounded-lg" />
      <Skeleton className="col-span-4 row-span-4 rounded-lg" />
      <Skeleton className="col-span-4 row-span-4 rounded-lg" />
      <Skeleton className="col-span-4 row-span-4 rounded-lg" />
    </div>
  );
}

function toInput(widget: HomeWidget) {
  return {
    id: widget.id,
    key: widget.key,
    title: widget.title,
    w: widget.w,
    h: widget.h,
    config: {
      metric: widget.config.metric,
      metrics: widget.config.metrics,
      definitionId: widget.config.definitionId,
      cannedKey: widget.config.cannedKey,
      chartId: widget.config.chartId,
      columnId: widget.config.columnId,
      dashboardId: widget.config.dashboardId,
      text: widget.config.text,
      limit: widget.config.limit,
      windowDays: widget.config.windowDays,
    },
  };
}
