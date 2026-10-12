import type { InboundMessageDetail } from "@/lib/graphql/inbox";
import { NovaMark } from "@trenova/shared/components/ui/nova-mark";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixTime } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { useState } from "react";
import { PersonAvatar } from "../person-avatar";

type Reply = InboundMessageDetail["replies"][number];

function SentReply({ reply }: { reply: Reply }) {
  const t = useT();
  const meId = useAuthStore((state) => state.user?.id ?? null);
  const [expanded, setExpanded] = useState(false);
  const sender = reply.sentBy;
  const who =
    sender === null ? t("Nova replied") : sender.id === meId ? t("You replied") : t("{0} replied", sender.name);

  return (
    <button
      type="button"
      aria-expanded={expanded}
      onClick={() => setExpanded((value) => !value)}
      className="border-border hover:border-border-strong ui-focus-ring animate-rise flex flex-col gap-2 rounded-xl border px-3.5 py-3 text-left transition-colors"
    >
      <span className="flex w-full items-center gap-2 text-base">
        <PersonAvatar name={sender?.name ?? "Nova"} />
        <span className="font-medium">{who}</span>
        {reply.sentAt !== null && (
          <span className="text-foreground-subtle font-mono text-2xs tabular-nums">
            {formatUnixTime(reply.sentAt)}
          </span>
        )}
        {reply.draftedByNova && (
          <span className="text-accent-violet ml-auto inline-flex items-center gap-1 text-xs">
            <NovaMark className="size-3" />
            {t("drafted by Nova")}
          </span>
        )}
      </span>
      <span
        className={cn(
          "text-foreground-muted text-base leading-relaxed whitespace-pre-wrap",
          !expanded && "line-clamp-3",
        )}
      >
        {reply.body}
      </span>
    </button>
  );
}

/** The replies that went out, newest last, each saying who sent it and whether Nova drafted it. */
export function SentReplies({ message }: { message: InboundMessageDetail }) {
  const sent = message.replies.filter((reply) => reply.status === "Sent");
  if (sent.length === 0) {
    return null;
  }

  return (
    <div className="flex flex-col gap-2">
      {sent.map((reply) => (
        <SentReply key={reply.id} reply={reply} />
      ))}
    </div>
  );
}
