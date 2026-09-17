import { useT } from "@trenova/shared/i18n/use-t";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatSecondsAgo } from "@trenova/shared/lib/date";
import { useAssistantStore } from "@/stores/assistant-store";
import type { AgentDefinitionRow } from "@/lib/graphql/agent-definition";
import type { AssistantThread } from "@/types/assistant";
import { ArrowRightIcon, BotIcon, ChevronRightIcon, PlugZapIcon } from "lucide-react";
import { m, useReducedMotion } from "motion/react";
import { useState } from "react";
import { Link } from "react-router";
import { AgentTile } from "@/components/agent-identity/agent-tile";
import { TrenovaSpark } from "./trenova-spark";

type AssistantHomeProps = {
  agents: AgentDefinitionRow[];
  threads: AssistantThread[];
  isLoading: boolean;
  isStarting: boolean;
  canManageAgents: boolean;
  onStart: (agentId: string) => void;
  onSelectThread: (id: string) => void;
};

const nowInSeconds = () => Math.floor(Date.now() / 1000);
const RECENT_LIMIT = 4;

/**
 * What the panel shows before a conversation is picked: who can be asked,
 * and what was asked recently. It is the launch pad, not a blank chat.
 */
export function AssistantHome({
  agents,
  threads,
  isLoading,
  isStarting,
  canManageAgents,
  onStart,
  onSelectThread,
}: AssistantHomeProps) {
  const t = useT();
  const closeWidget = useAssistantStore((state) => state.closeWidget);
  const reduceMotion = useReducedMotion();
  const [now] = useState(nowInSeconds);

  if (!isLoading && agents.length === 0) {
    return (
      <div className="flex flex-1 flex-col items-center justify-center gap-4 px-6 py-10 text-center">
        <span className="bg-muted text-muted-foreground flex size-12 items-center justify-center rounded-2xl">
          <BotIcon className="size-6" />
        </span>
        <div className="flex flex-col gap-1">
          <p className="text-sm font-semibold">{t("No agents are available")}</p>
          <p className="text-muted-foreground text-xs">
            {canManageAgents
              ? t("Connect an AI provider and enable an agent in AI Control, then come back here.")
              : t("An administrator needs to connect an AI provider and enable an agent first.")}
          </p>
        </div>
        {canManageAgents && (
          <Button
            size="sm"
            variant="outline"
            nativeButton={false}
            render={<Link to="/admin/agent-control" onClick={closeWidget} />}
          >
            <PlugZapIcon className="size-3.5" />
            {t("Open AI Control")}
          </Button>
        )}
      </div>
    );
  }

  const recent = threads.slice(0, RECENT_LIMIT);

  return (
    <ScrollArea className="flex-1" maskVariant="card">
      <div className="flex flex-col gap-5 px-4 py-4">
        {/* No corner flourishes and no blurred wash behind the text. This is
            the first thing a person sees every time they open the panel, and
            decoration that cannot be read is decoration they cannot skip. */}
        <m.div
          initial={reduceMotion ? false : { opacity: 0, y: 6 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.25 }}
          className="flex flex-col gap-2 px-1 pt-1"
        >
          <TrenovaSpark className="text-foreground size-6" />
          <p className="text-sm font-semibold">{t("Ask about anything you can see")}</p>
          <p className="text-muted-foreground text-xs leading-relaxed">
            {t(
              "Where a shipment is, who is free, what is holding an invoice, how to do something.",
            )}
          </p>
          <p className="text-muted-foreground text-xs leading-relaxed">
            {t("Reads only what you can already see. Changes wait for your approval.")}
          </p>
        </m.div>

        <section className="flex flex-col gap-2">
          <h3 className="text-muted-foreground text-xs font-medium">{t("Agents")}</h3>
          {isLoading ? (
            <div className="flex flex-col gap-2">
              <Skeleton className="h-14" />
              <Skeleton className="h-14" />
            </div>
          ) : (
            <div className="flex flex-col gap-1.5">
              {agents.map((agent, index) => (
                <m.button
                  key={agent.id}
                  type="button"
                  disabled={isStarting}
                  onClick={() => onStart(agent.id)}
                  initial={{ opacity: 0, y: 6 }}
                  animate={{ opacity: 1, y: 0 }}
                  transition={{ duration: 0.2, delay: 0.04 * index }}
                  className="border-border/70 bg-card hover:border-border hover:bg-muted/40 hover:bg-muted/40 focus-visible:ring-ring/50 group flex items-center gap-3 rounded-lg border px-3 py-2.5 text-left transition-colors outline-none focus-visible:ring-[3px] disabled:opacity-60"
                >
                  <AgentTile agent={agent} size="lg" />
                  <span className="flex min-w-0 flex-1 flex-col">
                    <span className="flex items-center gap-1.5 text-sm font-medium">
                      <span className="truncate">{agent.name}</span>
                      <Badge variant="secondary" className="h-4 px-1 text-[10px]">
                        {agent.toolNames.length === 0
                          ? t("Answers only")
                          : t("{0, plural, one {# tool} other {# tools}}", agent.toolNames.length)}
                      </Badge>
                    </span>
                    {agent.description && (
                      <span className="text-muted-foreground line-clamp-1 text-xs">
                        {agent.description}
                      </span>
                    )}
                  </span>
                  <ArrowRightIcon className="text-muted-foreground size-4 shrink-0 transition-transform group-hover:translate-x-0.5" />
                </m.button>
              ))}
            </div>
          )}
        </section>

        {recent.length > 0 && (
          <section className="flex flex-col gap-2">
            <h3 className="text-muted-foreground text-xs font-medium">
              {t("Pick up where you left off")}
            </h3>
            <div className="flex flex-col gap-0.5">
              {recent.map((thread) => {
                const touched = thread.lastMessageAt > 0 ? thread.lastMessageAt : thread.createdAt;
                return (
                  <button
                    key={thread.id}
                    type="button"
                    onClick={() => onSelectThread(thread.id)}
                    className="hover:bg-muted/60 flex items-center gap-2 rounded-md px-2 py-1.5 text-left transition-colors"
                  >
                    <span className="min-w-0 flex-1">
                      <span className="block truncate text-sm">
                        {thread.title || t("Untitled conversation")}
                      </span>
                      <span className="text-muted-foreground block text-xs">
                        {formatSecondsAgo(now - touched)}
                      </span>
                    </span>
                    <ChevronRightIcon className="text-muted-foreground size-4 shrink-0" />
                  </button>
                );
              })}
            </div>
          </section>
        )}
      </div>
    </ScrollArea>
  );
}
