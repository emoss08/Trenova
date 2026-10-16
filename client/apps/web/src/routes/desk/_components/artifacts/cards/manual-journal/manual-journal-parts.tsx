import type { JournalCardPeriod } from "@/lib/graphql/journal-cards";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { PeriodChip } from "../../record-card/card-journal";
import { accountingDay, accountingLeaf, periodIsShut } from "../../record-card/journal-values";
import { CardAvatar, CardIcon } from "../../record-card/record-card";
import { cardDay } from "../../record-card/record-card-values";
import type { PostingPlace } from "./manual-journal-card";

/** Why the journal was asked for, in the requester's own words, signed with their name. */
export function JournalWhy({ text, by }: { text: string; by: string | null }) {
  return (
    <blockquote className="dk-mj-why">
      <p>{text}</p>
      {by && (
        <footer>
          <CardAvatar name={by} size="sm" />
          {by}
        </footer>
      )}
    </blockquote>
  );
}

/** A period as a calendar leaf: the month over the booking day, with the period's name and state beside it. */
function PeriodLeaf({
  period,
  date,
  line,
  lands,
}: {
  period: JournalCardPeriod;
  date: number;
  line: string;
  lands: boolean;
}) {
  const leaf = accountingLeaf(date);
  return (
    <div className={cn("dk-mj-cal", lands ? "dk-lands" : periodIsShut(period) && "dk-shut")}>
      <span className="dk-mj-leaf" aria-hidden="true">
        <span>{leaf.month}</span>
        <b>{leaf.day}</b>
      </span>
      <span className="dk-what">
        <PeriodChip period={period} />
        <span>{line}</span>
      </span>
    </div>
  );
}

/**
 * Where the journal will post: the day asked for in its period, or that
 * period shut and the first open day it moves to under the organization's
 * policy, or why it cannot post until the period is reopened.
 */
export function JournalPostingPlace({ place, asked }: { place: PostingPlace; asked: number }) {
  const t = useT();
  switch (place.kind) {
    case "asked":
      return (
        <div className="dk-mj-post">
          <PeriodLeaf
            period={place.period}
            date={place.date}
            line={t("Posts on {0}, the date asked for", accountingDay(place.date))}
            lands
          />
        </div>
      );
    case "moved":
      return (
        <>
          <div className="dk-mj-post">
            {place.from && (
              <>
                <PeriodLeaf
                  period={place.from}
                  date={asked}
                  line={
                    place.from.closedAt
                      ? t("Closed {0}", cardDay(place.from.closedAt))
                      : t("Asked for {0}", accountingDay(asked))
                  }
                  lands={false}
                />
                <span className="dk-mj-arrow" aria-hidden="true">
                  <CardIcon name="forward" size={16} />
                </span>
              </>
            )}
            <PeriodLeaf
              period={place.to}
              date={place.date}
              line={t("Posts on {0}, its first open day", accountingDay(place.date))}
              lands
            />
          </div>
          <p className="dk-mj-postnote">
            {place.from
              ? t(
                  "{0} takes no more postings, and your accounting policy books late entries on the first open day.",
                  place.from.name,
                )
              : t("Your accounting policy books it on the first open day.")}
          </p>
        </>
      );
    case "refused":
      return (
        <div className="dk-mj-refused">
          <CardIcon name="warn" size={13} stroke={2.2} />
          <span>{place.reason}</span>
        </div>
      );
  }
}

/** Why a refused or withdrawn journal stands as it does, kept as written, with who and when. */
export function JournalSettledSlip({ reason, by }: { reason: string; by: string }) {
  return (
    <div className="dk-mj-slip">
      <p>{reason}</p>
      <span>{by}</span>
    </div>
  );
}
