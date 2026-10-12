import { useT } from "@trenova/shared/i18n/use-t";
import { SkeletonBar } from "../skeletons/record-card-skeleton";
import { CardCheck, CardChecks, CardSection, type CardCheckState } from "./record-card";

/** One thing a card checks on its record, drawn as a `CardCheck`. */
export type DueCheck = {
  key: string;
  state: CardCheckState;
  title: string;
  detail: string;
};

const RANK: Record<CardCheckState, number> = { bad: 0, warn: 1, ok: 2 };

/** Checks with what is wrong first, then what is close, then what holds, each kept in its order. */
export function worstFirst(checks: readonly DueCheck[]): DueCheck[] {
  return checks
    .map((check, index) => ({ check, index }))
    .sort((a, b) => RANK[a.check.state] - RANK[b.check.state] || a.index - b.index)
    .map(({ check }) => check);
}

/**
 * What falls due on a record, worst first, under a title that says how many
 * need a look. `waiting` holds a row for a check still being read, so the list
 * does not grow under the reader when it lands.
 */
export function CardDue({
  title,
  checks,
  waiting,
}: {
  title: string;
  checks: readonly DueCheck[];
  /** What the row held for a check still being read says to a screen reader. */
  waiting?: string;
}) {
  const t = useT();
  const open = checks.filter((check) => check.state !== "ok").length;
  return (
    <CardSection
      title={title}
      aside={
        open > 0 ? (
          <span className="dk-warn">
            {t("{0, plural, one {# needs a look} other {# need a look}}", open)}
          </span>
        ) : waiting ? undefined : (
          <span className="dk-ok">{t("Nothing due")}</span>
        )
      }
    >
      <CardChecks>
        {checks.map((check) => (
          <CardCheck
            key={check.key}
            state={check.state}
            title={check.title}
            detail={check.detail}
          />
        ))}
        {waiting && (
          <div className="dk-rec-ck dk-rec-ck-wait" aria-busy>
            <i className="dk-sk dk-rec-sk-mark" />
            <span className="dk-rec-sk-ckb">
              <SkeletonBar w="30%" className="dk-rec-sk-line" />
              <span className="sr-only">{waiting}</span>
            </span>
          </div>
        )}
      </CardChecks>
    </CardSection>
  );
}
