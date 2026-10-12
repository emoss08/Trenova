import { senderFirstName } from "@/lib/triage/message-state";
import { TextShimmer } from "@trenova/shared/components/ui/text-shimmer";
import { useRichT } from "@trenova/shared/i18n/rich";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { ReactNode } from "react";
import type { NovaLine } from "./briefing-model";

function Bold({ children }: { children: ReactNode }) {
  return <b className="text-foreground font-semibold">{children}</b>;
}

/**
 * What Nova says at the top of the briefing. It holds its height while the
 * briefing loads, so nothing under it moves when the words arrive. The name
 * of whoever waited longest opens their message.
 */
export function NovaLineText({
  line,
  now,
  onOpenMessage,
}: {
  /** Null while the briefing is on its way. */
  line: NovaLine | null;
  now: number;
  onOpenMessage: (id: string) => void;
}) {
  const t = useT();
  const rt = useRichT();
  const bold = { b: (chunks: ReactNode) => <Bold>{chunks}</Bold> };

  let words: ReactNode;
  if (line === null) {
    words = (
      <TextShimmer as="span" duration={1.6}>
        {t("Going through the day…")}
      </TextShimmer>
    );
  } else if (line.kind === "clear") {
    words = rt(
      "That's everything. <b>You're clear</b> — I'll bring the next one here when it lands.",
      bold,
    );
  } else if (line.kind === "endOfDay") {
    const closing = rt(
      "Wrapping up. I handled <b>{0}</b> and you closed <b>{1}</b>.",
      bold,
      line.handled,
      line.closed,
    );
    let carry: ReactNode;
    if (line.carry === 0) {
      carry = t("Nothing carries over.");
    } else if (line.onCall === null) {
      carry = rt(
        "<b>{0}</b> {1, plural, one {carries} other {carry}} over to tomorrow.",
        bold,
        line.carry,
        line.carry,
      );
    } else {
      carry = rt(
        "<b>{0}</b> {1, plural, one {carries} other {carry}} over to tomorrow — <b>{2}</b> is on overnight.",
        bold,
        line.carry,
        line.carry,
        senderFirstName({ fromName: line.onCall.name, fromAddress: line.onCall.emailAddress }),
      );
    }
    words = (
      <>
        {closing} {carry}
      </>
    );
  } else {
    const longest = line.longest;
    const over = longest !== null && longest.dueAt !== null && longest.dueAt < now;
    const reference = (chunks: ReactNode) =>
      longest === null ? (
        chunks
      ) : (
        <button
          type="button"
          onClick={() => onOpenMessage(longest.messageId)}
          className={cn(
            "ui-focus-ring cursor-pointer rounded-xs bg-linear-to-r bg-[length:100%_1.5px] bg-bottom bg-no-repeat transition-[background-size,color] duration-200 hover:bg-[length:100%_100%]",
            over
              ? "from-warning to-warning text-warning hover:text-warning-on-solid"
              : "from-foreground to-foreground text-foreground hover:text-canvas",
          )}
        >
          {chunks}
        </button>
      );
    const name = longest === null ? "" : senderFirstName(longest);
    let yours: ReactNode;
    if (line.waiting === 1) {
      yours =
        longest === null
          ? rt("<b>One is</b> yours.", bold)
          : rt("<b>One is</b> yours — <ref>{0}</ref> has waited longest.", { ...bold, ref: reference }, name);
    } else {
      yours =
        longest === null
          ? rt("<b>{0} are</b> yours.", bold, line.waiting)
          : rt(
              "<b>{0} are</b> yours — <ref>{1}</ref> has waited longest.",
              { ...bold, ref: reference },
              line.waiting,
              name,
            );
    }
    words = (
      <>
        {rt("I took care of <b>{0}</b> today.", bold, line.handled)} {yours}
        {line.minutes > 0 && (
          <span className="text-foreground-subtle">
            {" "}
            {t(
              "{0, plural, one {About # minute of work.} other {About # minutes of work.}}",
              line.minutes,
            )}
          </span>
        )}
      </>
    );
  }

  return (
    <p
      aria-live="polite"
      className="text-foreground-muted mt-2 min-h-16.5 text-3xl leading-snug text-pretty @max-2xl:min-h-13.5 @max-2xl:text-xl"
    >
      {words}
    </p>
  );
}
