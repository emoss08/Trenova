import { describeApiError } from "@/lib/api-error-message";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn, getNameInitials } from "@trenova/shared/lib/utils";
import { Fragment, type ReactNode } from "react";
import { Link } from "react-router";
import type { Neighbors } from "./record-card-values";

/*
The parts every Desk record card is built from, the billing queue item's and
the formula template's alike: a head with the record's name, its figure and
where it stands; titled sections; a list of checks; a timeline; and a bar of
actions held to the bottom. A card that opens from a table's row steps
through the table's other rows from its head.
*/

const CARD_PATHS = {
  check: <path d="M5 12.5l4.5 4.5L19 7" />,
  warn: (
    <>
      <path d="M12 4l9 15.5H3z" />
      <path d="M12 10v4M12 17v.01" />
    </>
  ),
  x: <path d="M7 7l10 10M17 7L7 17" />,
  info: (
    <>
      <path d="M12 11v6" />
      <path d="M12 7.5v.01" />
    </>
  ),
  file: (
    <>
      <path d="M6 3.5h8l4 4v13H6z" />
      <path d="M14 3.5v4h4" />
    </>
  ),
  pause: <path d="M9 6v12M15 6v12" />,
  truck: (
    <>
      <path d="M3.5 6.5h10v9h-10zM13.5 9.5h4l3 3v3h-7" />
      <circle cx="7" cy="17" r="1.6" />
      <circle cx="17" cy="17" r="1.6" />
    </>
  ),
  ext: (
    <path d="M14 4.5h5.5V10M19.5 4.5L11 13M17 14v4.5a1 1 0 0 1-1 1H6a1 1 0 0 1-1-1V8a1 1 0 0 1 1-1h4.5" />
  ),
  down: <path d="M6 9l6 6 6-6" />,
  up: <path d="M6 15l6-6 6 6" />,
  undo: <path d="M9 7L4.5 11.5 9 16M5 11.5h9a5 5 0 0 1 0 10h-2" />,
  back: <path d="M15 6l-6 6 6 6" />,
  forward: <path d="M9 6l6 6-6 6" />,
  rise: <path d="M12 19V5m0 0l-6 6m6-6l6 6" />,
  fall: <path d="M12 5v14m0 0l-6-6m6 6l6-6" />,
  copy: (
    <>
      <rect x="8.5" y="8.5" width="11" height="11" rx="2" />
      <path d="M15.5 8.5V6a1.5 1.5 0 0 0-1.5-1.5H6A1.5 1.5 0 0 0 4.5 6v8A1.5 1.5 0 0 0 6 15.5h2.5" />
    </>
  ),
  reset: <path d="M4.5 12a7.5 7.5 0 1 0 2.2-5.3M4.5 4.5v4h4" />,
  pin: (
    <>
      <path d="M12 21s-6.5-5.6-6.5-11a6.5 6.5 0 0 1 13 0c0 5.4-6.5 11-6.5 11z" />
      <circle cx="12" cy="10" r="2.3" />
    </>
  ),
} as const;

export type CardIconName = keyof typeof CARD_PATHS;

/** A record card's icons, drawn on one grid in the card's stroke. */
export function CardIcon({
  name,
  size = 13,
  stroke = 1.9,
}: {
  name: CardIconName;
  size?: number;
  stroke?: number;
}) {
  return (
    <svg
      className="size-auto"
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={stroke}
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      {CARD_PATHS[name]}
    </svg>
  );
}

/** The card itself: a column its sections stack in, with the bar held to the bottom. */
export function RecordCard({ className, children }: { className?: string; children: ReactNode }) {
  return <div className={cn("dk-rec", className)}>{children}</div>;
}

/**
 * A card's head: a kicker of short identifiers, the record's name with a line
 * under it, the figure it is read for on the right, and what it stands at.
 */
