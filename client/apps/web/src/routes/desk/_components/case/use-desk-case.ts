import { turnFailureDetail } from "@/components/assistant/follow-turn";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import type { AssistantThread, CaseParty, CaseSubjectType, SnoozeAnchor } from "@/types/assistant";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import { useCallback, useMemo } from "react";
import { toast } from "sonner";

type CaseAction =
  | { kind: "bind"; subjectType: CaseSubjectType; subjectId: string }
  | { kind: "unbind" }
  | { kind: "snooze"; anchor: SnoozeAnchor; until?: number }
  | { kind: "wake" }
  | { kind: "await"; party: CaseParty }
  | { kind: "tick"; itemKey: string; ticked: boolean };

/**
 * A conversation's case: where it stands and its checklist, read only while
 * the conversation is one, and the person's writes to it. Every write moves
 * the conversation, its place on the rail and its waits, so each refetches
 * all three; the realtime events catch up any other tab.
 */
export function useDeskCase(thread: Pick<AssistantThread, "id" | "case">) {
  const t = useT();
  const queryClient = useQueryClient();
  const threadId = thread.id;
  const isCase = thread.case !== undefined;
  const query = useQuery({ ...queries.assistant.case(threadId), enabled: isCase });

  const mutation = useMutation({
    mutationFn: async (action: CaseAction) => {
      const cases = apiService.assistantService;
      switch (action.kind) {
        case "bind":
          return cases.bindCase(threadId, {
            subjectType: action.subjectType,
            subjectId: action.subjectId,
          });
        case "unbind":
          return cases.unbindCase(threadId);
        case "snooze":
          return cases.snoozeCase(threadId, { anchor: action.anchor, until: action.until });
        case "wake":
          return cases.wakeCase(threadId);
        case "await":
          return cases.awaitCaseReply(threadId, {
            party: action.party.kind,
            partyId: action.party.id,
          });
        case "tick":
          return cases.tickCaseItem(threadId, action.itemKey, action.ticked);
      }
    },
    onError: (error) => {
      toast.error(t("The case could not be changed"), {
        description: turnFailureDetail(error, t("Try again in a moment.")),
      });
    },
    onSettled: () =>
      Promise.all([
        queryClient.invalidateQueries({ queryKey: queries.assistant.case(threadId).queryKey }),
        queryClient.invalidateQueries({ queryKey: queries.assistant.thread(threadId).queryKey }),
        queryClient.invalidateQueries({ queryKey: queries.assistant.threads().queryKey }),
        queryClient.invalidateQueries({ queryKey: queries.assistant.waits(threadId).queryKey }),
      ]),
  });

  const { mutate } = mutation;
  const bind = useCallback(
    (subjectType: CaseSubjectType, subjectId: string) =>
      mutate({ kind: "bind", subjectType, subjectId }),
    [mutate],
  );
  const unbind = useCallback(() => mutate({ kind: "unbind" }), [mutate]);
  const snooze = useCallback(
    (anchor: SnoozeAnchor, until?: number) => mutate({ kind: "snooze", anchor, until }),
    [mutate],
  );
  const wake = useCallback(() => mutate({ kind: "wake" }), [mutate]);
  const awaitReply = useCallback((party: CaseParty) => mutate({ kind: "await", party }), [mutate]);
  const tick = useCallback(
    (itemKey: string, ticked: boolean) => mutate({ kind: "tick", itemKey, ticked }),
    [mutate],
  );

  return useMemo(
    () => ({
      view: query.data ?? null,
      loading: isCase && query.isPending,
      failed: query.isError,
      busy: mutation.isPending,
      bind,
      unbind,
      snooze,
      wake,
      awaitReply,
      tick,
    }),
    [
      awaitReply,
      tick,
      bind,
      isCase,
      mutation.isPending,
      query.data,
      query.isError,
      query.isPending,
      snooze,
      unbind,
      wake,
    ],
  );
}

export type DeskCase = ReturnType<typeof useDeskCase>;
