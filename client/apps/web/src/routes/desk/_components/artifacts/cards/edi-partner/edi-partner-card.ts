import type {
  DeskEdiPartnerCard,
  EdiPartnerExchange,
  EdiPartnerMessage,
  EdiPartnerTransport,
} from "@/lib/graphql/edi-partner-card";
import { dateTimeFormatter, formatNumber } from "@trenova/shared/i18n/format";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatDurationFromSeconds, formatSecondsAgo } from "@trenova/shared/lib/date";
import { formatPercent } from "@trenova/shared/lib/utils";
import type {
  CardCheckState,
  CardFact,
  CardStatusMark,
  CardTone,
} from "../../record-card/record-card";
import { cardDate, cardMoment } from "../../record-card/record-card-values";

const DAY = 86_400;
/** How far ahead an AS2 certificate's expiry is worth a look. */
export const CERTIFICATE_NOTICE_DAYS = 30;
/** Below this share of delivered messages, the figure reads amber. */
export const DELIVERY_WARN_SHARE = 0.95;

/** What each transaction set the card can show is, in the partner's words. */
export function transactionSetName(set: string, t: TranslateFn): string {
  switch (set) {
    case "204":
      return t("Load tender");
    case "210":
      return t("Freight invoice");
    case "214":
      return t("Shipment status");
    case "990":
      return t("Response to a tender");
    case "997":
      return t("Functional acknowledgment");
    case "999":
      return t("Implementation acknowledgment");
    default:
      return t("Transaction set {0}", set);
  }
}

export type PartnerStanding = { label: string; tone: CardTone; mark: CardStatusMark };

/** Where the partner stands: switched off, waiting on a connection, not set up, in trouble, or trading. */
export function partnerStanding(
  card: DeskEdiPartnerCard,
  attentionCount: number,
  t: TranslateFn,
): PartnerStanding {
  const { partner } = card;
  if (partner.status === "Inactive") {
    return { label: t("Inactive"), tone: "muted", mark: "pause" };
  }
  if (partner.kind === "Internal") {
    switch (partner.connection?.status) {
      case "PendingAcceptance":
        return { label: t("Waiting on them to accept"), tone: "warn", mark: "dot" };
      case "Suspended":
        return { label: t("Suspended"), tone: "warn", mark: "pause" };
      case "Rejected":
        return { label: t("Rejected"), tone: "danger", mark: "x" };
      case "Revoked":
        return { label: t("Revoked"), tone: "danger", mark: "x" };
      case undefined:
        return { label: t("Not connected"), tone: "warn", mark: "dot" };
      default:
        break;
    }
  }
  if (!card.readiness.ready) {
    return { label: t("Not ready"), tone: "warn", mark: "dot" };
  }
  if (attentionCount > 0) {
    return { label: t("Needs a look"), tone: "warn", mark: "dot" };
  }
  return {
    label: partner.kind === "Internal" ? t("Connected") : t("Trading"),
    tone: "success",
    mark: "check",
  };
}

/** The partner's code, what kind of partner it is, and whose EDI it is. */
export function partnerKicker(card: DeskEdiPartnerCard, t: TranslateFn): string[] {
  const kicker = [card.partner.code];
  kicker.push(card.partner.kind === "Internal" ? t("Trenova to Trenova") : t("External partner"));
  if (card.customer) {
    kicker.push(t("Customer {0}", card.customer.name));
  }
  return kicker;
}

/** What the partner is for: its own description, else which ways documents travel. */
export function partnerSubtitle(card: DeskEdiPartnerCard, t: TranslateFn): string {
  const description = card.partner.description?.trim();
  if (description) {
    return description;
  }
  const { enabledForInbound: inbound, enabledForOutbound: outbound } = card.partner;
  if (card.partner.kind === "Internal") {
    const other = card.partner.internalOrganization?.name;
    return other
      ? t("Another Trenova organization, {0}, that we trade with directly", other)
      : t("Another Trenova organization we trade with directly");
  }
  if (inbound && outbound) {
    return t("Sends documents to us and takes ours back");
  }
  if (inbound) {
    return t("Sends documents to us");
  }
  if (outbound) {
    return t("Takes documents from us");
  }
  return t("Turned off both ways");
}

/** Whether the partner's interchanges go out marked for production or for test. */
export type UsageReading = "production" | "test" | null;

/** The usage the partner's active Lading version writes in ISA15. */
export function partnerUsage(card: DeskEdiPartnerCard): UsageReading {
  const usage = card.lading.active?.interchange?.usage;
  if (usage === "production") return "production";
  if (usage === "test") return "test";
  return null;
}

export function usageLabel(usage: UsageReading, t: TranslateFn): string {
  switch (usage) {
    case "production":
      return t("Production");
    case "test":
      return t("Test");
    default:
      return "";
  }
}

