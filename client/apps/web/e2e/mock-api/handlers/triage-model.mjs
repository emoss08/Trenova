// The Triage queue's state and rules, kept to the server's semantics
// (services/tms/internal/modules/documents/domain/{inboundmessage,triage}).
import {
  buildTriageWorld,
  DEFAULT_REPLY,
  DEFAULT_TARGETS,
  DOCUMENT_TYPES,
  FIRST_CREATED_PRO,
  KIND_LABEL,
  KIND_ORDER,
  LIVE_POOL,
  LOAD_DRAFTS,
  ME_ID,
  REFUSING_SENDERS,
  REPLY_DRAFTS,
  STOPS,
  STOPS_REPLIES,
  THEY_REPLY,
  VIEWING,
} from "../fixtures/triage.mjs";
import { publish, publishInvalidation } from "../realtime.mjs";

const MIN = 60;
const DAY = 86400;
const QUIET_AFTER_SECONDS = 3 * DAY;
const MAX_SNOOZE_SECONDS = 90 * DAY;
const MAX_AWAIT_SECONDS = 30 * DAY;
const MAX_BULK = 500;
const PREVIEW_WINDOW = 30 * DAY;
const LIVE_EVERY_MS = Number(process.env.MOCK_TRIAGE_LIVE_MS ?? 26000);
const PRESENCE_SCOPE = "inbox:messages";

export { ME_ID, PRESENCE_SCOPE };

/** A GraphQL error the way the server's presenter shapes one (code, problem type, field errors). */
export class TriageError extends Error {
  constructor(message, { code = "INVALID", field = null, type = "validation-error" } = {}) {
    super(message);
    this.code = code;
    this.field = field;
    this.problem = type;
  }
}

const notFound = (what) => new TriageError(`${what} not found`, { code: "NOT_FOUND", type: "not-found" });
const invalid = (field, message, code = "INVALID") => new TriageError(message, { code, field });

// --- predicates: inboundmessage.InboundMessage.NeedsReview / IsWaiting / IsSnoozed / IsAwaitingReply

export const needsReview = (m) => m.status === "InReview" || m.status === "Quarantined";
export const isWaiting = (m) => needsReview(m) && !m.parked;
export const isSnoozed = (m) => needsReview(m) && m.parked === "Snoozed";
export const isAwaitingReply = (m) => needsReview(m) && m.parked === "AwaitingReply";
export const isSettled = (m) => m.status === "Actioned" || m.status === "Ignored";
export const isProcessing = (m) => m.status === "Received" || m.status === "Processing" || m.status === "Classified";
export const hasRecord = (m) => !!(m.matchedShipmentId || m.matchedCustomerId || m.matchedCarrierId);
const readable = (a) => !a.failureText && !!a.documentId;
const filed = (a) => !!a.filedShipmentId || a.splitDocuments.length > 0;
const domainOf = (address) => (address.split("@")[1] ?? "").toLowerCase();
const domainAllowed = (domain, allowed) => allowed.some((d) => domain === d || domain.endsWith(`.${d}`));

// --- triage.Actions and triage.GroupFor, ported line for line from actions.go

const NEEDS_RECORD = new Set(["ProofOfDelivery", "RateConfirmation", "Invoice", "DetentionDispute", "StatusRequest"]);
const needsRecord = (c) => NEEDS_RECORD.has(c);
const filesAttachments = (c) => c !== "DetentionDispute" && c !== "StatusRequest";

const blankAction = (fields) => ({
  id: "",
  kind: "",
  done: false,
  novaCanDo: false,
  attachmentId: null,
  shipmentId: null,
  replyPurpose: null,
  refusedFileName: null,
  replyId: null,
  loadDrafted: false,
  createdShipmentId: null,
  stopsDecision: null,
  exceptionKind: null,
  completedAt: null,
  ...fields,
});

export function latestReply(m) {
  return m.replies.length ? m.replies[m.replies.length - 1] : null;
}

function replyAction(reply, purpose, novaDrafts) {
  const action = blankAction({ id: "reply", kind: "Reply", replyPurpose: purpose, novaCanDo: novaDrafts && !reply });
  if (reply) {
    action.replyId = reply.id;
    action.done = reply.status === "Sent";
    action.completedAt = reply.sentAt;
  }
  return action;
}

function fileAction(m, a) {
  return blankAction({
    id: `file:${a.id}`,
    kind: "File",
    done: filed(a),
    attachmentId: a.id,
    shipmentId: m.matchedShipmentId,
    completedAt: a.filedAt,
  });
}

function loadAction(m) {
  const state = m.actionState;
  return blankAction({
    id: "load",
    kind: "BuildLoad",
    done: !!state.createdShipmentId,
    novaCanDo: !m.loadDraft && !state.createdShipmentId,
    loadDrafted: !!m.loadDraft,
    createdShipmentId: state.createdShipmentId ?? null,
    completedAt: state.createdShipmentAt ?? null,
  });
}

export function actionsOf(m) {
  if (!m.novaRead) return [];
  const state = m.actionState;
  const reply = latestReply(m);
  const actions = [];

  if (m.status === "Quarantined") {
    actions.push(blankAction({ id: "block", kind: "BlockSender", done: !!state.senderBlockedAt, completedAt: state.senderBlockedAt ?? null }));
    return actions;
  }

  const link = blankAction({ id: "link", kind: "Link", done: !!state.linkedAt, completedAt: state.linkedAt ?? null });
  if (m.failureText && !hasRecord(m)) {
    actions.push(link);
    return actions;
  }

  const kind = m.classification;
  if (needsRecord(kind) && !hasRecord(m) && !state.linkedAt) {
    actions.push(link);
    if (kind === "StatusRequest") return actions;
  }

  if (filesAttachments(kind)) {
    for (const a of m.attachments) {
      if (!readable(a)) continue;
      actions.push(fileAction(m, a));
    }
  }

  switch (kind) {
    case "Tender":
      actions.push(loadAction(m));
      actions.push(replyAction(reply, m.attachments.some(readable) ? "AcceptTender" : "ConfirmTender", false));
      break;
    case "DetentionDispute":
      if (m.matchedShipmentId) {
        actions.push(blankAction({
          id: "stops",
          kind: "CheckStops",
          done: !!state.stopsDecision,
          shipmentId: m.matchedShipmentId,
          stopsDecision: state.stopsDecision ?? null,
          completedAt: state.stopsDecidedAt ?? null,
        }));
      }
      actions.push(replyAction(reply, "AnswerDispute", false));
      break;
    case "RateConfirmation":
      actions.push(replyAction(reply, "ReturnSigned", false));
      break;
    case "ProofOfDelivery": {
      if (m.facts.some((f) => f.tone === "Warning" && !f.flagged) && m.matchedShipmentId) {
        actions.push(blankAction({
          id: "exception",
          kind: "LogException",
          done: !!state.exceptionLoggedAt,
          shipmentId: m.matchedShipmentId,
          exceptionKind: state.exceptionKind ?? null,
          completedAt: state.exceptionLoggedAt ?? null,
        }));
      }
      const refused = m.attachments.find((a) => a.failureText);
      if (refused) {
        const ask = replyAction(reply, "AskResend", false);
        ask.refusedFileName = refused.fileName;
        actions.push(ask);
      }
      break;
    }
    case "StatusRequest":
      actions.push(replyAction(reply, "SendEta", true));
      break;
    default:
      break;
  }

  for (const action of actions) {
    if (!action.shipmentId && action.kind !== "Link") action.shipmentId = m.matchedShipmentId;
  }
  return actions;
}

export const nextOpen = (actions) => actions.find((a) => !a.done) ?? null;

export const DECK_GROUPS = ["File", "Draft", "Decide", "Link", "Held"];
const ESTIMATE_SECONDS = { File: 60, Draft: 120, Link: 120, Decide: 240, Held: 30 };

export function groupFor(m, actions) {
  if (m.status === "Quarantined") return { group: "Held", next: nextOpen(actions) };
  const next = nextOpen(actions);
  if (!next) return null;
  if (next.kind === "File" && m.matchedShipmentId) return { group: "File", next };
  if (next.novaCanDo) return { group: "Draft", next };
  if (next.kind === "Link") return { group: "Link", next };
  return { group: "Decide", next };
}

// --- state

export function createTriageState({ anchor, scenario }) {
  const world = buildTriageWorld({ anchor, scenario });
  const startedAt = Date.now();
  const t = {
    ...world,
    scenario,
    anchor,
    now: () => anchor + Math.floor((Date.now() - startedAt) / 1000),
    novaEnabled: scenario.triage !== "deskOff",
    targets: new Map(),
    settings: { onCallUserId: "usr_marcus", endOfDayMinute: 17 * 60, timezone: "UTC", version: 1 },
    corrections: 3,
    senderHints: new Map(),
    nextPro: FIRST_CREATED_PRO,
    livePool: 0,
    unread: new Map(),
    assistantThreads: new Map(),
    presence: new Map(),
    timers: new Set(),
    readingBacklog: false,
  };
  t.later = (ms, fn) => {
    const handle = setTimeout(() => {
      t.timers.delete(handle);
      fn();
    }, ms);
    t.timers.add(handle);
    return handle;
  };
  t.every = (ms, fn) => {
    const handle = setInterval(fn, ms);
    t.timers.add(handle);
    return handle;
  };
  t.dispose = () => {
    for (const handle of t.timers) {
      clearTimeout(handle);
      clearInterval(handle);
    }
    t.timers.clear();
  };

  for (const m of t.messages) applyDue(t, m);
  refreshMailboxReceipts(t);
  if (scenario.triage === "deskOff") {
    for (const m of t.messages) if (needsReview(m)) unread(t, m);
  }
  seedPresence(t);
  if (scenario.triage === "live") {
    t.every(LIVE_EVERY_MS, () => arriveFromPool(t));
  }
  return t;
}

function refreshMailboxReceipts(t) {
  for (const mailbox of t.mailboxes) {
    const latest = t.messages.filter((m) => m.mailboxId === mailbox.id).reduce((max, m) => Math.max(max, m.receivedAt), 0);
    if (latest && (!mailbox.lastReceivedAt || (mailbox.key !== "mb3" && latest > mailbox.lastReceivedAt))) {
      mailbox.lastReceivedAt = latest;
    }
  }
}

/** Nova was off when these arrived: no kind, no match, no facts, no actions. */
function unread(t, m) {
  t.unread.set(m.id, {
    classification: m.classification,
    confidence: m.confidence,
    matchedShipmentId: m.matchedShipmentId,
    matchedCustomerId: m.matchedCustomerId,
    matchedCarrierId: m.matchedCarrierId,
    matchReason: m.matchReason,
    facts: m.facts,
    conflicts: m.conflicts,
    readings: m.attachments.map((a) => a.reading),
    runId: m.runId,
  });
  m.novaRead = false;
  m.classification = null;
  m.confidence = 0;
  m.matchedShipmentId = null;
  m.matchedCustomerId = null;
  m.matchedCarrierId = null;
  m.matchReason = "";
  m.facts = [];
  m.conflicts = [];
  m.runId = null;
  for (const a of m.attachments) a.reading = null;
  m.activity = m.activity.filter((line) => line.kind !== "NovaRead");
  applyDue(t, m);
}

