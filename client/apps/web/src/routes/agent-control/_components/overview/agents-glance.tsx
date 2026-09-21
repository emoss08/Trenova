import { AgentTile } from "@/components/agent-identity/agent-tile";
import { SectionPanel, SectionPanelQuiet } from "@/components/section-panel";
import { queries } from "@/lib/queries";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatSecondsAgo } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { useQuery } from "@tanstack/react-query";
import { ArrowRightIcon } from "lucide-react";
import { useState } from "react";
import type { ActivityView } from "../rail-items";
import { groupAgentsByTrigger } from "../agents/agent-roster";
import { TRIGGER_LABELS } from "../agents/trigger-meta";

const nowInSeconds = () => Math.floor(Date.now() / 1000);

type AgentsGlanceProps = {
  onOpenAgents: () => void;
  onOpenActivity: (view: ActivityView) => void;
};

/**
 * Every agent on one panel: whether it is on, what starts it, when it last
 * ran and whether it has something waiting on a person. Enough to see the
 * shape of the organization's automation without opening the roster.
 */
export function AgentsGlance({ onOpenAgents, onOpenActivity }: AgentsGlanceProps) {
  const t = useT();
  const [now] = useState(nowInSeconds);
  const listQuery = useQuery(queries.assistant.agents(false));
  const agents = listQuery.data ?? [];
  const shelves = groupAgentsByTrigger(agents);

  return (
    <SectionPanel
      title={t("Agents")}
      count={agents.length}
      action={
        <Button variant="ghost" size="xs" onClick={onOpenAgents}>
          {t("Open roster")}
          <ArrowRightIcon className="size-3" />
        </Button>
      }
    >
      {listQuery.isLoading ? (
        <div className="flex flex-col gap-2 p-3">
          <Skeleton className="h-8" />
          <Skeleton className="h-8" />
          <Skeleton className="h-8" />
        </div>
      ) : agents.length === 0 ? (
        <SectionPanelQuiet>{t("No agents yet. Build one from the roster.")}</SectionPanelQuiet>
      ) : (
        <ul className="divide-border divide-y">
          {shelves.flatMap((shelf) =>
            shelf.agents.map((agent) => {
              const touched = agent.lastRunAt ?? 0;
              return (
                <li key={agent.id} className="flex items-center gap-3 px-3 py-2">
                  <AgentTile
                    agent={agent}
                    size="sm"
                    className={cn(!agent.enabled && "opacity-60 grayscale")}
                  />
                  <span className="flex min-w-0 flex-1 flex-col">
                    <span
                      className={cn("truncate text-sm", !agent.enabled && "text-muted-foreground")}
                    >
                      {agent.name}
                    </span>
                    <span className="text-muted-foreground truncate text-xs">
                      {t(TRIGGER_LABELS[shelf.trigger])}
                      {touched > 0 ? ` · ${t("ran {0}", formatSecondsAgo(now - touched))}` : ""}
                    </span>
                  </span>
                  {agent.pendingProposals > 0 ? (
                    <button
                      type="button"
                      onClick={() => onOpenActivity("proposals")}
                      className="ui-focus-ring rounded-full"
                    >
                      <Badge variant="warning">
                        {t(
                          "{0, plural, one {# awaiting} other {# awaiting}}",
                          agent.pendingProposals,
                        )}
                      </Badge>
                    </button>
                  ) : agent.shadowMode ? (
                    <Badge variant="neutral" appearance="outline">
                      {t("Shadow")}
                    </Badge>
                  ) : !agent.enabled ? (
                    <Badge variant="neutral" appearance="outline">
                      {t("Off")}
                    </Badge>
                  ) : null}
                </li>
              );
            }),
          )}
        </ul>
      )}
    </SectionPanel>
  );
}
