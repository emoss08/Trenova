import { usePermission } from "@/hooks/use-permission";
import { useAttentionSummary } from "@/hooks/use-attention";
import {
  DEFAULT_PANEL_SIZE,
  dockPositionClass,
  isLeftDock,
  panelSizeStyle,
  type AssistantLayout,
  type AssistantPanelSize,
} from "@/lib/assistant-dock";
import { useAssistantStore } from "@/stores/assistant-store";
import { useHotkey } from "@tanstack/react-hotkeys";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { AnimatePresence, m, useReducedMotion } from "motion/react";
import { Fragment, useEffect, useMemo, useState } from "react";
import { useSearchParams } from "react-router";
import { AssistantEdgeTab } from "./assistant-edge-tab";
import type { AssistantView } from "./assistant-header";
import { AssistantLauncher } from "./assistant-launcher";
import { AssistantPanel } from "./assistant-panel";
import { AssistantResizeHandle } from "./assistant-resize-handle";
import { ASSISTANT_SURFACE_ID } from "./assistant-surface";
import { beaconState } from "./beacon-state";
import { useLiveTurns } from "./use-active-turns";
import { useAssistantThreads } from "./use-assistant-threads";
import "./assistant.css";

const OPEN_PARAM = "assistant";

/** What the panel shows: in full screen the sidebar stands in for the history view. */
export function assistantView(
  layout: AssistantLayout,
  historyOpen: boolean,
  hasThread: boolean,
): AssistantView {
  if (historyOpen && layout !== "full") {
    return "history";
  }

  return hasThread ? "thread" : "home";
}

/**
 * The side the panel is docked to while it is open and docked, so the app's
 * page can make room beside it; null when it floats, fills the screen, is
 * closed, or cannot be used.
 */
export function useAssistantSideInset(): "left" | "right" | null {
  const { allowed } = usePermission(Resource.Assistant, Operation.Read);
  const open = useAssistantStore((state) => state.open);
  const layout = useAssistantStore((state) => state.layout);
  const dock = useAssistantStore((state) => state.dock);

  if (!allowed || !open || layout !== "side") {
    return null;
  }

  return isLeftDock(dock) ? "left" : "right";
}

/**
 * The assistant lives on every page, because a question about a shipment comes
 * up while looking at the shipment. It is mounted once in the shell; the
 * store remembers its layout (floating in a corner, docked down the side with
 * the page beside it, or full screen), which corner it sits in, how big the
 * floating panel is, and whether the beacon is tucked away, because whatever
 * it covers on one page it covers on every page.
 */
