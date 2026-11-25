import { DeskThread } from "@/components/desk-chat/desk-thread";
import { DeskIcon } from "@/components/desk-chat/desk-icons";
import { DeskTip } from "@/components/desk-chat/desk-tip";
import { deskIconClass } from "@/components/desk-chat/desk-button-styles";
import { conversationPath } from "@/lib/conversation-path";
import { assistant } from "@/lib/queries/assistant";
import { useDeskMarginStore } from "@/stores/desk-margin-store";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { NovaBadge } from "@trenova/shared/components/ui/nova-badge";
import { useT } from "@trenova/shared/i18n/use-t";
import { useNavigate } from "@tanstack/react-router";
import { useDesk } from "../desk-layout";
import { viewPath } from "../view/view-path";

/**
 * Nova's margin: a conversation that already exists, beside the work. It keeps
 * the page in view while the agent answers, and opens full width as the
 * Desk's own conversation. A new conversation never starts here.
 */
export function DeskMargin() {
  const t = useT();
  const navigate = useNavigate();
  const desk = useDesk();
  const margin = useDeskMarginStore((state) => state.margin);
  const closeMargin = useDeskMarginStore((state) => state.closeMargin);
  const subject = margin?.subject ?? null;
  const threadId = margin?.threadId ?? null;
  const threadQuery = useQuery({
    ...assistant.thread(threadId ?? ""),
    enabled: threadId !== null,
  });
  const thread = threadId !== null ? (threadQuery.data ?? null) : null;
  const agent = thread ? (desk.agentsById.get(thread.agentDefinitionId) ?? null) : null;

  if (!margin) {
    return null;
  }

  const title = subject ? (
    <>
      <b>{thread ? (agent?.name ?? t("Agent unavailable")) : t("Opening the conversation")}</b>
      <span className="dk-w-mg-sep">·</span>
      <span>{t("about {0}", subject.label)}</span>
    </>
  ) : (
    <b>{thread ? thread.title || t("Untitled conversation") : t("Opening the conversation")}</b>
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
                void navigate({ to: conversationPath(thread.id) });
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
      {thread === null ? (
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
            agentModelOnly={Boolean(subject?.agentId)}
            onOpenInDesk={(artifactId) => {
              // What the agent made opens on the canvas, beside the
              // conversation that made it; the conversation alone opens full width.
              if (artifactId) {
                void navigate({ to: viewPath(thread.id, artifactId) });
                return;
              }
              closeMargin();
              void navigate({ to: conversationPath(thread.id) });
            }}
          />
        </div>
      )}
    </aside>
  );
}
