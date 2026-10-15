import { fillCommand, SLASH_COMMANDS } from "@/components/assistant/composer-commands";
import { useAskableAgent } from "@/components/assistant/use-askable-agent";
import { usePermission } from "@/hooks/use-permission";
import type { AgentChoice, AgentRecommendation } from "@/lib/graphql/agent-definition";
import { assistant } from "@/lib/queries/assistant";
import type { AssistantEntityRef, AssistantThread } from "@/types/assistant";
import { useQuery } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useState, type ReactNode } from "react";
import { Link } from "react-router";
import { agentToAsk, offersRecommendation } from "./agent-recommendation-state";
import { DeskAgentRecommendation } from "./composer/desk-agent-recommendation";
import { useDeskAttachments, useHeldAttachments } from "./composer/desk-attachments";
import { DeskComposer } from "./composer/desk-composer";
import { DeskModelPicker } from "./composer/desk-model-picker";
import { DeskPageChip, useDeskPage } from "./composer/desk-page-chip";
import { useDeskDrop } from "./composer/desk-uploads";
import { DeskTermsNote } from "./desk-terms-note";
import { useAgentRecommendation } from "./use-agent-recommendation";
import "./desk-frame.css";
import "./desk-chat.css";
import "./desk-chat-compact.css";

/** What a question asked before its conversation exists carries to the conversation it starts. */
export type DeskStartExtras = {
  files: File[];
  mentions: AssistantEntityRef[];
  providerId: string;
};

export type DeskHomeAskOptions = {
  agents: readonly AgentChoice[];
  threads: readonly AssistantThread[];
  isLoading: boolean;
  /** The agent to offer first, when the person chose one in their settings. */
  preferId?: string;
  /** The page the question carries: the one the person came from, or the one on screen. */
  pageSource?: "recent" | "screen";
  onStart: (agentId: string, question?: string, extras?: DeskStartExtras) => void;
};

/**
 * Everything a question asked before its conversation exists is made of: the
 * agent it goes to, the words, the records named, the files held until the
 * conversation can take them, the page, and the model. The Desk's front page
 * and the assistant's home ask the same way, so both hold it here; the files
 * dragged over either surface land in the same list.
 *
 * As the question is typed it is matched to the person's agents by the tools
 * each holds that the person may use, and an agent that fits better than the
 * one chosen is offered at the top of the box. Until the person chooses, from the
 * picker or from that offer, a question sent goes to the agent that fits it,
 * waiting a moment for the match when it was sent before one came back. One
 * of the chosen agent's own starter questions always goes to that agent.
 */
export function useDeskHomeAsk({
  agents,
  threads,
  isLoading,
  preferId,
  pageSource = "recent",
  onStart,
}: DeskHomeAskOptions) {
  const askable = useAskableAgent({ threads, preferId });
  const noAgents = !isLoading && (agents.length === 0 || askable.noneAvailable);
  const [draft, setDraft] = useState("");
  const [mentions, setMentions] = useState<AssistantEntityRef[]>([]);
  const [providerId, setProviderId] = useState("");
  const providersQuery = useQuery(assistant.providers());
  const held = useHeldAttachments();
  const attachments = useDeskAttachments(held);
  const deskPage = useDeskPage({ onScreen: pageSource === "screen" });
  const drag = useDeskDrop(noAgents ? null : attachments.add);
  const agentId = askable.agent?.id ?? null;
  const advice = useAgentRecommendation({
    draft,
    agentId,
    enabled: !noAgents && agentId !== null,
  });
  // The person has said where this question goes: a pick from the picker, or
  // either answer to the offer. Until then a question goes where it fits.
  const [chosen, setChosen] = useState(false);
  const [keptOverId, setKeptOverId] = useState<string | null>(null);
  const [routing, setRouting] = useState(false);
  const offered: AgentRecommendation | null = offersRecommendation(
    advice.recommendation,
    agentId,
    keptOverId,
  )
    ? advice.recommendation
    : null;

  const choose = (agent: AgentChoice) => {
    askable.choose(agent);
    setChosen(true);
  };

  const acceptRecommendation = () => {
    if (offered) {
      choose(offered.agent);
    }
  };

  const keepAgent = () => {
    if (offered) {
      setKeptOverId(offered.agent.id);
      setChosen(true);
    }
  };

  const send = async (content: string, payloadMentions: AssistantEntityRef[]) => {
    const agent = askable.agent;
    if (!agent || routing) {
      return;
    }
    const files = held.files();
    const starter = agent.starters?.some((item) => item.prompt === content.trim()) ?? false;
    let target = agent;
    if (!chosen && !starter) {
      setRouting(true);
      const recommendation = await advice.resolve(content);
      setRouting(false);
      target = agentToAsk({ agent, recommendation, chosen });
    }
    onStart(target.id, content, {
      files,
      mentions: payloadMentions,
      providerId,
    });
    held.clear();
    attachments.clear();
  };

  return {
    askable,
    noAgents,
    choose,
    offered,
    routing,
    acceptRecommendation,
    keepAgent,
    draft,
    setDraft,
    mentions,
    setMentions,
    providerId,
    setProviderId,
    providers: providersQuery.data ?? [],
    attachments,
    deskPage,
    drag,
    send,
  };
}