function readNow(t, m) {
  const saved = t.unread.get(m.id);
  if (!saved) return;
  t.unread.delete(m.id);
  Object.assign(m, {
    classification: saved.classification,
    confidence: saved.confidence,
    matchedShipmentId: saved.matchedShipmentId,
    matchedCustomerId: saved.matchedCustomerId,
    matchedCarrierId: saved.matchedCarrierId,
    matchReason: saved.matchReason,
    facts: saved.facts,
    conflicts: saved.conflicts,
    runId: saved.runId,
    novaRead: true,
  });
  m.attachments.forEach((a, i) => (a.reading = saved.readings[i] ?? null));
  log(t, m, "NovaRead", { subject: KIND_LABEL[m.classification] ?? "" }, { kind: "Nova" });
  applyDue(t, m);
}

export function targetMinutes(t, classification) {
  const kind = classification ?? "Other";
  return t.targets.get(kind) ?? DEFAULT_TARGETS[kind] ?? DEFAULT_TARGETS.Other;
}

/** due_at = received_at + target(kind), absent once the message is settled; a held one is not on the clock. */
function applyDue(t, m) {
  m.dueAt = m.status === "InReview" ? m.receivedAt + targetMinutes(t, m.classification) * MIN : null;
}

export function healthOf(t, mailbox) {
  if (mailbox.status === "Inactive") return "Paused";
  const now = t.now();
  if (!mailbox.lastReceivedAt) return now - mailbox.createdAt > QUIET_AFTER_SECONDS ? "NeverReceived" : "Healthy";
  return now - mailbox.lastReceivedAt > QUIET_AFTER_SECONDS ? "Quiet" : "Healthy";
}

export function mailboxOf(t, id) {
  return t.mailboxes.find((mb) => mb.id === id) ?? null;
}

export function messageOf(t, id) {
  const m = t.messages.find((x) => x.id === id);
  if (!m) throw notFound("Message");
  return m;
}

function userOf(t, id) {
  const user = t.users.get(id);
  if (!user) throw invalid("ownerId", "That person cannot be assigned messages");
  return user;
}

/** Writes one line of the activity trace, the way every server write records its own. */
function log(t, m, kind, extra = {}, actor = { kind: "Person", id: ME_ID }) {
  m.activity.push(t.activity(kind, actor.kind, actor.kind === "Person" ? actor.id ?? ME_ID : null, t.now(), extra));
}

/** Bumps the version and tells open inboxes, as TriageMessageNotifier does after every write. */
function touch(t, m, action = "updated") {
  m.version += 1;
  m.updatedAt = t.now();
  publishInvalidation({ resource: "inbound_message", action, recordId: m.id });
}

function touchMailbox(t, mailbox) {
  mailbox.version += 1;
  mailbox.updatedAt = t.now();
  publishInvalidation({ resource: "inbound_mailbox", action: "updated", recordId: mailbox.id });
}

/** Wakes what a timer would have woken: snoozes and waits whose time has come. */
export function wakeDue(t) {
  const now = t.now();
  for (const m of t.messages) {
    if (isSnoozed(m) && m.snoozedUntil && m.snoozedUntil <= now) {
      unpark(m);
      log(t, m, "Woken", {}, { kind: "System" });
      touch(t, m);
    } else if (isAwaitingReply(m) && m.awaitingUntil && m.awaitingUntil <= now) {
      unpark(m);
      log(t, m, "Woken", {}, { kind: "System" });
      touch(t, m);
    }
  }
}

function unpark(m) {
  Object.assign(m, {
    parked: "",
    snoozedUntil: null,
    snoozeEvent: null,
    snoozeWatchId: null,
    snoozeDocumentTypeId: null,
    snoozedAt: null,
    snoozedBy: null,
    awaitingSince: null,
    awaitingUntil: null,
  });
}

// --- the list: lanes, filters, sort, threads and cursors

const LANE = {
  Waiting: isWaiting,
  WaitingOnThem: isAwaitingReply,
  AssignedToMe: (m) => isWaiting(m) && m.ownerId === ME_ID,
  Snoozed: isSnoozed,
  Handled: (m) => m.status === "Actioned",
  Ignored: (m) => m.status === "Ignored",
  Everything: () => true,
};

function matchesQuery(m, query) {
  const words = query.toLowerCase().split(/\s+/).filter(Boolean);
  if (!words.length) return true;
  const haystack = `${m.subject} ${m.fromAddress} ${m.fromName} ${m.classification ?? ""} ${m.status}`.toLowerCase();
  return words.every((word) => haystack.includes(word));
}

function byNewest(a, b) {
  return b.receivedAt - a.receivedAt || (a.id < b.id ? 1 : -1);
}

function byDue(a, b) {
  if (a.dueAt != null && b.dueAt != null && a.dueAt !== b.dueAt) return a.dueAt - b.dueAt;
  if (a.dueAt != null && b.dueAt == null) return -1;
  if (a.dueAt == null && b.dueAt != null) return 1;
  return byNewest(a, b);
}

/** The folder's messages, in order, before paging: the rows inboundMessages pages through. */
export function folderMessages(t, input = {}) {
  let list = t.messages;
  if (input.threadKey) {
    return list.filter((m) => m.threadKey === input.threadKey).sort(byNewest);
  }
  if (input.lane) {
    const lane = LANE[input.lane];
    if (!lane) throw invalid("lane", "That folder does not exist");
    list = list.filter(lane);
  } else if (input.statuses?.length) {
    list = list.filter((m) => input.statuses.includes(m.status));
  }
  if (input.ownerId) list = list.filter((m) => m.ownerId === input.ownerId);
  if (input.classification) list = list.filter((m) => m.classification === input.classification);
  if (input.mailboxId) list = list.filter((m) => m.mailboxId === input.mailboxId);
  if (input.shipmentId) list = list.filter((m) => m.matchedShipmentId === input.shipmentId);
  if (input.since) list = list.filter((m) => m.receivedAt >= input.since);
  if (input.query) list = list.filter((m) => matchesQuery(m, input.query));
  list = [...list].sort((input.sort ?? "Newest") === "Due" ? byDue : byNewest);
  if (input.collapseThreads) {
    const newest = new Map();
    for (const m of list) {
      const current = newest.get(m.threadKey);
      if (!current || m.receivedAt > current.receivedAt) newest.set(m.threadKey, m);
    }
    list = list.filter((m) => newest.get(m.threadKey) === m);
  }
  return list;
}

const encodeCursor = (offset) => Buffer.from(`offset:${offset}`).toString("base64url");
function decodeCursor(cursor) {
  if (!cursor) return 0;
  const match = /^offset:(\d+)$/.exec(Buffer.from(cursor, "base64url").toString());
  if (!match) throw invalid("after", "The cursor is not one this list gave out");
  return Number(match[1]);
}

export function pageMessages(t, input = {}) {
  const all = folderMessages(t, input);
  const first = Math.max(1, Math.min(input.first ?? 25, 200));
  const start = decodeCursor(input.after);
  const page = all.slice(start, start + first);
  return {
    edges: page.map((m, i) => ({ cursor: encodeCursor(start + i + 1), node: m })),
    hasNextPage: start + first < all.length,
    endCursor: page.length ? encodeCursor(start + page.length) : null,
    totalCount: all.length,
  };
}

export function threadCount(t, threadKey) {
  return t.messages.filter((m) => m.threadKey === threadKey).length;
}

export function counts(t) {
  const now = t.now();
  const byClassification = KIND_ORDER.map((classification) => {
    const ofKind = t.messages.filter((m) => m.classification === classification);
    return { classification, total: ofKind.length, waiting: ofKind.filter(isWaiting).length };
  });
  const byMailbox = t.mailboxes
    .map((mb) => {
      const ofMailbox = t.messages.filter((m) => m.mailboxId === mb.id);
      return { mailboxId: mb.id, total: ofMailbox.length, waiting: ofMailbox.filter(isWaiting).length };
    })
    .filter((row) => row.total > 0);
  return {
    waiting: t.messages.filter(isWaiting).length,
    waitingOnThem: t.messages.filter(isAwaitingReply).length,
    snoozed: t.messages.filter(isSnoozed).length,
    assignedToMe: t.messages.filter(LANE.AssignedToMe).length,
    overdue: t.messages.filter((m) => isWaiting(m) && m.dueAt != null && m.dueAt < now).length,
    processing: t.messages.filter(isProcessing).length,
    handled: t.messages.filter((m) => m.status === "Actioned").length,
    ignored: t.messages.filter((m) => m.status === "Ignored").length,
    quarantined: t.messages.filter((m) => m.status === "Quarantined").length,
    total: t.messages.length,
    byClassification,
    byMailbox,
  };
}

function selectionOf(t, selection = {}) {
  if (selection.ids?.length) {
    return selection.ids.map((id) => messageOf(t, id));
  }
  if (selection.folder) return folderMessages(t, selection.folder).slice(0, MAX_BULK);
  throw invalid("selection", "Choose the messages, or a folder");
}

// --- the briefing aggregate, worked out from the queue as it is now

export function referenceOf(t, m) {
  if (m.matchedShipmentId) return t.shipments.get(m.matchedShipmentId)?.proNumber ?? "";
  const po = m.facts.find((f) => /purchase order|po/i.test(f.label));
  if (po) return po.value;
  return m.attachments[0]?.fileName ?? "";
}

function sampleOf(t, m) {
  return { m, reference: referenceOf(t, m) };
}

function deckItemOf(t, m) {
  const group = groupFor(m, actionsOf(m));
  return { m, action: group?.next ?? null };
}

const settledByPerson = (m) => isSettled(m) && !m.handledByNova && !!m.reviewedBy;

export function phaseOf(t) {
  const now = t.now();
  const minute = Math.floor((now % DAY) / MIN);
  return minute >= t.settings.endOfDayMinute ? "EndOfDay" : "Day";
}