/** When the latest message either way was made, of those inside the window. */
export function lastMessageAt(card: DeskEdiPartnerCard): number | null {
  let latest: number | null = null;
  for (const set of card.activity.sets) {
    if (set.lastAt && (latest === null || set.lastAt > latest)) {
      latest = set.lastAt;
    }
  }
  return latest;
}

/** The facts under the head: production or test, the latest message, and the ways it is reached. */
export function headFacts(card: DeskEdiPartnerCard, now: number, t: TranslateFn): string[] {
  const facts = [usageLabel(partnerUsage(card), t)];
  const latest = lastMessageAt(card);
  if (latest !== null) {
    facts.push(t("Last message {0}", formatSecondsAgo(Math.max(0, now - latest))));
  } else if (card.partner.kind === "External") {
    facts.push(t("Nothing sent or received in 7 days"));
  }
  const methods = [
    ...new Set(
      card.transports
        .filter((transport) => transport.profile.status === "Active")
        .map((transport) => transport.profile.method),
    ),
  ];
  facts.push(...methods.filter((method) => method !== "Internal"));
  return facts.filter(Boolean);
}

export type DeliveryFigure = { value: string; caption: string; warn: boolean };

/** The share of outbound messages delivered in the window, when any was tried. */
export function deliveryFigure(card: DeskEdiPartnerCard, t: TranslateFn): DeliveryFigure | null {
  const rate = card.activity.scorecard.deliverySuccessRate;
  if (rate === null || rate === undefined) {
    return null;
  }
  return {
    value: formatPercent(rate * 100, rate >= 0.999 || rate === 0 ? 0 : 1),
    caption: t("delivered · last 7 days"),
    warn: rate < DELIVERY_WARN_SHARE,
  };
}

/** One transaction set as one lane of the exchange draws it. */
export type LaneSet = {
  key: string;
  set: string;
  name: string;
  /** The answer it waits for, or what has gone wrong with it. */
  note: string;
  state: CardCheckState | null;
  count: number | null;
  test: boolean;
  off: boolean;
};

export type Exchange = {
  /** Whether each set's traffic is counted; a Trenova-to-Trenova exchange has no messages to count. */
  counted: boolean;
  us: { name: string; id: string };
  them: { name: string; id: string };
  outbound: LaneSet[];
  inbound: LaneSet[];
};

function outboundNote(
  exchange: EdiPartnerExchange | undefined,
  activity: DeskEdiPartnerCard["activity"]["sets"][number] | undefined,
  t: TranslateFn,
): Pick<LaneSet, "note" | "state"> {
  if (activity && activity.deadLetteredCount > 0) {
    return {
      note: t(
        "{0, plural, one {# dead-lettered} other {# dead-lettered}}",
        activity.deadLetteredCount,
      ),
      state: "bad",
    };
  }
  if (activity && activity.failedCount > 0) {
    return {
      note: t("{0, plural, one {# failed to send} other {# failed to send}}", activity.failedCount),
      state: "warn",
    };
  }
  if (exchange?.acknowledgment) {
    return exchange.acknowledgmentWithin
      ? {
          note: t(
            "Waits on a {0} within {1}",
            exchange.acknowledgment,
            formatDurationFromSeconds(exchange.acknowledgmentWithin),
          ),
          state: null,
        }
      : { note: t("Waits on a {0}", exchange.acknowledgment), state: null };
  }
  return { note: t("No acknowledgment asked"), state: null };
}

function inboundNote(
  set: string,
  exchange: EdiPartnerExchange | undefined,
  t: TranslateFn,
): Pick<LaneSet, "note" | "state"> {
  if (set === "997" || set === "999") {
    return { note: t("Answers what we send"), state: null };
  }
  if (exchange?.acknowledgment) {
    return { note: t("We send a {0} back", exchange.acknowledgment), state: null };
  }
  return { note: t("No acknowledgment sent back"), state: null };
}

function laneSets(
  card: DeskEdiPartnerCard,
  direction: "Outbound" | "Inbound",
  t: TranslateFn,
): LaneSet[] {
  const exchanges = new Map<string, EdiPartnerExchange>();
  for (const exchange of card.lading.active?.exchanges ?? []) {
    if (exchange.direction === direction && !exchanges.has(exchange.transactionSet)) {
      exchanges.set(exchange.transactionSet, exchange);
    }
  }
  const test = partnerUsage(card) === "test";
  const activity = new Map(
    card.activity.sets
      .filter((set) => set.direction === direction)
      .map((set) => [set.transactionSet, set] as const),
  );
  const sets = [...new Set([...exchanges.keys(), ...activity.keys()])].sort();
  const traded = card.partner.kind === "Internal" || card.activity.sets.length > 0;
  return sets.map((set) => {
    const exchange = exchanges.get(set);
    const seen = activity.get(set);
    const off = exchange === undefined && set !== "997" && set !== "999";
    const note = off
      ? { note: t("Not in the active Lading version"), state: null }
      : direction === "Outbound"
        ? outboundNote(exchange, seen, t)
        : inboundNote(set, exchange, t);
    return {
      key: `${direction}:${set}`,
      set,
      name: transactionSetName(set, t),
      ...note,
      count: seen ? seen.count : traded ? 0 : null,
      test: !off && test,
      off,
    };
  });
}