export function CardHead({
  kicker,
  onBack,
  backLabel,
  stepper,
  title,
  subtitle,
  lead,
  figure,
  under,
  meta,
}: {
  /** Short identifiers, set apart by dots; the first reads strongest. */
  kicker: readonly ReactNode[];
  /** Back to the table the card was opened from. */
  onBack?: () => void;
  backLabel?: string;
  stepper?: ReactNode;
  title: ReactNode;
  subtitle?: ReactNode;
  /** A mark before the title that says who the record is, such as a person's initials. */
  lead?: ReactNode;
  figure?: ReactNode;
  /** What sits under the name, above where the record stands: a plan's track. */
  under?: ReactNode;
  meta?: ReactNode;
}) {
  return (
    <div className="dk-rec-head">
      <div className="dk-rec-kick">
        {onBack && (
          <Button
            variant="quiet"
            size="icon-xs"
            className="-ml-1"
            onClick={onBack}
            title={backLabel}
            aria-label={backLabel}
          >
            <CardIcon name="back" size={12} stroke={2.2} />
          </Button>
        )}
        {kicker.map((item, index) => (
          // oxlint-disable-next-line react/no-array-index-key -- the kicker is a fixed run of slots
          <Fragment key={index}>
            {index > 0 && <i />}
            <span>{item}</span>
          </Fragment>
        ))}
        {stepper}
      </div>
      <div className="dk-rec-hrow">
        {lead}
        <div className={cn("dk-rec-who", subtitle === undefined && "dk-alone")}>
          <h2>{title}</h2>
          {subtitle !== undefined && <span>{subtitle}</span>}
        </div>
        {figure && <div className="dk-rec-amt">{figure}</div>}
      </div>
      {under}
      {meta && <div className="dk-rec-meta">{meta}</div>}
    </div>
  );
}

/** A person or a company as their initials in a round mark, where a card names them. */
export function CardAvatar({ name, size = "md" }: { name: string; size?: "sm" | "md" | "lg" }) {
  return (
    <span className={cn("dk-rec-av", `dk-${size}`)} aria-hidden>
      {getNameInitials(name, "•")}
    </span>
  );
}

/**
 * A card whose record could not be read: what kind of record it was, that it
 * could not be read and why, and the way back to where it was opened from.
 */
export function CardUnreadable({
  kind,
  title,
  error,
  onBack,
  backLabel,
}: {
  /** The record's kind, in the kicker. */
  kind: string;
  /** "This tractor could not be read", in the record's own words. */
  title: string;
  error: unknown;
  onBack?: () => void;
  backLabel?: string;
}) {
  const t = useT();
  return (
    <RecordCard>
      <CardHead
        onBack={onBack}
        backLabel={backLabel ?? t("Back to the table")}
        kicker={[kind]}
        title={title}
        subtitle={describeApiError(
          error,
          t("It may have been removed, or you may not be allowed to see it."),
        )}
      />
    </RecordCard>
  );
}

/** Up and down through the list the card was opened from, with where it sits in it. */
export function CardStepper({
  neighbors,
  onStep,
  previousLabel,
  nextLabel,
}: {
  neighbors: Neighbors;
  onStep: (id: string) => void;
  previousLabel: string;
  nextLabel: string;
}) {
  const t = useT();
  return (
    <span className="dk-rec-nav">
      <Button
        variant="quiet"
        size="icon-xs"
        className="disabled:opacity-30"
        title={previousLabel}
        aria-label={previousLabel}
        disabled={!neighbors.prevId}
        onClick={() => neighbors.prevId && onStep(neighbors.prevId)}
      >
        <CardIcon name="up" size={12} stroke={2.2} />
      </Button>
      {neighbors.position > 0 && <em>{t("{0} of {1}", neighbors.position, neighbors.total)}</em>}
      <Button
        variant="quiet"
        size="icon-xs"
        className="disabled:opacity-30"
        title={nextLabel}
        aria-label={nextLabel}
        disabled={!neighbors.nextId}
        onClick={() => neighbors.nextId && onStep(neighbors.nextId)}
      >
        <CardIcon name="down" size={12} stroke={2.2} />
      </Button>
    </span>
  );
}

/**
 * How a status reads on a card: still moving, done, closed out, held, retired
 * or not yet started.
 */
export type CardTone = "progress" | "success" | "ink" | "warn" | "danger" | "muted" | "draft";

/** What a status pill is marked with. */
export type CardStatusMark = "dot" | "check" | "pause" | "x";

const STATUS_STROKE: Record<Exclude<CardStatusMark, "dot">, number> = {
  check: 2.8,
  pause: 2.6,
  x: 2.6,
};

const TONE_CLASS: Record<CardTone, string> = {
  progress: "",
  success: "dk-s1",
  ink: "dk-s2",
  warn: "dk-st-warn",
  danger: "dk-st-bad",
  muted: "dk-off",
  draft: "dk-draft",
};