export function briefing(t) {
  const now = t.now();
  const dayStart = now - (now % DAY);
  const waiting = t.messages.filter(isWaiting);
  const novaToday = t.messages.filter((m) => m.handledByNova && m.reviewedAt >= dayStart);

  const handled = new Map();
  for (const m of [...novaToday].sort(byNewest)) {
    const key = `${m.handledAction}|${m.classification}`;
    if (!handled.has(key)) handled.set(key, { action: m.handledAction, classification: m.classification, list: [] });
    handled.get(key).list.push(m);
  }

  const firstArrival = t.messages.filter((m) => m.receivedAt >= dayStart).reduce((min, m) => Math.min(min, m.receivedAt), now);
  const startHour = Math.min(dayStart + 6 * 3600, firstArrival - (firstArrival % 3600));
  const arrivals = [];
  for (let hour = startHour; hour <= now; hour += 3600) {
    const inHour = t.messages.filter((m) => m.receivedAt >= hour && m.receivedAt < hour + 3600);
    arrivals.push({
      hourStart: hour,
      handledByNova: inHour.filter((m) => m.handledByNova).length,
      leftForPerson: inHour.filter((m) => !m.handledByNova).length,
    });
  }

  const cards = new Map();
  for (const m of waiting) {
    const group = groupFor(m, actionsOf(m));
    if (!group) continue;
    if (!cards.has(group.group)) cards.set(group.group, []);
    cards.get(group.group).push({ m, action: group.next });
  }
  const deck = DECK_GROUPS.filter((g) => cards.has(g)).map((g) => {
    const items = cards.get(g).sort((a, b) => byDue(a.m, b.m));
    return {
      group: g,
      count: items.length,
      estimateMinutes: Math.max(1, Math.round((items.length * ESTIMATE_SECONDS[g]) / 60)),
      items: items.slice(0, 4),
      messageIds: items.map((item) => item.m.id),
    };
  });
  const estimateSeconds = deck.reduce((sum, card) => sum + card.count * ESTIMATE_SECONDS[card.group], 0);

  const over = waiting.filter((m) => m.status !== "Quarantined" && m.dueAt != null && m.dueAt < now).sort(byDue);
  const phase = phaseOf(t);
  const accuracyChange = Math.min(4, 1 + Math.floor(t.corrections / 2));

  return {
    generatedAt: now,
    dayStart,
    phase,
    firstDay: t.mailboxes.length === 0 || t.messages.length === 0,
    novaEnabled: t.novaEnabled,
    activeSources: t.mailboxes.filter((mb) => mb.status === "Active").length,
    waiting: waiting.length,
    overdue: waiting.filter((m) => m.dueAt != null && m.dueAt < now).length,
    closedByPeopleToday: t.messages.filter((m) => settledByPerson(m) && m.reviewedAt >= dayStart).length,
    handledByNovaToday: novaToday.length,
    handledByNova: [...handled.values()]
      .sort((a, b) => b.list.length - a.list.length)
      .map((g) => ({ action: g.action, classification: g.classification, count: g.list.length, samples: g.list.slice(0, 3).map((m) => sampleOf(t, m)) })),
    arrivals,
    novaPoints: [...novaToday].sort((a, b) => a.receivedAt - b.receivedAt).map((m) => ({ m, done: true })),
    yours: t.messages
      .filter((m) => needsReview(m) || (settledByPerson(m) && m.reviewedAt >= dayStart))
      .sort((a, b) => a.receivedAt - b.receivedAt)
      .map((m) => ({ m, done: !isWaiting(m) })),
    deck,
    longestWaiting: over[0] ? deckItemOf(t, over[0]) : null,
    estimateMinutes: Math.round(estimateSeconds / 60),
    corrections: { thisWeek: t.corrections, accuracyChange, accuracy: Math.round((90.8 + accuracyChange * 0.9) * 10) / 10 },
    comingBack: t.messages.filter((m) => isSnoozed(m) || isAwaitingReply(m)).length,
    unhealthySources: t.mailboxes.filter((mb) => ["Quiet", "NeverReceived"].includes(healthOf(t, mb))),
    suggestion: ruleSuggestion(t),
    onCallUserId: t.settings.onCallUserId,
    carryOver: phase === "EndOfDay" ? [...waiting].sort(byDue).map((m) => deckItemOf(t, m)) : [],
  };
}

// --- rules: triage.Rule.Covers / Matches / Specificity / FirstMatch, and what they would have done

export function ruleCovers(rule, m) {
  if (m.classification !== rule.classification) return false;
  if (rule.mailboxId && rule.mailboxId !== m.mailboxId) return false;
  switch (rule.senderScope) {
    case "Anyone":
      return true;
    case "AnyCustomer":
      return !!m.matchedCustomerId;
    case "AnyCarrier":
      return !!m.matchedCarrierId;
    case "Domain":
      return domainAllowed(domainOf(m.fromAddress), [rule.sender]);
    case "Address":
      return m.fromAddress.trim().toLowerCase() === rule.sender;
    default:
      return false;
  }
}

const SUGGESTED_ACTION = {
  ProofOfDelivery: "FileOnShipment",
  RateConfirmation: "FileOnShipment",
  Invoice: "FileAndQueuePay",
  Tender: "DraftLoad",
  StatusRequest: "DraftReply",
  DetentionDispute: "DraftReply",
  Other: "Ignore",
};

/** What people keep doing by hand: the same kind, from the same domain, on the same source, settled the same way. */
function ruleSuggestion(t) {
  const since = t.now() - PREVIEW_WINDOW;
  const groups = new Map();
  for (const m of t.messages) {
    if (!isSettled(m) || m.handledByNova || !m.classification || m.receivedAt < since) continue;
    const key = `${m.classification}|${m.senderDomain}|${m.mailboxId}`;
    if (!groups.has(key)) groups.set(key, { classification: m.classification, sender: m.senderDomain, mailboxId: m.mailboxId, total: 0, byHand: 0 });
    const g = groups.get(key);
    g.total += 1;
    if (m.status === "Actioned" && m.reviewedBy) g.byHand += 1;
  }
  const best = [...groups.values()]
    .filter((g) => g.byHand >= 5 && g.byHand / g.total >= 0.9)
    .filter((g) => !t.rules.some((r) => r.enabled && r.classification === g.classification && (!r.mailboxId || r.mailboxId === g.mailboxId) && (r.senderScope === "Anyone" || (r.senderScope === "Domain" && domainAllowed(g.sender, [r.sender])))))
    .sort((a, b) => b.byHand - a.byHand)[0];
  if (!best) return null;
  return {
    classification: best.classification,
    senderScope: "Domain",
    sender: best.sender,
    mailboxId: best.mailboxId,
    action: SUGGESTED_ACTION[best.classification],
    handledByHand: best.byHand,
    total: best.total,
  };
}

const SETTLING = new Set(["FileOnShipment", "FileAndQueuePay", "Ignore"]);
const NEEDS_A_RECORD = new Set(["FileOnShipment", "FileAndQueuePay", "LinkBySender", "DraftReply"]);
const PREVIEW_SENDERS = ["northwindfoods.example", "acme.example", "harborline.example", "crestline.example", "blueridgebrokerage.example"];

function mismatchOf(rule, m) {
  if (m.classificationCorrected) return "KindCorrected";
  if (NEEDS_A_RECORD.has(rule.action) && !hasRecord(m)) return "NoMatchedRecord";
  if (isSettled(m) && !m.handledByNova) {
    if (rule.action === "Ignore" && m.status === "Actioned") return "PersonDisagreed";
    if (SETTLING.has(rule.action) && rule.action !== "Ignore" && m.status === "Ignored") return "PersonDisagreed";
  }
  return null;
}

/**
 * What a rule would have matched over thirty days: the real mail it covers, filled out to the
 * prototype's previewRule figure (a hash of the rule's sentence) so a preview always has a shape.
 */
export function rulePreview(t, input) {
  const rule = normalizeRule(input);
  const now = t.now();
  const since = now - PREVIEW_WINDOW;
  const real = t.messages
    .filter((m) => m.receivedAt >= since && m.novaRead && ruleCovers(rule, m))
    .map((m) => ({ messageId: m.id, fromName: m.fromName, fromAddress: m.fromAddress, subject: m.subject, receivedAt: m.receivedAt, reference: referenceOf(t, m), mismatch: mismatchOf(rule, m) }));
  const h = [...`${rule.classification}${rule.sender || rule.senderScope}${rule.mailboxId ?? ""}${rule.action}`].reduce((s, c) => s + c.charCodeAt(0), 0);
  const n = 8 + (h % 46);
  const wrong = h % 4 === 0 ? 0 : h % 3;
  const senders = rule.senderScope === "Domain" ? [rule.sender] : rule.senderScope === "Address" ? [rule.sender.split("@")[1]] : PREVIEW_SENDERS;
  const synthetic = [];
  for (let i = 0; real.length + synthetic.length < n; i++) {
    const domain = senders[i % senders.length];
    const reference = [`S-1041${20 + i}`, `ref ${4400 + i * 7}`, `PO ${88100 + i * 13}`][i % 3];
    synthetic.push({
      messageId: `imsg_pv_${h}_${i}`,
      fromName: "",
      fromAddress: rule.senderScope === "Address" ? rule.sender : `${["ops", "dispatch", "billing", "loads"][i % 4]}@${domain}`,
      subject: `${KIND_LABEL[rule.classification]} · ${reference}`,
      receivedAt: now - Math.round((i * 2.4 + 0.6) * DAY),
      reference,
      mismatch: i < wrong ? (i % 2 ? "KindCorrected" : "NoMatchedRecord") : null,
    });
  }
  const matches = [...real, ...synthetic].sort((a, b) => (!!b.mismatch - !!a.mismatch) || b.receivedAt - a.receivedAt);
  return {
    matched: matches.length,
    off: matches.filter((x) => x.mismatch).length,
    since,
    matches: matches.slice(0, 12),
  };
}

const ACTIONS = new Set(["FileOnShipment", "FileAndQueuePay", "DraftReply", "DraftLoad", "LinkBySender", "Ignore", "Assign"]);
const SCOPES = new Set(["Anyone", "AnyCustomer", "AnyCarrier", "Domain", "Address"]);

function normalizeRule(input = {}) {
  return {
    classification: input.classification,
    senderScope: input.senderScope,
    sender: (input.sender ?? "").trim().toLowerCase(),
    mailboxId: input.mailboxId ?? null,
    action: input.action,
    assigneeId: input.assigneeId ?? null,
    enabled: input.enabled ?? true,
  };
}

/** triage.Rule.Validate. */
function validateRule(t, rule) {
  if (!KIND_ORDER.includes(rule.classification)) throw invalid("classification", "Kind of message is required", "REQUIRED");
  if (!SCOPES.has(rule.senderScope)) throw invalid("senderScope", "Sender is required", "REQUIRED");
  if (!ACTIONS.has(rule.action)) throw invalid("action", "Action is required", "REQUIRED");
  if (rule.senderScope === "Domain" && !/^[a-z0-9-]+(\.[a-z0-9-]+)+$/.test(rule.sender)) throw invalid("sender", "Sender must be a domain");
  if (rule.senderScope === "Address" && !/^[^@\s]+@[^@\s]+\.[^@\s]+$/.test(rule.sender)) throw invalid("sender", "Sender must be an email address");
  if (!["Domain", "Address"].includes(rule.senderScope) && rule.sender) throw invalid("sender", "A sender is only named for a domain or an address");
  if (rule.action === "Assign" && !rule.assigneeId) throw invalid("assigneeId", "Say who to assign it to", "REQUIRED");
  if (rule.action !== "Assign" && rule.assigneeId) throw invalid("assigneeId", "Only an assigning rule names a person");
  if (rule.assigneeId && !t.users.has(rule.assigneeId)) throw invalid("assigneeId", "That person cannot be assigned messages");
  if (rule.mailboxId && !mailboxOf(t, rule.mailboxId)) throw invalid("mailboxId", "That source does not exist");
}

export function ruleOf(t, id) {
  const rule = t.rules.find((r) => r.id === id);
  if (!rule) throw notFound("Rule");
  return rule;
}


