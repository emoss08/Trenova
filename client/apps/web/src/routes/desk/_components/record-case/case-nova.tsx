import { DeskThread } from "@/components/desk-chat/desk-thread";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { assistant } from "@/lib/queries/assistant";
import { assistantService } from "@/services/assistant";
import { useDeskAttachedPageStore } from "@/stores/desk-attached-page-store";
import { useDeskSetting } from "@/stores/desk-settings-store";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { NovaBadge } from "@trenova/shared/components/ui/nova-badge";
import { useT } from "@trenova/shared/i18n/use-t";
import { useState } from "react";
import { useDesk } from "../desk-layout";
import { casePageContext, caseSubjectOf } from "./case-subject";

/**
 * Nova beside the work, not in front of it: the person's conversation about
 * this record, the same thread the Desk keeps, drawn compact in the case's
 * margin. A record a case can be about binds the conversation to it, so the
 * thread is the record's case; every question carries the record as the page
 * it was asked from. Nothing here starts a conversation until a question is
 * sent.
 */
export function CaseNova({
  kind,
  id,
  label,
  path,
}: {
  kind: string;
  id: string;
  label: string;
  path: string;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const { threads, agents, agentsById, agentsUnavailable } = useDesk();
  const preferred = useDeskSetting("agent");
  const attach = useDeskAttachedPageStore((state) => state.attach);
  const subject = caseSubjectOf(kind);
  const existing =
    subject === null
      ? null
      : (threads.find((thread) => thread.subjectType === subject && thread.subjectId === id) ??
        null);
  const [startedId, setStartedId] = useState<string | null>(null);
  const [opening, setOpening] = useState<string | undefined>(undefined);
  const [draft, setDraft] = useState("");
  const threadId = startedId ?? existing?.id ?? null;

  const threadQuery = useQuery({ ...assistant.thread(threadId ?? ""), enabled: threadId !== null });
  const thread = threadQuery.data ?? (existing?.id === threadId ? existing : null);
  const agent =
    (thread ? agentsById.get(thread.agentDefinitionId) : undefined) ??
    agents.find((choice) => choice.id === preferred) ??
    agents[0] ??
    null;

  const start = useApiMutation({
    mutationFn: async (question: string) => {
      if (agent === null) {
        throw new Error("no agent");
      }
      const created = await assistantService.startThread(agent.id, {
        origin: "Desk",
        title: label,
        subjectType: subject ?? undefined,
        subjectId: subject ? id : undefined,
      });
      return { created, question };
    },
    onSuccess: ({ created, question }) => {
      attach(created.id, casePageContext(path, kind, id, label));
      queryClient.setQueryData(assistant.thread(created.id).queryKey, created);
      void queryClient.invalidateQueries({ queryKey: assistant.threads().queryKey });
      setOpening(question);
      setStartedId(created.id);
      setDraft("");
    },
    resourceName: "Conversation",
  });

  if (thread && agent) {
    return (
      <div className="dk-rc-nova">
        <DeskThread
          key={thread.id}
          thread={thread}
          agent={agent}
          agentsUnavailable={agentsUnavailable}
          threads={threads}
          density="compact"
          pageSource="none"
          openingQuestion={opening}
          onOpeningQuestionSent={() => setOpening(undefined)}
        />
      </div>
    );
  }

  const send = () => {
    const question = draft.trim();
    if (question === "" || start.isPending) {
      return;
    }
    start.mutate(question);
  };

  return (
    <div className="dk-rc-ask">
      <div className="dk-rc-ask-h">
        <NovaBadge size="lg" />
        <b>{t("Ask about this record")}</b>
      </div>
      <p>
        {agent
          ? t("{0} reads the record with your question. Nothing changes until you approve it.", agent.name)
          : t("No agent is available to you here.")}
      </p>
      <label className="sr-only" htmlFor="dk-rc-ask-box">
        {t("Your question")}
      </label>
      <textarea
        id="dk-rc-ask-box"
        className="dk-rc-ask-box ui-field"
        rows={3}
        value={draft}
        disabled={agent === null}
        placeholder={t("Why is this late, and what are my options?")}
        onChange={(event) => setDraft(event.target.value)}
        onKeyDown={(event) => {
          if (event.key === "Enter" && !event.shiftKey) {
            event.preventDefault();
            send();
          }
        }}
      />
      <div className="dk-rc-ask-f">
        <Button
          size="sm"
          onClick={send}
          isLoading={start.isPending}
          disabled={agent === null || draft.trim() === ""}
        >
          {t("Ask")}
        </Button>
      </div>
    </div>
  );
}