/** The two ends of the exchange and the sets that travel each way between them. */
export function partnerExchange(card: DeskEdiPartnerCard, t: TranslateFn): Exchange {
  const address = partnerAddress(card);
  const internal = card.partner.kind === "Internal";
  return {
    counted: true,
    us: {
      name: t("Us"),
      id: internal ? t("On Trenova") : address ? `${address.us.qualifier} · ${address.us.id}` : "",
    },
    them: {
      name: card.partner.internalOrganization?.name ?? card.partner.name,
      id: internal
        ? t("On Trenova")
        : address
          ? `${address.them.qualifier} · ${address.them.id}`
          : "",
    },
    outbound: laneSets(card, "Outbound", t),
    inbound: laneSets(card, "Inbound", t),
  };
}

/**
 * What a Trenova-to-Trenova partner trades, from its connection's
 * capabilities: each side's Trenova hands the other the record, so nothing is
 * counted as a message and nothing waits on an acknowledgment.
 */
export function connectionExchange(card: DeskEdiPartnerCard, t: TranslateFn): Exchange {
  const capabilities = card.partner.connection?.capabilities;
  const lane = (direction: "Outbound" | "Inbound", set: string, note: string): LaneSet => ({
    key: `${direction}:${set}`,
    set,
    name: transactionSetName(set, t),
    note,
    state: null,
    count: null,
    test: false,
    off: false,
  });
  const outbound: LaneSet[] = [];
  const inbound: LaneSet[] = [];
  if (capabilities?.loadTenderOutbound) {
    outbound.push(lane("Outbound", "204", t("Their dispatch accepts or declines")));
  }
  if (capabilities?.shipmentStatus) {
    outbound.push(lane("Outbound", "214", t("As stops are made")));
  }
  if (capabilities?.loadTenderInbound) {
    inbound.push(lane("Inbound", "204", t("We accept or decline")));
  }
  if (capabilities?.invoice) {
    inbound.push(lane("Inbound", "210", t("Straight to accounts payable")));
  }
  return {
    counted: false,
    us: { name: t("Us"), id: t("On Trenova") },
    them: {
      name: card.partner.internalOrganization?.name ?? card.partner.name,
      id: t("On Trenova"),
    },
    outbound,
    inbound,
  };
}

export type AddressEnd = { qualifier: string; id: string; application: string };

export type PartnerAddress = {
  us: AddressEnd;
  them: AddressEnd;
  /** ISA15 as written: P or T. */
  usage: string;
  separators: { element: string; segment: string; component: string; repetition: string };
};

/** How interchanges with the partner are addressed, from us to them, by the active Lading version. */
export function partnerAddress(card: DeskEdiPartnerCard): PartnerAddress | null {
  const interchange = card.lading.active?.interchange;
  if (!interchange || !interchange.theirId || !interchange.ourId) {
    return null;
  }
  const [element = "", segment = "", component = "", repetition = ""] = interchange.separators;
  return {
    us: {
      qualifier: interchange.ourQualifier ?? "",
      id: interchange.ourId,
      application: interchange.ourGroupCode ?? "",
    },
    them: {
      qualifier: interchange.theirQualifier ?? "",
      id: interchange.theirId,
      application: interchange.theirGroupCode ?? "",
    },
    usage: interchange.usage === "production" ? "P" : "T",
    separators: { element, segment, component, repetition },
  };
}

/** One local day of the series: what went out, what came in, and what failed. */
export type PulseDay = {
  start: number;
  sent: number;
  received: number;
  failed: number;
  today: boolean;
};

function localDayStart(seconds: number): number {
  const date = new Date(seconds * 1000);
  date.setHours(0, 0, 0, 0);
  return Math.floor(date.getTime() / 1000);
}

/**
 * The hourly series folded into the viewer's own days, the last `days` of them
 * ending today, so a day starts at the viewer's midnight and an empty day is
 * still drawn.
 */
export function pulseDays(
  series: DeskEdiPartnerCard["activity"]["series"],
  now: number,
  days = 14,
): PulseDay[] {
  const today = localDayStart(now);
  const starts: number[] = [];
  for (let index = days - 1; index >= 0; index--) {
    const noon = today - index * DAY + DAY / 2;
    starts.push(localDayStart(noon));
  }
  const byDay = new Map<number, PulseDay>(
    starts.map((start) => [
      start,
      { start, sent: 0, received: 0, failed: 0, today: start === today },
    ]),
  );
  for (const point of series) {
    const day = byDay.get(localDayStart(point.bucketStart));
    if (!day) continue;
    day.sent += point.outboundCount;
    day.received += point.receivedCount;
    day.failed += point.failedCount;
  }
  return starts.map((start) => byDay.get(start) as PulseDay);
}

export type PulseFigure = {
  key: string;
  label: string;
  value: string;
  state: CardCheckState | null;
};

