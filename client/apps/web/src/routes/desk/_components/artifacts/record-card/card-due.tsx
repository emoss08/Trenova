import { useT } from "@trenova/shared/i18n/use-t";
import { SkeletonBar } from "../skeletons/record-card-skeleton";
import type { DueCheck } from "./due-checks";
import { CardCheck, CardChecks, CardSection, needsLook } from "./record-card";

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
  const open = checks.filter((check) => needsLook(check.state)).length;
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
