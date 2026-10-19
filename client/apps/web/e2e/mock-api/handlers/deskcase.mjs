// Two Desk conversations: a case about a delivered shipment (its card,
// ready-to-bill checklist, snooze, the parties it can wait on and the agent's
// waits), and a reply still under way that a person steers and queues
// follow-ups for. Writes follow assistantcaseservice, agentwaitservice and
// assistantqueueservice: the same limits, the same refusals.

export const CASE_THREAD_ID = "athr_desk_case";
export const LIVE_THREAD_ID = "athr_desk_live";

const SHIPMENT_ID = "shp_s2610_1342";
const LIVE_TURN_ID = "aturn_desk_live";
const MIN_SNOOZE_SECONDS = 60;
const MAX_SNOOZE_SECONDS = 30 * 24 * 60 * 60;
const DEFAULT_REPLY_HOURS = 72;
const MAX_REPLY_HOURS = 168;
const MAX_OPEN_WAITS = 10;
const MAX_QUEUED = 20;
const MAX_QUEUED_RUNES = 16_000;
const CASE_QUESTION = "What's left before we can bill S2610-1342?";
const LIVE_QUESTION = "Which delivered loads from last week still can't be billed, and why?";
const FOLLOW_UPS = {
  athr_desk_case:
    "Noted. The case is still waiting on Red River Freight for the signed POD and the rate con, and I'll pick it back up as soon as either comes in.",
  athr_desk_live:
    "Nine of last week's delivered loads can't be billed yet. Six are waiting on a POD and three on a carrier's rate confirmation.",
};

const PARTIES = [
  { kind: "Customer", id: "cus_acme", name: "Acme Foods" },
  { kind: "Carrier", id: "car_redriver", name: "Red River Freight" },
];

/** The organization's added step a person ticks on the ready-to-bill checklist. */
const MANUAL_STEPS = new Set(["custom:lumper"]);

const PARTY_RANK = { Customer: 0, Carrier: 1, Reply: 2, Event: 3 };

function problem(status, type, title, detail, errors) {
  return { __status: status, body: { type, title, status, detail, ...(errors ? { errors } : {}) } };
}

function invalid(field, code, message) {
  return problem(400, "validation-error", "Validation Failed", message, [{ field, code, message }]);
}

function business(message) {
  return problem(422, "business-rule-violation", "Business Rule Violation", message);
}

function notFound(what) {
  return problem(
    404,
    "resource-not-found",
    "Resource Not Found",
    `${what} not found within your organization`,
  );
}

/** 9:00 tomorrow in the fixture organization's zone, UTC-5 on the board's anchor day. */
function tomorrowAtNine(anchor) {
  return Math.floor(anchor / 86400) * 86400 + 86400 + 14 * 3600;
}

function message(id, threadId, sequence, role, content, createdAt) {
  return {
    id,
    threadId,
    sequence,
    role,
    content,
    kind: "Message",
    toolFailed: false,
    refused: false,
    createdAt,
  };
}

function queued(threadId, id, content, position, createdAt) {
  return {
    id,
    threadId,
    content,
    request: { mentions: [], attachmentDocumentIds: [] },
    position,
    steer: false,
    version: 1,
    createdAt,
  };
}

