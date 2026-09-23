import { createContext, useContext, type ReactNode } from "react";

/**
 * Asks for the turn that follows a decision on one of the thread's proposals.
 * Null outside a conversation that can answer one — a Desk artifact or the
 * decisions queue records the decision and stops there.
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
