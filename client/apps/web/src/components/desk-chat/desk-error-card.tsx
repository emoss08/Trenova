import type { ReactNode } from "react";
import { DeskCard, DeskCardButton, DeskCardLink, type DeskCardTone } from "./desk-card";
import type { DeskIconName } from "./desk-icons";

/** What an error card is about: something failed, something to watch, a boundary, or news. */
export type DeskErrorTone = Exclude<DeskCardTone, "ask">;

/**
 * A card in the conversation for something that did not go to plan: a reply
 * that failed or stopped, a question the agent may not take, a limit reached.
 * Its tone colours it; its actions are the one or two ways forward.
 */
export function DeskErrorCard({
  tone = "err",
  icon = "alert",
  title,
  sub,
  compact = false,
  actions,
  children,
}: {
  tone?: DeskErrorTone;
  icon?: DeskIconName;
  title?: ReactNode;
  sub?: ReactNode;
  compact?: boolean;
  actions?: ReactNode;
  children?: ReactNode;
}) {
  return (
    <DeskCard tone={tone} icon={icon} title={title} sub={sub} compact={compact} actions={actions}>
      {children}
    </DeskCard>
  );
}

/** A button in an error card: ink for the way forward, quiet for the rest. */
export const DeskErrorButton = DeskCardButton;

/** A link-like action in an error card or lock line: underlined text, no fill. */
export const DeskErrorLink = DeskCardLink;