/** The scenario's Desk state, built on first use so every scenario starts from the fixture. */
function desk(state) {
  if (state.deskCase) return state.deskCase;
  const anchor = state.anchor;
  const working = state.scenario.deskReply === "working";
  state.deskCase = {
    nextId: 1,
    subject: { type: "Shipment", id: SHIPMENT_ID },
    snooze: null,
    ticks: new Map(),
    waits: [
      {
        id: "awt_desk_pod",
        kind: "Reply",
        party: "Carrier",
        description: "Red River Freight's reply with the signed POD",
        nextStep: "Attach the POD and tick it off the checklist",
        threadId: CASE_THREAD_ID,
        status: "Waiting",
        expiresAt: anchor + 24 * 3600,
        createdAt: anchor - 3 * 60,
      },
      {
        id: "awt_desk_ratecon",
        kind: "Time",
        party: "Event",
        description:
          "Tomorrow at 9:00 AM, chase the rate confirmation again if it's still unsigned",
        nextStep: "Chase Red River Freight for the signed rate con",
        threadId: CASE_THREAD_ID,
        status: "Waiting",
        dueAt: tomorrowAtNine(anchor),
        expiresAt: tomorrowAtNine(anchor) + 3600,
        createdAt: anchor - 2 * 60,
      },
    ],
    messages: {
      [CASE_THREAD_ID]: [
        message("amsg_desk_c1", CASE_THREAD_ID, 1, "User", CASE_QUESTION, anchor - 6 * 60),
        message(
          "amsg_desk_c2",
          CASE_THREAD_ID,
          2,
          "Assistant",
          "Two things hold it up. **Red River Freight hasn't sent the signed POD**, and they haven't confirmed the rate con either. Everything else is in: it delivered yesterday, the BOL and lumper slip are attached, and the charges match the rate con to the cent.\n\nI've asked Red River for the POD and I'll pick this back up as soon as they reply. If the rate con is still unsigned by 9:00 tomorrow, I'll chase it again.",
          anchor - 3 * 60,
        ),
      ],
      [LIVE_THREAD_ID]: [
        message(
          "amsg_desk_l1",
          LIVE_THREAD_ID,
          1,
          "User",
          "How many loads delivered last week?",
          anchor - 9 * 60,
        ),
        message(
          "amsg_desk_l2",
          LIVE_THREAD_ID,
          2,
          "Assistant",
          "**42 loads** delivered between Monday and Sunday, for 11 customers. 33 of them are already invoiced. Acme Foods had the most, with 14.",
          anchor - 8 * 60,
        ),
      ],
    },
    turns: new Map(),
    working,
    queue: working
      ? [
          queued(
            LIVE_THREAD_ID,
            "aqm_desk_seed_1",
            "Then draft the POD requests for any that are missing one",
            1,
            anchor - 12,
          ),
          queued(
            LIVE_THREAD_ID,
            "aqm_desk_seed_2",
            "And put the totals in a table I can export",
            2,
            anchor - 6,
          ),
        ]
      : [],
    steers: working
      ? [{ id: "aqm_desk_steer", content: "Only the Acme Foods loads, please", mentions: [] }]
      : [],
    streams: new Set(),
  };
  return state.deskCase;
}

function nextId(d, prefix) {
  const id = `${prefix}desk_${d.nextId}`;
  d.nextId += 1;
  return id;
}

function record(d) {
  if (!d.subject) return null;
  return {
    type: "Shipment",
    id: d.subject.id,
    label: "S2610-1342",
    status: "Completed",
    closed: false,
    customerId: "cus_acme",
    carrierIds: ["car_redriver"],
  };
}

function openWaits(d) {
  return d.waits.filter((wait) => wait.status === "Waiting");
}

/** deskcase.Resolve: settled, then snoozed, then waiting, then working. */
function summary(state) {
  const d = desk(state);
  const waits = openWaits(d);
  const out = { state: "Working", openWaits: waits.length, record: record(d) };
  if (waits.length > 0) {
    out.waitingOn = waits.reduce((best, wait) =>
      PARTY_RANK[wait.party] < PARTY_RANK[best.party] ? wait : best,
    ).party;
    const due = waits.map((wait) => wait.dueAt).filter((at) => at != null);
    if (due.length > 0) out.nextWaitDue = Math.min(...due);
  }
  if (d.snooze && d.snooze.until > state.now()) {
    out.state = "Snoozed";
    out.snoozedUntil = d.snooze.until;
    out.snoozeAnchor = d.snooze.anchor;
  } else if (waits.length > 0) {
    out.state = "Waiting";
  }
  return out;
}

function checklist(state) {
  const d = desk(state);
  const at = (hoursAgo) => state.anchor - hoursAgo * 3600;
  const tick = d.ticks.get("custom:lumper");
  const lumper = { key: "custom:lumper", label: "Lumper receipt on file", manual: true };
  const items = [
    { key: "delivered", state: "Done", at: at(17) },
    { key: "pod", state: "Blocked", step: "request_pod", names: ["Red River Freight"] },
    { key: "paperwork", state: "Done", at: at(16) },
    { key: "rateConfirmation", state: "Done", at: at(15) },
    {
      key: "carrierRateConfirmed",
      state: "Blocked",
      step: "confirm_rate",
      names: ["Red River Freight"],
    },
    { key: "accessorials", state: "Done", at: at(14) },
    { key: "customerNotified", state: "Done", at: at(16), optional: true },
    { key: "billingHolds", state: "Done" },
    tick
      ? { ...lumper, state: "Done", at: tick.at, tickedBy: tick.by }
      : { ...lumper, state: "Pending" },
  ];
  return { kind: "ReadyToBill", ready: false, next: "request_pod", items };
}

