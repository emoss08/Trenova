import { useUserTimezone } from "@/hooks/use-user-timezone";
import { shipmentBoard } from "@/lib/queries/shipment-board";
import { useShipmentCapabilities } from "@/lib/shipment-board/capabilities";
import { useQuery } from "@tanstack/react-query";
import { PlugIcon, XCloseIcon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Tabs, TabsList, TabsPanel, TabsTab } from "@trenova/shared/components/ui/tabs";
import { useT } from "@trenova/shared/i18n/use-t";
import { useState } from "react";
import { useNavigate } from "react-router";
import { type PanelTab, useShipmentBoardUrl } from "../url-state";
import { ActionQueue } from "./action-queue";
import { ActivityTimeline } from "./activity/activity-timeline";
import { SIDE_PANEL_FRAME_CLASS } from "./side-panel-skeleton";
import { Watchlist } from "./watchlist";

const AI_SETUP_PATH = "/admin/agent-control?tab=providers";

export function ShipmentSidePanel({ onClose }: { onClose: () => void }) {
  const t = useT();
  const navigate = useNavigate();
  const { ai } = useShipmentCapabilities();
  const timezone = useUserTimezone();
  const [{ tab }, setUrl] = useShipmentBoardUrl();
  const { data: suggestions } = useQuery({
    ...shipmentBoard.suggestions(timezone),
    staleTime: 15_000,
  });
  const openCount = suggestions?.items.length ?? 0;
  const [activityViewport, setActivityViewport] = useState<HTMLDivElement | null>(null);

  return (
    <aside aria-label={t("Brief and activity")} className={SIDE_PANEL_FRAME_CLASS}>
      <Tabs
        value={tab}
        onValueChange={(value) => void setUrl({ tab: value as PanelTab })}
        className="flex min-h-0 flex-1 flex-col gap-0"
      >
        <div className="border-border flex items-center gap-2 border-b px-3 py-2">
          <TabsList className="flex-1">
            <TabsTab value="brief">
              {ai ? t("Brief") : t("Overview")}
              {openCount > 0 ? (
                <span className="bg-danger text-foreground-on-solid ml-1 rounded-full px-1.5 font-mono text-2xs tabular-nums">
                  {openCount}
                </span>
              ) : null}
            </TabsTab>
            <TabsTab value="activity">{t("Activity")}</TabsTab>
          </TabsList>
          <Button variant="ghost" size="icon-sm" aria-label={t("Close panel")} onClick={onClose}>
            <XCloseIcon className="size-4" />
          </Button>
        </div>
        <TabsPanel value="brief" className="flex min-h-0 flex-1 flex-col">
          <ScrollArea className="min-h-0 flex-1" maskVariant="raised">
            <div className="flex flex-col gap-5 p-4">
              {!ai ? (
                <button
                  type="button"
                  onClick={() => void navigate(AI_SETUP_PATH)}
                  className="ui-focus-ring border-border text-muted-foreground hover:text-foreground flex items-center gap-2 rounded-md border border-dashed p-2.5 text-left text-xs"
                >
                  <PlugIcon className="size-3.5 shrink-0" />
                  {t(
                    "Connect an AI provider to get a written brief, drafted messages and fit scores.",
                  )}
                </button>
              ) : null}
              <ActionQueue />
              <Watchlist />
            </div>
          </ScrollArea>
        </TabsPanel>
        <TabsPanel value="activity" className="flex min-h-0 flex-1 flex-col">
          <ScrollArea
            className="min-h-0 flex-1"
            maskVariant="raised"
            viewportRef={setActivityViewport}
          >
            <ActivityTimeline scrollElement={activityViewport} />
          </ScrollArea>
        </TabsPanel>
      </Tabs>
    </aside>
  );
}
