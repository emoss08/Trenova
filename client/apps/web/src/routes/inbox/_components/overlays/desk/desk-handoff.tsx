import { AgentPicker } from "@/components/assistant/agent-picker";
import { useAssistantThreads } from "@/components/assistant/use-assistant-threads";
import { DeskThread, type DeskDock } from "@/components/desk-chat/desk-thread";
import { conversationPath } from "@/lib/conversation-path";
import type { AgentChoice } from "@/lib/graphql/agent-definition";
import type { InboundMessageDetail } from "@/lib/graphql/inbox";
import { agentRecency } from "@/lib/recent-agents";
import { inbox } from "@/lib/queries/inbox";
import { useAssistantStore } from "@/stores/assistant-store";
import { useDeskHandoffStore } from "@/stores/desk-handoff-store";
import { useTriageStore } from "@/stores/triage-store";
import type { AssistantEntityRef, AssistantThread } from "@/types/assistant";
import { useQuery } from "@tanstack/react-query";
import { LinkExternal01Icon, XCloseIcon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetTitle,
} from "@trenova/shared/components/ui/sheet";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useNavigate } from "react-router";
import { USER_FORMATTERS, actionCopy } from "../../action-copy";
import { inlineErrorText } from "../../inline-error";
import { useTriageActions } from "../../use-triage-actions";
import { pickAgentFor } from "./desk-agent";

const RECENT_AGENTS = 6;

/** What the conversation is handed besides the words: the message, and the shipment it is about. */
function handedOver(message: InboundMessageDetail): AssistantEntityRef[] {
  const refs: AssistantEntityRef[] = [
    {
      type: "inbound_message",
      id: message.id,
      label: message.subject.trim() === "" ? message.fromAddress : message.subject,
    },
  ];
  if (message.matchedShipment !== null) {
    refs.push({
      type: "shipment",
      id: message.matchedShipment.id,
      label: message.matchedShipment.proNumber,
    });
  }

  return refs;
}

/**
 * "Work on it in Desk": the message and the steps still open on it, handed to
 * one of the organization's own agents in a real Desk conversation, drawn
 * beside the reader. The agent is picked for the kind of message — dispatch
 * for tenders and rate cons, billing for invoices, disputes and PODs,
 * customer service for the rest — by what each agent is called and does, and
 * can be changed, which starts the conversation over with the one chosen.
 * Once the agent has answered, "Yes, do it" closes this and starts the next
 * step in Triage. The Desk is drawn in its own voice: the agent's mark, never
 * Nova's.
 */
export function DeskHandoff({ messageId, onClose }: { messageId: string; onClose: () => void }) {
  const t = useT();
  const message = useQuery(inbox.message(messageId)).data;

  return (
    <Sheet open onOpenChange={(open) => (open ? undefined : onClose())}>
      <SheetContent
        side="right"
        showCloseButton={false}
        className="bg-overlay flex w-[min(440px,calc(100%-1rem))] flex-col gap-0 overflow-hidden p-0 data-[side=right]:top-2 data-[side=right]:right-2 data-[side=right]:bottom-2 data-[side=right]:sm:max-w-110"
      >
        <SheetDescription className="sr-only">
          {t("Hand this message to one of your agents in Desk")}
        </SheetDescription>
        {message === undefined ? (
          <>
            <SheetTitle className="sr-only">{t("Desk")}</SheetTitle>
            <div className="flex flex-col gap-3 p-4">
              <Skeleton className="h-7 w-1/2" />
              <Skeleton className="h-16 w-4/5 self-end" />
              <Skeleton className="h-20 w-full" />
            </div>
          </>
        ) : (
          <HandoffBody message={message} onClose={onClose} />
        )}
      </SheetContent>
    </Sheet>
  );
}