export function ruleLastMatch(t, rule) {
  return t.messages.filter((m) => m.handledByRuleId === rule.id).sort((a, b) => b.reviewedAt - a.reviewedAt)[0] ?? null;
}

export function createRule(t, input) {
  const rule = normalizeRule(input);
  validateRule(t, rule);
  const created = {
    id: `trule_${t.nextId("activity", "r").split("_")[1]}`,
    ...rule,
    createdById: ME_ID,
    baseThisMonth: 0,
    baseTotal: 0,
    version: 1,
    createdAt: t.now(),
  };
  t.rules.unshift(created);
  publishInvalidation({ resource: "inbound_message", action: "updated" });
  return created;
}

export function updateRule(t, id, version, input) {
  const rule = ruleOf(t, id);
  if (rule.version !== version) throw new TriageError("Someone changed this rule since you opened it", { code: "VERSION_MISMATCH", type: "conflict" });
  const next = normalizeRule(input);
  validateRule(t, next);
  Object.assign(rule, next, { version: rule.version + 1 });
  return rule;
}

export function setRuleEnabled(t, id, enabled) {
  const rule = ruleOf(t, id);
  rule.enabled = enabled;
  rule.version += 1;
  return rule;
}

export function deleteRule(t, id) {
  ruleOf(t, id);
  t.rules = t.rules.filter((r) => r.id !== id);
  return true;
}

// --- sender profile and peeks

export function senderProfile(t, messageId) {
  const m = messageOf(t, messageId);
  const domain = m.senderDomain;
  const fromDomain = t.messages.filter((x) => x.senderDomain === domain);
  const kinds = new Map();
  for (const x of fromDomain) if (x.classification) kinds.set(x.classification, (kinds.get(x.classification) ?? 0) + 1);
  const h = [...domain].reduce((s, c) => s + c.charCodeAt(0), 0);
  const customer = m.matchedCustomerId ? t.customers.get(m.matchedCustomerId) : null;
  const role = customer ? (customer.broker ? "Broker" : "Customer") : /drivers\./.test(domain) ? "Driver" : m.matchedCarrierId || /edi|carrier/.test(domain) ? "Carrier" : "Unknown";
  return {
    address: m.fromAddress,
    name: m.fromName,
    domain,
    role,
    since: customer ? customer.since : m.matchedCarrierId ? Date.UTC(2019 + (h % 5), 0, 1) / 1000 : null,
    messagesLast90Days: fromDomain.length + (h % 30) + 4,
    usualReplyMinutes: 12 + (h % 70),
    kinds: [...kinds.entries()].sort((a, b) => b[1] - a[1]).map(([classification, count]) => ({ classification, count })),
    rules: t.rules.filter((r) => (r.senderScope === "Domain" && domainAllowed(domain, [r.sender])) || (r.senderScope === "Address" && r.sender === m.fromAddress.toLowerCase())),
    recent: fromDomain.filter((x) => x.id !== m.id).sort(byNewest).slice(0, 3).map((x) => sampleOf(t, x)),
  };
}

export function shipmentOf(t, id) {
  const shipment = t.shipments.get(id);
  if (!shipment) throw notFound("Shipment");
  return shipment;
}

export function shipmentPeek(t, id) {
  const s = shipmentOf(t, id);
  const customer = s.customerId ? t.customers.get(s.customerId) : null;
  const filedHere = t.messages.flatMap((m) => m.attachments).filter((a) => a.filedShipmentId === s.id).map((a) => a.filedAs);
  const pickupAt = (s.deliveredAt ?? s.eta ?? t.now()) - Math.round(DAY * 0.9);
  const delivered = s.progress >= 1;
  return {
    id: s.id,
    proNumber: s.proNumber,
    bol: s.bol,
    poNumber: s.poNumber,
    status: s.status,
    customerName: customer?.name ?? "",
    equipment: s.equipment,
    driverName: s.driverName,
    progress: s.progress,
    late: s.late,
    eta: delivered ? null : s.eta,
    deliveredAt: s.deliveredAt,
    stops: [
      { kind: "Pickup", city: s.originCity, state: s.originState, scheduledAt: pickupAt, arrivedAt: s.progress > 0 ? pickupAt + 12 * MIN : null, departedAt: s.progress > 0 ? pickupAt + 70 * MIN : null, done: s.progress > 0 },
      { kind: "Delivery", city: s.destinationCity, state: s.destinationState, scheduledAt: s.deliveredAt ?? s.eta, arrivedAt: delivered ? s.deliveredAt - 40 * MIN : null, departedAt: delivered ? s.deliveredAt : null, done: delivered },
    ],
    documentsOnFile: [...new Set([...s.docs, ...filedHere])],
  };
}

export function customerPeek(t, id) {
  const customer = t.customers.get(id);
  if (!customer) throw notFound("Customer");
  const shipments = [...t.shipments.values()].filter((s) => s.customerId === id && (s.progress < 1 || s.deliveredAt > t.now() - 7 * DAY));
  const messages = t.messages.filter((m) => m.matchedCustomerId === id);
  return {
    id: customer.id,
    code: customer.code,
    name: customer.name,
    since: customer.since,
    shipmentsThisWeek: shipments.length,
    waitingInTriage: messages.filter(isWaiting).length,
    shipments: shipments.map((s) => shipmentPeek(t, s.id)),
    messages: [...messages].sort(byNewest).slice(0, 4).map((m) => sampleOf(t, m)),
  };
}

export function stopTimes(t, messageId) {
  const m = messageOf(t, messageId);
  const stops = STOPS[m.id.replace("imsg_", "")];
  if (!stops || !m.matchedShipmentId || m.classification !== "DetentionDispute") return null;
  const day = t.now() - (t.now() % DAY) + stops.day * DAY;
  return {
    shipmentId: m.matchedShipmentId,
    stopName: stops.stopName,
    appointmentStart: day + stops.appointmentStart,
    appointmentEnd: day + stops.appointmentEnd,
    arrivedAt: day + stops.arrivedAt,
    checkedInAt: day + stops.checkedInAt,
    departedAt: day + stops.departedAt,
    lateMinutes: stops.lateMinutes,
    billedDetentionMinutes: stops.billedDetentionMinutes,
    billedDetentionAmount: stops.billedDetentionAmount,
    detentionPaysWhenLate: stops.detentionPaysWhenLate,
    arrivalSource: stops.arrivalSource,
  };
}

// --- templates

export function templateOf(t, id) {
  const template = t.templates.find((x) => x.id === id);
  if (!template) throw notFound("Template");
  return template;
}

/** data2.jsx fillT, with the server's placeholder set ({first} {pro} {status} {eta} {lane} {customer} {po}). */
export function renderTemplate(t, templateId, messageId) {
  const template = templateOf(t, templateId);
  const m = messageOf(t, messageId);
  const s = m.matchedShipmentId ? t.shipments.get(m.matchedShipmentId) : null;
  const customer = m.matchedCustomerId ? t.customers.get(m.matchedCustomerId) : s?.customerId ? t.customers.get(s.customerId) : null;
  const po = s?.poNumber || m.facts.find((f) => /purchase order/i.test(f.label))?.value.replace(/^PO\s*/, "") || "";
  const fields = {
    first: (m.fromName || "there").split(" ")[0],
    pro: s ? s.proNumber : "your load",
    status: s ? (s.progress >= 1 ? "delivered" : s.late ? "running about two hours behind" : "on schedule") : "moving",
    eta: s ? s.etaText.replace(/^ETA /, "") : "to follow",
    lane: s ? `${s.originCity} → ${s.destinationCity}` : "the load",
    customer: customer?.name ?? "your team",
    po: po || "your order",
  };
  return template.body.replace(/\{(\w+)\}/g, (all, key) => fields[key] ?? all);
}

export function createTemplate(t, input) {
  validateTemplate(input);
  const template = { id: `ttpl_${t.nextId("activity", "t").split("_")[1]}`, name: input.name.trim(), body: input.body, createdById: ME_ID, version: 1, createdAt: t.now() };
  t.templates.push(template);
  return template;
}

export function updateTemplate(t, id, version, input) {
  const template = templateOf(t, id);
  if (template.version !== version) throw new TriageError("Someone changed this template since you opened it", { code: "VERSION_MISMATCH", type: "conflict" });
  validateTemplate(input);
  Object.assign(template, { name: input.name.trim(), body: input.body, version: template.version + 1 });
  return template;
}

export function deleteTemplate(t, id) {
  templateOf(t, id);
  t.templates = t.templates.filter((x) => x.id !== id);
  return true;
}

function validateTemplate(input) {
  if (!input.name?.trim()) throw invalid("name", "Name is required", "REQUIRED");
  if (input.name.length > 100) throw invalid("name", "Name must be 100 characters or fewer");
  if (!input.body?.trim()) throw invalid("body", "Write the reply", "REQUIRED");
}

// --- views, targets, settings

export function viewWaiting(t, view) {
  return t.messages.filter((m) => isWaiting(m) && matchesQuery(m, view.query)).length;
}

export function createView(t, name, query) {
  if (!name?.trim()) throw invalid("name", "Name is required", "REQUIRED");
  if (!query?.trim()) throw invalid("query", "Search for something first", "REQUIRED");
  const view = { id: `tview_${t.nextId("activity", "v").split("_")[1]}`, name: name.trim(), query: query.trim(), position: t.views.reduce((max, v) => Math.max(max, v.position), 0) + 1, createdAt: t.now() };
  t.views.push(view);
  return view;
}

export function renameView(t, id, name) {
  const view = t.views.find((v) => v.id === id);
  if (!view) throw notFound("View");
  if (!name?.trim()) throw invalid("name", "Name is required", "REQUIRED");
  view.name = name.trim();
  return view;
}

export function deleteView(t, id) {
  if (!t.views.some((v) => v.id === id)) throw notFound("View");
  t.views = t.views.filter((v) => v.id !== id);
  return true;
}

export function responseTargets(t) {
  return KIND_ORDER.map((classification) => ({
    classification,
    minutes: targetMinutes(t, classification),
    isDefault: !t.targets.has(classification),
  }));
}

export function updateTargets(t, input) {
  for (const row of input) {
    if (!KIND_ORDER.includes(row.classification)) throw invalid("classification", "Kind is not a known one");
    if (row.minutes != null && (row.minutes < 5 || row.minutes > 30 * 1440)) {
      throw invalid("minutes", row.minutes < 5 ? "A target is at least five minutes" : "A target is at most thirty days");
    }
  }
  for (const row of input) {
    if (row.minutes == null) t.targets.delete(row.classification);
    else t.targets.set(row.classification, row.minutes);
  }
  for (const m of t.messages) {
    if (!needsReview(m)) continue;
    const before = m.dueAt;
    applyDue(t, m);
    if (before !== m.dueAt) touch(t, m);
  }
  return responseTargets(t);
}