/** A record's status as a pill, marked with a dot, a tick, a pause or a cross. */
export function CardStatus({
  tone,
  mark = "dot",
  children,
}: {
  tone: CardTone;
  mark?: CardStatusMark;
  children: ReactNode;
}) {
  return (
    <span className={cn("dk-rec-st", TONE_CLASS[tone])}>
      {mark === "dot" ? <i /> : <CardIcon name={mark} size={11} stroke={STATUS_STROKE[mark]} />}
      {children}
    </span>
  );
}

/** Facts set side by side under a card's head, apart by dots. */
export function CardFacts({ children }: { children: readonly ReactNode[] }) {
  const shown = children.filter((child) => child !== null && child !== false && child !== "");
  return (
    <span className="dk-rec-q">
      {shown.map((child, index) => (
        // oxlint-disable-next-line react/no-array-index-key -- the facts are a fixed run of slots
        <Fragment key={index}>
          {index > 0 && <b aria-hidden="true">·</b>}
          <span>{child}</span>
        </Fragment>
      ))}
    </span>
  );
}

/** A titled block of a card, with what sits beside its title. */
export function CardSection({
  title,
  aside,
  children,
}: {
  title: string;
  aside?: ReactNode;
  children: ReactNode;
}) {
  return (
    <section className="dk-rec-sec">
      <div className="dk-rec-sh">
        <h3>{title}</h3>
        {aside}
      </div>
      {children}
    </section>
  );
}

/** One fact a card lists about its record: what it is and what it says. */
export type CardFact = { key: string; label: string; value: ReactNode };

/** A record's facts, two to a row, in a ruled grid: the plate, the VIN, who owns it. */
export function CardFactGrid({ facts }: { facts: readonly CardFact[] }) {
  return (
    <dl className="dk-rec-facts">
      {facts.map((fact) => (
        <div key={fact.key}>
          <dt>{fact.label}</dt>
          <dd>{fact.value}</dd>
        </div>
      ))}
    </dl>
  );
}

/** One figure a card reads about its record, marked when it is close to or past a line. */
export type CardStat = { key: string; label: string; value: ReactNode; state?: CardCheckState };

/** A record's figures side by side in a ruled strip, four to a row and two in a narrow card. */
export function CardStats({
  stats,
  className,
}: {
  stats: readonly CardStat[];
  className?: string;
}) {
  return (
    <dl className={cn("dk-rec-stats", className)}>
      {stats.map((stat) => (
        <div key={stat.key} className={cn(stat.state && stat.state !== "ok" && `dk-${stat.state}`)}>
          <dt>{stat.label}</dt>
          <dd>{stat.value}</dd>
        </div>
      ))}
    </dl>
  );
}

/** The bordered list a card's checks sit in. */
export function CardChecks({ children }: { children: ReactNode }) {
  return <div className="dk-rec-checks">{children}</div>;
}

export type CardCheckState = "ok" | "info" | "warn" | "bad";

/** Whether a check asks for a look: what is wrong or close, not what holds or is only worth knowing. */
export function needsLook(state: CardCheckState): boolean {
  return state === "bad" || state === "warn";
}

/** One check: whether it holds, what it is, what it found, and anything to do about it. */
export function CardCheck({
  state,
  title,
  detail,
  children,
}: {
  state: CardCheckState;
  title: ReactNode;
  detail: ReactNode;
  children?: ReactNode;
}) {
  return (
    <div className={cn("dk-rec-ck", `dk-${state}`)}>
      <span className="dk-rec-ci">
        <CardIcon
          name={
            state === "ok" ? "check" : state === "info" ? "info" : state === "warn" ? "warn" : "x"
          }
          size={state === "warn" ? 12 : 11}
          stroke={state === "warn" ? 2 : 2.8}
        />
      </span>
      <div className="dk-rec-cb">
        <b>{title}</b>
        <span>{detail}</span>
        {children}
      </div>
    </div>
  );
}

export type CardTimelineEntry = {
  id: string;
  text: ReactNode;
  /** Who and when, under the entry. */
  meta: ReactNode;
  /** What the entry changed, under its sentence. */
  detail?: ReactNode;
  /** The person's own entry, marked solid. */
  mine?: boolean;
};

/** What has happened to the record, oldest first, down a line. */
export function CardTimeline({ entries }: { entries: readonly CardTimelineEntry[] }) {
  return (
    <ol className="dk-rec-log">
      {entries.map((entry) => (
        <li key={entry.id} className={cn(entry.mine && "dk-you")}>
          <i />
          <span>{entry.text}</span>
          {entry.detail}
          <em>{entry.meta}</em>
        </li>
      ))}
    </ol>
  );
}

