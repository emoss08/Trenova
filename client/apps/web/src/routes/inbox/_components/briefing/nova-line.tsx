import { senderFirstName, senderName } from "@/lib/triage/message-state";
import { TextShimmer } from "@trenova/shared/components/ui/text-shimmer";
import { useRichT } from "@trenova/shared/i18n/rich";
import { useT } from "@trenova/shared/i18n/use-t";
import type { ReactNode } from "react";
import type { NovaLine } from "./briefing-model";

function Bold({ children }: { children: ReactNode }) {
  return <b className="font-medium text-dsk-fg">{children}</b>;
}

/**
 * What Nova says under the briefing's headline, set as the Desk's Today sets
 * the day's line. The name of whoever waited longest opens their message.
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
        senderFirstName({ fromName: line.onCall.name, fromAddress: line.onCall.username }),
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
          className={over ? "dk-dy-lnk ui-focus-ring text-dsk-warn-fg" : "dk-dy-lnk ui-focus-ring"}
        >
          {chunks}
        </button>
      );
    // The whole name: a sender here is as often a company as a person, and the
    // first word of "Partner Carrier EDI" names nobody.
    const name = longest === null ? "" : senderName(longest);
    // The headline above already says how many are waiting; this line adds
    // what the count does not: who has waited longest, and how long it takes.
    const yours: ReactNode =
      longest === null
        ? null
        : rt("<ref>{0}</ref> has waited longest.", { ...bold, ref: reference }, name);
    words = (
      <>
        {line.handled > 0 && <>{rt("I took care of <b>{0}</b> today.", bold, line.handled)} </>}
        {yours}
        {line.minutes > 0 && (
          <span>
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
    <p aria-live="polite" className="dk-dy-hl">
      {words}
    </p>
  );
}
