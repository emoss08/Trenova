import { Badge } from "@trenova/shared/components/ui/badge";
import type { DeskCase } from "@/lib/graphql/desk-case";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn, toSentenceFragment } from "@trenova/shared/lib/utils";
import { Fragment, type ReactNode } from "react";
import { Link } from "react-router";
import { dueTone } from "../attention/attention-row";
import { formatCountdown, minutesLeft } from "../watchtower/lanes";
import { badgeTone } from "./case-tone";

/**
 * The case's head: what the record is and whose, its route or amount as the
 * heading, where it stands, who is on it, and on the right how long is left
 * and until what. When the plan has run, the clock gives way to what it came
 * to.
 */
export function CaseHead({
  data,
  now,
  recordPath,
  owner,
}: {
  data: DeskCase;
  now: number;
  recordPath: string | null;
  /** Whose case it is, and the control that changes it. */
  owner: ReactNode;
}) {
  const t = useT();
  const { head, record } = data;
  const minutes = minutesLeft(head.deadline?.at, now);
  const countdown = formatCountdown(t, minutes);
  const done = data.decision?.phase === "Done";
  const chosen = data.options.find((option) => option.chosen) ?? null;
  const key = [record.noun, record.label, ...head.references].filter((part) => part !== "");

  return (
    <div className="dk-cz-head">
      <div>
        <div className="dk-cz-key">
          {key.map((part, index) => (
            <Fragment key={`${index}:${part}`}>
              {index > 0 && <span aria-hidden>·</span>}
              <span className={cn(index === 1 && "dk-cz-num")}>
                {index === 0 ? capitalize(part) : part}
              </span>
            </Fragment>
          ))}
        </div>
        <h1>{head.title}</h1>
        <div className="dk-cz-sub">
          {head.status !== "" && (
            <Badge variant={badgeTone(head.statusTone)} dot>
              {head.status}
            </Badge>
          )}
          {head.details.map((detail, index) => (
            <Fragment key={detail}>
              {index > 0 && <span aria-hidden>·</span>}
              <span>{detail}</span>
            </Fragment>
          ))}
          {recordPath && (
            <Link className="dk-w-alnk ui-focus-ring" to={recordPath}>
              {t("Open the record")}
            </Link>
          )}
          {owner}
        </div>
      </div>
      {done && chosen && chosen.outcome !== "" ? (
        <div className="dk-cz-clk">
          <div className="dk-cz-clk-v dk-ok">{chosen.outcome}</div>
          <div className="dk-cz-clk-l">
            {chosen.outcomeNote !== "" && <>{capitalize(chosen.outcomeNote)}. </>}
            {t("After the plan you approved ran.")}
          </div>
        </div>
      ) : (
        countdown !== null && (
          <div className="dk-cz-clk">
            <div className={cn("dk-cz-clk-v", dueTone(minutes))}>{countdown}</div>
            <div className="dk-cz-clk-l">
              {head.deadline && head.deadline.label !== ""
                ? minutes !== null && minutes < 0
                  ? t("since: {0}", toSentenceFragment(head.deadline.label))
                  : t("until: {0}", toSentenceFragment(head.deadline.label))
                : t("until its deadline")}
            </div>
          </div>
        )
      )}
    </div>
  );
}

function capitalize(text: string): string {
  return text === "" ? text : text.charAt(0).toUpperCase() + text.slice(1);
}
