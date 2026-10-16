import {
  AlertCircleIcon,
  AlertTriangleIcon,
  ArrowDownIcon,
  ArrowLeftIcon,
  ArrowRightIcon,
  ArrowUpIcon,
  Certificate01Icon,
  ClockIcon,
  Key01Icon,
  Mail01Icon,
  Phone01Icon,
} from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatNumber } from "@trenova/shared/i18n/format";
import { cn, telHref } from "@trenova/shared/lib/utils";
import type { ReactNode } from "react";
import { Link } from "react-router";
import { CardAvatar, CardIcon, type CardCheckState } from "../../record-card/record-card";
import type {
  AckRuler,
  AttentionItem,
  CertificateReading,
  CredentialChip,
  Exchange,
  LaneSet,
  MappingChip,
  PartnerAddress,
  PulseDay,
  PulseFigure,
  ReadinessCheck,
  TransportEnds,
  TransportLine,
  UsageReading,
} from "./edi-partner-card";

/**
 * The partner's port: one square with its two ways, in and out, each lit when
 * documents travel that way and faint when it is turned off.
 */
export function PartnerPort({ inbound, outbound }: { inbound: boolean; outbound: boolean }) {
  return (
    <span className="dk-edp-port" aria-hidden="true">
      <i className={cn(!inbound && "dk-shut")}>
        <ArrowDownIcon className="size-4.5" />
      </i>
      <i className={cn(!outbound && "dk-shut")}>
        <ArrowUpIcon className="size-4.5" />
      </i>
    </span>
  );
}

function Lane({
  direction,
  label,
  sets,
  empty,
  counted,
}: {
  direction: "out" | "in";
  label: string;
  sets: readonly LaneSet[];
  empty: string;
  counted: boolean;
}) {
  return (
    <div className={cn("dk-edp-lane", `dk-${direction}`)}>
      <div className="dk-edp-wire">
        <span>{label}</span>
      </div>
      {sets.length > 0 ? (
        <ul aria-label={label}>
          {sets.map((set) => (
            <li key={set.key} className={cn(set.state && `dk-${set.state}`, set.off && "dk-off")}>
              <span className={cn("dk-edp-stamp", set.test && "dk-test")}>
                <b>{set.set}</b>
                {set.test && <em aria-label="test">T</em>}
              </span>
              <span className="dk-what">
                <b>{set.name}</b>
                <span>{set.note}</span>
              </span>
              {counted && (
                <span className="dk-n">{set.count === null ? "—" : formatNumber(set.count)}</span>
              )}
            </li>
          ))}
        </ul>
      ) : (
        <p className="dk-edp-off">{empty}</p>
      )}
    </div>
  );
}

/**
 * What the two sides trade: our end and theirs, and between them a lane each
 * way with every transaction set that travels it, how many went in the window
 * and what answer it waits for.
 */
export function PartnerExchange({ exchange, note }: { exchange: Exchange; note?: string }) {
  const t = useT();
  return (
    <>
      <div className="dk-edp-x">
        <div className="dk-edp-end dk-us">
          <b>{exchange.us.name}</b>
          {exchange.us.id && <code>{exchange.us.id}</code>}
        </div>
        <div className="dk-edp-lanes">
          <Lane
            direction="out"
            label={t("We send")}
            sets={exchange.outbound}
            empty={t("Nothing goes out to them")}
            counted={exchange.counted}
          />
          <Lane
            direction="in"
            label={t("They send")}
            sets={exchange.inbound}
            empty={t("Nothing comes in from them")}
            counted={exchange.counted}
          />
        </div>
        <div className="dk-edp-end dk-them">
          <b>{exchange.them.name}</b>
          {exchange.them.id && <code>{exchange.them.id}</code>}
        </div>
      </div>
      {note && <p className="dk-edp-xnote">{note}</p>}
    </>
  );
}

function Figures({ figures, className }: { figures: readonly PulseFigure[]; className: string }) {
  return (
    <dl className={className}>
      {figures.map((figure) => (
        <div key={figure.key} className={cn(figure.state && `dk-${figure.state}`)}>
          <dt>{figure.label}</dt>
          <dd>{figure.value}</dd>
        </div>
      ))}
    </dl>
  );
}

function share(value: number, scale: number): number {
  return scale > 0 ? Math.min(100, Math.max(0, (value / scale) * 100)) : 0;
}