/** The window's figures: sent, received, failed, dead-lettered. */
export function pulseFigures(card: DeskEdiPartnerCard, t: TranslateFn): PulseFigure[] {
  const score = card.activity.scorecard;
  return [
    { key: "sent", label: t("Sent"), value: formatNumber(score.sentCount), state: null },
    {
      key: "received",
      label: t("Received"),
      value: formatNumber(score.receivedCount),
      state: null,
    },
    {
      key: "failed",
      label: t("Failed"),
      value: formatNumber(score.failedCount),
      state: score.failedCount > 0 ? "warn" : null,
    },
    {
      key: "dead",
      label: t("Dead-lettered"),
      value: formatNumber(score.deadLetteredCount),
      state: score.deadLetteredCount > 0 ? "bad" : null,
    },
  ];
}

export type AckRuler = {
  /** Seconds the ruler spans, from nothing to a little past the slowest mark. */
  scale: number;
  average: number | null;
  slowest: number | null;
  /** The promised turnaround: the shortest any outbound set waits on, if any set waits. */
  promise: number | null;
  line: string;
};

/** How quickly the partner's acknowledgments came back, against what the flow promises. */
export function ackRuler(card: DeskEdiPartnerCard, t: TranslateFn): AckRuler | null {
  const score = card.activity.scorecard;
  const average = score.avgAckSeconds ?? null;
  const slowest = score.p95AckSeconds ?? null;
  const promises = (card.lading.active?.exchanges ?? [])
    .filter((exchange) => exchange.direction === "Outbound" && exchange.acknowledgmentWithin)
    .map((exchange) => exchange.acknowledgmentWithin ?? 0);
  const promise = promises.length > 0 ? Math.min(...promises) : null;
  if (average === null && slowest === null) {
    return promise === null
      ? null
      : {
          scale: promise * 1.25,
          average: null,
          slowest: null,
          promise,
          line: t("No acknowledgment came back in 7 days"),
        };
  }
  const reach = Math.max(slowest ?? 0, average ?? 0, promise ?? 0);
  let line = t(
    "Back in {0} on average, {1} at the slowest 5%",
    formatDurationFromSeconds(Math.round(average ?? 0)),
    formatDurationFromSeconds(Math.round(slowest ?? average ?? 0)),
  );
  if (promise !== null && (slowest ?? 0) > promise) {
    line = t(
      "Back in {0} on average, but the slowest 5% take longer than the {1} promised",
      formatDurationFromSeconds(Math.round(average ?? 0)),
      formatDurationFromSeconds(promise),
    );
  }
  return { scale: Math.max(reach * 1.15, 60), average, slowest, promise, line };
}

/** The window's acknowledgments by how they came back, in a fixed order. */
export function ackCounts(card: DeskEdiPartnerCard, t: TranslateFn): PulseFigure[] {
  const count = (status: string) =>
    card.activity.ackStatusCounts.find((entry) => entry.status === status)?.count ?? 0;
  const overdue = card.attention.overdueAckCount;
  return [
    {
      key: "accepted",
      label: t("Accepted"),
      value: formatNumber(count("Accepted")),
      state: count("Accepted") > 0 ? "ok" : null,
    },
    {
      key: "errors",
      label: t("With errors"),
      value: formatNumber(count("AcceptedWithErrors")),
      state: count("AcceptedWithErrors") > 0 ? "warn" : null,
    },
    {
      key: "rejected",
      label: t("Rejected"),
      value: formatNumber(count("Rejected")),
      state: count("Rejected") > 0 ? "bad" : null,
    },
    { key: "waiting", label: t("Waiting"), value: formatNumber(count("Pending")), state: null },
    {
      key: "overdue",
      label: t("Overdue"),
      value: formatNumber(overdue),
      state: overdue > 0 ? "warn" : null,
    },
  ];
}

/** Something about the partner that wants a person, with what it is about. */
export type AttentionItem = {
  key: string;
  state: "warn" | "bad";
  title: string;
  body: string;
  when: string;
  /** The words the line or the partner sent back, kept as written. */
  quote?: string;
  messages?: readonly EdiPartnerMessage[];
  files?: readonly DeskEdiPartnerCard["attention"]["quarantinedFiles"][number][];
};

function pollFailing(transport: EdiPartnerTransport): boolean {
  if (!transport.lastPollError || transport.profile.status !== "Active") {
    return false;
  }
  const attempt = transport.lastPollAttemptAt ?? 0;
  const success = transport.lastPollSuccessAt ?? 0;
  return attempt > success;
}

