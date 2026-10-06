import { useT } from "@trenova/shared/i18n/use-t";
import { PlugIcon, XCloseIcon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { Tabs, TabsList, TabsPanel, TabsTab } from "@trenova/shared/components/ui/tabs";
import { useUserTimezone } from "@/hooks/use-user-timezone";
import { queries } from "@/lib/queries";
import { useShipmentCapabilities } from "@/lib/shipment-board/capabilities";
import { useQuery } from "@tanstack/react-query";
import { useNavigate } from "react-router";
import { ActionQueue } from "./action-queue";
import { ActivityTimeline } from "./activity/activity-timeline";
import { Watchlist } from "./watchlist";
import { type PanelTab, useShipmentBoardUrl } from "../url-state";

const AI_SETUP_PATH = "/admin/agent-control?tab=providers";

/**
 * The floating panel over the board: the brief (the action queue and the
 * watchlist) and the activity feed. It floats rather than takes a column, so
 * the table keeps its width; it draws no shadow, only a stronger border.
 */
export function ShipmentSidePanel({ onClose }: { onClose: () => void }) {
  const t = useT();
  const navigate = useNavigate();
  const { ai } = useShipmentCapabilities();
  const timezone = useUserTimezone();
  const [{ tab }, setUrl] = useShipmentBoardUrl();
  const { data: suggestions } = useQuery({ ...queries.shipmentBoard.suggestions(timezone), staleTime: 15_000 });
  const openCount = suggestions?.items.length ?? 0;

  return (
    <aside
      aria-label={t("Brief and activity")}
      className="animate-panel-in bg-raised border-border-strong absolute top-2 right-3 bottom-3 z-30 flex w-[min(372px,calc(100%-28px))] flex-col overflow-hidden rounded-lg border @max-[820px]/board:inset-x-2 @max-[820px]/board:w-auto"
    >
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
        <TabsPanel value="brief" className="min-h-0 flex-1 overflow-y-auto">
          <div className="flex flex-col gap-5 p-4">
            {!ai ? (
              <button
                type="button"
                onClick={() => void navigate(AI_SETUP_PATH)}
                className="ui-focus-ring border-border text-muted-foreground hover:text-foreground flex items-center gap-2 rounded-md border border-dashed p-2.5 text-left text-xs"
              >
                <PlugIcon className="size-3.5 shrink-0" />
                {t("Connect an AI provider to get a written brief, drafted messages and fit scores.")}
              </button>
            ) : null}
            <ActionQueue />
            <Watchlist />
          </div>
        </TabsPanel>
        <TabsPanel value="activity" className="min-h-0 flex-1 overflow-y-auto">
          <ActivityTimeline />
        </TabsPanel>
      </Tabs>
    </aside>
  );
}
