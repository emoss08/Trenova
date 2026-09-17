import { useT } from "@trenova/shared/i18n/use-t";
import { usePermission } from "@/hooks/use-permission";
import { fetchPendingProposalCount } from "@/lib/graphql/agent-activity";
import { useAssistantStore } from "@/stores/assistant-store";
import { useQuery } from "@tanstack/react-query";
import { useHotkey } from "@tanstack/react-hotkeys";
import { cn } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { AnimatePresence, m } from "motion/react";
import { useEffect } from "react";
import { useSearchParams } from "react-router";
import { AssistantLauncher } from "./assistant-launcher";
import { AssistantPanel } from "./assistant-panel";

const OPEN_PARAM = "assistant";

/**
 * The assistant lives in the bottom-right corner of every page, because a
 * question about a shipment comes up while looking at the shipment. It is
 * mounted once in the shell; the store remembers whether it was expanded.
 */
export function AssistantWidget() {
  const t = useT();
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
    <>
      <AnimatePresence>
        {!open && (
          <AssistantLauncher
            key="launcher"
            pendingCount={pendingQuery.data ?? 0}
            onClick={openWidget}
          />
        )}
      </AnimatePresence>

      <AnimatePresence>
        {open && (
          <m.section
            key="panel"
            role="dialog"
            aria-label={t("Assistant")}
            aria-modal={expanded}
            layout
            initial={{ opacity: 0, y: 16, scale: 0.96 }}
            animate={{ opacity: 1, y: 0, scale: 1 }}
            exit={{ opacity: 0, y: 16, scale: 0.96 }}
            transition={{ type: "spring", stiffness: 380, damping: 32 }}
            className={cn(
              "bg-background/95 border-border fixed z-50 flex flex-col overflow-hidden rounded-2xl border shadow-2xl shadow-black/25 backdrop-blur-sm",
              expanded
                ? "inset-4 md:inset-x-[max(1rem,calc((100vw-1100px)/2))] md:inset-y-4"
                : "right-4 bottom-4 h-[min(600px,calc(100dvh-2rem))] w-[min(400px,calc(100vw-2rem))]",
            )}
          >
            <AssistantPanel
              expanded={expanded}
              onToggleExpanded={toggleExpanded}
              onClose={closeWidget}
            />
          </m.section>
        )}
      </AnimatePresence>
    </>
  );
}
