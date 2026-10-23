import { AlertTriangleIcon, Link01Icon, Mail01Icon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { WithheldValue } from "@trenova/shared/components/withheld-value";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatDetentionMinutes } from "@trenova/shared/lib/detention";
import { cn } from "@trenova/shared/lib/utils";
import { Link } from "react-router";
import { CardIcon } from "../../record-card/record-card";
import { money } from "../../record-card/record-card-values";
import type {
  ClockModel,
  NoticeRow,
  ProofEvent,
  ProofFactor,
  ReceiptRow,
} from "./detention-occurrence-card";

/** How near an edge of the ruler, in percent, a moment's label sets off from that edge rather than centring on it. */
const EDGE = 10;

function share(model: Pick<ClockModel, "from" | "to">, at: number): number {
  return Math.min(100, Math.max(0, ((at - model.from) / (model.to - model.from)) * 100));
}

/**
 * The stop's clock as one ruler: the time before the clock started hatched,
 * free time faint, billable time solid; the notice flagged above it, green
 * once sent and dashed amber while due; and, while the truck is still there,
 * a mark at now. Each moment is named under the ruler, every other crowded
 * one a row lower, and the row is tall only when one is; a label near
 * either end sets off from that end so it stays on the ruler. Beneath, how long
 * the truck was there, the free time, and the billable time before and after
 * rounding.
 */
export function DetentionClock({ model }: { model: ClockModel }) {
  const t = useT();
  const stacked = model.ticks.some((tick) => tick.low);
  const billable =
    model.billable.minutes === model.billable.rounded
      ? formatDetentionMinutes(model.billable.minutes)
      : t(
          "{0} → {1}",
          formatDetentionMinutes(model.billable.minutes),
          formatDetentionMinutes(model.billable.rounded),
        );
  const dwell = formatDetentionMinutes(model.dwellMinutes);
  const free = formatDetentionMinutes(model.freeMinutes);
  return (
    <div className="dk-dtn-clock">
      <div
        className="dk-dtn-rule"
        role="img"
        aria-label={t("At the dock {0}, free time {1}, billable {2}", dwell, free, billable)}
      >
        {model.spans.map((span) => (
          <i
            key={span.kind}
            className={`dk-${span.kind}`}
            style={{
              left: `${share(model, span.from)}%`,
              width: `${share(model, span.to) - share(model, span.from)}%`,
            }}
          />
        ))}
        {model.flag && (
          <b
            className={`dk-flag dk-${model.flag.tone}`}
            style={{ left: `${share(model, model.flag.at)}%` }}
          >
            <span>{model.flag.label}</span>
          </b>
        )}
        {model.live && <em className="dk-now" style={{ left: "100%" }} />}
      </div>
      <ol className={cn("dk-dtn-ticks", stacked && "dk-two")} aria-label={t("The stop's moments")}>
        {model.ticks.map((tick) => (
          <li
            key={tick.key}
            className={cn(
              share(model, tick.at) < EDGE && "dk-start",
              share(model, tick.at) > 100 - EDGE && "dk-end",
              tick.low && "dk-low",
            )}
            style={{ left: `${share(model, tick.at)}%` }}
          >
            <b>{tick.label}</b>
            <span>{tick.time}</span>
          </li>
        ))}
      </ol>
      <dl className="dk-dtn-sum">
        <div>
          <dt>{t("At the dock")}</dt>
          <dd>{dwell}</dd>
        </div>
        <div>
          <dt>{t("Free time")}</dt>
          <dd>{free}</dd>
        </div>
        <div className="dk-bill">
          <dt>{model.live ? t("Billable so far") : t("Billable")}</dt>
          <dd>{billable}</dd>
        </div>
      </dl>
    </div>
  );
}

/**
 * The calculation as the server recorded it, a receipt line a step: the step,
 * the engine's own words for what it did and the policy term behind it, and
 * the minutes, money or moment it settled on; the charge at its foot.
 */
export function DetentionReceipt({ rows, total }: { rows: readonly ReceiptRow[]; total: string }) {
  const t = useT();
  return (
    <>
      <ol className="dk-dtn-rcpt">
        {rows.map((row) => (
          <li key={row.key} className={cn(row.cut && "dk-cut")}>
            <b className="dk-step">{row.label}</b>
            <span className="dk-what">
              <span>{row.detail}</span>
              {row.term && <em>{row.term}</em>}
            </span>
            <span className="dk-val">{row.value}</span>
          </li>
        ))}
      </ol>
      <dl className="dk-rec-stmt-foot">
        <dt>{t("Charge")}</dt>
        <dd>{total}</dd>
      </dl>
    </>
  );
}

/**
 * Who pays and who is paid: the charge as one bar, the worker's detention pay
 * as its share, what is kept as the rest; beneath, each as a figure.
 */
export function DetentionPaySplit({
  bill,
  pay,
  margin,
  payNote,
  payShare,
  currency,
}: {
  bill: number;
  pay: number | null;
  margin: number | null;
  payNote: string;
  payShare: number;
  currency: string;
}) {
  const t = useT();
  const payPercent = payShare * 100;
  const withheld = pay === null || margin === null;
  return (
    <div className="dk-dtn-split">
      {!withheld && (
        <div className="dk-dtn-splitbar" aria-hidden>
          <i className="dk-pay" style={{ width: `${payPercent}%` }} />
          {margin > 0 && <i className="dk-margin" style={{ width: `${100 - payPercent}%` }} />}
        </div>
      )}
      <dl>
        <div>
          <dt>{t("Bill the customer")}</dt>
          <dd>{money(bill, currency)}</dd>
        </div>
        <div className="dk-pay">
          <dt>{t("Worker's detention pay")}</dt>
          <dd>
            {pay === null ? <WithheldValue group="Margins" /> : money(pay, currency)}
            <span>{payNote}</span>
          </dd>
        </div>
        {margin === null ? (
          <div>
            <dt>{t("Kept")}</dt>
            <dd>
              <WithheldValue group="Margins" />
            </dd>
          </div>
        ) : (
          <div className={cn(margin >= 0 ? "dk-margin" : "dk-loss")}>
            <dt>{margin >= 0 ? t("Kept") : t("Short")}</dt>
            <dd>{money(margin, currency)}</dd>
          </div>
        )}
      </dl>
    </div>
  );
}