/** What wants a person, worst first: failing pickups and lines, dead letters, late acks, set-aside files. */
export function attentionItems(
  card: DeskEdiPartnerCard,
  now: number,
  t: TranslateFn,
): AttentionItem[] {
  const items: AttentionItem[] = [];
  for (const transport of card.transports) {
    if (pollFailing(transport)) {
      const since = transport.lastPollSuccessAt;
      items.push({
        key: `poll:${transport.profile.id}`,
        state: "bad",
        title: t("We can't pick up their files"),
        body: since
          ? t(
              "Every pickup through {0} since {1} failed.",
              transport.profile.name,
              cardMoment(since),
            )
          : t("No pickup through {0} has worked yet.", transport.profile.name),
        when: since ? formatSecondsAgo(Math.max(0, now - since)) : "",
        quote: transport.lastPollError ?? undefined,
      });
    }
    if (transport.profile.status !== "Active") continue;
    for (const credential of transport.credentials) {
      if (credential.required && !credential.onFile) {
        items.push({
          key: `credential:${transport.profile.id}:${credential.key}`,
          state: "bad",
          title: t("{0} is missing", credentialLabel(credential.key, transport.profile.method, t)),
          body: t(
            "{0} cannot connect until it is entered on the partner's page.",
            transport.profile.name,
          ),
          when: "",
        });
      }
    }
    for (const certificate of transport.certificates) {
      const reading = certificateReading(certificate, now, t);
      if (reading.state === "warn" || reading.state === "bad") {
        items.push({
          key: `certificate:${transport.profile.id}:${certificate.role}`,
          state: reading.state,
          title: reading.title,
          body: reading.body,
          when: reading.when,
        });
      }
    }
  }
  const { attention } = card;
  if (attention.deadLettered.length > 0) {
    const latest = attention.deadLettered[0];
    items.push({
      key: "dead-lettered",
      state: "bad",
      title: t(
        "{0, plural, one {A message was dead-lettered} other {# messages were dead-lettered}}",
        attention.deadLettered.length,
      ),
      body: t("Delivery gave up on it, and it has not reached {0}.", card.partner.name),
      when: cardMoment(latest.deliveryLastAttemptAt ?? latest.generatedAt),
      messages: attention.deadLettered,
    });
  }
  if (attention.overdueAckCount > 0) {
    const age = attention.oldestOverdueAt ? Math.max(0, now - attention.oldestOverdueAt) : 0;
    items.push({
      key: "overdue",
      state: "warn",
      title: t(
        "{0, plural, one {# message is waiting on an acknowledgment} other {# messages are waiting on an acknowledgment}} past {1}",
        attention.overdueAckCount,
        formatDurationFromSeconds(attention.overdueAckAgeSeconds),
      ),
      body: age > 0 ? t("The oldest has waited {0}.", formatDurationFromSeconds(age)) : "",
      when: attention.oldestOverdueAt ? t("Since {0}", cardMoment(attention.oldestOverdueAt)) : "",
      messages: attention.overdueAcks,
    });
  }
  if (attention.quarantinedFiles.length > 0) {
    items.push({
      key: "quarantined",
      state: "warn",
      title: t(
        "{0, plural, one {A file was set aside} other {# files were set aside}}",
        attention.quarantinedFiles.length,
      ),
      body: t("Nothing in it was read."),
      when: cardMoment(attention.quarantinedFiles[0].receivedAt),
      files: attention.quarantinedFiles,
    });
  }
  return items.sort((a, b) => (a.state === b.state ? 0 : a.state === "bad" ? -1 : 1));
}

/** What each credential a transport keeps is called, by the key it is stored under. */
export function credentialLabel(key: string, method: string, t: TranslateFn): string {
  switch (key) {
    case "knownHostKey":
      return t("Pinned host key");
    case "password":
      return t("Password or private key");
    case "privateKey":
      return method === "AS2" ? t("Our private key") : t("Private key");
    case "basicAuthPassword":
      return t("Basic auth password");
    case "localCertificate":
      return t("Our signing certificate");
    case "partnerSigningCertificate":
      return t("Their signing certificate");
    case "partnerEncryptionCertificate":
      return t("Their encryption certificate");
    default:
      return key;
  }
}

const CERTIFICATE_KEYS = new Set([
  "localCertificate",
  "partnerSigningCertificate",
  "partnerEncryptionCertificate",
]);

export type CredentialChip = { key: string; label: string; onFile: boolean };

/**
 * The credentials worth naming on the card: every one required, and every
 * optional one that is on file. Certificates are drawn as their own tiles.
 */
export function credentialChips(transport: EdiPartnerTransport, t: TranslateFn): CredentialChip[] {
  return transport.credentials
    .filter((credential) => !CERTIFICATE_KEYS.has(credential.key))
    .filter((credential) => credential.required || credential.onFile)
    .map((credential) => ({
      key: credential.key,
      label: credentialLabel(credential.key, transport.profile.method, t),
      onFile: credential.onFile,
    }));
}

export function certificateRoleLabel(role: string, t: TranslateFn): string {
  switch (role) {
    case "Local":
      return t("Our signing certificate");
    case "PartnerSigning":
      return t("Their signing certificate");
    case "PartnerEncryption":
      return t("Their encryption certificate");
    default:
      return role;
  }
}