function caseView(state) {
  return {
    summary: summary(state),
    abilities: {
      request_pod: { via: "Agent" },
      confirm_rate: { via: "Ask", agentId: "agdef_dispatch", agentName: "Dispatch" },
      mark_ready: { via: "Agent" },
    },
    checklist: checklist(state),
    parties: PARTIES,
  };
}

function thread(state, id, title, extra = {}) {
  const messages = desk(state).messages[id];
  const anchor = state.anchor;
  return {
    id,
    businessUnitId: "bu_mock",
    organizationId: "org_mock",
    userId: "usr_mock",
    agentDefinitionId: "agdef_billing",
    title,
    status: "Active",
    lastMessageAt: messages.at(-1).createdAt,
    lastReadAt: anchor,
    origin: "Desk",
    pinned: false,
    canContinue: true,
    pinnedFacts: ["Acme Foods needs a lumper receipt with every invoice"],
    version: 3,
    createdAt: anchor - 40 * 60,
    updatedAt: messages.at(-1).createdAt,
    ...extra,
  };
}

function caseThread(state) {
  const d = desk(state);
  return thread(state, CASE_THREAD_ID, CASE_QUESTION, {
    subjectType: d.subject?.type ?? "",
    subjectId: d.subject?.id ?? "",
    ...(d.snooze ? { snoozedUntil: d.snooze.until, snoozeAnchor: d.snooze.anchor } : {}),
    ...(d.subject ? { case: summary(state) } : {}),
  });
}

function liveThread(state) {
  return desk(state).working
    ? thread(state, LIVE_THREAD_ID, LIVE_QUESTION, { lastMessageAt: state.anchor - 40 })
    : thread(state, LIVE_THREAD_ID, "How many loads delivered last week?");
}

/** The conversations this file serves, for the rail's list and shelf counts. */
export function deskCaseThreads(state) {
  return [caseThread(state), liveThread(state)];
}

function binding(state) {
  const t = caseThread(state);
  return {
    threadId: t.id,
    subjectType: t.subjectType,
    subjectId: t.subjectId,
    ...(t.snoozedUntil ? { snoozedUntil: t.snoozedUntil, snoozeAnchor: t.snoozeAnchor } : {}),
    ...(t.case ? { case: t.case } : {}),
  };
}

/** The Billing agent's allowance this month and today, and the person's own. */
function budget(state) {
  const anchor = state.anchor;
  const monthStart = Math.floor(new Date(anchor * 1000).setUTCDate(1) / 1000 / 86400) * 86400;
  const nextMonth = new Date(monthStart * 1000);
  nextMonth.setUTCMonth(nextMonth.getUTCMonth() + 1);
  const tomorrow = Math.floor(anchor / 86400) * 86400 + 86400;
  return {
    agentName: "Billing",
    spentUsd: "18.40",
    limitUsd: "200.00",
    share: 0.092,
    near: false,
    monthStart,
    resetsAt: Math.floor(nextMonth.getTime() / 1000),
    runsToday: 14,
    dailyRunLimit: 200,
    budgetUsed: false,
    dailyUsed: false,
    dayResetsAt: tomorrow,
    disabledBy: "",
    disabledAt: 0,
    person: { used: 23, limit: 500, resetsAt: tomorrow },
  };
}

function page(results) {
  return { results, hasMore: false, total: results.length, limit: 50 };
}

// --- Turns ----------------------------------------------------------------