/**
 * The window's traffic: its figures, a pair of columns a day (what went out,
 * what came in, the failures capped red), and how quickly acknowledgments came
 * back against what was promised.
 */
export function PartnerPulse({
  figures,
  days,
  ruler,
  acks,
}: {
  figures: readonly PulseFigure[];
  days: readonly PulseDay[];
  ruler: AckRuler | null;
  acks: readonly PulseFigure[];
}) {
  const t = useT();
  const peak = Math.max(1, ...days.flatMap((day) => [day.sent, day.received]));
  const height = (value: number) => (value > 0 ? Math.max(2, (value / peak) * 100) : 0);
  return (
    <div className="dk-edp-pulse">
      <Figures figures={figures} className="dk-edp-figs" />
      <div className="dk-edp-cols" role="img" aria-label={t("Messages a day, last 14 days")}>
        {days.map((day) => (
          <div key={day.start} className={cn("dk-edp-day", day.today && "dk-today")}>
            <span>
              <i className="dk-o" style={{ height: `${height(day.sent)}%` }}>
                {day.failed > 0 && day.sent > 0 && (
                  <em style={{ height: `${Math.max(8, (day.failed / day.sent) * 100)}%` }} />
                )}
              </i>
              <i className="dk-i" style={{ height: `${height(day.received)}%` }} />
            </span>
          </div>
        ))}
      </div>
      <div className="dk-edp-key" aria-hidden="true">
        <span className="dk-o">{t("Sent")}</span>
        <span className="dk-i">{t("Received")}</span>
        <span className="dk-f">{t("Failed")}</span>
      </div>
      {ruler && (
        <div className="dk-edp-ack">
          <div className="dk-edp-ack-h">
            <b>{t("Acknowledgments")}</b>
            <span>{ruler.line}</span>
          </div>
          <div className="dk-edp-ackrule" role="img" aria-label={ruler.line}>
            {ruler.slowest !== null && (
              <i className="dk-span" style={{ width: `${share(ruler.slowest, ruler.scale)}%` }} />
            )}
            {ruler.average !== null && (
              <b
                className={cn("dk-pin", share(ruler.average, ruler.scale) < 8 && "dk-lead")}
                style={{ left: `${share(ruler.average, ruler.scale)}%` }}
              >
                <span>{t("avg")}</span>
              </b>
            )}
            {ruler.slowest !== null && (
              <b
                className="dk-pin dk-p95"
                style={{ left: `${share(ruler.slowest, ruler.scale)}%` }}
              >
                <span>{t("p95")}</span>
              </b>
            )}
            {ruler.promise !== null && (
              <b className="dk-sla" style={{ left: `${share(ruler.promise, ruler.scale)}%` }}>
                <span>{t("Promised")}</span>
              </b>
            )}
          </div>
          <Figures figures={acks} className="dk-edp-acks" />
        </div>
      )}
    </div>
  );
}

/** Something with the partner that wants a person, with the records it is about beneath. */
export function PartnerNeed({ item, children }: { item: AttentionItem; children?: ReactNode }) {
  const Icon = item.state === "bad" ? AlertCircleIcon : AlertTriangleIcon;
  return (
    <li className={`dk-${item.state}`}>
      <div className="dk-edp-need-h">
        <Icon className="size-3.5" aria-hidden />
        <span className="dk-what">
          <b>{item.title}</b>
          {item.body && <span>{item.body}</span>}
        </span>
        {item.when && <span className="dk-when">{item.when}</span>}
      </div>
      {item.quote && (
        <blockquote>
          <code>{item.quote}</code>
        </blockquote>
      )}
      {children}
    </li>
  );
}

export function PartnerNeeds({ children }: { children: ReactNode }) {
  return <ul className="dk-edp-needs">{children}</ul>;
}

/** A transaction set or a file kind as a small tag at the start of a listed row. */
export function RowTag({ children }: { children: ReactNode }) {
  return <span className="dk-edp-mini">{children}</span>;
}

/** Which way a listed message travelled. */
export function RowWay({ direction }: { direction: string }) {
  const t = useT();
  const Icon = direction === "Outbound" ? ArrowRightIcon : ArrowLeftIcon;
  return (
    <span
      className="dk-edp-dir"
      title={direction === "Outbound" ? t("Sent to them") : t("Received from them")}
    >
      <Icon className="size-3" aria-hidden />
    </span>
  );
}

