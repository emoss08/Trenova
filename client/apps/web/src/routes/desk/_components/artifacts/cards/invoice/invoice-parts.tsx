import { useClipped } from "@/hooks/use-clipped";
import { Button } from "@trenova/shared/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import {
  BankIcon,
  BookOpen01Icon,
  CheckDone01Icon,
  FileMinus02Icon,
  Mail01Icon,
  SwitchHorizontal01Icon,
  type IconComponent,
} from "@trenova/shared/components/icons";
import { formatList } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { ReactNode } from "react";
import { Link } from "react-router";
import { CardActionButton } from "../../record-card/card-action";
import { CardIcon } from "../../record-card/record-card";
import type {
  ChannelReading,
  ChargeRow,
  ClosedCase,
  FootRow,
  InvoiceKind,
  OpenCase,
  OwedSegment,
  RemitRow,
  TermsRuler,
  VoidNote,
} from "./invoice-card";

/** The invoice as a receipt slip: what kind of bill it is over a torn foot. */
export function InvoiceSlip({ mark }: { mark: InvoiceKind["mark"] }) {
  return (
    <span className="dk-inv-slip" aria-hidden>
      <b>{mark}</b>
      <i />
      <i />
    </span>
  );
}

/**
 * Text cut off with an ellipsis where it would grow its box, told in full in a
 * tooltip only when it was cut. The text takes focus while it is cut, so the
 * whole of it can be read from the keyboard too.
 */
export function ClippedText({ text, className }: { text: string; className?: string }) {
  const [ref, clipped] = useClipped<HTMLSpanElement>();
  return (
    <Tooltip disabled={!clipped}>
      <TooltipTrigger
        render={
          <span
            key={text}
            ref={ref}
            className={cn("dk-inv-clip", className)}
            tabIndex={clipped ? 0 : undefined}
          />
        }
      >
        {text}
      </TooltipTrigger>
      <TooltipContent side="top">{text}</TooltipContent>
    </Tooltip>
  );
}

/**
 * The total as one strip, paid, credited, in dispute and still open, with each
 * part's amount keyed beneath it; a credit memo's as applied against left.
 */
export function OwedStrip({ segments }: { segments: readonly OwedSegment[] }) {
  const t = useT();
  const label = formatList(
    segments.map((segment) => t("{0} {1}", segment.label, segment.amount)),
    "unit",
  );
  return (
    <div className="dk-inv-owed">
      <div className="dk-inv-strip" role="img" aria-label={label}>
        {segments
          .filter((segment) => !segment.zero)
          .map((segment) => (
            <i
              key={segment.key}
              className={`dk-${segment.key}`}
              style={{ flexGrow: segment.share }}
            />
          ))}
      </div>
      <dl
        className={cn("dk-inv-keys", segments.length === 2 && "dk-two")}
        aria-label={t("What is owed")}
      >
        {segments.map((segment) => (
          <div key={segment.key} className={cn(`dk-${segment.key}`, segment.zero && "dk-zero")}>
            <dt>{segment.label}</dt>
            <dd>{segment.amount}</dd>
          </div>
        ))}
      </dl>
    </div>
  );
}

function percent(position: number): string {
  return `${Math.min(100, Math.max(0, position * 100)).toFixed(2)}%`;
}

/** Where a label under the ruler sits: at its mark, but never over the labels at either end. */
function labelAt(position: number): string {
  return percent(Math.min(0.7, Math.max(0.3, position)));
}

/**
 * The terms as a ruler: from the invoice date to the due date with today on
 * the way, past it in amber to today once late, or a draft's waiting days in
 * amber up to today with the rest of its terms still ahead.
 */
export function TermsRulerView({ ruler }: { ruler: TermsRuler }) {
  const due = ruler.duePosition ?? 1;
  const now = ruler.nowPosition;
  const late = ruler.kind === "late";
  const unposted = ruler.kind === "unposted";
  return (
    <div className={cn("dk-inv-terms", `dk-${ruler.kind}`)}>
      <div className="dk-inv-ruler" aria-hidden>
        {late && (
          <>
            <i className="dk-term" style={{ width: percent(due) }} />
            <i className="dk-over" style={{ left: percent(due), width: percent(1 - due) }} />
          </>
        )}
        {unposted && (
          <>
            <i className="dk-wait" style={{ width: percent(now) }} />
            <i className="dk-ahead" style={{ left: percent(now), width: percent(1 - now) }} />
          </>
        )}
        {ruler.kind === "running" && (
          <>
            <i className="dk-term" style={{ width: percent(now) }} />
            <i className="dk-ahead" style={{ left: percent(now), width: percent(1 - now) }} />
          </>
        )}
        <i className="dk-tick dk-start" />
        {(late || unposted) && <i className="dk-tick dk-due" style={{ left: percent(due) }} />}
        {!late && <i className="dk-tick dk-end" />}
        <i
          className={cn("dk-pin", (late || unposted) && "dk-warn")}
          style={{ left: percent(late ? 1 : now) }}
        />
      </div>
      <div className="dk-inv-marks">
        <span className="dk-start">
          <b>{ruler.startLabel}</b>
          {ruler.startDay}
        </span>
        {unposted && (
          <span className="dk-mid dk-warn" style={{ left: labelAt(now) }}>
            <b>{ruler.nowLabel}</b>
            {ruler.nowDay}
          </span>
        )}
        {late && (
          <span className="dk-mid" style={{ left: labelAt(due) }}>
            <b>{ruler.dueLabel}</b>
            {ruler.dueDay}
          </span>
        )}
        <span className={cn("dk-end", late && "dk-warn")}>
          <b>{ruler.endLabel}</b>
          {ruler.endDay}
        </span>
      </div>
    </div>
  );
}

