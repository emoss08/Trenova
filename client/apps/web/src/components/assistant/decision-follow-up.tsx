import { createContext, useContext, type ReactNode } from "react";

/**
 * Tells the conversation a card in it was just decided, so it picks up the
 * turn the server starts to report the outcome without waiting for its lists
 * to refetch. Null outside a conversation; a decision made there still
 * reaches the thread, which notices the proposal stop waiting.
 */
export type DecisionFollowUp = (proposalId: string) => void;

const DecisionFollowUpContext = createContext<DecisionFollowUp | null>(null);

export function DecisionFollowUpProvider({
  value,
  children,
}: {
  value: DecisionFollowUp;
  children: ReactNode;
}) {
  return <DecisionFollowUpContext value={value}>{children}</DecisionFollowUpContext>;
}

export function useDecisionFollowUp(): DecisionFollowUp | null {
  return useContext(DecisionFollowUpContext);
}
