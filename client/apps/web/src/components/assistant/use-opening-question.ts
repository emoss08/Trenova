import { useAssistantStore } from "@/stores/assistant-store";
import { useCallback } from "react";

/**
 * Picks up a question asked before this conversation existed.
 *
 * Both surfaces that can start a conversation — the Desk's front page and
 * the corner panel — ask first and get a thread second, so both need the
 * same three lines to collect the question on the other side. They are here
 * rather than in each of them, so the two cannot drift into handing it over
 * differently.
 */
export function useOpeningQuestion(threadId: string): {
  openingQuestion: string | undefined;
  onOpeningQuestionSent: () => void;
} {
  const openingQuestion = useAssistantStore((state) =>
    state.openingQuestion?.threadId === threadId ? state.openingQuestion.text : undefined,
  );
  const setOpeningQuestion = useAssistantStore((state) => state.setOpeningQuestion);

  return {
    openingQuestion,
    onOpeningQuestionSent: useCallback(() => setOpeningQuestion(null), [setOpeningQuestion]),
  };
}
