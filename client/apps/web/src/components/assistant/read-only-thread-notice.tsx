import type { CannotContinueReason } from "@/types/assistant";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { useT } from "@trenova/shared/i18n/use-t";
import {
  CalendarClockIcon,
  CircleSlashIcon,
  LockIcon,
  PowerOffIcon,
  type LucideIcon,
} from "lucide-react";
import { FloatingSlot } from "./floating-slot";

type Notice = { icon: LucideIcon; text: string };

/**
 * What the notice says for each reason the server gives. A thread served
 * without a reason, by a server from before it gave one, says only that the
 * conversation cannot continue: guessing "lost access" told people whose
 * agent was merely switched off to go and ask for a role.
 */
export function readOnlyThreadNotice(
  reason: CannotContinueReason | undefined,
  t: ReturnType<typeof useT>,
): Notice {
  switch (reason) {
    case "AgentDisabled":
      return {
        icon: PowerOffIcon,
        text: t("This agent is turned off. An administrator can turn it back on."),
      };
    case "NoAccess":
      return {
        icon: LockIcon,
        text: t(
          "You no longer have access to this agent. An administrator can give one of your roles access to it.",
        ),
      };
    case "AgentDeleted":
      return { icon: CircleSlashIcon, text: t("This agent was removed.") };
    case "AgentNotConversational":
      return {
        icon: CalendarClockIcon,
        text: t(
          "This agent now runs on its own and no longer takes conversations. An administrator can change how it runs.",
        ),
      };
    default:
      return { icon: LockIcon, text: t("This conversation can no longer continue.") };
  }
}

/**
 * What stands where the composer was, in a conversation whose reader may no
 * longer ask its agent anything. It sits where the box sat, floating over the
 * foot of the thread the same way, so the transcript above reads to its end
 * and is padded clear of it by the same measurement.
 *
 * It is a statement rather than an error: nothing went wrong, the
 * conversation is simply a record now. It says why, and who can change that,
 * and nothing on it offers to send.
 */
export function ReadOnlyThreadNotice({
  reason,
  compact = false,
  notice,
  ref,
}: {
  /** Why the conversation cannot continue, as the server served it. */
  reason?: CannotContinueReason;
  compact?: boolean;
  /** A line above the notice: the decisions put off for later, for one. */
  notice?: React.ReactNode;
  /** Measured by the thread, as the composer is, so the last message is never hidden. */
  ref?: React.Ref<HTMLDivElement>;
}) {
  const t = useT();
  const statement = readOnlyThreadNotice(reason, t);
  const Icon = statement.icon;

  return (
    <FloatingSlot ref={ref} compact={compact}>
      {notice}
      <Alert size="sm" role="status" data-testid="read-only-thread-notice">
        <Icon aria-hidden />
        <AlertDescription>{statement.text}</AlertDescription>
      </Alert>
    </FloatingSlot>
  );
}