export function updateSettings(t, input) {
  if (input.version !== t.settings.version) throw new TriageError("Someone changed these settings since you opened them", { code: "VERSION_MISMATCH", type: "conflict" });
  if (input.endOfDayMinute < 0 || input.endOfDayMinute >= 1440) throw invalid("endOfDayMinute", "Pick a time of day");
  if (input.onCallUserId && !t.users.has(input.onCallUserId)) throw invalid("onCallUserId", "That person cannot be on call");
  t.settings = { ...t.settings, onCallUserId: input.onCallUserId ?? null, endOfDayMinute: input.endOfDayMinute, version: t.settings.version + 1 };
  publishInvalidation({ resource: "inbound_message", action: "updated" });
  return t.settings;
}

// --- writes to messages

function settle(t, m, status, note, actor = { kind: "Person", id: ME_ID }) {
  unpark(m);
  m.status = status;
  m.reviewedBy = actor.kind === "Person" ? actor.id : null;
  m.reviewedAt = t.now();
  m.reviewNote = note ?? "";
  m.handledByNova = actor.kind === "Nova";
  applyDue(t, m);
  log(t, m, status === "Actioned" ? "Handled" : "Ignored", { detail: m.reviewNote }, actor);
}

function requireReviewStatus(status) {
  if (status !== "Actioned" && status !== "Ignored") {
    throw invalid("status", "Only Actioned and Ignored are a person's to choose");
  }
}

export function review(t, id, input) {
  requireReviewStatus(input.status);
  const m = messageOf(t, id);
  if (!needsReview(m)) throw new TriageError("This message is already settled", { code: "INVALID_OPERATION" });
  settle(t, m, input.status, input.note);
  touch(t, m);
  return m;
}

export function reviewMany(t, selection, input) {
  requireReviewStatus(input.status);
  const targets = selectionOf(t, selection).filter(needsReview);
  let ruleId = null;
  if (input.teach && targets.length) {
    if (!ACTIONS.has(input.teach)) throw invalid("teach", "Action is not a known one");
    const first = targets[0];
    if (!first.classification) throw invalid("teach", "Nova has not read this message, so there is no kind to make a rule for");
    const rule = createRule(t, {
      classification: first.classification,
      senderScope: "Domain",
      sender: first.senderDomain,
      mailboxId: first.mailboxId,
      action: input.teach,
      assigneeId: input.teach === "Assign" ? ME_ID : null,
    });
    ruleId = rule.id;
    log(t, first, "RuleCreated", { subject: input.teach, detail: first.senderDomain, recordId: rule.id });
  }
  for (const m of targets) {
    settle(t, m, input.status, input.note);
    touch(t, m);
  }
  return { messages: targets, affected: targets.length, ruleId };
}

export function reopenMany(t, selection) {
  const targets = selectionOf(t, selection).filter(isSettled);
  for (const m of targets) {
    m.status = "InReview";
    m.reviewedBy = null;
    m.reviewedAt = null;
    m.reviewNote = "";
    m.handledByNova = false;
    m.handledByRuleId = null;
    m.handledAction = null;
    applyDue(t, m);
    log(t, m, "Reopened");
    touch(t, m);
  }
  return { messages: targets, affected: targets.length, ruleId: null };
}

export function assignMany(t, selection, ownerId) {
  const owner = ownerId ? userOf(t, ownerId) : null;
  const targets = selectionOf(t, selection).filter(needsReview);
  for (const m of targets) {
    if ((m.ownerId ?? null) === (ownerId ?? null)) continue;
    m.ownerId = ownerId ?? null;
    log(t, m, owner ? "Assigned" : "Unassigned", owner ? { subject: owner.name, recordId: owner.id } : {});
    touch(t, m);
  }
  return { messages: targets, affected: targets.length, ruleId: null };
}

export function snoozeMany(t, selection, input) {
  const now = t.now();
  if ((input.until == null) === (input.event == null)) throw invalid("until", "Snooze until a time or until something happens");
  if (input.until != null && (input.until <= now || input.until > now + MAX_SNOOZE_SECONDS)) {
    throw invalid("until", input.until <= now ? "Pick a time in the future" : "Snooze for at most ninety days");
  }
  if (input.event && !["ShipmentDelivered", "DocumentAttached", "InvoicePaid"].includes(input.event)) throw invalid("event", "That event is not one Triage watches");
  if (input.event === "DocumentAttached" && !input.documentTypeId) throw invalid("documentTypeId", "Say which document to wait for", "REQUIRED");
  const targets = selectionOf(t, selection).filter(needsReview);
  if (input.event) {
    const missing = targets.find((m) => !m.matchedShipmentId);
    if (missing) throw invalid("event", "Snoozing until something happens needs a shipment to watch");
  }
  const documentType = input.documentTypeId ? DOCUMENT_TYPES.find((d) => d.id === input.documentTypeId) : null;
  if (input.documentTypeId && !documentType) throw invalid("documentTypeId", "That document type does not exist");
  for (const m of targets) {
    unpark(m);
    Object.assign(m, {
      parked: "Snoozed",
      snoozedUntil: input.until ?? null,
      snoozeEvent: input.event ?? null,
      snoozeWatchId: input.event ? m.matchedShipmentId : null,
      snoozeDocumentTypeId: documentType?.id ?? null,
      snoozedAt: now,
      snoozedBy: ME_ID,
    });
    log(t, m, "Snoozed", { until: input.until ?? null, subject: input.event ?? "", detail: documentType?.name ?? "" });
    touch(t, m);
  }
  return { messages: targets, affected: targets.length, ruleId: null };
}

export function wake(t, id) {
  const m = messageOf(t, id);
  if (!isSnoozed(m)) throw new TriageError("This message is not snoozed", { code: "INVALID_OPERATION" });
  unpark(m);
  log(t, m, "Woken");
  touch(t, m);
  return m;
}

export function waitForReply(t, id, until) {
  const m = messageOf(t, id);
  const now = t.now();
  if (!needsReview(m)) throw new TriageError("Only a message still in the queue can wait on a reply", { code: "INVALID_OPERATION" });
  if (until <= now || until > now + MAX_AWAIT_SECONDS) throw invalid("until", until <= now ? "Pick a time in the future" : "Wait for at most thirty days");
  unpark(m);
  m.parked = "AwaitingReply";
  m.awaitingSince = now;
  m.awaitingUntil = until;
  log(t, m, "WaitingOnReply", { until, subject: m.fromName || m.fromAddress });
  touch(t, m);
  t.later(Number(t.scenario.triageReplyAfterMs ?? 6500), () => theyReply(t, m.id));
  return m;
}

/** app.jsx wait(): the sender answers on the thread; the original comes back and the answer lands. */
function theyReply(t, id) {
  const m = t.messages.find((x) => x.id === id);
  if (!m || !isAwaitingReply(m)) return;
  unpark(m);
  log(t, m, "TheyReplied", { subject: m.fromName || m.fromAddress }, { kind: "System" });
  touch(t, m);
  const answer = t.makeMessage(
    {
      id: `reply_${m.id.replace("imsg_", "")}_${m.activity.length}`,
      from: [m.fromName, m.fromAddress],
      to: [mailboxOf(t, m.mailboxId)?.address ?? m.toAddresses[0]],
      subject: `Re: ${m.subject || "your email"}`,
      min: 0,
      status: "InReview",
      kind: m.classification ?? "Other",
      conf: 0.9,
      mb: m.mailboxId,
      shp: m.matchedShipmentId,
      reason: "A reply in a thread you were waiting on.",
      body: [THEY_REPLY[m.classification] ?? THEY_REPLY.default],
      atts: [],
      run: "run_9001",
    },
    { receivedAt: t.now() },
  );
  answer.threadKey = m.threadKey;
  answer.matchedCustomerId = m.matchedCustomerId;
  arrive(t, answer);
}

export function stopWaiting(t, id) {
  const m = messageOf(t, id);
  if (!isAwaitingReply(m)) throw new TriageError("This message is not waiting on a reply", { code: "INVALID_OPERATION" });
  unpark(m);
  log(t, m, "StoppedWaiting");
  touch(t, m);
  return m;
}

export function link(t, id, input) {
  const m = messageOf(t, id);
  if (!input.shipmentId && !input.customerId && !input.carrierId) throw invalid("shipmentId", "Choose a shipment, a customer, or both", "REQUIRED");
  if (!input.reason?.trim()) throw invalid("reason", "Say why this is the right record", "REQUIRED");
  if (input.shipmentId) shipmentOf(t, input.shipmentId);
  if (input.customerId && !t.customers.has(input.customerId)) throw notFound("Customer");
  if (input.carrierId && !t.carriers.has(input.carrierId)) throw notFound("Carrier");
  if (input.shipmentId) m.matchedShipmentId = input.shipmentId;
  if (input.customerId) m.matchedCustomerId = input.customerId;
  if (input.carrierId) m.matchedCarrierId = input.carrierId;
  m.matchReason = input.reason.trim();
  m.actionState.linkedAt = t.now();
  if (m.status !== "Quarantined") {
    m.failureText = "";
    m.failureCode = "";
  }
  const names = [
    input.shipmentId && t.shipments.get(input.shipmentId).proNumber,
    input.customerId && t.customers.get(input.customerId).name,
    input.carrierId && t.carriers.get(input.carrierId).name,
  ].filter(Boolean);
  log(t, m, "Linked", { subject: names.join(", "), detail: m.matchReason, recordId: input.shipmentId ?? input.customerId ?? input.carrierId });
  touch(t, m);
  return m;
}

export function unlink(t, id, record) {
  const m = messageOf(t, id);
  const field = { Shipment: "matchedShipmentId", Customer: "matchedCustomerId", Carrier: "matchedCarrierId" }[record];
  if (!field) throw invalid("record", "That record kind cannot be unlinked");
  const recordId = m[field];
  if (!recordId) throw new TriageError(`This message is not linked to a ${record.toLowerCase()}`, { code: "INVALID_OPERATION" });
  const name = record === "Shipment" ? t.shipments.get(recordId)?.proNumber : record === "Customer" ? t.customers.get(recordId)?.name : t.carriers.get(recordId)?.name;
  m[field] = null;
  if (!hasRecord(m)) delete m.actionState.linkedAt;
  log(t, m, "Unlinked", { subject: name ?? recordId, recordId });
  touch(t, m);
  return m;
}

export function correct(t, id, classification) {
  const m = messageOf(t, id);
  if (!KIND_ORDER.includes(classification)) throw invalid("classification", "Kind is not a known one");
  if (m.classification === classification) return m;
  const from = m.classification;
  m.classification = classification;
  m.classificationCorrected = true;
  t.corrections += 1;
  t.senderHints.set(m.senderDomain, classification);
  log(t, m, "Corrected", { fromClassification: from, toClassification: classification, subject: m.senderDomain });
  if (needsReview(m)) {
    const before = m.dueAt;
    applyDue(t, m);
    if (before !== m.dueAt) log(t, m, "DueChanged", { until: m.dueAt }, { kind: "System" });
  }
  touch(t, m);
  return m;
}