export function AssistantWidget() {
  const t = useT();
  const reduceMotion = useReducedMotion();
  const { allowed } = usePermission(Resource.Assistant, Operation.Read);
  const { allowed: canSeeProposals } = usePermission(Resource.AgentProposal, Operation.Read);

  const open = useAssistantStore((state) => state.open);
  const layout = useAssistantStore((state) => state.layout);
  const setLayout = useAssistantStore((state) => state.setLayout);
  const openWidget = useAssistantStore((state) => state.openWidget);
  const closeWidget = useAssistantStore((state) => state.closeWidget);
  const toggleWidget = useAssistantStore((state) => state.toggleWidget);
  const dock = useAssistantStore((state) => state.dock);
  const setDock = useAssistantStore((state) => state.setDock);
  const launcherHidden = useAssistantStore((state) => state.launcherHidden);
  const setLauncherHidden = useAssistantStore((state) => state.setLauncherHidden);
  const savedSize = useAssistantStore((state) => state.panelSize);
  const setPanelSize = useAssistantStore((state) => state.setPanelSize);
  const activeThreadId = useAssistantStore((state) => state.activeThreadId);
  const setActiveThreadId = useAssistantStore((state) => state.setActiveThreadId);
  const lastAgentId = useAssistantStore((state) => state.lastAgentId);
  const repliedThreadId = useAssistantStore((state) => state.repliedThreadId);
  const sidebarCollapsed = useAssistantStore((state) => state.sidebarCollapsed);
  // While a resize is under way the size lives here, and is saved once when
  // it ends rather than on every pointer move.
  const [liveSize, setLiveSize] = useState<AssistantPanelSize | null>(null);
  const panelSize = liveSize ?? savedSize ?? DEFAULT_PANEL_SIZE;
  const resizing = liveSize !== null;
  const [historyOpen, setHistoryOpen] = useState(false);

  const data = useAssistantThreads(activeThreadId);
  const view = assistantView(layout, historyOpen, data.activeThread !== null);
  const full = layout === "full";
  const tall = layout === "compact" && view !== "home";

  const [searchParams, setSearchParams] = useSearchParams();
  const requestedOpen = searchParams.get(OPEN_PARAM) === "open";

  useEffect(() => {
    if (!allowed || !requestedOpen) {
      return;
    }
    openWidget();
    setSearchParams(
      (current) => {
        const next = new URLSearchParams(current);
        next.delete(OPEN_PARAM);
        return next;
      },
      { replace: true },
    );
  }, [allowed, openWidget, requestedOpen, setSearchParams]);

  useHotkey("Mod+J", () => toggleWidget(), {
    ignoreInputs: false,
    preventDefault: true,
    enabled: allowed,
  });

  // Esc shrinks the full-screen layout back into the corner; otherwise it closes.
  useHotkey("Escape", () => (full ? setLayout("compact") : closeWidget()), {
    ignoreInputs: false,
    preventDefault: false,
    enabled: allowed && open,
  });

  // The beacon is a signal: it says what the assistant is doing and, above
  // all, when a change is waiting on someone. The count is the same one the
  // sidebar and the Desk show, read once through the attention summary.
  const { data: attention } = useAttentionSummary();
  const pendingCount = allowed && canSeeProposals ? (attention?.agentDecisions ?? 0) : 0;
  const liveTurns = useLiveTurns();
  const { agents, agentsById, threads } = data;
  const repliedThread = repliedThreadId
    ? (threads.find((thread) => thread.id === repliedThreadId) ?? null)
    : null;
  const beacon = useMemo(
    () =>
      beaconState({
        pendingCount,
        liveTurns,
        repliedAgentName:
          repliedThreadId === null
            ? null
            : (agentsById.get(repliedThread?.agentDefinitionId ?? "")?.name ?? t("The assistant")),
        lastAgentName: agentsById.get(lastAgentId ?? "")?.name ?? agents[0]?.name ?? "",
      }),
    [agents, agentsById, lastAgentId, liveTurns, pendingCount, repliedThread, repliedThreadId, t],
  );

  // Opened from the beacon, the panel goes where the beacon pointed: the
  // conversation a change waits on, or the one that replied.
  const openFromBeacon = () => {
    const target =
      beacon.mode === "pending"
        ? threads.find((thread) => (thread.attention?.pendingDecisions ?? 0) > 0)
        : beacon.mode === "replied"
          ? repliedThread
          : null;
    if (target) {
      setActiveThreadId(target.id);
      setHistoryOpen(false);
    }
    openWidget();
  };

  if (!allowed) {
    return null;
  }

  const sizeStyle = panelSizeStyle(dock, panelSize);

  return (
    <AnimatePresence initial={false} mode="popLayout">
      {open ? (
        <Fragment key="panel">
          {/* Full screen, the panel is the only thing being used, so it says so. */}
          {full && (
            <m.div
              key="scrim"
              aria-hidden
              initial={reduceMotion ? false : { opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
              transition={{ duration: 0.15 }}
              onClick={() => setLayout("compact")}
              className="as-scrim"
            />
          )}
          <m.section
            role="dialog"
            aria-label={t("Assistant")}
            aria-modal={full}
            layout
            layoutId={ASSISTANT_SURFACE_ID}
            data-side={isLeftDock(dock) ? "left" : "right"}
            style={
              layout === "compact"
                ? {
                    borderRadius: 18,
                    width: sizeStyle.width,
                    ...(tall ? { height: sizeStyle.height } : {}),
                  }
                : { borderRadius: full ? 18 : 0 }
            }
            transition={
              reduceMotion || resizing
                ? { duration: 0 }
                : { type: "spring", stiffness: 380, damping: 36 }
            }
            className={cn(
              "as-pn dk-chat",
              `as-${layout}`,
              tall && "as-tall",
              full && sidebarCollapsed && "as-sbx",
              layout === "compact" && dockPositionClass(dock, "panel"),
            )}
          >
            {layout === "compact" && (
              <AssistantResizeHandle
                dock={dock}
                size={panelSize}
                onResize={setLiveSize}
                onResizeEnd={(size) => {
                  setPanelSize(size);
                  setLiveSize(null);
                }}
                onReset={() => {
                  setPanelSize(null);
                  setLiveSize(null);
                }}
              />
            )}
            <AssistantPanel
              layout={layout}
              view={view}
              data={data}
              onHistory={setHistoryOpen}
              onLayout={(next) => {
                setLayout(next);
                setHistoryOpen(false);
              }}
              onClose={closeWidget}
            />
          </m.section>
        </Fragment>
      ) : launcherHidden ? (
        <AssistantEdgeTab key="tab" dock={dock} beacon={beacon} onClick={openFromBeacon} />
      ) : (
        <AssistantLauncher
          key="launcher"
          dock={dock}
          beacon={beacon}
          onClick={openFromBeacon}
          onMove={setDock}
          onHide={() => setLauncherHidden(true)}
        />
      )}
    </AnimatePresence>
  );
}
