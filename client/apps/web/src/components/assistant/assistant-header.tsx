import { AgentTile } from "@/components/agent-identity/agent-tile";
import type { AgentDefinitionRow } from "@/lib/graphql/agent-definition";
import type { AssistantThread } from "@/types/assistant";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Kbd } from "@trenova/shared/components/ui/kbd";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useT } from "@trenova/shared/i18n/use-t";
import {
  CheckIcon,
  ChevronDownIcon,
  DownloadIcon,
  HistoryIcon,
  Maximize2Icon,
  Minimize2Icon,
  PlusIcon,
  XIcon,
} from "lucide-react";
import { AnimatePresence, m, useReducedMotion } from "motion/react";
import { useMemo, useState } from "react";
import { groupThreadsByRecency } from "./thread-grouping";
import { ThreadList } from "./thread-sidebar";

type AssistantHeaderProps = {
  agents: AgentDefinitionRow[];
  activeAgent: AgentDefinitionRow | null;
  activeThread: AssistantThread | null;
  threads: AssistantThread[];
  expanded: boolean;
  isStarting: boolean;
  onStart: (agentId: string) => void;
  onSelectThread: (id: string) => void;
  onDeleteThread: (thread: AssistantThread) => void;
  /** Saves the open conversation as a file. Offered only while one is open. */
  onDownloadTranscript: (thread: AssistantThread) => void;
  onToggleExpanded: () => void;
  onClose: () => void;
};

const nowInSeconds = () => Math.floor(Date.now() / 1000);

/**
 * One slim bar: who is being talked to on the left, the panel's controls on
 * the right.
 *
 * The conversation's title used to sit here too. It is the first question
 * asked, which is already the first line of the thread below, and a long one
 * pushed the controls off the edge. The history list and the sidebar are
 * where titles belong.
 */