export function flagFact(t, id, key) {
  const m = messageOf(t, id);
  const fact = m.facts.find((f) => f.key === key);
  if (!fact) throw notFound("Fact");
  if (!fact.flagged) {
    fact.flagged = true;
    t.corrections += 1;
    log(t, m, "FactFlagged", { subject: fact.label, detail: fact.value });
    touch(t, m);
  }
  return m;
}

const FITS = { DuplicateDocument: ["ReplaceExisting", "KeepBoth"], OverlappingLoad: ["SkipDuplicate", "ConfirmNew"] };

export function resolveConflict(t, id, conflictId, resolution) {
  const m = messageOf(t, id);
  const conflict = m.conflicts.find((c) => c.id === conflictId);
  if (!conflict) throw notFound("Conflict");
  if (!FITS[conflict.kind].includes(resolution)) throw invalid("resolution", "That choice does not settle this conflict");
  Object.assign(conflict, { resolution, resolvedAt: t.now(), resolvedBy: ME_ID });
  log(t, m, "ConflictResolved", { subject: resolution, detail: conflict.existingShipmentId ? t.shipments.get(conflict.existingShipmentId)?.proNumber ?? "" : "", recordId: conflict.existingShipmentId });
  touch(t, m);
  return m;
}

export function reopenConflict(t, id, conflictId) {
  const m = messageOf(t, id);
  const conflict = m.conflicts.find((c) => c.id === conflictId);
  if (!conflict) throw notFound("Conflict");
  if (!conflict.resolution) throw new TriageError("This conflict is not settled", { code: "INVALID_OPERATION" });
  const was = conflict.resolution;
  Object.assign(conflict, { resolution: null, resolvedAt: null, resolvedBy: null });
  log(t, m, "ConflictReopened", { subject: was });
  touch(t, m);
  return m;
}

// --- reply attachments: documentmailservice (Resolve, Mailable) and inboundreplyservice.Attachables

/** emailports.MaxAttachmentBytes: what fits in the smallest provider's message once base64-encoded. */
export const MAX_REPLY_ATTACHMENT_BYTES = Math.floor((10 * 1024 * 1024 - 16 * 1024) / 4) * 3;
const MAX_REPLY_ATTACHMENTS = 20;
const SHIPMENT_DOCUMENT_FILES = {
  "Bill of lading": { prefix: "BOL", byteSize: 182_000 },
  "Proof of delivery": { prefix: "POD", byteSize: 241_000 },
  "Rate confirmation": { prefix: "RateCon", byteSize: 96_000 },
  Invoice: { prefix: "Invoice", byteSize: 88_000 },
};
const EXTRA_SHIPMENT_DOCUMENTS = {
  shp_seedpay002: [
    { fileName: "ELD_SEED-PAY-002_Oct3.pdf", documentType: "ELD record", byteSize: 214_000, detention: true },
    { fileName: "GateLog_SEED-PAY-002_Oct3.jpg", documentType: "Gate log", byteSize: 1_640_000, contentType: "image/jpeg" },
  ],
};
const CUSTOMER_DOCUMENTS = {
  "Harborline Grocers": [{ fileName: "Harborline_detention_terms_2026.pdf", documentType: "Rate agreement", byteSize: 128_000 }],
  "Crestline Retail": [{ fileName: "Crestline_routing_guide.pdf", documentType: "Routing guide", byteSize: 2_310_000 }],
};

function storedDocuments(t) {
  if (t.documents) return t.documents;
  t.documents = new Map();
  const at = t.now() - 6 * DAY;
  const add = (doc) => t.documents.set(doc.id, { contentType: "application/pdf", createdAt: at, ...doc });
  for (const shipment of t.shipments.values()) {
    for (const type of shipment.docs ?? []) {
      const file = SHIPMENT_DOCUMENT_FILES[type] ?? { prefix: type.replace(/\s+/g, ""), byteSize: 64_000 };
      add({ id: `doc_${shipment.id}_${file.prefix.toLowerCase()}`, fileName: `${file.prefix}_${shipment.proNumber}.pdf`, documentType: type, byteSize: file.byteSize, resourceType: "shipment", resourceId: shipment.id });
    }
    for (const [index, extra] of (EXTRA_SHIPMENT_DOCUMENTS[shipment.id] ?? []).entries()) {
      add({ id: `doc_${shipment.id}_extra${index + 1}`, resourceType: "shipment", resourceId: shipment.id, ...extra, createdAt: t.now() - 4 * DAY });
    }
  }
  for (const customer of t.customers.values()) {
    for (const [index, extra] of (CUSTOMER_DOCUMENTS[customer.name] ?? []).entries()) {
      add({ id: `doc_${customer.id}_${index + 1}`, resourceType: "customer", resourceId: customer.id, ...extra, createdAt: t.now() - 40 * DAY });
    }
  }
  return t.documents;
}

function messageDocuments(t, m) {
  return m.attachments.filter(readable).map((a) => ({
    id: a.documentId,
    fileName: a.fileName,
    contentType: a.contentType,
    documentType: a.reading?.documentType ?? "",
    byteSize: a.byteSize,
    createdAt: a.createdAt,
    resourceType: "inbound_message",
    resourceId: m.id,
  }));
}

function documentById(t, id) {
  const stored = storedDocuments(t).get(id);
  if (stored) return stored;
  for (const m of t.messages) {
    const found = messageDocuments(t, m).find((doc) => doc.id === id);
    if (found) return found;
  }
  return null;
}

function fileOf(doc) {
  return { documentId: doc.id, fileName: doc.fileName, contentType: doc.contentType, byteSize: doc.byteSize };
}

function detentionRecordOf(t, shipmentId) {
  return [...storedDocuments(t).values()].find((doc) => doc.resourceType === "shipment" && doc.resourceId === shipmentId && doc.detention) ?? null;
}

function humanBytes(bytes) {
  if (bytes < 1024) return `${bytes} B`;
  const units = ["KB", "MB", "GB"];
  let value = bytes / 1024;
  let unit = 0;
  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024;
    unit += 1;
  }
  return `${value.toFixed(1)} ${units[unit]}`;
}

function resolveAttachments(t, documentIds, field = "documentIds") {
  const ids = [...new Set(documentIds)];
  if (ids.length > MAX_REPLY_ATTACHMENTS) throw invalid(field, `An email can carry at most ${MAX_REPLY_ATTACHMENTS} files`);
  const files = ids.map((id) => {
    const doc = documentById(t, id);
    if (!doc) throw invalid(`${field}[${documentIds.indexOf(id)}]`, "This file no longer exists");
    return fileOf(doc);
  });
  const total = files.reduce((sum, file) => sum + file.byteSize, 0);
  if (total > MAX_REPLY_ATTACHMENT_BYTES) {
    throw invalid(field, `These files add up to ${humanBytes(total)}; an email can carry at most ${humanBytes(MAX_REPLY_ATTACHMENT_BYTES)}`);
  }
  return files;
}

const ATTACHABLE_SOURCES = [
  ["inbound_message", "Message"],
  ["shipment", "Shipment"],
  ["customer", "Customer"],
  ["carrier", "Carrier"],
];

export function replyAttachables(t, messageId) {
  const m = messageOf(t, messageId);
  const records = { inbound_message: m.id, shipment: m.matchedShipmentId, customer: m.matchedCustomerId, carrier: m.matchedCarrierId };
  const stored = [...storedDocuments(t).values()];
  const out = [];
  const seen = new Set();
  for (const [resourceType, source] of ATTACHABLE_SOURCES) {
    const resourceId = records[resourceType];
    if (!resourceId) continue;
    const docs = resourceType === "inbound_message" ? [...messageDocuments(t, m), ...stored.filter((doc) => doc.resourceType === resourceType && doc.resourceId === resourceId)] : stored.filter((doc) => doc.resourceType === resourceType && doc.resourceId === resourceId);
    for (const doc of [...docs].sort((a, b) => b.createdAt - a.createdAt)) {
      if (seen.has(doc.id)) continue;
      seen.add(doc.id);
      out.push({ ...doc, source });
    }
  }
  return out;
}

/** POST /documents/upload/: the document route, kept to what a reply's upload needs. */
export function uploadDocument(t, { fileName, contentType, byteSize, resourceType, resourceId }) {
  if (!fileName) throw invalid("file", "Choose a file to upload", "REQUIRED");
  if (!resourceType || !resourceId) throw invalid("resourceId", "Say what the file is for", "REQUIRED");
  if (byteSize === 0) throw invalid("file", "File cannot be empty", "REQUIRED");
  const doc = {
    id: t.nextId("document", "doc"),
    fileName,
    contentType: contentType || "application/octet-stream",
    documentType: "",
    byteSize,
    createdAt: t.now(),
    resourceType,
    resourceId,
  };
  storedDocuments(t).set(doc.id, doc);
  return doc;
}

// --- replies

function openDraft(m) {
  const reply = latestReply(m);
  return reply && (reply.status === "Draft" || reply.status === "Failed") ? reply : null;
}

export function replyDraftOf(m) {
  return openDraft(m);
}

function newReply(t, m, fields) {
  const now = t.now();
  const reply = {
    id: t.nextId("reply", "irpl"),
    messageId: m.id,
    status: "Draft",
    fromAddress: mailboxOf(t, m.mailboxId)?.address ?? m.toAddresses[0] ?? "",
    toAddress: m.fromAddress,
    ccAddresses: [],
    subject: `Re: ${m.subject || "your email"}`,
    body: "",
    draftedByNova: false,
    templateId: null,
    failureText: "",
    suggestedAddress: null,
    sentById: null,
    sentAt: null,
    attachments: [],
    createdAt: now,
    updatedAt: now,
    ...fields,
  };
  m.replies.push(reply);
  return reply;
}

function novaReplyBody(m) {
  return REPLY_DRAFTS[m.id] ?? DEFAULT_REPLY;
}

function writeNovaDraft(t, m, body = novaReplyBody(m), evidence = []) {
  const draft = openDraft(m);
  const attachments = [...(draft?.attachments ?? [])];
  for (const file of evidence) if (!attachments.some((a) => a.documentId === file.documentId)) attachments.push(file);
  const reply = draft
    ? Object.assign(draft, { body, draftedByNova: true, status: "Draft", failureText: "", suggestedAddress: null, attachments, updatedAt: t.now() })
    : newReply(t, m, { body, draftedByNova: true, attachments });
  log(t, m, "ReplyDrafted", { subject: reply.toAddress }, { kind: "Nova" });
  touch(t, m);
  return reply;
}

export async function draftReply(t, messageId) {
  const m = messageOf(t, messageId);
  await new Promise((resolve) => t.later(1300, resolve));
  return writeNovaDraft(t, m);
}

export function saveDraft(t, messageId, subject, body, documentIds) {
  const m = messageOf(t, messageId);
  if (!subject?.trim()) throw invalid("subject", "Subject is required", "REQUIRED");
  if (!body?.trim()) throw invalid("body", "Write something before sending", "REQUIRED");
  const draft = openDraft(m);
  const attachments = documentIds == null ? draft?.attachments ?? [] : resolveAttachments(t, documentIds);
  const reply = draft
    ? Object.assign(draft, { subject, body, attachments, updatedAt: t.now() })
    : newReply(t, m, { subject, body, attachments });
  touch(t, m);
  return reply;
}