/** A reader that rejoins the reply under way is sent it from its start, steers included, and held open. */
function followLive(state) {
  const d = desk(state);
  const start = (callId, name, args) => ({
    event: "tool_started",
    data: { callId, name, arguments: args, effect: "Read" },
    delay: 500,
  });
  const finish = (callId, name, text) => ({
    event: "tool_finished",
    data: { callId, name, effect: "Read", summary: text },
    delay: 700,
  });
  return {
    __sse: [
      { event: "accepted", data: { content: LIVE_QUESTION }, delay: 200 },
      start("call_desk_1", "shipment_list", { status: "Completed", deliveredFrom: "last week" }),
      finish("call_desk_1", "shipment_list", "42 delivered loads"),
      start("call_desk_2", "get_billing_readiness", { shipments: 42 }),
      finish("call_desk_2", "get_billing_readiness", "9 not ready to bill"),
      ...d.steers.map((steer) => ({ event: "steered", data: steer, delay: 300 })),
      start("call_desk_3", "shipment_list", { customer: "Acme Foods", status: "Completed" }),
    ],
    __sseHold: (write) => {
      d.streams.add(write);
      return () => d.streams.delete(write);
    },
  };
}

/** A message sent while nothing is under way is answered at once, as a new turn. */
function startTurn(state, threadId, item) {
  const d = desk(state);
  const messages = d.messages[threadId];
  const at = state.now();
  const answer = FOLLOW_UPS[threadId];
  const user = message(nextId(d, "amsg_"), threadId, messages.length + 1, "User", item.content, at);
  const reply = message(
    nextId(d, "amsg_"),
    threadId,
    messages.length + 2,
    "Assistant",
    answer,
    at + 2,
  );
  messages.push(user);
  const turnId = nextId(d, "aturn_");
  d.turns.set(turnId, { threadId, user, reply });
  return { turnId, threadId, streamUrl: `/assistant/turns/${turnId}/stream/`, status: "Running" };
}

function followTurn(state, path) {
  const d = desk(state);
  const turn = d.turns.get(/\/turns\/([^/]+)\/stream/.exec(path)[1]);
  if (!turn) return notFound("Turn");
  const messages = d.messages[turn.threadId];
  if (!messages.includes(turn.reply)) messages.push(turn.reply);
  const answer = turn.reply.content;
  const words = answer.split(/(?<= )/);
  const frames = [{ event: "accepted", data: { content: turn.user.content }, delay: 300 }];
  for (let i = 0; i < words.length; i += 3) {
    frames.push({
      event: "delta",
      data: { text: words.slice(i, i + 3).join("") },
      delay: i === 0 ? 700 : 60,
    });
  }
  frames.push({ event: "message", data: { content: answer, model: "mock" }, delay: 40 });
  const t = turn.threadId === CASE_THREAD_ID ? caseThread(state) : liveThread(state);
  frames.push({
    event: "done",
    data: { thread: t, messages: [turn.user, turn.reply], reply: answer, refused: false },
    delay: 40,
  });
  return { __sse: frames };
}

function steer(d, item) {
  const event = { id: item.id, content: item.content, mentions: item.request.mentions };
  d.steers.push(event);
  for (const write of d.streams) write("steered", event);
}

// --- Case writes ----------------------------------------------------------

function checkSnoozeTime(until, now) {
  if (until < now + MIN_SNOOZE_SECONDS)
    return invalid("until", "INVALID", "Snooze until a time that has not come yet");
  if (until > now + MAX_SNOOZE_SECONDS)
    return invalid("until", "INVALID", "A case can be snoozed for at most 30 days");
  return null;
}

function bindCase(state, _path, body) {
  const d = desk(state);
  if (!["Shipment", "Invoice", "InvoiceDispute"].includes(body?.subjectType)) {
    return invalid("subjectType", "INVALID", "A case is about a shipment, an invoice or a dispute");
  }
  if (body.subjectType !== "Shipment" || body.subjectId !== SHIPMENT_ID) {
    return invalid("subjectId", "NOT_FOUND", "That record is gone");
  }
  if (d.subject?.id !== body.subjectId) d.snooze = null;
  d.subject = { type: body.subjectType, id: body.subjectId };
  return binding(state);
}

function unbindCase(state) {
  const d = desk(state);
  d.subject = null;
  d.snooze = null;
  return binding(state);
}

/** The shipment has delivered, so it has no stop ahead and no ETA: only a time or a fallback holds. */
function snoozeCase(state, _path, body) {
  const d = desk(state);
  if (!d.subject) return notFound("Case");
  const until = Number(body?.until ?? 0);
  switch (body?.anchor) {
    case "Time":
    case "ETA": {
      if (body.anchor === "ETA" && until <= 0) {
        return invalid(
          "anchor",
          "INVALID_OPERATION",
          "The shipment has no ETA yet; snooze to a time instead",
        );
      }
      const refusal = checkSnoozeTime(until, state.now());
      if (refusal) return refusal;
      d.snooze = { until, anchor: body.anchor };
      return binding(state);
    }
    case "Appointment":
      return invalid("anchor", "INVALID_OPERATION", "The shipment has no stop ahead to snooze to");
    default:
      return invalid("anchor", "INVALID", "Snooze until a time, the next appointment or the ETA");
  }
}

