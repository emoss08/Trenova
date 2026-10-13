import type { InboundMessageFilter } from "@/lib/graphql/inbox";
import { createContext, useContext } from "react";

/**
 * How the page lets the rest of Triage move the reader without re-rendering
 * anyone: every member is a function that reads the page's state at the
 * moment it is called, and the object itself never changes identity.
 */
export type TriageNavigation = {
  /** The open message, if any. */
  openId: () => string | null;
  /** The open folder's messages in the order the list draws them. */
  orderedIds: () => readonly string[];
  /** The open folder as the server is asked for it; null on the Briefing or Rules. */
  filter: () => InboundMessageFilter | null;
  /** Opens a message (or closes the reader with null), keeping the folder. */
  open: (id: string | null) => void;
  /** Opens a message in Waiting on you, as the Briefing does when it hands one over. */
  openInQueue: (id: string) => void;
  /** Whether the reader sits beside the list, so leaving a message should open the next. */
  isWide: () => boolean;
};

const NOWHERE: TriageNavigation = {
  openId: () => null,
  orderedIds: () => [],
  filter: () => null,
  open: () => undefined,
  openInQueue: () => undefined,
  isWide: () => false,
};

export const TriageNavigationContext = createContext<TriageNavigation>(NOWHERE);

export function useTriageNavigation(): TriageNavigation {
  return useContext(TriageNavigationContext);
}