export function AssistantHeader({
  agents,
  activeAgent,
  activeThread,
  threads,
  expanded,
  isStarting,
  onStart,
  onSelectThread,
  onDeleteThread,
  onDownloadTranscript,
  onToggleExpanded,
  onClose,
}: AssistantHeaderProps) {
  const t = useT();
  const reduceMotion = useReducedMotion();
  const [agentMenuOpen, setAgentMenuOpen] = useState(false);
  const [historyOpen, setHistoryOpen] = useState(false);
  const [now] = useState(nowInSeconds);

  const agentsById = useMemo(() => new Map(agents.map((agent) => [agent.id, agent])), [agents]);
  const groups = useMemo(() => groupThreadsByRecency(threads, now), [now, threads]);

  return (
    <div className="border-border flex h-11 shrink-0 items-center justify-between gap-2 border-b pr-1.5 pl-2">
      <Popover open={agentMenuOpen} onOpenChange={setAgentMenuOpen}>
        <PopoverTrigger
          render={
            <button
              type="button"
              className="hover:bg-surface-hover ui-focus-ring flex h-8 min-w-0 max-w-full items-center gap-2 rounded-md px-1.5 text-left transition-colors"
              aria-label={t("Choose an agent")}
              disabled={agents.length === 0}
            />
          }
        >
          {/* The tile crossfades when the agent changes: the one thing in the
              bar that answers a choice. */}
          <AnimatePresence mode="popLayout" initial={false}>
            <m.span
              key={activeAgent?.id ?? "none"}
              initial={reduceMotion ? false : { opacity: 0, scale: 0.85 }}
              animate={{ opacity: 1, scale: 1 }}
              exit={{ opacity: 0, scale: 0.85 }}
              transition={{ duration: 0.14 }}
              className="flex shrink-0"
            >
              <AgentTile agent={activeAgent} size="md" />
            </m.span>
          </AnimatePresence>
          <span className="truncate text-sm font-semibold">
            {activeAgent?.name ?? t("Assistant")}
          </span>
          {agents.length > 1 && (
            <ChevronDownIcon className="text-muted-foreground size-3.5 shrink-0" />
          )}
        </PopoverTrigger>
        <PopoverContent align="start" className="w-80 p-1.5">
          <p className="text-muted-foreground px-2 py-1 text-xs font-medium">
            {t("Start a conversation with")}
          </p>
          <ScrollArea viewportClassName="max-h-72">
            <div className="flex flex-col gap-0.5">
              {agents.map((agent) => (
                <button
                  key={agent.id}
                  type="button"
                  disabled={isStarting}
                  onClick={() => {
                    setAgentMenuOpen(false);
                    onStart(agent.id);
                  }}
                  className="hover:bg-surface-hover ui-focus-ring flex items-start gap-2.5 rounded-md px-2 py-1.5 text-left transition-colors disabled:opacity-60"
                >
                  <AgentTile agent={agent} size="md" className="mt-0.5" />
                  <span className="flex min-w-0 flex-1 flex-col">
                    <span className="flex items-center gap-1.5 text-sm font-medium">
                      <span className="truncate">{agent.name}</span>
                      <Badge variant="neutral" className="h-4 px-1 text-2xs">
                        {agent.toolNames.length === 0
                          ? t("Answers only")
                          : t("{0, plural, one {# tool} other {# tools}}", agent.toolNames.length)}
                      </Badge>
                    </span>
                    {agent.description && (
                      <span className="text-muted-foreground line-clamp-2 text-xs">
                        {agent.description}
                      </span>
                    )}
                  </span>
                  {activeAgent?.id === agent.id && (
                    <CheckIcon className="text-foreground mt-1 size-4 shrink-0" />
                  )}
                </button>
              ))}
            </div>
          </ScrollArea>
        </PopoverContent>
      </Popover>

      <div className="flex shrink-0 items-center gap-0.5">
        {!expanded && (
          <Popover open={historyOpen} onOpenChange={setHistoryOpen}>
            <Tooltip>
              <TooltipTrigger
                render={
                  <PopoverTrigger
                    render={
                      <Button
                        variant="ghost"
                        size="icon-sm"
                        aria-label={t("Conversations")}
                        className="text-muted-foreground hover:text-foreground"
                      />
                    }
                  />
                }
              >
                <HistoryIcon className="size-4" />
              </TooltipTrigger>
              <TooltipContent>{t("Conversations")}</TooltipContent>
            </Tooltip>
            <PopoverContent align="end" className="w-80 p-0">
              <ScrollArea viewportClassName="max-h-80">
                <ThreadList
                  groups={groups}
                  agentsById={agentsById}
                  activeThreadId={activeThread?.id ?? null}
                  now={now}
                  emptyText={t("No conversations yet.")}
                  onSelect={(id) => {
                    setHistoryOpen(false);
                    onSelectThread(id);
                  }}
                  onDelete={(thread) => {
                    setHistoryOpen(false);
                    onDeleteThread(thread);
                  }}
                />
              </ScrollArea>
            </PopoverContent>
          </Popover>
        )}

        {activeThread && (
          <Tooltip>
            <TooltipTrigger
              render={
                <Button
                  variant="ghost"
                  size="icon-sm"
                  aria-label={t("Download transcript")}
                  className="text-muted-foreground hover:text-foreground"
                  onClick={() => onDownloadTranscript(activeThread)}
                />
              }
            >
              <DownloadIcon className="size-4" />
            </TooltipTrigger>
            <TooltipContent>{t("Download transcript")}</TooltipContent>
          </Tooltip>
        )}

        <Tooltip>
          <TooltipTrigger
            render={
              <Button
                variant="ghost"
                size="icon-sm"
                aria-label={t("New conversation")}
                className="text-muted-foreground hover:text-foreground"
                disabled={agents.length === 0 || isStarting}
                onClick={() => {
                  if (activeAgent) {
                    onStart(activeAgent.id);
                  } else if (agents.length === 1) {
                    onStart(agents[0].id);
                  } else {
                    setAgentMenuOpen(true);
                  }
                }}
              />
            }
          >
            <PlusIcon className="size-4" />
          </TooltipTrigger>
          <TooltipContent>{t("New conversation")}</TooltipContent>
        </Tooltip>

        <Tooltip>
          <TooltipTrigger
            render={
              <Button
                variant="ghost"
                size="icon-sm"
                aria-label={expanded ? t("Collapse") : t("Expand")}
                className="text-muted-foreground hover:text-foreground"
                onClick={onToggleExpanded}
              />
            }
          >
            {expanded ? <Minimize2Icon className="size-4" /> : <Maximize2Icon className="size-4" />}
          </TooltipTrigger>
          <TooltipContent>{expanded ? t("Collapse") : t("Expand")}</TooltipContent>
        </Tooltip>

        <Tooltip>
          <TooltipTrigger
            render={
              <Button
                variant="ghost"
                size="icon-sm"
                aria-label={t("Close")}
                className="text-muted-foreground hover:text-foreground"
                onClick={onClose}
              />
            }
          >
            <XIcon className="size-4" />
          </TooltipTrigger>
          <TooltipContent className="flex items-center gap-2">
            {t("Close")} <Kbd>Esc</Kbd>
          </TooltipContent>
        </Tooltip>
      </div>
    </div>
  );
}
