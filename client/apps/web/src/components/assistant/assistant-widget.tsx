import { useT } from "@trenova/shared/i18n/use-t";
import { usePermission } from "@/hooks/use-permission";
import { fetchPendingProposalCount } from "@/lib/graphql/agent-activity";
import { useAssistantStore } from "@/stores/assistant-store";
import { useQuery } from "@tanstack/react-query";
import { useHotkey } from "@tanstack/react-hotkeys";
import { cn } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { AnimatePresence, m, useReducedMotion } from "motion/react";
import { useEffect } from "react";
import { useSearchParams } from "react-router";
import { AssistantLauncher } from "./assistant-launcher";
import { ASSISTANT_SURFACE_ID } from "./assistant-surface";
import { AssistantPanel } from "./assistant-panel";

const OPEN_PARAM = "assistant";

/**
 * The assistant lives in the bottom-right corner of every page, because a
 * question about a shipment comes up while looking at the shipment. It is
 * mounted once in the shell; the store remembers whether it was expanded.
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

  const pendingQuery = useQuery({
    queryKey: ["assistant", "pending-proposals"],
    queryFn: ({ signal }) => fetchPendingProposalCount({ signal }),
    enabled: allowed && canSeeProposals,
    refetchInterval: 60_000,
  });

  if (!allowed) {
    return null;
  }

  return (
    <AnimatePresence initial={false} mode="popLayout">
      {open ? (
        <m.section
          key="panel"
          role="dialog"
          aria-label={t("Assistant")}
          aria-modal={expanded}
          layout
          layoutId={ASSISTANT_SURFACE_ID}
          style={{ borderRadius: 16 }}
          transition={
            reduceMotion ? { duration: 0 } : { type: "spring", stiffness: 380, damping: 34 }
          }
          className={cn(
            "bg-popover ring-foreground/10 fixed z-50 flex flex-col overflow-hidden shadow-xl shadow-black/15 ring-1 backdrop-blur-sm",
            expanded
              ? "inset-4 md:inset-x-[max(1rem,calc((100vw-1100px)/2))] md:inset-y-4"
              : "right-4 bottom-4 h-[min(600px,calc(100dvh-2rem))] w-[min(400px,calc(100vw-2rem))]",
          )}
        >
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
      ) : (
        <AssistantLauncher
          key="launcher"
          pendingCount={pendingQuery.data ?? 0}
          onClick={openWidget}
        />
      )}
    </AnimatePresence>
  );
}
