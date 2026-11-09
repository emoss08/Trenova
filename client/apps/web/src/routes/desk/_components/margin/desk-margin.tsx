import { DeskThread } from "@/components/desk-chat/desk-thread";
import {
  DeskHomeComposer,
  useDeskHomeAsk,
  type DeskStartExtras,
} from "@/components/desk-chat/desk-home-ask";
import { DeskIcon } from "@/components/desk-chat/desk-icons";
import { DeskTip } from "@/components/desk-chat/desk-tip";
import { deskIconClass } from "@/components/desk-chat/desk-button-styles";
import { conversationPath } from "@/lib/conversation-path";
import { assistant } from "@/lib/queries/assistant";
import { useDeskSetting } from "@/stores/desk-settings-store";
import { useDeskMarginStore, type DeskSubject } from "@/stores/desk-margin-store";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { NovaBadge } from "@trenova/shared/components/ui/nova-badge";
import { useT } from "@trenova/shared/i18n/use-t";
import { useNavigate } from "react-router";
import { useDesk } from "../desk-layout";

/** The record a margin question carries, as the composer's mention of it. */
function subjectMention(subject: DeskSubject | null) {
  return subject ? [{ type: subject.kind, id: subject.id, label: subject.label }] : [];
}

/**
 * Nova's margin: the conversation beside the work, never in front of it. It
 * opens on what the page is about, keeps the page in view while the agent
 * answers, and opens full width as the Desk's own conversation.
 */
export function DeskMargin() {
  const t = useT();
  const navigate = useNavigate();
  const desk = useDesk();
  const margin = useDeskMarginStore((state) => state.margin);
  const closeMargin = useDeskMarginStore((state) => state.closeMargin);
  const threadId = margin?.threadId ?? null;
  const subject = margin?.subject ?? null;
  const threadQuery = useQuery({
    ...assistant.thread(threadId ?? ""),
    enabled: threadId !== null,
  });
  const thread = threadId !== null ? (threadQuery.data ?? null) : null;
  const agent = thread ? (desk.agentsById.get(thread.agentDefinitionId) ?? null) : null;

  if (!margin) {
    return null;
  }

  const title = thread ? (
    <>
      <b>{agent?.name ?? t("Agent unavailable")}</b>
      <span className="dk-w-mg-sep">·</span>
      <span>{subject ? t("about {0}", subject.label) : thread.title}</span>
    </>
  ) : subject ? (
    <>
      <b>{t("New conversation")}</b>
      <span className="dk-w-mg-sep">·</span>
      <span>{t("about {0}", subject.label)}</span>
    </>
  ) : (
    <b>{t("New conversation")}</b>
  );

  return (
    <aside className="dk-w-mg" aria-label={t("Nova")}>
      <div className="dk-w-mg-h">
        <NovaBadge size="sm" />
        <span className="dk-w-mg-t">{title}</span>
        {thread && (
          <DeskTip label={t("Open full width")}>
            <Button
              variant="quiet"
              size="icon-sm"
              className={deskIconClass}
              aria-label={t("Open full width")}
              onClick={() => {
                closeMargin();
                void navigate(conversationPath(thread.id));
              }}
            >
              <DeskIcon name="ext" size={14} />
            </Button>
          </DeskTip>
        )}
        <DeskTip label={t("Close")}>
          <Button
            variant="quiet"
            size="icon-sm"
            className={deskIconClass}
            aria-label={t("Close Nova's margin")}
            onClick={closeMargin}
          >
            <DeskIcon name="x" size={15} />
          </Button>
        </DeskTip>
      </div>
      {threadId === null ? (
        <MarginAsk subject={subject} />
      ) : thread === null ? (
        <div className="dk-w-mg-load" aria-busy aria-label={t("Opening the conversation")}>
          <i className="dk-sk" />
          <i className="dk-sk" />
        </div>
      ) : (
        <div className="dk-w-mg-tx">
          <DeskThread
            key={thread.id}
            thread={thread}
            agent={agent}
            agentsUnavailable={desk.agentsUnavailable}
            density="compact"
            threads={desk.threads}
            pageSource="none"
            surface="Desk"
            onOpenInDesk={(artifactId) => {
              closeMargin();
              void navigate(
                artifactId
                  ? `${conversationPath(thread.id)}?a=${encodeURIComponent(artifactId)}`
                  : conversationPath(thread.id),
              );
            }}
          />
        </div>
      )}
    </aside>
  );
}

/**
 * The margin before its conversation exists: the Desk's box, with the record
 * the page is about riding on the question. Asking starts the conversation
 * here, beside the work.
 */
function MarginAsk({ subject }: { subject: DeskSubject | null }) {
  const t = useT();
  const desk = useDesk();
  const preferredAgent = useDeskSetting("agent");
  const ask = useDeskHomeAsk({
    agents: desk.agents,
    threads: desk.threads,
    isLoading: desk.isLoading,
    preferId: preferredAgent,
    pageSource: "screen",
    onStart: (agentId: string, question?: string, extras?: DeskStartExtras) =>
      desk.startInMargin(agentId, question, {
        files: extras?.files ?? [],
        providerId: extras?.providerId ?? "",
        mentions: [...subjectMention(subject), ...(extras?.mentions ?? [])],
      }),
  });

  return (
    <div className="dk-w-mg-new">
      <p className="dk-w-mg-hint">
        {subject
          ? t("Ask anything about {0}. The record goes with your question.", subject.label)
          : t("Ask anything. Whatever is open on the canvas goes with the question.")}
      </p>
      <div className="dk-w-mg-f">
        <DeskHomeComposer ask={ask} isStarting={desk.isStarting} />
      </div>
    </div>
  );
}
