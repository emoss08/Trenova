import { useT } from "@trenova/shared/i18n/use-t";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Kbd } from "@trenova/shared/components/ui/kbd";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { formatSecondsAgo } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import type { AgentDefinitionRow } from "@/lib/graphql/agent-definition";
import type { AssistantThread } from "@/types/assistant";
import {
  CheckIcon,
  ChevronDownIcon,
  HistoryIcon,
  Maximize2Icon,
  Minimize2Icon,
  PlusIcon,
  Trash2Icon,
  XIcon,
} from "lucide-react";
import { useState } from "react";
import { AgentTile } from "@/components/agent-identity/agent-tile";

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
  onToggleExpanded: () => void;
  onClose: () => void;
};

const nowInSeconds = () => Math.floor(Date.now() / 1000);

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
  onToggleExpanded,
  onClose,
}: AssistantHeaderProps) {
  const t = useT();
  const [agentMenuOpen, setAgentMenuOpen] = useState(false);
  const [historyOpen, setHistoryOpen] = useState(false);
  const [now] = useState(nowInSeconds);

  const title = activeAgent?.name ?? t("Assistant");
  const subtitle = activeThread
    ? activeThread.title || t("Untitled conversation")
    : t("Ask about anything you can see in Trenova");

  return (
    <div className="border-border/70 flex items-center gap-2 border-b px-3 py-2">
      <Popover open={agentMenuOpen} onOpenChange={setAgentMenuOpen}>
        <PopoverTrigger
          render={
            <button
              type="button"
              className="hover:bg-muted/60 flex min-w-0 flex-1 items-center gap-2 rounded-md px-1 py-1 text-left transition-colors"
              aria-label={t("Choose an agent")}
              disabled={agents.length === 0}
            />
          }
        >
          <AgentTile agent={activeAgent} size="lg" />
          <span className="flex min-w-0 flex-1 flex-col leading-tight">
            <span className="flex items-center gap-1 text-sm font-semibold">
              <span className="truncate">{title}</span>
              {agents.length > 1 && (
                <ChevronDownIcon className="text-muted-foreground size-3.5 shrink-0" />
              )}
            </span>
            <span className="text-muted-foreground truncate text-xs">{subtitle}</span>
          </span>
        </PopoverTrigger>
        <PopoverContent align="start" className="w-80 p-1.5">
          <p className="text-muted-foreground px-2 py-1 text-xs font-medium">
            {t("Start a conversation with")}
          </p>
          <ScrollArea className="max-h-72">
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
                  className="hover:bg-muted flex items-start gap-2.5 rounded-md px-2 py-1.5 text-left transition-colors disabled:opacity-60"
                >
                  <AgentTile agent={agent} size="md" className="mt-0.5" />
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
                      <span className="text-muted-foreground line-clamp-2 text-xs">
                        {agent.description}
                      </span>
                    )}
                  </span>
                  {activeAgent?.id === agent.id && (
                    <CheckIcon className="text-primary mt-1 size-4 shrink-0" />
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
            <PopoverContent align="end" className="w-80 p-1.5">
              <p className="text-muted-foreground px-2 py-1 text-xs font-medium">
                {t("Recent conversations")}
              </p>
              {threads.length === 0 ? (
                <p className="text-muted-foreground px-2 py-4 text-center text-xs">
                  {t("No conversations yet.")}
                </p>
              ) : (
                <ScrollArea className="max-h-72">
                  <div className="flex flex-col gap-0.5">
                    {threads.map((thread) => {
                      const touched =
                        thread.lastMessageAt > 0 ? thread.lastMessageAt : thread.createdAt;
                      const active = thread.id === activeThread?.id;
                      return (
                        <div
                          key={thread.id}
                          className={cn(
                            "group flex items-center gap-1 rounded-md px-2 py-1.5 transition-colors",
                            active ? "bg-muted" : "hover:bg-muted/60",
                          )}
                        >
                          <button
                            type="button"
                            className="min-w-0 flex-1 text-left"
                            onClick={() => {
                              setHistoryOpen(false);
                              onSelectThread(thread.id);
                            }}
                          >
                            <span className="block truncate text-sm">
                              {thread.title || t("Untitled conversation")}
                            </span>
                            <span className="text-muted-foreground block text-xs">
                              {formatSecondsAgo(now - touched)}
                            </span>
                          </button>
                          <Button
                            variant="ghost"
                            size="icon-xxs"
                            aria-label={t("Delete conversation")}
                            className="text-muted-foreground hover:text-destructive opacity-0 group-hover:opacity-100 focus-visible:opacity-100"
                            onClick={() => {
                              setHistoryOpen(false);
                              onDeleteThread(thread);
                            }}
                          >
                            <Trash2Icon className="size-3.5" />
                          </Button>
                        </div>
                      );
                    })}
                  </div>
                </ScrollArea>
              )}
            </PopoverContent>
          </Popover>
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