function HandoffBody({ message, onClose }: { message: InboundMessageDetail; onClose: () => void }) {
  const t = useT();
  const navigate = useNavigate();
  const actions = useTriageActions();
  const lastAgentId = useAssistantStore((store) => store.lastAgentId);
  const [thread, setThread] = useState<AssistantThread | null>(null);
  const [chosen, setChosen] = useState<AgentChoice | null>(null);
  const [refusal, setRefusal] = useState<string | null>(null);
  const [asked, setAsked] = useState<string | null>(null);
  const startedFor = useRef<string | null>(null);
  const data = useAssistantThreads(thread?.id ?? null);
  const { agents, threads, agentsUnavailable } = data;

  const picked = useMemo(
    () => chosen ?? pickAgentFor(agents, message.classification, lastAgentId),
    [agents, chosen, lastAgentId, message.classification],
  );
  const recency = useMemo(
    () => agentRecency(threads, { limit: RECENT_AGENTS, preferId: lastAgentId }),
    [threads, lastAgentId],
  );
  const open = message.actions.filter((action) => !action.done);
  const next = open[0] ?? null;

  const start = useCallback(
    (agent: AgentChoice) => {
      startedFor.current = agent.id;
      setRefusal(null);
      setThread(null);
      actions.startDeskConversation(agent.id, message).then(
        (started) => {
          useDeskHandoffStore.getState().setHandoff({
            threadId: started.id,
            files: [],
            mentions: handedOver(message),
            providerId: "",
          });
          setAsked(null);
          setThread(started);
        },
        (error: unknown) => {
          startedFor.current = null;
          setRefusal(inlineErrorText(t, error));
        },
      );
    },
    [actions, message, t],
  );

  useEffect(() => {
    if (picked === null || startedFor.current === picked.id) {
      return;
    }
    start(picked);
  }, [picked, start]);

  const agent =
    thread === null ? picked : (data.agentsById.get(thread.agentDefinitionId) ?? picked);

  const dock = (desk: DeskDock) => {
    if (desk.busy || asked === null) {
      return null;
    }
    if (next === null) {
      return (
        <div className="flex flex-wrap items-center gap-1.5 pb-2">
          <Button
            size="sm"
            onClick={() => {
              onClose();
              actions.handle([message.id]);
            }}
          >
            {t("Mark handled")}
          </Button>
        </div>
      );
    }
    const copy = actionCopy(t, next, message, USER_FORMATTERS);
    return (
      <div className="flex flex-wrap items-center gap-1.5 pb-2">
        <span className="text-foreground-subtle min-w-0 flex-1 truncate text-xs">
          {t("Next step: {0}", copy.title)}
        </span>
        <Button
          size="sm"
          onClick={() => {
            onClose();
            useTriageStore.getState().readerCommands.doNextStep?.();
          }}
        >
          {next.novaCanDo ? t("Yes, do it") : t("Yes, open it")}
        </Button>
        {open.length > 1 && (
          <Button
            variant="ghost"
            size="sm"
            disabled={desk.busy}
            onClick={() => desk.ask(t("Show me the plan for what's left."))}
          >
            {t("Show the plan")}
          </Button>
        )}
      </div>
    );
  };

  return (
    <>
      <header className="border-border-subtle flex shrink-0 items-center gap-1.5 border-b py-2 pr-2 pl-3.5">
        <SheetTitle className="text-base font-semibold">{t("Desk")}</SheetTitle>
        <AgentPicker
          agent={agent}
          recentIds={recency.ids}
          lastUsedAt={recency.lastUsedAt}
          disabled={agentsUnavailable}
          onSelect={(picking) => {
            if (picking.id !== agent?.id) {
              setChosen(picking);
            }
          }}
        />
        <span className="flex-1" />
        <Button
          variant="ghost"
          size="sm"
          disabled={thread === null}
          onClick={() => {
            if (thread !== null) {
              onClose();
              void navigate(conversationPath(thread.id));
            }
          }}
        >
          {t("Open in Desk")}
          <LinkExternal01Icon className="size-3" />
        </Button>
        <Button variant="ghost" size="icon-sm" aria-label={t("Close")} onClick={onClose}>
          <XCloseIcon className="size-3.5" />
        </Button>
      </header>

      {open.length > 0 && (
        <div
          aria-label={t("Handed over")}
          className="border-border-subtle flex shrink-0 flex-wrap gap-1 border-b px-3.5 py-2"
        >
          {open.map((action) => (
            <span
              key={action.id}
              className="bg-surface-hover text-foreground-muted inline-flex h-5.5 max-w-full items-center truncate rounded-md px-2 text-xs"
            >
              {actionCopy(t, action, message, USER_FORMATTERS).title}
            </span>
          ))}
        </div>
      )}

      <div className="dk-chat flex min-h-0 flex-1 flex-col">
        {refusal !== null ? (
          <div className="flex flex-col items-start gap-2 p-4">
            <p role="alert" className="text-danger text-sm">
              {refusal}
            </p>
            {picked !== null && (
              <Button variant="outline" size="sm" onClick={() => start(picked)}>
                {t("Try again")}
              </Button>
            )}
          </div>
        ) : agents.length === 0 && !data.isLoading ? (
          <p className="text-foreground-subtle p-4 text-sm">
            {t("You have no agents to ask yet. An administrator sets them up in AI control.")}
          </p>
        ) : thread === null ? (
          <div className="flex flex-col gap-3 p-4">
            <Skeleton className="h-12 w-3/4 self-end" />
            <Skeleton className="h-20 w-full" />
          </div>
        ) : (
          <DeskThread
            key={thread.id}
            thread={thread}
            agent={agent}
            agentsUnavailable={agentsUnavailable}
            density="compact"
            threads={threads}
            pageSource="none"
            surface="Desk"
            openingQuestion={asked === thread.id ? undefined : t("Help me finish this one.")}
            onOpeningQuestionSent={() => setAsked(thread.id)}
            onSwitchAgent={(agentId) => {
              const switched = agents.find((row) => row.id === agentId);
              if (switched !== undefined) {
                setChosen(switched);
              }
            }}
            onOpenInDesk={(artifactId) => {
              onClose();
              const path = conversationPath(thread.id);
              void navigate(artifactId ? `${path}?a=${encodeURIComponent(artifactId)}` : path);
            }}
            dockTop={dock}
            disclaimer={t(
              "Agents can make mistakes. Check important details before you act on them.",
            )}
          />
        )}
      </div>
    </>
  );
}