function wakeCase(state) {
  const d = desk(state);
  if (!d.subject) return notFound("Case");
  d.snooze = null;
  return binding(state);
}

function awaitReply(state, _path, body) {
  const d = desk(state);
  if (!d.subject) return notFound("Case");
  if (body?.party !== "Customer" && body?.party !== "Carrier") {
    return invalid("party", "INVALID", "Wait on the customer or a carrier");
  }
  const party = PARTIES.find((p) => p.kind === body.party && p.id === body.partyId);
  if (!party) return invalid("partyId", "NOT_FOUND", "That customer or carrier is gone");
  if (openWaits(d).length >= MAX_OPEN_WAITS) {
    return business(`A conversation can wait on up to ${MAX_OPEN_WAITS} things at once`);
  }
  const requested = Number(body.giveUpAfterHours ?? 0);
  const hours = requested <= 0 ? DEFAULT_REPLY_HOURS : Math.min(requested, MAX_REPLY_HOURS);
  const at = state.now();
  const wait = {
    id: nextId(d, "awt_"),
    kind: "Reply",
    party: party.kind,
    description: `A reply from ${party.name} about shipment S2610-1342`,
    nextStep: "Read what they said and pick the case back up from there.",
    threadId: CASE_THREAD_ID,
    status: "Waiting",
    expiresAt: at + hours * 3600,
    createdAt: at,
  };
  d.waits.push(wait);
  return { __status: 201, body: publicWait(wait) };
}

function tickItem(state, _path, body) {
  const d = desk(state);
  if (!d.subject) return notFound("Case");
  if (!MANUAL_STEPS.has(body?.itemKey)) {
    return invalid(
      "itemKey",
      "INVALID",
      "Only a step your organization added for a person to tick can be ticked",
    );
  }
  if (body.ticked) d.ticks.set(body.itemKey, { at: state.now(), by: "Maya Chen" });
  else d.ticks.delete(body.itemKey);
  return caseView(state);
}

// --- Waits ----------------------------------------------------------------

function publicWait({ party: _party, ...wait }) {
  return wait;
}

function cancelWait(state, path) {
  const d = desk(state);
  const id = /\/waits\/([^/]+)\/cancel/.exec(path)[1];
  const wait = d.waits.find((w) => w.id === id && w.threadId === threadOf(path));
  if (!wait) return notFound("Wait");
  if (wait.status === "Waiting") {
    wait.status = "Cancelled";
    wait.resolvedAt = state.now();
  }
  return publicWait(wait);
}

// --- Queue ----------------------------------------------------------------

function threadOf(path) {
  return /\/threads\/([^/]+)\//.exec(path)[1];
}

function itemOf(path) {
  return /\/queue\/([^/]+)\//.exec(path)[1];
}

function queueOf(d, threadId) {
  return d.queue
    .filter((item) => item.threadId === threadId)
    .sort((a, b) => a.position - b.position);
}

/** Characters as the server counts them: Unicode code points (Go's utf8.RuneCountInString). */
function runeCount(text) {
  let count = 0;
  for (const _ of text) count += 1;
  return count;
}

function contentRefusal(content) {
  if (typeof content !== "string" || content.trim() === "") {
    return invalid("content", "REQUIRED", "Message cannot be empty");
  }
  if (runeCount(content) > MAX_QUEUED_RUNES) {
    return invalid("content", "INVALID", `A message can be at most ${MAX_QUEUED_RUNES} characters`);
  }
  return null;
}

