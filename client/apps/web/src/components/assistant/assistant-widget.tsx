import { useT } from "@trenova/shared/i18n/use-t";
import { usePermission } from "@/hooks/use-permission";
import { useAttentionSummary } from "@/hooks/use-attention";
import {
  DEFAULT_PANEL_SIZE,
  dockPositionClass,
  panelSizeStyle,
  type AssistantPanelSize,
} from "@/lib/assistant-dock";
import { useAssistantStore } from "@/stores/assistant-store";
import { useHotkey } from "@tanstack/react-hotkeys";
import { cn } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { AnimatePresence, m, useReducedMotion } from "motion/react";
import { Fragment, useEffect, useState } from "react";
import { useSearchParams } from "react-router";
import { AssistantEdgeTab } from "./assistant-edge-tab";
import { AssistantLauncher } from "./assistant-launcher";
import { AssistantResizeHandle } from "./assistant-resize-handle";
import { ASSISTANT_SURFACE_ID } from "./assistant-surface";
import { AssistantPanel } from "./assistant-panel";
import { useLiveReplyCount } from "./use-active-turns";

const OPEN_PARAM = "assistant";

/**
 * The assistant lives in a corner of every page, because a question about a
 * shipment comes up while looking at the shipment. It is mounted once in the
 * shell; the store remembers whether it was expanded, which corner it sits
 * in, how big the panel is, and whether the launcher is tucked away, because
 * whatever it covers on one page it covers on every page.
 */
export function AssistantWidget() {
  const t = useT();
  const reduceMotion = useReducedMotion();
  const { allowed } = usePermission(Resource.Assistant, Operation.Read);
  const { allowed: canSeeProposals } = usePermission(Resource.AgentProposal, Operation.Read);

  const open = useAssistantStore((state) => state.open);
  const expanded = useAssistantStore((state) => state.expanded);
  const openWidget = useAssistantStore((state) => state.openWidget);
  const closeWidget = useAssistantStore((state) => state.closeWidget);
  const toggleWidget = useAssistantStore((state) => state.toggleWidget);
  const toggleExpanded = useAssistantStore((state) => state.toggleExpanded);
  const setExpanded = useAssistantStore((state) => state.setExpanded);
  const dock = useAssistantStore((state) => state.dock);
  const setDock = useAssistantStore((state) => state.setDock);
  const launcherHidden = useAssistantStore((state) => state.launcherHidden);
  const setLauncherHidden = useAssistantStore((state) => state.setLauncherHidden);
  const savedSize = useAssistantStore((state) => state.panelSize);
  const setPanelSize = useAssistantStore((state) => state.setPanelSize);
  // While a resize is under way the size lives here, and is saved once when
  // it ends rather than on every pointer move.
  const [liveSize, setLiveSize] = useState<AssistantPanelSize | null>(null);
  const panelSize = liveSize ?? savedSize ?? DEFAULT_PANEL_SIZE;
  const resizing = liveSize !== null;

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

  useHotkey("Escape", () => (expanded ? setExpanded(false) : closeWidget()), {
    ignoreInputs: false,
    preventDefault: false,
    enabled: allowed && open,
  });

  // The launcher is a signal: it moves only for a decision waiting on
  // someone. The count is the same one the sidebar and the Desk show, read
  // once through the attention summary rather than polled on its own.
  const { data: attention } = useAttentionSummary();
  const pendingCount = allowed && canSeeProposals ? (attention?.agentDecisions ?? 0) : 0;
  // Replies keep being written with the panel closed; the launcher is where
  // that shows until the panel is opened again.
  const writingCount = useLiveReplyCount();

  if (!allowed) {
    return null;
  }

  return (
    <AnimatePresence initial={false} mode="popLayout">
      {open ? (
        <Fragment key="panel">
          {/* Expanded, the panel is the only thing being used, so it says so.
              Floating it a few pixels below the app header instead left the two
              overlapping at the top of the screen, which read as a mistake
              rather than as a mode. */}
          {expanded && (
            <m.div
              key="scrim"
              aria-hidden
              initial={reduceMotion ? false : { opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
              transition={{ duration: 0.15 }}
              onClick={() => setExpanded(false)}
              className="fixed inset-0 z-40 bg-scrim backdrop-blur-[2px]"
            />
          )}
          <m.section
            role="dialog"
            aria-label={t("Assistant")}
            aria-modal={expanded}
            layout
            layoutId={ASSISTANT_SURFACE_ID}
            style={
              expanded
                ? { borderRadius: 16 }
                : { borderRadius: 16, ...panelSizeStyle(dock, panelSize) }
            }
            transition={
              reduceMotion || resizing
                ? { duration: 0 }
                : { type: "spring", stiffness: 380, damping: 34 }
            }
            className={cn(
              "group/panel bg-popover ring-foreground/10 fixed z-50 flex flex-col overflow-hidden ring-1",
              expanded
                ? "inset-x-3 inset-y-3 md:inset-x-[max(2rem,calc((100vw-1180px)/2))] md:inset-y-[max(2rem,calc((100dvh-820px)/2))]"
                : dockPositionClass(dock, "panel"),
            )}
          >
            {!expanded && (
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
            <m.div
              className="flex min-h-0 flex-1 flex-col"
              initial={reduceMotion ? false : { opacity: 0 }}
              animate={{ opacity: 1 }}
              transition={reduceMotion ? { duration: 0 } : { duration: 0.12, delay: 0.08 }}
            >
              <AssistantPanel
                expanded={expanded}
                onToggleExpanded={toggleExpanded}
                onClose={closeWidget}
              />
            </m.div>
          </m.section>
        </Fragment>
      ) : launcherHidden ? (
        <AssistantEdgeTab
          key="tab"
          dock={dock}
          pendingCount={pendingCount}
          writingCount={writingCount}
          onClick={openWidget}
        />
      ) : (
        <AssistantLauncher
          key="launcher"
          dock={dock}
          pendingCount={pendingCount}
          writingCount={writingCount}
          onClick={openWidget}
          onMove={setDock}
          onHide={() => setLauncherHidden(true)}
        />
      )}
    </AnimatePresence>
  );
}