export type CertificateReading = {
  state: CardCheckState;
  title: string;
  body: string;
  when: string;
  /** How much of its life the certificate has used, 0 to 1. */
  used: number;
};

/** A certificate's standing: unreadable, expired, expiring soon, or good until a date. */
export function certificateReading(
  certificate: EdiPartnerTransport["certificates"][number],
  now: number,
  t: TranslateFn,
): CertificateReading {
  const name = certificateRoleLabel(certificate.role, t);
  if (!certificate.readable || !certificate.notAfter) {
    return {
      state: "warn",
      title: t("{0} can't be read", name),
      body: t("What is stored is not a certificate. Paste it again on the partner's page."),
      when: "",
      used: 0,
    };
  }
  const notAfter = certificate.notAfter;
  const notBefore = certificate.notBefore ?? notAfter;
  const used =
    notAfter > notBefore ? Math.min(1, Math.max(0, (now - notBefore) / (notAfter - notBefore))) : 1;
  if (certificate.expired || notAfter <= now) {
    return {
      state: "bad",
      title: t("{0} has expired", name),
      body: t(
        "It ran out on {0}. Messages signed or encrypted with it are refused.",
        cardDate(notAfter),
      ),
      when: cardDate(notAfter),
      used: 1,
    };
  }
  const daysLeft = Math.floor((notAfter - now) / DAY);
  if (daysLeft <= CERTIFICATE_NOTICE_DAYS) {
    return {
      state: "warn",
      title: t("{0} expires in {1, plural, one {# day} other {# days}}", name, daysLeft),
      body: t("Ask them for its replacement before {0}.", cardDate(notAfter)),
      when: cardDate(notAfter),
      used,
    };
  }
  return {
    state: "ok",
    title: name,
    body: certificate.subject ?? "",
    when: t("Until {0}", cardDate(notAfter)),
    used,
  };
}

function configText(config: unknown, key: string): string {
  if (!config || typeof config !== "object") return "";
  const value = (config as Record<string, unknown>)[key];
  if (typeof value === "string") return value.trim();
  if (typeof value === "number") return String(value);
  return "";
}

function configNumber(config: unknown, key: string): number | null {
  const text = configText(config, key);
  if (!text) return null;
  const value = Number(text);
  return Number.isFinite(value) ? value : null;
}

export type TransportEnds = { usLabel: string; us: string; themLabel: string; them: string };

/** Who each end of the line is: the AS2 IDs, or the account we sign in as and their server. */
export function transportEnds(
  transport: EdiPartnerTransport,
  t: TranslateFn,
): TransportEnds | null {
  const config = transport.profile.config;
  switch (transport.profile.method) {
    case "AS2": {
      const us = configText(config, "localAS2Id");
      const them = configText(config, "partnerAS2Id");
      return us || them
        ? {
            usLabel: t("Our AS2 ID"),
            us: us || "—",
            themLabel: t("Their AS2 ID"),
            them: them || "—",
          }
        : null;
    }
    case "SFTP":
    case "VAN": {
      const host = configText(config, "host");
      const port = configText(config, "port");
      const user = configText(config, "username");
      if (!host && !user) return null;
      return {
        usLabel: t("We sign in as"),
        us: user || "—",
        themLabel: transport.profile.method === "VAN" ? t("Their network") : t("Their server"),
        them: host ? (port ? `${host}:${port}` : host) : "—",
      };
    }
    default:
      return null;
  }
}

function receiptMode(mode: string, t: TranslateFn): string {
  switch (mode) {
    case "sync":
      return t("Signed receipt, right away");
    case "async":
      return t("Signed receipt, sent back later");
    default:
      return "";
  }
}

const SIGNING_LABELS: Record<string, string> = {
  sha256: "SHA-256",
  sha384: "SHA-384",
  sha512: "SHA-512",
  sha1: "SHA-1",
};

const ENCRYPTION_LABELS: Record<string, string> = {
  "aes256-cbc": "AES-256-CBC",
  "aes128-cbc": "AES-128-CBC",
  "aes256-gcm": "AES-256-GCM",
  "aes128-gcm": "AES-128-GCM",
  "3des": "3DES",
};

function retryLine(config: unknown, t: TranslateFn): string {
  const attempts = configNumber(config, "retryMaxAttempts");
  if (!attempts) return "";
  const first = configNumber(config, "retryInitialIntervalSeconds");
  const most = configNumber(config, "retryMaxIntervalSeconds");
  if (first && most && most > first) {
    return t(
      "{0, plural, one {# try} other {# tries}}, {1} backing off to {2}",
      attempts,
      formatDurationFromSeconds(first),
      formatDurationFromSeconds(most),
    );
  }
  return t("{0, plural, one {# try} other {# tries}}", attempts);
}