/** With a reply under way a message waits for it, or steers it; with none it is sent at once. */
function enqueue(state, path, body) {
  const d = desk(state);
  const threadId = threadOf(path);
  const refusal = contentRefusal(body?.content);
  if (refusal) return refusal;
  const items = queueOf(d, threadId);
  if (items.length >= MAX_QUEUED) {
    return business(
      `Up to ${MAX_QUEUED} messages can wait in a conversation. Send or remove one first.`,
    );
  }
  const item = {
    ...queued(
      threadId,
      nextId(d, "aqm_"),
      body.content.trim(),
      (items.at(-1)?.position ?? 0) + 1,
      state.now(),
    ),
    request: {
      mentions: body.mentions ?? [],
      attachmentDocumentIds: body.attachmentDocumentIds ?? [],
    },
  };
  if (threadId !== LIVE_THREAD_ID || !d.working) {
    return { __status: 202, body: { steering: false, turn: startTurn(state, threadId, item) } };
  }
  if (body.steer && item.request.attachmentDocumentIds.length === 0 && !body.directedAgentId) {
    item.steer = true;
    steer(d, item);
    return { __status: 202, body: { item, steering: true } };
  }
  d.queue.push(item);
  return { __status: 202, body: { item, steering: false } };
}

function editQueued(state, path, body) {
  const d = desk(state);
  const item = d.queue.find((q) => q.id === itemOf(path) && q.threadId === threadOf(path));
  if (!item) return notFound("Queued message");
  const refusal = contentRefusal(body?.content);
  if (refusal) return refusal;
  if (Number(body.version) !== item.version) {
    return problem(
      409,
      "resource-conflict",
      "Resource Conflict",
      "The message changed since you opened it. Reload and try again.",
    );
  }
  item.content = body.content.trim();
  item.version += 1;
  return item;
}

function removeQueued(state, path) {
  const d = desk(state);
  const index = d.queue.findIndex((q) => q.id === itemOf(path) && q.threadId === threadOf(path));
  if (index === -1) return notFound("Queued message");
  d.queue.splice(index, 1);
  return { __status: 204, body: undefined };
}

function reorderQueue(state, path, body) {
  const d = desk(state);
  const items = queueOf(d, threadOf(path));
  const ids = Array.isArray(body?.ids) ? body.ids : [];
  const same =
    ids.length === items.length &&
    new Set(ids).size === ids.length &&
    items.every((q) => ids.includes(q.id));
  if (!same) {
    return invalid(
      "ids",
      "INVALID",
      "The order must name every waiting message once. Reload the conversation and try again.",
    );
  }
  ids.forEach((id, index) => {
    d.queue.find((q) => q.id === id).position = index + 1;
  });
  return { items: queueOf(d, threadOf(path)) };
}

/** Sending a waiting message now hands it to the reply under way, unless it carries files. */
function sendQueued(state, path) {
  const d = desk(state);
  const index = d.queue.findIndex((q) => q.id === itemOf(path) && q.threadId === threadOf(path));
  if (index === -1) return notFound("Queued message");
  const item = d.queue[index];
  if (!d.working) {
    d.queue.splice(index, 1);
    return {
      __status: 202,
      body: { steering: false, turn: startTurn(state, item.threadId, item) },
    };
  }
  if (item.request.attachmentDocumentIds.length > 0) {
    return business(
      "A message with files is sent once the reply under way ends; it cannot steer it.",
    );
  }
  d.queue.splice(index, 1);
  item.steer = true;
  steer(d, item);
  return { __status: 202, body: { item, steering: true } };
}

// --- Case checklist settings ---------------------------------------------

function checklistTemplates(kind) {
  const bill = [
    ["delivered", "Required"],
    ["pod", "Required"],
    ["paperwork", "Required"],
    ["rateConfirmation", "Required"],
    ["carrierRateConfirmed", "Required"],
    ["accessorials", "Required"],
    ["customerNotified", "Optional"],
    ["billingHolds", "Required"],
  ].map(([key, mode]) => ({ key, mode }));
  bill.push({
    key: "custom:lumper",
    mode: "Required",
    custom: {
      label: "Lumper receipt on file",
      check: "Manual",
      stepLabel: "Request the lumper receipt",
      prompt: "Ask the carrier for the lumper receipt and attach it to the shipment.",
    },
  });
  const close = ["posted", "sent", "dispute", "paid"].map((key) => ({ key, mode: "Required" }));
  return {
    kind,
    organization: {
      id: "cct_desk_org",
      kind,
      items: kind === "ReadyToBill" ? bill : close,
      version: 2,
      updatedAt: 1791700000,
    },
    customers:
      kind === "ReadyToBill"
        ? [
            {
              id: "cct_desk_acme",
              kind,
              customerId: "cus_acme",
              customerName: "Acme Foods",
              items: bill,
              version: 1,
              updatedAt: 1791700000,
            },
          ]
        : [],
    locked: [
      "delivered",
      "pod",
      "paperwork",
      "rateConfirmation",
      "billingHolds",
      "posted",
      "dispute",
      "paid",
    ],
  };
}