export function discardDraft(t, messageId) {
  const m = messageOf(t, messageId);
  const draft = openDraft(m);
  if (draft) {
    m.replies = m.replies.filter((r) => r !== draft);
    touch(t, m);
  }
  return m;
}

export async function sendReply(t, input) {
  const m = messageOf(t, input.messageId);
  if (!input.body?.trim()) throw invalid("body", "Write something before sending", "REQUIRED");
  if (!input.subject?.trim()) throw invalid("subject", "Subject is required", "REQUIRED");
  if (input.toAddress && !/^[^@\s]+@[^@\s]+\.[^@\s]+$/.test(input.toAddress)) throw invalid("toAddress", "That is not an email address");
  if (input.templateId) templateOf(t, input.templateId);
  const attachments = resolveAttachments(t, input.documentIds ?? [], "input.documentIds");
  await new Promise((resolve) => t.later(600, resolve));
  const to = (input.toAddress ?? m.fromAddress).toLowerCase();
  const draft = openDraft(m);
  const reply = draft ?? newReply(t, m, {});
  Object.assign(reply, {
    subject: input.subject,
    body: input.body,
    toAddress: to,
    templateId: input.templateId ?? null,
    draftedByNova: !!input.draftedByNova,
    attachments,
    updatedAt: t.now(),
  });
  const refusal = REFUSING_SENDERS[to];
  if (refusal) {
    Object.assign(reply, { status: "Failed", failureText: refusal.failureText, suggestedAddress: refusal.suggestedAddress });
    log(t, m, "ReplyFailed", { subject: to, detail: refusal.failureText });
  } else {
    Object.assign(reply, { status: "Sent", failureText: "", suggestedAddress: null, sentById: ME_ID, sentAt: t.now() });
    log(t, m, "Replied", { subject: reply.fromAddress, detail: attachments.map((a) => a.fileName).join(", "), recordId: reply.id });
  }
  touch(t, m);
  return reply;
}

export function draftMany(t, messageIds) {
  const targets = messageIds.map((id) => messageOf(t, id));
  let scheduled = 0;
  targets.forEach((m) => {
    const next = nextOpen(actionsOf(m));
    if (!next || !next.novaCanDo) return;
    const delay = 1200 + scheduled * 400;
    scheduled += 1;
    t.later(delay, () => {
      if (!t.messages.includes(m)) return;
      if (next.kind === "BuildLoad") writeLoadDraft(t, m);
      else writeNovaDraft(t, m);
    });
  });
  return scheduled;
}

// --- loads

function writeLoadDraft(t, m) {
  const draft = LOAD_DRAFTS[m.id];
  const customer = m.matchedCustomerId ? t.customers.get(m.matchedCustomerId)?.name ?? "" : "";
  m.loadDraft = {
    source: draft?.source ?? "",
    fields: (draft?.fields ?? [["Customer", customer], ["Origin", ""], ["Destination", ""], ["Pickup", ""]]).map(([key, value]) => ({ key, value })),
    draftedAt: t.now(),
    draftedByNova: true,
  };
  log(t, m, "LoadDrafted", { subject: m.loadDraft.source }, { kind: "Nova" });
  touch(t, m);
  return m;
}

export async function draftLoad(t, messageId) {
  const m = messageOf(t, messageId);
  if (m.actionState.createdShipmentId) throw new TriageError("The load is already on the board", { code: "INVALID_OPERATION" });
  await new Promise((resolve) => t.later(1200, resolve));
  return writeLoadDraft(t, m);
}

export function discardLoad(t, messageId) {
  const m = messageOf(t, messageId);
  if (m.loadDraft) {
    m.loadDraft = null;
    touch(t, m);
  }
  return m;
}

const LOAD_KEYS = new Set(["Customer", "Origin", "Destination", "Pickup", "Delivery", "Equipment", "Weight", "Rate", "Loads", "WeekOf", "RespondBy", "Reference", "Commodity", "Instructions"]);

export function createLoad(t, messageId, fields) {
  const m = messageOf(t, messageId);
  if (m.actionState.createdShipmentId) throw new TriageError("The load is already on the board", { code: "INVALID_OPERATION" });
  for (const field of fields) if (!LOAD_KEYS.has(field.key)) throw invalid("fields", "That field is not part of a load");
  const value = (key) => fields.find((f) => f.key === key)?.value?.trim() ?? "";
  const place = (text) => {
    const match = /^(.*?)[,\s]+([A-Z]{2})$/.exec(text);
    return match ? { city: match[1].replace(/^.*,\s*/, ""), state: match[2] } : { city: text.split(",")[0], state: "" };
  };
  const origin = place(value("Origin"));
  const destination = place(value("Destination"));
  const customerName = value("Customer");
  const customer = [...t.customers.values()].find((c) => c.name === customerName) ?? (m.matchedCustomerId ? t.customers.get(m.matchedCustomerId) : null);
  const pro = `S-1042${t.nextPro++}`;
  const shipment = t.addShipment({
    id: `shp_${pro.replace(/\D/g, "")}`,
    pro,
    o: `${origin.city}${origin.state ? `, ${origin.state}` : ""}`,
    d: `${destination.city}${destination.state ? `, ${destination.state}` : ""}`,
    bol: "",
    po: value("Reference"),
    status: "New",
    p: 0,
    drv: "",
    eq: value("Equipment") || "Dry van 53'",
    cust: customer?.name,
    etaText: "Not dispatched",
    eta: 2 * DAY,
    docs: [],
    ageDays: 0,
  });
  shipment.createdAt = t.now();
  m.actionState.createdShipmentId = shipment.id;
  m.actionState.createdShipmentAt = t.now();
  m.loadDraft = null;
  log(t, m, "LoadCreated", { subject: pro, recordId: shipment.id });
  touch(t, m);
  return m;
}

// --- documents

function documentTypeName(id, fallback) {
  if (!id) return fallback;
  const type = DOCUMENT_TYPES.find((d) => d.id === id);
  if (!type) throw invalid("documentTypeId", "That document type does not exist");
  return type.name;
}

function attachmentOf(t, attachmentId) {
  for (const m of t.messages) {
    const attachment = m.attachments.find((a) => a.id === attachmentId);
    if (attachment) return { m, attachment };
  }
  throw notFound("Attachment");
}

function fileOne(t, m, a, shipment, { documentTypeId, pages, actor }) {
  const now = t.now();
  if (pages?.length) {
    for (const range of pages) {
      if (range.fromPage < 1 || range.toPage < range.fromPage || range.toPage > a.pageCount) throw invalid("pages", `Pages run from 1 to ${a.pageCount}`);
    }
    a.splitDocuments = pages.map((range) => ({ fromPage: range.fromPage, toPage: range.toPage, documentId: t.nextId("document", "doc"), documentType: documentTypeName(range.documentTypeId, "Other") }));
    a.filedAs = `${pages.length} documents (${[...new Set(a.splitDocuments.map((d) => d.documentType.toLowerCase()))].join(", ")})`;
    log(t, m, "Split", { subject: a.fileName, detail: `${pages.length} documents`, recordId: shipment.id }, actor);
  } else {
    a.filedAs = documentTypeName(documentTypeId, a.reading?.documentType ?? "Other");
  }
  a.filedShipmentId = shipment.id;
  a.filedAt = now;
  a.filedBy = actor.kind === "Person" ? actor.id : null;
  if (!m.matchedShipmentId) {
    m.matchedShipmentId = shipment.id;
    if (!m.matchReason) m.matchReason = `Filed by hand on ${shipment.proNumber}.`;
  }
  log(t, m, "Filed", { subject: a.fileName, detail: shipment.proNumber, recordId: shipment.id }, actor);
}

export function fileAttachment(t, input) {
  const { m, attachment } = attachmentOf(t, input.attachmentId);
  const shipment = shipmentOf(t, input.shipmentId);
  if (!readable(attachment)) throw new TriageError("This file could not be read, so it cannot be filed", { code: "INVALID_OPERATION", field: "attachmentId" });
  fileOne(t, m, attachment, shipment, { documentTypeId: input.documentTypeId, pages: input.pages, actor: { kind: "Person", id: ME_ID } });
  touch(t, m);
  return m;
}

export function fileMany(t, messageIds) {
  const changed = [];
  for (const id of messageIds) {
    const m = messageOf(t, id);
    if (!m.matchedShipmentId) continue;
    const shipment = shipmentOf(t, m.matchedShipmentId);
    const open = m.attachments.filter((a) => readable(a) && !filed(a));
    if (!open.length) continue;
    for (const a of open) fileOne(t, m, a, shipment, { actor: { kind: "Person", id: ME_ID } });
    touch(t, m);
    changed.push(m);
  }
  return { messages: changed, affected: changed.length, ruleId: null };
}

export function unfile(t, attachmentId) {
  const { m, attachment } = attachmentOf(t, attachmentId);
  if (!filed(attachment)) throw new TriageError("This file is not filed", { code: "INVALID_OPERATION" });
  const pro = t.shipments.get(attachment.filedShipmentId)?.proNumber ?? "";
  Object.assign(attachment, { filedShipmentId: null, filedAs: null, filedAt: null, filedBy: null, splitDocuments: [] });
  log(t, m, "Unfiled", { subject: attachment.fileName, detail: pro });
  touch(t, m);
  return m;
}

export function logException(t, messageId, input) {
  const m = messageOf(t, messageId);
  if (!m.matchedShipmentId) throw new TriageError("Link a shipment before logging an exception on it", { code: "INVALID_OPERATION" });
  if (!["Shortage", "Overage", "Damage"].includes(input.kind)) throw invalid("kind", "That exception is not a known one");
  if (!(input.quantity > 0)) throw invalid("quantity", "Say how many", "REQUIRED");
  if (!input.unit?.trim()) throw invalid("unit", "Say what was short, over or damaged", "REQUIRED");
  m.actionState.exceptionKind = input.kind;
  m.actionState.exceptionLoggedAt = t.now();
  log(t, m, "ExceptionLogged", { subject: `${input.quantity} ${input.unit.trim()}`, detail: input.kind, recordId: m.matchedShipmentId });
  touch(t, m);
  return m;
}

export function blockSender(t, messageId) {
  const m = messageOf(t, messageId);
  const mailbox = mailboxOf(t, m.mailboxId);
  if (!mailbox) throw notFound("Source");
  if (!mailbox.blockedSenders.includes(m.senderDomain)) mailbox.blockedSenders.push(m.senderDomain);
  touchMailbox(t, mailbox);
  m.actionState.senderBlockedAt = t.now();
  log(t, m, "SenderBlocked", { subject: m.senderDomain, detail: mailbox.name });
  if (needsReview(m)) settle(t, m, "Ignored", "Blocked sender.");
  touch(t, m);
  return m;
}

export function unblockSender(t, mailboxId, sender) {
  const mailbox = mailboxOf(t, mailboxId);
  if (!mailbox) throw notFound("Source");
  const normalized = sender.trim().toLowerCase();
  if (!mailbox.blockedSenders.includes(normalized)) throw new TriageError("That sender is not blocked on this source", { code: "INVALID_OPERATION" });
  mailbox.blockedSenders = mailbox.blockedSenders.filter((s) => s !== normalized);
  touchMailbox(t, mailbox);
  return mailbox;
}