/** The settings of a line that a person checking it reads, by method. */
export function transportFacts(
  transport: EdiPartnerTransport,
  now: number,
  t: TranslateFn,
): CardFact[] {
  const config = transport.profile.config;
  const facts: CardFact[] = [];
  const add = (key: string, label: string, value: string) => {
    if (value) facts.push({ key, label, value });
  };
  if (transport.profile.method === "AS2") {
    add("endpoint", t("Endpoint"), configText(config, "endpointUrl"));
    add("receipts", t("Receipts"), receiptMode(configText(config, "mdnMode"), t));
    const signing = configText(config, "signingAlgorithm");
    add("signing", t("Signing"), SIGNING_LABELS[signing] ?? signing);
    const encryption = configText(config, "encryptionAlgorithm");
    add("encryption", t("Encryption"), ENCRYPTION_LABELS[encryption] ?? encryption);
  } else {
    const auth = configText(config, "authMode");
    add(
      "auth",
      t("Signs in with"),
      auth === "password" ? t("Password") : auth === "privateKey" ? t("Private key") : "",
    );
    add("mailbox", t("Mailbox"), configText(config, "mailboxId"));
    add("drops", t("Drops files in"), configText(config, "outboundDirectory"));
    const inbound = configText(config, "inboundDirectory");
    const archive = configText(config, "archiveDirectory");
    add(
      "picks",
      t("Picks up from"),
      inbound && archive ? t("{0}, then moves them to {1}", inbound, archive) : inbound,
    );
    add("names", t("Files named"), configText(config, "fileNamingPattern"));
    if (transport.lastPollSuccessAt) {
      add(
        "last-pickup",
        t("Last pickup that worked"),
        formatSecondsAgo(Math.max(0, now - transport.lastPollSuccessAt)),
      );
    }
  }
  add("retries", t("Retries"), retryLine(config, t));
  return facts;
}

export type TransportLine = { state: CardCheckState; text: string } | null;

/** The line's last word: pickups failing, or when one last worked. */
export function transportLine(
  transport: EdiPartnerTransport,
  now: number,
  t: TranslateFn,
): TransportLine {
  if (transport.profile.status !== "Active") {
    return { state: "info", text: t("Turned off") };
  }
  if (pollFailing(transport)) {
    return transport.lastPollSuccessAt
      ? {
          state: "warn",
          text: t("Pickup failing since {0}", cardMoment(transport.lastPollSuccessAt)),
        }
      : { state: "warn", text: t("No pickup has worked yet") };
  }
  if (transport.lastPollSuccessAt) {
    return {
      state: "ok",
      text: t("Picked up {0}", formatSecondsAgo(Math.max(0, now - transport.lastPollSuccessAt))),
    };
  }
  return null;
}

/** Whether a transport fetches files from the partner, so a pickup can be asked for by hand. */
export function transportPolls(transport: EdiPartnerTransport): boolean {
  return (
    transport.profile.status === "Active" &&
    (transport.profile.method === "SFTP" || transport.profile.method === "VAN") &&
    configText(transport.profile.config, "inboundDirectory") !== ""
  );
}

/** The transport the card's actions run against: the active one, else the first. */
export function leadTransport(card: DeskEdiPartnerCard): EdiPartnerTransport | null {
  return (
    card.transports.find((transport) => transport.profile.status === "Active") ??
    card.transports[0] ??
    null
  );
}

const READINESS_ORDER = [
  "details",
  "communication-profile",
  "mappings",
  "inbound-document-profile",
  "outbound-document-profile",
  "test-case",
];

export type ReadinessCheck = { key: string; label: string; complete: boolean; note: string };

/** Each check the partner page runs before a partner trades, with what is missing. */
export function readinessChecks(card: DeskEdiPartnerCard, t: TranslateFn): ReadinessCheck[] {
  const label = (key: string) => {
    switch (key) {
      case "details":
        return { label: t("Details"), note: t("A contact email and the partner's timezone.") };
      case "communication-profile":
        return {
          label: t("Transport"),
          note: t("No AS2, SFTP or VAN line is set up to reach them."),
        };
      case "mappings":
        return {
          label: t("Mappings"),
          note: t("None of their codes are matched to our records yet."),
        };
      case "inbound-document-profile":
        return { label: t("Inbound sets"), note: t("Nothing they send is set up to be read.") };
      case "outbound-document-profile":
        return { label: t("Outbound sets"), note: t("Nothing we send them is set up.") };
      case "test-case":
        return {
          label: t("Test case"),
          note: t("Keep one of their documents passing in the designer."),
        };
      default:
        return { label: key, note: "" };
    }
  };
  return [...card.readiness.items]
    .sort((a, b) => READINESS_ORDER.indexOf(a.key) - READINESS_ORDER.indexOf(b.key))
    .map((item) => {
      const { label: name, note } = label(item.key);
      return {
        key: item.key,
        label: name,
        complete: item.complete,
        note: item.complete ? "" : note,
      };
    });
}

export type MappingChip = { key: string; text: string; count: number };