/**
 * The notice the policy asks for: a warning when one is overdue or failed,
 * what the policy asks in a sentence, and each notice sent or due.
 */
export function DetentionNotices({
  slip,
  rule,
  rows,
}: {
  slip: { title: string; body: string } | null;
  rule: string;
  rows: readonly NoticeRow[];
}) {
  const t = useT();
  return (
    <>
      {slip && (
        <div className="dk-dtn-slip" role="alert">
          <AlertTriangleIcon className="size-3.5" aria-hidden />
          <span>
            <b>{slip.title}</b>
            {slip.body}
          </span>
        </div>
      )}
      {rule && <p className="dk-dtn-rule-line">{rule}</p>}
      {rows.length > 0 ? (
        <ul className="dk-dtn-notices">
          {rows.map((row) => (
            <li key={row.key} className={`dk-${row.tone}`}>
              <Mail01Icon className="size-3.5" aria-hidden />
              <span className="dk-what">
                <b>{row.title}</b>
                <span>{row.to}</span>
              </span>
              <span className="dk-when">{row.when}</span>
            </li>
          ))}
        </ul>
      ) : (
        <p className="dk-dtn-none">{t("No notice has gone out for this stop.")}</p>
      )}
    </>
  );
}

/**
 * Whether the charge would hold up to a dispute: the score and its band on a
 * meter, each factor with its points and, where it fell short, what would
 * earn them; then the evidence in the order it happened, and whether the
 * chain still verifies.
 */
export function DetentionProof({
  score,
  band,
  weak,
  factors,
  events,
  chain,
  chainValid,
}: {
  score: number;
  band: string;
  weak: boolean;
  factors: readonly ProofFactor[];
  events: readonly ProofEvent[];
  chain: string;
  chainValid: boolean;
}) {
  const t = useT();
  return (
    <>
      <div className="dk-dtn-proof">
        <div className={cn("dk-dtn-score", weak && "dk-warn")}>
          <b>{score}</b>
          <span>{band}</span>
          <i
            role="meter"
            aria-label={t("Collectability")}
            aria-valuemin={0}
            aria-valuemax={100}
            aria-valuenow={score}
          >
            <em style={{ width: `${score}%` }} />
          </i>
        </div>
        <ul className="dk-dtn-factors">
          {factors.map((factor) => (
            <li key={factor.key} className={cn(factor.short && "dk-short")}>
              <span className="dk-what">
                <b>{factor.label}</b>
                {factor.remedy && <span>{factor.remedy}</span>}
              </span>
              <span className="dk-pts">{factor.points}</span>
            </li>
          ))}
        </ul>
      </div>
      {events.length > 0 ? (
        <ol className="dk-dtn-chain" aria-label={t("Evidence")}>
          {events.map((event) => (
            <li key={event.key}>
              <i aria-hidden />
              <span className="dk-what">
                <b>{event.title}</b>
                <span>{event.summary}</span>
              </span>
              <span className="dk-when">{event.when}</span>
            </li>
          ))}
        </ol>
      ) : (
        <p className="dk-dtn-none">{t("No evidence has been recorded yet.")}</p>
      )}
      <p className={cn("dk-dtn-chainok", !chainValid && "dk-bad")}>
        <Link01Icon className="size-3" aria-hidden />
        {chain}
      </p>
    </>
  );
}

/** One record the stop belongs to, as a tile: opened in the card's place, or a link to its page. */
export type StopRef = {
  key: string;
  kind: string;
  name: string;
  line: string;
  to: string | null;
  onOpen?: () => void;
};

/** The facility and the customer, each a tile that opens it. */
export function DetentionStopRefs({ refs }: { refs: readonly StopRef[] }) {
  return (
    <div className="dk-dtn-where">
      {refs.map((ref) => {
        const body = (
          <>
            <span className="dk-k">{ref.kind}</span>
            <b>{ref.name}</b>
            {ref.line && <span>{ref.line}</span>}
          </>
        );
        if (ref.onOpen) {
          return (
            <Button
              key={ref.key}
              variant="bare"
              size="bare"
              className="dk-dtn-ref"
              onClick={ref.onOpen}
            >
              {body}
              <CardIcon name="forward" size={11} />
            </Button>
          );
        }
        if (ref.to) {
          return (
            <Link key={ref.key} className="ui-focus-ring dk-dtn-ref" to={ref.to}>
              {body}
              <CardIcon name="ext" size={11} />
            </Link>
          );
        }
        return (
          <div key={ref.key} className="dk-dtn-ref">
            {body}
          </div>
        );
      })}
    </div>
  );
}

/**
 * Why a charge was waived or what the customer disputes, kept as written,
 * with who, when and how far it had got.
 */
export function DetentionSettledSlip({
  note,
  facts,
}: {
  note: string;
  facts: readonly { key: string; label: string; value: string }[];
}) {
  return (
    <div className="dk-rec-voidslip">
      {note && <blockquote>{note}</blockquote>}
      <dl>
        {facts.map((fact) => (
          <div key={fact.key}>
            <dt>{fact.label}</dt>
            <dd>{fact.value}</dd>
          </div>
        ))}
      </dl>
    </div>
  );
}
