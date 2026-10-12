import type { InboundMessageDetail, InboundThreadMessage } from "@/lib/graphql/inbox";
import { queries } from "@/lib/queries";
import { senderName } from "@/lib/triage/message-state";
import { useQuery } from "@tanstack/react-query";
import { CheckIcon } from "@trenova/shared/components/icons";
import { useT } from "@trenova/shared/i18n/use-t";
import {
  formatSecondsAgo,
  formatUnixDateTimeShort,
  formatUnixTime,
} from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { useMemo, useState } from "react";
import { PersonAvatar } from "../person-avatar";
import { splitQuotedBody } from "./message-text";
import { SenderName } from "./sender-card";

const DAY = 86_400;

/** "13:39" today, "Oct 6, 10:19" before. */
function stamp(seconds: number, now: number): string {
  return now - seconds < DAY ? formatUnixTime(seconds) : formatUnixDateTimeShort(seconds);
}

function ThreadRow({
  message,
  expanded,
  onToggle,
  now,
}: {
  message: InboundThreadMessage;
  expanded: boolean;
  onToggle: () => void;
  now: number;
}) {
  const t = useT();
  const meId = useAuthStore((state) => state.user?.id ?? null);
  const own = useMemo(() => splitQuotedBody(message.textBody).own, [message.textBody]);
  const name = senderName(message);
  const settled = message.status === "Actioned" || message.status === "Ignored";
  const note = message.reviewNote.trim();
  const by =
    message.reviewer === null
      ? message.handledByNova
        ? t("Nova")
        : t("A teammate")
      : message.reviewer.id === meId
        ? t("You")
        : (message.reviewer.name.split(/\s+/)[0] ?? message.reviewer.name);

  return (
    <div className="border-border-subtle not-first:border-t">
      <button
        type="button"
        aria-expanded={expanded}
        onClick={onToggle}
        className="hover:bg-surface-hover ui-inset-focus-ring flex w-full items-center gap-2.5 px-3 py-2 text-left text-sm"
      >
        <PersonAvatar name={name} />
        <span className="shrink-0 font-medium">{name}</span>
        <span className="text-foreground-subtle min-w-0 flex-1 truncate">
          {expanded ? "" : message.preview || own.split("\n")[0]}
        </span>
        <span className="text-foreground-subtle shrink-0 font-mono text-xs tabular-nums">
          {stamp(message.receivedAt, now)}
        </span>
      </button>
      {expanded && (
        <div className="animate-rise flex flex-col gap-1.5 pr-3 pb-2.5 pl-10.5">
          <p className="text-foreground-muted text-sm leading-relaxed whitespace-pre-wrap">{own}</p>
          {settled && note !== "" && (
            <span className="text-success flex items-center gap-1.5 text-xs">
              <CheckIcon aria-hidden className="size-3" strokeWidth={2.4} />
              {t("{0}: {1}", by, note)}
            </span>
          )}
        </div>
      )}
    </div>
  );
}

/** The older messages of the thread, folded to one line each; one opens at a time. */
function OlderInThread({ message, now }: { message: InboundMessageDetail; now: number }) {
  const t = useT();
  const [expanded, setExpanded] = useState<string | null>(null);
  const query = useQuery({
    ...queries.inbox.thread(message.threadKey),
    enabled: message.threadCount > 1 && message.threadKey !== "",
  });
  const older = useMemo(
    () =>
      (query.data ?? [])
        .filter((row) => row.id !== message.id && row.receivedAt <= message.receivedAt)
        .sort((a, b) => a.receivedAt - b.receivedAt),
    [query.data, message.id, message.receivedAt],
  );
  if (older.length === 0) {
    return null;
  }

  return (
    <div
      aria-label={t("Earlier in this thread")}
      className="border-border-subtle overflow-hidden rounded-lg border"
    >
      {older.map((row) => (
        <ThreadRow
          key={row.id}
          message={row}
          now={now}
          expanded={expanded === row.id}
          onToggle={() => setExpanded((current) => (current === row.id ? null : row.id))}
        />
      ))}
    </div>
  );
}

/** The subject, the older messages of its thread, and who sent it when. */
export function MessageHeader({ message, now }: { message: InboundMessageDetail; now: number }) {
  const t = useT();
  const name = senderName(message);

  return (
    <header className="flex flex-col gap-3.5">
      <h1
        className={cn(
          "text-3xl leading-tight font-semibold text-balance wrap-anywhere",
          message.subject === "" && "text-foreground-subtle font-medium",
        )}
      >
        {message.subject === "" ? t("(no subject)") : message.subject}
      </h1>
      <OlderInThread message={message} now={now} />
      <div className="flex min-w-0 items-center gap-2.5 text-base">
        <PersonAvatar name={name} size="md" />
        <SenderName message={message} />
        {message.fromName.trim() !== "" && (
          <span className="text-foreground-subtle min-w-0 truncate text-sm">
            {message.fromAddress}
          </span>
        )}
        <span className="flex-1" />
        <span className="text-foreground-subtle shrink-0 font-mono text-xs tabular-nums">
          {t(
            "{0} · {1}",
            formatUnixTime(message.receivedAt),
            formatSecondsAgo(now - message.receivedAt),
          )}
        </span>
      </div>
    </header>
  );
}