function mappingCount(entityType: string, count: number, t: TranslateFn): string {
  switch (entityType) {
    case "Customer":
      return t("{0, plural, one {# customer} other {# customers}}", count);
    case "ServiceType":
      return t("{0, plural, one {# service type} other {# service types}}", count);
    case "ShipmentType":
      return t("{0, plural, one {# shipment type} other {# shipment types}}", count);
    case "FormulaTemplate":
      return t("{0, plural, one {# formula template} other {# formula templates}}", count);
    case "Location":
      return t("{0, plural, one {# location} other {# locations}}", count);
    case "Commodity":
      return t("{0, plural, one {# commodity} other {# commodities}}", count);
    case "AccessorialCharge":
      return t("{0, plural, one {# accessorial charge} other {# accessorial charges}}", count);
    case "ServiceFailureReasonCode":
      return t("{0, plural, one {# failure reason} other {# failure reasons}}", count);
    default:
      return t("{0, plural, one {# code} other {# codes}}", count);
  }
}

/** Their codes matched to our records, by kind, most first. */
export function mappingChips(card: DeskEdiPartnerCard, t: TranslateFn): MappingChip[] {
  return [...card.mappings.counts]
    .filter((entry) => entry.count > 0)
    .sort((a, b) => b.count - a.count)
    .map((entry) => ({
      key: entry.entityType,
      text: mappingCount(entry.entityType, entry.count, t),
      count: entry.count,
    }));
}

/** The partner's own clock, read for someone about to call them. */
export function partnerLocalTime(
  timezone: string | null | undefined,
  now: number,
  t: TranslateFn,
): string {
  if (!timezone) return "";
  try {
    const time = dateTimeFormatter({
      hour: "numeric",
      minute: "2-digit",
      timeZone: timezone,
    }).format(new Date(now * 1000));
    return t("{0} their time", time);
  } catch {
    return "";
  }
}

export type MessageRowStatus = { text: string; state: CardCheckState | null };

/** Where a listed message stands, in a word or two. */
export function messageRowStatus(message: EdiPartnerMessage, t: TranslateFn): MessageRowStatus {
  if (message.status === "Failed") {
    return {
      text: message.direction === "Inbound" ? t("Could not be read") : t("Not generated"),
      state: "warn",
    };
  }
  if (message.direction === "Inbound") {
    return { text: t("Received"), state: null };
  }
  switch (message.deliveryStatus) {
    case "DeadLettered":
      return { text: t("Dead-lettered"), state: "bad" };
    case "Failed":
      return { text: t("Delivery failed"), state: "warn" };
    case "Queued":
      return { text: t("Queued"), state: null };
    case "Sending":
      return { text: t("Sending"), state: null };
    default:
      break;
  }
  switch (message.ackStatus) {
    case "Accepted":
      return { text: t("Accepted"), state: "ok" };
    case "AcceptedWithErrors":
      return { text: t("Accepted with errors"), state: "warn" };
    case "Rejected":
      return { text: t("Rejected"), state: "bad" };
    case "Failed":
      return { text: t("Acknowledgment unreadable"), state: "warn" };
    case "Pending":
      return { text: t("Waiting on an acknowledgment"), state: null };
    default:
      return { text: t("Sent"), state: null };
  }
}

/** The line under a listed message: its interchange number and what went wrong, if anything. */
export function messageRowLine(message: EdiPartnerMessage, t: TranslateFn): string {
  const problem =
    message.deliveryStatus === "DeadLettered" || message.deliveryStatus === "Failed"
      ? message.deliveryLastError?.trim()
      : message.ackStatus === "Rejected"
        ? message.ackLastError?.trim()
        : "";
  return problem
    ? t("Interchange {0} · {1}", message.interchangeControlNumber, problem)
    : t("Interchange {0}", message.interchangeControlNumber);
}

/** When a listed message moved last: delivered, acknowledged, or made. */
export function messageRowTime(message: EdiPartnerMessage): string {
  return cardMoment(message.ackReceivedAt ?? message.deliverySentAt ?? message.generatedAt);
}

/** The connection facts a Trenova-to-Trenova partner is read for. */
export function connectionFacts(card: DeskEdiPartnerCard, t: TranslateFn): CardFact[] {
  const connection = card.partner.connection;
  if (!connection) return [];
  const facts: CardFact[] = [
    { key: "asked", label: t("Asked for"), value: cardDate(connection.requestedAt) },
  ];
  if (connection.acceptedAt) {
    facts.push({ key: "accepted", label: t("Accepted"), value: cardDate(connection.acceptedAt) });
  }
  if (connection.suspendedAt) {
    facts.push({
      key: "suspended",
      label: t("Suspended"),
      value: cardDate(connection.suspendedAt),
    });
  }
  if (connection.rejectedAt) {
    facts.push({ key: "rejected", label: t("Rejected"), value: cardDate(connection.rejectedAt) });
  }
  if (connection.revokedAt) {
    facts.push({ key: "revoked", label: t("Revoked"), value: cardDate(connection.revokedAt) });
  }
  if (connection.rejectionReason?.trim()) {
    facts.push({ key: "reason", label: t("Why"), value: connection.rejectionReason.trim() });
  }
  return facts;
}