/** A row's status word, coloured by how it stands. */
export function RowState({
  state,
  children,
}: {
  state: CardCheckState | null;
  children: ReactNode;
}) {
  return <span className={cn("dk-rec-sub", state && `dk-${state}`)}>{children}</span>;
}

/**
 * How we reach the partner: the line as a patch with its method and name, how
 * it last went, each end's identity, the settings a person checks it by, its
 * certificates with how much life they have left, and its credentials, named
 * and said to be on file or not, never shown.
 */
export function TransportPatch({
  method,
  name,
  line,
  ends,
  children,
}: {
  method: string;
  name: string;
  line: TransportLine;
  ends: TransportEnds | null;
  children: ReactNode;
}) {
  return (
    <div className="dk-edp-patch">
      <div className="dk-edp-patch-h">
        <span className="dk-edp-meth">{method}</span>
        <b>{name}</b>
        {line && (
          <span className={cn("dk-edp-last", `dk-${line.state}`)}>
            {line.state === "ok" ? (
              <CardIcon name="check" size={11} stroke={2.6} />
            ) : line.state === "warn" ? (
              <AlertTriangleIcon className="size-3" aria-hidden />
            ) : null}
            {line.text}
          </span>
        )}
      </div>
      {ends && (
        <div className="dk-edp-ends">
          <div>
            <span>{ends.usLabel}</span>
            <code>{ends.us}</code>
          </div>
          <i aria-hidden="true">
            <ArrowRightIcon className="size-3" />
          </i>
          <div>
            <span>{ends.themLabel}</span>
            <code>{ends.them}</code>
          </div>
        </div>
      )}
      {children}
    </div>
  );
}

export function CertificateTile({ reading }: { reading: CertificateReading }) {
  return (
    <div className={cn("dk-edp-cert", `dk-${reading.state}`)}>
      <Certificate01Icon className="size-3.5" aria-hidden />
      <span className="dk-what">
        <b>{reading.title}</b>
        {reading.body && <span>{reading.body}</span>}
      </span>
      <span className="dk-edp-life" aria-hidden="true">
        <i style={{ width: `${Math.round(reading.used * 100)}%` }} />
      </span>
      {reading.when && <span className="dk-when">{reading.when}</span>}
    </div>
  );
}

export function CredentialChips({ chips }: { chips: readonly CredentialChip[] }) {
  const t = useT();
  if (chips.length === 0) return null;
  return (
    <ul className="dk-edp-creds" aria-label={t("Credentials")}>
      {chips.map((chip) => (
        <li key={chip.key} className={chip.onFile ? "dk-ok" : "dk-warn"}>
          <Key01Icon className="size-3" aria-hidden />
          <span>{chip.label}</span>
          <b>{chip.onFile ? t("On file") : t("Missing")}</b>
        </li>
      ))}
    </ul>
  );
}

/**
 * How every interchange with the partner is addressed, from us to them: the
 * ISA ids with their qualifiers, production or test, the GS codes, and the
 * separators the X12 is written with.
 */
export function AddressLabel({ address, usage }: { address: PartnerAddress; usage: UsageReading }) {
  const t = useT();
  const production = address.usage === "P";
  return (
    <div className="dk-edp-label">
      <div className="dk-edp-route">
        <div>
          <span>{t("ISA from")}</span>
          <code>
            <em>{address.us.qualifier}</em>
            {address.us.id}
          </code>
        </div>
        <i aria-hidden="true">
          <ArrowRightIcon className="size-3" />
        </i>
        <div>
          <span>{t("ISA to")}</span>
          <code>
            <em>{address.them.qualifier}</em>
            {address.them.id}
          </code>
        </div>
        <span className={cn("dk-edp-usage", production ? "dk-prod" : "dk-test")}>
          <b aria-hidden="true">{address.usage}</b>
          {production ? t("Production") : t("Test")}
        </span>
      </div>
      <div className="dk-edp-route dk-gs">
        <div>
          <span>{t("GS from")}</span>
          <code>{address.us.application}</code>
        </div>
        <i aria-hidden="true">
          <ArrowRightIcon className="size-3" />
        </i>
        <div>
          <span>{t("GS to")}</span>
          <code>{address.them.application}</code>
        </div>
      </div>
      <ul className="dk-edp-seps" aria-label={t("Separators")}>
        <li>
          <kbd>{address.separators.element}</kbd>
          <span>{t("Element")}</span>
        </li>
        <li>
          <kbd>{address.separators.segment}</kbd>
          <span>{t("Segment")}</span>
        </li>
        <li>
          <kbd>{address.separators.component}</kbd>
          <span>{t("Component")}</span>
        </li>
        {address.separators.repetition && (
          <li>
            <kbd>{address.separators.repetition}</kbd>
            <span>{t("Repetition")}</span>
          </li>
        )}
      </ul>
      {(address.differing > 0 || usage === "mixed") && (
        <p className="dk-edp-differs">
          {address.differing > 0
            ? t(
                "{0, plural, one {# other set is addressed differently} other {# other sets are addressed differently}}; each is on the partner's page.",
                address.differing,
              )
            : t("Some sets still go out marked for test.")}
        </p>
      )}
    </div>
  );
}