const CASE = CASE_THREAD_ID;
const LIVE = LIVE_THREAD_ID;
const EITHER = `(?:${CASE}|${LIVE})`;
const route = (rest) => new RegExp(`^/api/v1/assistant/threads/${rest}/?$`);

export const DESK_CASE_ROUTES = [
  [
    "GET",
    /^\/api\/v1\/case-checklists\/?$/,
    (_s, _p, _b, query) => {
      const kind = query?.get("kind");
      return kind === "ReadyToBill" || kind === "ReadyToClose"
        ? checklistTemplates(kind)
        : invalid("kind", "INVALID", "Pick ReadyToBill or ReadyToClose");
    },
  ],
  ["GET", route(CASE), caseThread],
  ["GET", route(LIVE), liveThread],
  ["GET", route(`${EITHER}/messages`), (s, path) => page(desk(s).messages[threadOf(path)])],
  ["GET", route(`${CASE}/case`), (s) => (desk(s).subject ? caseView(s) : notFound("Case"))],
  ["PUT", route(`${CASE}/case`), bindCase],
  ["DELETE", route(`${CASE}/case`), unbindCase],
  ["POST", route(`${CASE}/case/snooze`), snoozeCase],
  ["POST", route(`${CASE}/case/wake`), wakeCase],
  ["POST", route(`${CASE}/case/await-reply`), awaitReply],
  ["POST", route(`${CASE}/case/ticks`), tickItem],
  [
    "GET",
    route(`${EITHER}/waits`),
    (s, path) => ({
      items: openWaits(desk(s))
        .filter((w) => w.threadId === threadOf(path))
        .map(publicWait),
    }),
  ],
  ["POST", route(`${EITHER}/waits/[^/]+/cancel`), cancelWait],
  ["GET", route(`${EITHER}/queue`), (s, path) => ({ items: queueOf(desk(s), threadOf(path)) })],
  ["POST", route(`${EITHER}/queue`), enqueue],
  ["PUT", route(`${EITHER}/queue/order`), reorderQueue],
  ["POST", route(`${EITHER}/queue/[^/]+/send`), sendQueued],
  ["PATCH", route(`${EITHER}/queue/[^/]+`), editQueued],
  ["DELETE", route(`${EITHER}/queue/[^/]+`), removeQueued],
  [
    "GET",
    route(`${EITHER}/artifacts`),
    () => ({ results: [], total: 0, nextCursor: "", counts: { all: 0, pinned: 0, families: {} } }),
  ],
  ["GET", route(`${EITHER}/schedules`), () => ({ items: [], total: 0 })],
  ["GET", route(`${EITHER}/proposals`), () => ({ results: [] })],
  ["GET", route(`${EITHER}/plans`), () => ({ results: [] })],
  ["GET", route(`${EITHER}/budget`), budget],
  ["GET", route(`${CASE}/turns/active`), () => ({ turn: null })],
  [
    "GET",
    route(`${LIVE}/turns/active`),
    (s) => ({
      turn: desk(s).working
        ? {
            id: LIVE_TURN_ID,
            threadId: LIVE,
            status: "Running",
            origin: "Person",
            input: LIVE_QUESTION,
          }
        : null,
    }),
  ],
  [
    "GET",
    new RegExp(`^/api/v1/assistant/turns/${LIVE_TURN_ID}/stream/?$`),
    (s) => (desk(s).working ? followLive(s) : notFound("Turn")),
  ],
  ["GET", /^\/api\/v1\/assistant\/turns\/aturn_desk_\d+\/stream\/?$/, followTurn],
  [
    "GET",
    /^\/api\/v1\/assistant\/turns\/active\/?$/,
    (s) => ({
      items: desk(s).working
        ? [
            {
              turnId: LIVE_TURN_ID,
              threadId: LIVE,
              threadTitle: LIVE_QUESTION,
              origin: "Person",
              startedAt: s.anchor - 40,
            },
          ]
        : [],
    }),
  ],
];
