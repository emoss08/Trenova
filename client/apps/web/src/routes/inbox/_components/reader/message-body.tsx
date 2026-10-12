import type { InboundMessageDetail } from "@/lib/graphql/inbox";
import { useTriageStore } from "@/stores/triage-store";
import { AssistMark } from "@trenova/shared/components/ui/assist-mark";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { memo, useMemo, useState } from "react";
import {
  isRawEdi,
  splitQuotedBody,
  underlineParagraphs,
  type BodySegment,
  type UnderlineFact,
} from "./message-text";

function FactMark({ text, fact }: { text: string; fact: UnderlineFact }) {
  const t = useT();
  const warning = fact.tone === "Warning" || fact.flagged;

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <span
            className={cn(
              "cursor-help rounded-xs decoration-dotted decoration-[1.5px] underline-offset-4 transition-colors hover:decoration-solid",
              "underline",
              warning
                ? "decoration-warning hover:bg-warning-subtle"
                : "decoration-accent-violet hover:bg-accent-violet-subtle",
            )}
          />
        }
      >
        {text}
      </TooltipTrigger>
      <TooltipContent className="flex items-center gap-1.5">
        <AssistMark aria-hidden className="size-3" />
        {fact.flagged ? t("{0} · flagged as wrong", fact.label) : fact.label}
      </TooltipContent>
    </Tooltip>
  );
}

function Paragraph({ segments }: { segments: BodySegment[] }) {
  return (
    <p className="max-w-[68ch] text-lg leading-[1.7] text-pretty whitespace-pre-wrap wrap-anywhere">
      {segments.map((segment, index) =>
        segment.kind === "text" ? (
          segment.text
        ) : (
          <FactMark key={`${segment.fact.key}:${index}`} text={segment.text} fact={segment.fact} />
        ),
      )}
    </p>
  );
}

/**
 * What the sender wrote: prose in paragraphs with the words Nova read a fact
 * from underlined (when the viewer asks for it), raw EDI as the code it is,
 * and the thread they replied on folded away. A held message is drawn quieter
 * so it is not read as trustworthy.
 */
export const MessageBody = memo(function MessageBody({
  message,
}: {
  message: Pick<InboundMessageDetail, "textBody" | "facts" | "status">;
}) {
  const t = useT();
  const underline = useTriageStore((state) => state.underlineFacts);
  const [showQuoted, setShowQuoted] = useState(false);
  const body = useMemo(() => splitQuotedBody(message.textBody), [message.textBody]);
  const raw = useMemo(() => isRawEdi(body.own), [body.own]);
  const paragraphs = useMemo(
    () => underlineParagraphs(body.own, underline && !raw ? message.facts : []),
    [body.own, underline, raw, message.facts],
  );
  const held = message.status === "Quarantined";

  return (
    <section
      aria-label={t("Message")}
      className={cn("flex flex-col gap-3 pt-1", held && "text-foreground-muted")}
    >
      {raw ? (
        <pre className="bg-sunken border-border-subtle text-foreground-muted rounded-lg border px-3 py-2.5 font-mono text-sm leading-relaxed break-all whitespace-pre-wrap">
          {body.own}
        </pre>
      ) : paragraphs.length === 0 ? (
        <p className="text-foreground-subtle text-base">{t("The message has no text.")}</p>
      ) : (
        paragraphs.map((segments, index) => <Paragraph key={index} segments={segments} />)
      )}
      {body.quoted !== "" && (
        <>
          <button
            type="button"
            aria-expanded={showQuoted}
            onClick={() => setShowQuoted((value) => !value)}
            className="bg-surface-hover text-foreground-subtle hover:text-foreground ui-focus-ring inline-flex h-5.5 w-fit items-center rounded-full px-2 text-sm"
          >
            {showQuoted ? t("Hide earlier thread") : t("··· Earlier thread")}
          </button>
          {showQuoted && (
            <pre className="border-border text-foreground-subtle animate-rise border-l-2 pl-3 font-sans text-base leading-relaxed whitespace-pre-wrap">
              {body.quoted}
            </pre>
          )}
        </>
      )}
    </section>
  );
});