/** Whether the partner is set up to trade: one line when every check passes, else each with what is missing. */
export function ReadinessList({ checks }: { checks: readonly ReadinessCheck[] }) {
  const t = useT();
  const done = checks.filter((check) => check.complete).length;
  if (checks.length > 0 && done === checks.length) {
    return (
      <p className="dk-edp-allready">
        <CardIcon name="check" size={11} stroke={2.6} />
        <span>
          <b>
            {t("{0, plural, one {The # check passes} other {All # checks pass}}", checks.length)}
          </b>{" "}
          {checks.map((check) => check.label).join(" · ")}
        </span>
      </p>
    );
  }
  return (
    <div className="dk-edp-ready">
      <div
        className="dk-edp-readybar"
        role="meter"
        aria-label={t("Ready to trade")}
        aria-valuemin={0}
        aria-valuemax={checks.length}
        aria-valuenow={done}
        style={{ gridTemplateColumns: `repeat(${Math.max(1, checks.length)}, minmax(0, 1fr))` }}
      >
        {checks.map((check) => (
          <i key={check.key} className={cn(check.complete && "dk-done")} />
        ))}
      </div>
      <ul>
        {checks.map((check) => (
          <li key={check.key} className={check.complete ? "dk-done" : "dk-miss"}>
            <span className="dk-tick" aria-hidden="true">
              {check.complete && <CardIcon name="check" size={10} stroke={2.8} />}
            </span>
            <span className="dk-what">
              <b>{check.label}</b>
              {check.note && <span>{check.note}</span>}
            </span>
          </li>
        ))}
      </ul>
    </div>
  );
}

export function MappingChips({ chips }: { chips: readonly MappingChip[] }) {
  return (
    <ul className="dk-edp-maps">
      {chips.map((chip) => (
        <li key={chip.key}>{chip.text}</li>
      ))}
    </ul>
  );
}

/** A record the partner belongs to, as a tile: opened in the card's place, or a link to its page. */
export function PartnerRef({
  kind,
  name,
  line,
  to,
  onOpen,
}: {
  kind: string;
  name: string;
  line: string;
  to: string;
  onOpen?: () => void;
}) {
  const body = (
    <>
      <span className="dk-k">{kind}</span>
      <b>{name}</b>
      {line && <span>{line}</span>}
    </>
  );
  return onOpen ? (
    <Button variant="bare" size="bare" className="dk-edp-ref" onClick={onOpen}>
      {body}
      <CardIcon name="forward" size={11} />
    </Button>
  ) : (
    <Link className="ui-focus-ring dk-edp-ref" to={to}>
      {body}
      <CardIcon name="ext" size={11} />
    </Link>
  );
}

/** Who to call at the partner, with the ways to reach them and their own clock. */
export function PartnerContact({
  name,
  role,
  email,
  phone,
  local,
}: {
  name: string;
  role: string;
  email: string;
  phone: string;
  local: string;
}) {
  return (
    <div className="dk-edp-who">
      <CardAvatar name={name} />
      <span className="dk-rec-main">
        <b>{name}</b>
        {role && <span className="dk-rec-sub">{role}</span>}
      </span>
      <span className="dk-edp-reach">
        {email && (
          <a className="ui-focus-ring" href={`mailto:${email}`}>
            <Mail01Icon className="size-3" aria-hidden />
            {email}
          </a>
        )}
        {phone && (
          <a className="ui-focus-ring" href={telHref(phone)}>
            <Phone01Icon className="size-3" aria-hidden />
            {phone}
          </a>
        )}
        {local && (
          <span>
            <ClockIcon className="size-3" aria-hidden />
            {local}
          </span>
        )}
      </span>
    </div>
  );
}
