import { useUserTimezone } from "@/hooks/use-user-timezone";
import { AI_SETUP_PATH } from "@/lib/ai-setup";
import { shipmentBoard } from "@/lib/queries/shipment-board";
import { useShipmentCapabilities } from "@/lib/shipment-board/capabilities";
import { useQuery } from "@tanstack/react-query";
import { FloatingPanel, FloatingPanelHeader } from "@/components/floating-panel";
import { PlugIcon } from "@trenova/shared/components/icons";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Tabs, TabsList, TabsPanel, TabsTab } from "@trenova/shared/components/ui/tabs";
import { useT } from "@trenova/shared/i18n/use-t";
import { useState } from "react";
import { useNavigate } from "react-router";
import { type PanelTab, useShipmentBoardUrl } from "../url-state";
import { ActionQueue } from "./action-queue";
import { ActivityTimeline } from "./activity/activity-timeline";
import { Watchlist } from "./watchlist";

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
    <FloatingPanel label={t("Brief and activity")}>
      <Tabs
        value={tab}
        onValueChange={(value) => void setUrl({ tab: value as PanelTab })}
        className="flex min-h-0 flex-1 flex-col gap-0"
      >
        <FloatingPanelHeader onClose={onClose}>
          <TabsList className="flex-1">
            <TabsTab value="brief">
              {ai ? t("Brief") : t("Overview")}
              {openCount > 0 ? (
                <span className="ml-1 rounded-badge bg-danger px-1.5 font-mono text-2xs text-foreground-on-solid tabular-nums">
                  {openCount}
                </span>
              ) : null}
            </TabsTab>
            <TabsTab value="activity">{t("Activity")}</TabsTab>
          </TabsList>
        </FloatingPanelHeader>
        <TabsPanel value="brief" className="flex min-h-0 flex-1 flex-col">
          <ScrollArea className="min-h-0 flex-1" maskVariant="raised">
            <div className="flex flex-col gap-5 p-4">
              {!ai ? (
                <button
                  type="button"
                  onClick={() => void navigate(AI_SETUP_PATH)}
                  className="flex items-center gap-2 rounded-md border border-dashed ui-focus-ring border-border p-2.5 text-left text-xs text-muted-foreground hover:text-foreground"
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
    </FloatingPanel>
  );
}