export function decideStops(t, messageId, decision) {
  const m = messageOf(t, messageId);
  if (decision !== "Waive" && decision !== "StandBy") throw invalid("decision", "Waive or stand by the charge");
  if (m.classification !== "DetentionDispute" || !m.matchedShipmentId) throw new TriageError("Only a detention dispute on a shipment has stop times to check", { code: "INVALID_OPERATION" });
  m.actionState.stopsDecision = decision;
  m.actionState.stopsDecidedAt = t.now();
  log(t, m, "StopsDecided", { subject: decision, recordId: m.matchedShipmentId });
  if (decision === "StandBy") {
    const record = detentionRecordOf(t, m.matchedShipmentId);
    writeNovaDraft(t, m, record ? STOPS_REPLIES.StandBy : STOPS_REPLIES.StandByNoRecord, record ? [fileOf(record)] : []);
  } else {
    writeNovaDraft(t, m, STOPS_REPLIES[decision]);
  }
  return m;
}

// --- sources

const PROVIDERS = new Set(["Postmark", "Resend"]);
const POLICIES = new Set(["AlwaysReview", "ReviewBelowConfidence", "AutoHandle"]);

function validateMailbox(t, input, ignoreId = null) {
  if (!input.name?.trim()) throw invalid("name", "Name is required", "REQUIRED");
  if (!/^[^@\s]+@[^@\s]+\.[^@\s]+$/.test(input.address ?? "")) throw invalid("address", "That is not an email address");
  if (!PROVIDERS.has(input.provider)) throw invalid("provider", "Provider is required", "REQUIRED");
  if (!POLICIES.has(input.reviewPolicy)) throw invalid("reviewPolicy", "Review policy is required", "REQUIRED");
  if (!(input.minConfidence >= 0 && input.minConfidence <= 1)) throw invalid("minConfidence", "Confidence is between 0 and 1");
  if (t.mailboxes.some((mb) => mb.id !== ignoreId && mb.address.toLowerCase() === input.address.toLowerCase())) {
    throw new TriageError("Another source already listens on this address", { code: "DUPLICATE", field: "address", type: "conflict" });
  }
}

function newToken() {
  return `imtk_${Math.random().toString(36).slice(2, 12)}${Math.random().toString(36).slice(2, 12)}`;
}

export function createMailbox(t, input, signingSecret, apiKey) {
  validateMailbox(t, input);
  const now = t.now();
  const mailbox = {
    id: `imb_${input.name.toLowerCase().replace(/[^a-z0-9]+/g, "")}_${t.mailboxes.length + 1}`,
    key: null,
    name: input.name.trim(),
    address: input.address.trim().toLowerCase(),
    provider: input.provider,
    purpose: input.purpose ?? "",
    reviewPolicy: input.reviewPolicy,
    minConfidence: input.minConfidence,
    status: input.status ?? "Active",
    hasSigningSecret: !!signingSecret,
    hasApiKey: !!apiKey,
    lastReceivedAt: null,
    testSentAt: null,
    testReceivedAt: null,
    blockedSenders: [],
    version: 1,
    createdAt: now,
    updatedAt: now,
  };
  t.mailboxes.push(mailbox);
  publishInvalidation({ resource: "inbound_mailbox", action: "created", recordId: mailbox.id });
  if (t.messages.length === 0 && !t.readingBacklog) readBacklog(t, mailbox);
  return { mailbox, token: newToken() };
}

export function updateMailbox(t, id, version, input) {
  const mailbox = mailboxOf(t, id);
  if (!mailbox) throw notFound("Source");
  if (mailbox.version !== version) throw new TriageError("Someone changed this source since you opened it", { code: "VERSION_MISMATCH", type: "conflict" });
  validateMailbox(t, input, id);
  if (input.provider !== mailbox.provider) {
    mailbox.hasSigningSecret = false;
    mailbox.hasApiKey = false;
  }
  Object.assign(mailbox, {
    name: input.name.trim(),
    address: input.address.trim().toLowerCase(),
    provider: input.provider,
    purpose: input.purpose ?? mailbox.purpose,
    reviewPolicy: input.reviewPolicy,
    minConfidence: input.minConfidence,
    status: input.status,
  });
  touchMailbox(t, mailbox);
  return mailbox;
}

export function rotateToken(t, id) {
  const mailbox = mailboxOf(t, id);
  if (!mailbox) throw notFound("Source");
  touchMailbox(t, mailbox);
  return { mailbox, token: newToken() };
}

export function setSecret(t, id, secret, field) {
  const mailbox = mailboxOf(t, id);
  if (!mailbox) throw notFound("Source");
  if (!secret?.trim()) throw invalid(field === "hasApiKey" ? "apiKey" : "secret", "Paste the key", "REQUIRED");
  if (field === "hasApiKey" && !secret.startsWith("re_")) throw invalid("apiKey", "A Resend API key starts with re_");
  mailbox[field] = true;
  touchMailbox(t, mailbox);
  return mailbox;
}

export function sendMailboxTest(t, id) {
  const mailbox = mailboxOf(t, id);
  if (!mailbox) throw notFound("Source");
  mailbox.testSentAt = t.now();
  touchMailbox(t, mailbox);
  t.later(1500, () => {
    const now = t.now();
    mailbox.testReceivedAt = now;
    mailbox.lastReceivedAt = now;
    touchMailbox(t, mailbox);
    const test = t.makeMessage(
      {
        id: `test_${mailbox.id.replace("imb_", "")}_${now}`,
        from: ["System Administrator", "admin@trenova.app"],
        to: [mailbox.address],
        subject: "Test — does this arrive?",
        min: 0,
        status: "InReview",
        kind: "Other",
        conf: 0.99,
        mb: mailbox.id,
        reason: "A test you sent.",
        body: [`Testing the ${mailbox.name} source.`],
        atts: [],
        run: "run_test",
      },
      { receivedAt: now },
    );
    log(t, test, "TestReceived", { subject: mailbox.address }, { kind: "System" });
    arrive(t, test);
  });
  return mailbox;
}

export function handOff(t, toUserId) {
  const user = t.users.get(toUserId);
  if (!user) throw invalid("toUserId", "That person cannot take the queue");
  const targets = t.messages.filter(isWaiting);
  const now = t.now();
  for (const m of targets) {
    m.ownerId = user.id;
    m.actionState.handedOffTo = user.id;
    m.actionState.handedOffAt = now;
    log(t, m, "HandedOff", { subject: user.name, recordId: user.id });
    touch(t, m);
  }
  return { user, affected: targets.length };
}

export function readUnread(t) {
  const unreadIds = [...t.unread.keys()];
  t.novaEnabled = true;
  t.later(1600, () => {
    for (const id of unreadIds) {
      const m = t.messages.find((x) => x.id === id);
      if (!m) continue;
      readNow(t, m);
      touch(t, m);
    }
  });
  return unreadIds.length;
}

// --- arrivals

/** A message lands: it joins the queue, its source counts it, and open inboxes are told. */
function arrive(t, m) {
  const hint = t.senderHints.get(m.senderDomain);
  if (hint && m.classification !== hint) {
    m.classification = hint;
    m.classificationCorrected = false;
  }
  applyDue(t, m);
  t.messages.push(m);
  const mailbox = mailboxOf(t, m.mailboxId);
  if (mailbox) {
    mailbox.lastReceivedAt = m.receivedAt;
    touchMailbox(t, mailbox);
  }
  publishInvalidation({ resource: "inbound_message", action: "created", recordId: m.id });
  return m;
}

function arriveFromPool(t) {
  const template = LIVE_POOL[t.livePool];
  if (!template) return;
  t.livePool += 1;
  const m = t.makeMessage({ ...template, id: `live_${t.livePool}`, min: 0, status: "InReview", run: "run_live" }, { receivedAt: t.now() });
  arrive(t, m);
  const marcus = { userId: "usr_marcus", connectionId: "conn_marcus", name: "Marcus Lee", messageId: m.id, activity: "viewing" };
  t.later(5000, () => {
    t.presence.set(marcus.connectionId, marcus);
    publish("presence", { scope: PRESENCE_SCOPE, action: "enter", ...marcus }, { resumable: false });
  });
  t.later(20000, () => {
    t.presence.delete(marcus.connectionId);
    publish("presence", { scope: PRESENCE_SCOPE, action: "leave", ...marcus }, { resumable: false });
  });
}

/** First day: a source is added and Nova reads what was already waiting there, over about three seconds. */
function readBacklog(t, mailbox) {
  t.readingBacklog = true;
  const backlog = buildTriageWorld({ anchor: t.anchor, scenario: { ...t.scenario, triage: "live" } });
  for (const [id, shipment] of backlog.shipments) if (!t.shipments.has(id)) t.shipments.set(id, shipment);
  for (const kind of ["attachment", "document", "reply"]) t.ids[kind] = Math.max(t.ids[kind], backlog.ids[kind]);
  const known = new Set(t.mailboxes.map((mb) => mb.id));
  const messages = [...backlog.messages].sort((a, b) => a.receivedAt - b.receivedAt);
  messages.forEach((m, index) => {
    if (!known.has(m.mailboxId)) m.mailboxId = mailbox.id;
    m.activity = m.activity.map((line) => ({ ...line, id: t.nextId("activity", "iact") }));
    const final = { status: m.status, novaRead: m.novaRead };
    t.later(Math.round((3000 * index) / messages.length), () => {
      m.status = "Processing";
      m.novaRead = false;
      t.messages.push(m);
      applyDue(t, m);
      publishInvalidation({ resource: "inbound_message", action: "created", recordId: m.id });
      t.later(700, () => {
        Object.assign(m, final);
        applyDue(t, m);
        for (const mb of t.mailboxes) {
          if (mb.id === m.mailboxId && (!mb.lastReceivedAt || m.receivedAt > mb.lastReceivedAt)) mb.lastReceivedAt = m.receivedAt;
        }
        publishInvalidation({ resource: "inbound_message", action: "updated", recordId: m.id });
        if (index === messages.length - 1) {
          t.readingBacklog = false;
          if (t.scenario.triage !== "noMailbox") {
            for (const presence of VIEWING) if (t.messages.some((x) => x.id === presence.messageId)) t.presence.set(presence.connectionId, presence);
          }
        }
      });
    });
  });
}

// --- presence

export function seedPresence(t) {
  if (t.scenario.triage !== "live" && t.scenario.triage !== "deskOff") return;
  for (const presence of VIEWING) if (t.messages.some((m) => m.id === presence.messageId)) t.presence.set(presence.connectionId, presence);
}

export function presenceSnapshot(t) {
  return { scope: PRESENCE_SCOPE, members: [...t.presence.values()].map((p) => ({ userId: p.userId, connectionId: p.connectionId, name: p.name, messageId: p.messageId ?? null, activity: p.activity ?? "viewing" })) };
}