export type DeskHomeAsk = ReturnType<typeof useDeskHomeAsk>;

/**
 * The box a conversation starts from: the terms note on first use, then the
 * composer with its slow ring and the agent's starter questions typed out in
 * the placeholder. Tab takes the one on screen and ⌘1–9 asks it outright.
 * With no agent to ask, it says so, and where to turn one on.
 */
export function DeskHomeComposer({ ask, isStarting }: { ask: DeskHomeAsk; isStarting: boolean }) {
  const t = useT();
  const { allowed: canManageAgents } = usePermission(Resource.AgentDefinition, Operation.Read);
  const explain = SLASH_COMMANDS.find((command) => command.name === "explain");
  const { askable, deskPage } = ask;

  if (ask.noAgents) {
    return (
      <div className="dk-ec-offmsg dk-home-none">
        <span>
          <b>{t("No agents are available.")}</b>{" "}
          {canManageAgents
            ? t("Connect an AI provider and enable an agent in AI Control.")
            : t("An administrator needs to connect an AI provider and enable an agent first.")}
        </span>
        {canManageAgents && (
          <Link className="dk-ec-link" to="/admin/agent-control">
            {t("Open AI Control")}
          </Link>
        )}
      </div>
    );
  }

  return (
    <>
      <DeskTermsNote />
      <DeskComposer
        home
        value={ask.draft}
        onChange={ask.setDraft}
        agent={askable.agent}
        onAgentChange={ask.choose}
        recentAgentIds={askable.recency.ids}
        agentLastUsedAt={askable.recency.lastUsedAt}
        busy={false}
        disabled={isStarting || ask.routing || askable.agent === null}
        placeholder={ask.routing ? t("Finding the agent with the tools for this…") : undefined}
        presets={askable.agent?.starters?.map((starter) => starter.prompt) ?? []}
        suggestions={askable.agent?.starters?.map((starter) => ({
          label: starter.label,
          prompt: starter.prompt,
        }))}
        attachments={ask.attachments}
        mentions={ask.mentions}
        onMentionsChange={ask.setMentions}
        drag={ask.drag}
        extras={
          <DeskPageChip
            page={deskPage.page}
            share={deskPage.share}
            onShareChange={deskPage.setShare}
            onExplain={explain ? () => ask.setDraft(fillCommand(explain, [])) : undefined}
            onScreen={deskPage.onScreen}
          />
        }
        model={
          <DeskModelPicker
            options={ask.providers}
            value={ask.providerId}
            onChange={ask.setProviderId}
            hasReplies={false}
          />
        }
        onSend={(content, payload) => {
          void ask.send(content, payload.mentions);
        }}
        notice={
          ask.offered && askable.agent ? (
            <DeskAgentRecommendation
              recommendation={ask.offered}
              agent={askable.agent}
              onUse={ask.acceptRecommendation}
              onKeep={ask.keepAgent}
            />
          ) : undefined
        }
      />
    </>
  );
}

/**
 * The block a front page opens with: the date in mono, a heading and one line
 * under it. The Desk greets the person by the time of day; the assistant says
 * what it can do. `pulse` breathes a dot beside the date.
 */
export function DeskGreeting({
  date,
  title,
  line,
  pending = false,
  pulse = false,
  className,
}: {
  date: string;
  title: string;
  line: ReactNode;
  /** The line is still being worked out; a skeleton holds its place. */
  pending?: boolean;
  pulse?: boolean;
  className?: string;
}) {
  return (
    <div className={cn("dk-greeting", className)}>
      <div className="dk-date">
        {pulse && <span className="dk-date-pulse" aria-hidden />}
        {date}
      </div>
      <h1 className="dk-greet">{title}</h1>
      <p className="dk-headline" data-slot="desk-headline">
        {pending ? <span className="dk-headline-sk ui-shimmer" data-slot="skeleton" /> : line}
      </p>
    </div>
  );
}