/**
 * The formula template that rated the freight, its name cut to fit and told in
 * full on hover, opening the template's card when it is one the Desk draws.
 */
function FormulaTemplateName({ name, onOpen }: { name: string; onOpen?: () => void }) {
  const t = useT();
  const [ref, clipped] = useClipped<HTMLSpanElement>();
  const body = (
    <>
      <i className="dk-inv-fx-mark" aria-hidden>
        ƒ
      </i>
      <span key={name} ref={ref} className="dk-inv-clip">
        {name}
      </span>
      {onOpen && <CardIcon name="forward" size={10} />}
    </>
  );
  const trigger = onOpen ? (
    <Button
      variant="bare"
      size="bare"
      className="dk-inv-fx"
      onClick={onOpen}
      aria-label={t("Open formula template {0}", name)}
    />
  ) : (
    <span className="dk-inv-fx dk-still" tabIndex={clipped ? 0 : undefined} />
  );
  return (
    <Tooltip disabled={!clipped}>
      <TooltipTrigger render={trigger}>{body}</TooltipTrigger>
      <TooltipContent side="top">{name}</TooltipContent>
    </Tooltip>
  );
}

function templateOpener(
  templateId: string | null,
  onOpenTemplate: ((templateId: string) => void) | undefined,
): (() => void) | undefined {
  return templateId && onOpenTemplate ? () => onOpenTemplate(templateId) : undefined;
}

/**
 * The charges as the bill reads them, one row for each way a charge was
 * billed, then the foot from the total down to the balance due.
 */
export function ChargeBill({
  rows,
  foot,
  note,
  onOpenTemplate,
}: {
  rows: readonly ChargeRow[];
  foot: readonly FootRow[];
  note?: string;
  /** Opens a formula template's card; absent, the template's name is only read. */
  onOpenTemplate?: (templateId: string) => void;
}) {
  const t = useT();
  return (
    <div className="dk-inv-bill">
      <table>
        <thead>
          <tr>
            <th scope="col">{t("Charge")}</th>
            <th scope="col">{t("Rated by")}</th>
            <th scope="col">{t("Amount")}</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((row) => (
            <tr key={row.key}>
              <td>
                <span className="dk-inv-desc">
                  <span>
                    {row.name}
                    {row.code && <code className="dk-inv-code">{row.code}</code>}
                  </span>
                  {row.detail && <em>{row.detail}</em>}
                </span>
              </td>
              <td className="dk-inv-basis">
                {row.basis.kind === "formula" ? (
                  <FormulaTemplateName
                    name={row.basis.name}
                    onOpen={templateOpener(row.basis.templateId, onOpenTemplate)}
                  />
                ) : (
                  row.basis.text
                )}
              </td>
              <td className="dk-inv-amt">{row.amount}</td>
            </tr>
          ))}
        </tbody>
      </table>
      <dl className="dk-inv-foot">
        {foot.map((row) => (
          <div key={row.label} className={`dk-${row.kind}`}>
            <dt>{row.label}</dt>
            <dd>{row.value}</dd>
          </div>
        ))}
      </dl>
      {note && <p className="dk-inv-note">{note}</p>}
    </div>
  );
}

const REMIT_ICON: Record<RemitRow["kind"], IconComponent> = {
  payment: BankIcon,
  credit: FileMinus02Icon,
};

/**
 * What settled an invoice, or what a credit memo settled, as remittance stubs:
 * a payment reversed or a credit taken back kept and struck through, a credit
 * memo or an invoice opening in the card's place.
 */
export function RemitList({
  rows,
  onOpen,
}: {
  rows: readonly RemitRow[];
  onOpen?: (row: RemitRow) => (() => void) | undefined;
}) {
  return (
    <ul className="dk-inv-remit">
      {rows.map((row) => {
        const Icon = REMIT_ICON[row.kind];
        const open = onOpen?.(row);
        const body = (
          <>
            <span className="dk-inv-mark" aria-hidden>
              <Icon className="size-4" />
            </span>
            <span className="dk-inv-what">
              <b>
                <span>{row.title}</span>
                {row.tag && <span className="dk-inv-tag dk-warn">{row.tag}</span>}
              </b>
              <span>{row.line}</span>
            </span>
            <span className="dk-inv-amount">{row.amount}</span>
            {open && <CardIcon name="forward" size={12} />}
          </>
        );
        return (
          <li key={row.key} className={cn(row.undone && "dk-undone")}>
            {open ? (
              <Button variant="bare" size="bare" className="dk-inv-stub" onClick={open}>
                {body}
              </Button>
            ) : (
              <div className="dk-inv-stub">{body}</div>
            )}
          </li>
        );
      })}
    </ul>
  );
}

