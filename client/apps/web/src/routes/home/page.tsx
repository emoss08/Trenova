import { Metadata } from "@/components/metadata";
import {
  useHomeLayout,
  useHomeWidgetCatalog,
  useResetHomeLayout,
  useUpdateHomeLayout,
} from "@/hooks/use-home-layout";
import {
  DEFAULT_HOME_LAYOUT,
  toWidgetInput,
  type HomeWidget,
} from "@/lib/graphql/home-layout";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { graphQLErrorMessage } from "@trenova/shared/lib/graphql";
import { Undo2Icon } from "lucide-react";
import { useEffect, useState } from "react";
import { toast } from "sonner";
import { BriefingBar } from "./_components/briefing-bar";
import { HomeCanvas } from "./_components/home-canvas";
import { useHomeData } from "./_components/use-home-data";

function sameWidgets(a: HomeWidget[], b: HomeWidget[]): boolean {
  if (a.length !== b.length) return false;
  return a.every(
    (widget, index) =>
      JSON.stringify(toWidgetInput(widget)) === JSON.stringify(toWidgetInput(b[index])),
  );
}

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
  const data = useHomeData(widgets, { briefing: true });

  const startEditing = () => {
    setDraft(layout.widgets);
    setEditing(true);
  };

  const discard = () => {
    setDraft(null);
    setEditing(false);
  };

  const save = async () => {
    // Looking around in edit mode and pressing Done must not count as a
    // customization: saving an untouched draft would diverge the user from
    // their preset and cut them off from its future updates.
    if (draft == null || sameWidgets(draft, layout.widgets)) {
      discard();
      return;
    }

    try {
      await updateLayout.mutateAsync({
        version: layout.version,
        customized: true,
        density: layout.density,
        widgets: draft.map(toWidgetInput),
      });
      setDraft(null);
      setEditing(false);
      toast.success("Home screen saved");
    } catch (error) {
      toast.error(graphQLErrorMessage(error, "Could not save your home screen"));
    }
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
        analyticsReady={data.shipmentAnalyticsReady}
        loading={layoutQuery.isLoading || data.attentionLoading || data.shipmentAnalyticsLoading}
        canCustomize={layout.canCustomize}
        locked={layout.locked}
        editing={editing}
        saving={updateLayout.isPending}
        onCustomize={() => (editing ? void save() : startEditing())}
      />

      <div className="flex flex-col gap-3 p-4">
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
            data={data}
            catalog={catalog.data}
            catalogLoading={catalog.isLoading}
            editing={editing}
            onChange={setDraft}
            dock={{
              dirty: draft != null && !sameWidgets(draft, layout.widgets),
              saving: updateLayout.isPending,
              onSave: () => void save(),
              onDiscard: discard,
            }}
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
    <div className="flex flex-wrap items-center gap-1.5 self-start rounded-full border border-border/70 bg-card py-0.5 pr-1 pl-2.5 text-2xs text-muted-foreground">
      <span>Customized{presetName ? ` from ${presetName}` : ""}</span>
      <Button
        variant="ghost"
        size="xxs"
        className="h-4.5 rounded-full px-1.5"
        onClick={onReset}
        disabled={pending}
      >
        <Undo2Icon className="size-2.5" />
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