/** The card's actions, held to the bottom as it scrolls. */
export function CardBar({ children }: { children: ReactNode }) {
  return <div className="dk-rec-bar">{children}</div>;
}

/** A line in the bar saying why its actions stand as they do. */
export function CardBarHint({ children }: { children: ReactNode }) {
  return <span className="dk-rec-bh">{children}</span>;
}

/** A way out of the card to the record's own page, beside a section's title. */
export function CardLink({ to, children }: { to: string; children: ReactNode }) {
  return (
    <Link className="dk-rec-lnk" to={to}>
      {children}
      <CardIcon name="ext" size={11} />
    </Link>
  );
}

/**
 * A way to a record beside a section's title: opened in the card's place when
 * its kind has a card here (`onOpen`), or a link to its page when it does not.
 */
export function CardOpenLink({
  to,
  onOpen,
  children,
}: {
  to: string;
  onOpen?: () => void;
  children: ReactNode;
}) {
  return onOpen ? (
    <Button variant="bare" size="bare" className="dk-rec-lnk" onClick={onOpen}>
      {children}
      <CardIcon name="forward" size={11} />
    </Button>
  ) : (
    <CardLink to={to}>{children}</CardLink>
  );
}

/**
 * One record a card lists: opened in the card's place when its kind has a
 * card here (`useListedRecords`), or a link to the record's page when it does
 * not, so a listed record opens the same way a table's row does.
 */
export function CardRecordRow({
  to,
  onOpen,
  className,
  children,
}: {
  /** The record's own page. */
  to: string;
  /** Opens the record in the card's place; absent, the row links to its page. */
  onOpen?: () => void;
  className?: string;
  children: ReactNode;
}) {
  return onOpen ? (
    <Button variant="bare" size="bare" className={cn("dk-rec-row", className)} onClick={onOpen}>
      {children}
    </Button>
  ) : (
    <Link className={cn("dk-rec-row", className)} to={to}>
      {children}
    </Link>
  );
}

/** A section that could not be read says so in its own place, and the rest of the card stands. */
export function CardSectionError({ children }: { children: ReactNode }) {
  return (
    <div className="dk-ax-warn">
      <CardIcon name="warn" size={13} />
      <span>{children}</span>
    </div>
  );
}

/**
 * The foot of a list that has more to read: how many are shown, and the way to
 * read the next page, which says so while it is read.
 */
export function CardMore({
  shown,
  onMore,
  loading,
  label,
  loadingLabel,
}: {
  /** How many are shown, and of how many when that is known. */
  shown: ReactNode;
  onMore: () => void;
  loading: boolean;
  label: string;
  loadingLabel: string;
}) {
  return (
    <div className="dk-rec-more">
      <span>{shown}</span>
      <Button variant="quiet" size="xs" disabled={loading} onClick={onMore}>
        {loading ? loadingLabel : label}
      </Button>
    </div>
  );
}

/**
 * A share of something drawn as a meter under what it measures: the label on
 * the left, the reading and its figure on the right, and a bar that turns as it
 * nears its end. `share` is 0 to 1; past 1 the bar stays full.
 */
export function CardMeter({
  label,
  reading,
  figure,
  share,
  state,
}: {
  label: string;
  reading: ReactNode;
  figure?: ReactNode;
  share: number;
  state: CardCheckState;
}) {
  const percent = Math.round(Math.min(1, Math.max(0, share)) * 100);
  return (
    <div className={cn("dk-rec-gauge", state !== "ok" && `dk-${state}`)}>
      <div className="dk-rec-gauge-h">
        <span>{label}</span>
        <span>
          {reading}
          {figure !== undefined && <b>{figure}</b>}
        </span>
      </div>
      <div
        className="dk-rec-meter"
        role="meter"
        aria-label={label}
        aria-valuemin={0}
        aria-valuemax={100}
        aria-valuenow={percent}
      >
        <i style={{ width: `${percent}%` }} />
      </div>
    </div>
  );
}

/** Rows standing in for a section's list while it is read. */
export function CardSectionWait({ rows }: { rows: number }) {
  return (
    <div className="dk-rec-wait" aria-busy>
      {Array.from({ length: rows }, (_, index) => (
        // oxlint-disable-next-line react/no-array-index-key -- a fixed outline that never reorders
        <i key={index} className="dk-sk" />
      ))}
    </div>
  );
}