/** The open dispute as a case slip, the ones already closed beneath it with their outcomes. */
export function DisputeCases({
  open,
  closed,
}: {
  open: OpenCase | null;
  closed: readonly ClosedCase[];
}) {
  return (
    <>
      {open && (
        <div className="dk-inv-case" role="note" aria-label={open.reason}>
          <div className="dk-inv-case-h">
            <b>{open.reason}</b>
            <span>{open.amount}</span>
          </div>
          {open.notes && <blockquote>{open.notes}</blockquote>}
          <em>{open.opened}</em>
        </div>
      )}
      {closed.length > 0 && (
        <ul className={cn("dk-inv-cases", !open && "dk-alone")}>
          {closed.map((item) => (
            <li key={item.key}>
              <b>{item.title}</b>
              <span>{item.amount}</span>
              <em>{item.line}</em>
            </li>
          ))}
        </ul>
      )}
    </>
  );
}

/** Why the invoice was voided, kept as written, with what became of its freight and when. */
export function VoidSlip({ note, reversal }: { note: VoidNote; reversal?: ReactNode }) {
  const t = useT();
  return (
    <div className="dk-inv-void">
      {note.reason && <blockquote>{note.reason}</blockquote>}
      <dl>
        {note.freight && (
          <div>
            <dt>{t("Freight")}</dt>
            <dd>{note.freight}</dd>
          </div>
        )}
        {reversal && (
          <div>
            <dt>{t("Reversal")}</dt>
            <dd>{reversal}</dd>
          </div>
        )}
        <div>
          <dt>{t("Voided")}</dt>
          <dd>{note.when}</dd>
        </div>
      </dl>
    </div>
  );
}

const CHANNEL_ICON = {
  email: Mail01Icon,
  edi: SwitchHorizontal01Icon,
  books: BookOpen01Icon,
} as const satisfies Record<string, IconComponent>;

/**
 * What a slip leads to: a record opened in the card's place, a page here, the
 * system outside, or a step the person takes on the channel from the card.
 */
export type ChannelAct =
  | { kind: "open"; label: string; onClick: () => void }
  | { kind: "page"; label: string; to: string }
  | { kind: "outside"; label: string; href: string }
  | { kind: "do"; label: string; onClick: () => void; blocker: string | null; busy: boolean };

export type ChannelSlip = ChannelReading & {
  key: keyof typeof CHANNEL_ICON;
  label: string;
  actions?: readonly ChannelAct[];
};

function ChannelAction({ action }: { action: ChannelAct }) {
  switch (action.kind) {
    case "do":
      return (
        <CardActionButton
          variant="outline"
          size="xxs"
          blocker={action.blocker}
          busy={action.busy}
          onClick={action.onClick}
        >
          {action.label}
        </CardActionButton>
      );
    case "open":
      return (
        <Button variant="bare" size="bare" className="dk-inv-chan-a" onClick={action.onClick}>
          {action.label}
          <CardIcon name="forward" size={10} />
        </Button>
      );
    case "page":
      return (
        <Link className="dk-inv-chan-a" to={action.to}>
          {action.label}
          <CardIcon name="forward" size={10} />
        </Link>
      );
    case "outside":
      return (
        <a className="dk-inv-chan-a" href={action.href} target="_blank" rel="noreferrer noopener">
          {action.label}
          <CardIcon name="ext" size={10} />
        </a>
      );
  }
}

/**
 * Where the invoice went, one slip for each way out: email, the EDI 210 and
 * the accounting system. Each slip's line is cut to one row and told in full
 * on hover, so a long reason never grows the slip.
 */
export function ChannelSlips({ slips }: { slips: readonly ChannelSlip[] }) {
  return (
    <div className={cn("dk-inv-went", slips.length === 2 && "dk-two")}>
      {slips.map((slip) => {
        const Icon = CHANNEL_ICON[slip.key];
        return (
          <div key={slip.key} className={cn("dk-inv-chan", `dk-${slip.tone}`)}>
            <span className="dk-inv-chan-h">
              <Icon className="size-3.5" aria-hidden />
              {slip.label}
            </span>
            <b>
              {slip.tone === "ok" ? (
                <CheckDone01Icon className="size-3.5" aria-hidden />
              ) : (
                <i aria-hidden />
              )}
              {slip.state}
            </b>
            {slip.detail && (
              <ClippedText text={slip.detail} className={cn(slip.why && "dk-inv-why")} />
            )}
            {slip.actions && slip.actions.length > 0 && (
              <span className="dk-inv-chan-acts">
                {slip.actions.map((action) => (
                  <ChannelAction key={action.label} action={action} />
                ))}
              </span>
            )}
          </div>
        );
      })}
    </div>
  );
}
