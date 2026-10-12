// Sends every operation in client/packages/graphql/src/operations/inbox/{inbox,triage}.graphql to
// the mock API and holds each response to its operation's selection set: every selected field
// present, nothing unselected (other than __typename), no null on a non-null type, enums and
// scalars of the right kind. Also walks the scenarios and the realtime stream.
//
//   node e2e/mock-api/check-triage.mjs            (mock on :8080, or MOCK_API_URL)
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { buildSchema, getNamedType, isEnumType, isListType, isNonNullType, isScalarType, Kind, parse } from "graphql";

const MOCK = process.env.MOCK_API_URL ?? "http://localhost:8080";
const GRAPHQL_SRC = fileURLToPath(new URL("../../../../packages/graphql/src/", import.meta.url));
const schema = buildSchema(readFileSync(`${GRAPHQL_SRC}schema.graphql`, "utf8"));

const CONTRACT = ["operations/inbox/inbox.graphql", "operations/inbox/triage.graphql"];
const SUPPORT = ["operations/agent/access.graphql", "operations/equipment/table.graphql", "operations/select-options/options.graphql"];
const documents = [...CONTRACT, ...SUPPORT].map((file) => ({ file, doc: parse(readFileSync(`${GRAPHQL_SRC}${file}`, "utf8")) }));

const fragments = new Map();
const operations = new Map();
for (const { file, doc } of documents) {
  for (const def of doc.definitions) {
    if (def.kind === Kind.FRAGMENT_DEFINITION) fragments.set(def.name.value, def);
    if (def.kind === Kind.OPERATION_DEFINITION) operations.set(def.name.value, { def, contract: CONTRACT.includes(file) });
  }
}
const contractOperations = [...operations.entries()].filter(([, op]) => op.contract).map(([name]) => name);

const failures = [];
const exercised = new Set();
const fail = (message) => {
  failures.push(message);
  console.log(`  ✗ ${message}`);
};

function directiveAllows(node, variables, defaults) {
  for (const directive of node.directives ?? []) {
    const arg = directive.arguments.find((a) => a.name.value === "if");
    if (!arg) continue;
    const value = arg.value.kind === Kind.VARIABLE ? variables[arg.value.name.value] ?? defaults[arg.value.name.value] : arg.value.value;
    if (directive.name.value === "include" && !value) return false;
    if (directive.name.value === "skip" && value) return false;
  }
  return true;
}

function collect(selectionSet, typeName, variables, defaults, into = new Map()) {
  for (const selection of selectionSet.selections) {
    if (!directiveAllows(selection, variables, defaults)) continue;
    if (selection.kind === Kind.FIELD) {
      const key = selection.alias?.value ?? selection.name.value;
      if (!into.has(key)) into.set(key, []);
      into.get(key).push(selection);
    } else if (selection.kind === Kind.FRAGMENT_SPREAD) {
      const fragment = fragments.get(selection.name.value);
      if (!fragment) throw new Error(`unknown fragment ${selection.name.value}`);
      if (fragment.typeCondition.name.value === typeName) collect(fragment.selectionSet, typeName, variables, defaults, into);
    } else if (selection.kind === Kind.INLINE_FRAGMENT) {
      if (!selection.typeCondition || selection.typeCondition.name.value === typeName) collect(selection.selectionSet, typeName, variables, defaults, into);
    }
  }
  return into;
}

function checkScalar(value, type, path) {
  const name = type.name;
  const ok = {
    Int: Number.isInteger(value),
    Timestamp: Number.isInteger(value),
    Float: typeof value === "number" && Number.isFinite(value),
    String: typeof value === "string",
    ID: typeof value === "string" && value.length > 0,
    Decimal: typeof value === "string" && /^-?\d+(\.\d+)?$/.test(value),
    Boolean: typeof value === "boolean",
  }[name];
  if (ok === false) fail(`${path}: ${JSON.stringify(value)} is not a ${name}`);
}

function check(value, type, nodes, path, ctx) {
  if (isNonNullType(type)) {
    if (value === null || value === undefined) {
      fail(`${path}: null on non-null ${type}`);
      return;
    }
    check(value, type.ofType, nodes, path, ctx);
    return;
  }
  if (value === null) return;
  if (value === undefined) {
    fail(`${path}: undefined (a nullable field must still be present, as null)`);
    return;
  }
  if (isListType(type)) {
    if (!Array.isArray(value)) {
      fail(`${path}: expected a list`);
      return;
    }
    value.forEach((item, i) => check(item, type.ofType, nodes, `${path}[${i}]`, ctx));
    return;
  }
  if (isEnumType(type)) {
    if (!type.getValue(value)) fail(`${path}: ${JSON.stringify(value)} is not a ${type.name}`);
    return;
  }
  if (isScalarType(type)) {
    checkScalar(value, type, path);
    return;
  }
  if (typeof value !== "object" || Array.isArray(value)) {
    fail(`${path}: expected an object of ${type.name}`);
    return;
  }
  const selected = new Map();
  for (const node of nodes) if (node.selectionSet) collect(node.selectionSet, type.name, ctx.variables, ctx.defaults, selected);
  const fields = type.getFields();
  for (const [key, fieldNodes] of selected) {
    const fieldName = fieldNodes[0].name.value;
    if (fieldName === "__typename") continue;
    if (!(key in value)) {
      fail(`${path}.${key}: missing`);
      continue;
    }
    check(value[key], fields[fieldName].type, fieldNodes, `${path}.${key}`, ctx);
  }
  for (const key of Object.keys(value)) {
    if (key !== "__typename" && !selected.has(key)) fail(`${path}.${key}: not selected by the operation`);
  }
}

async function post(path, body, method = "POST") {
  const response = await fetch(`${MOCK}${path}`, { method, headers: { "content-type": "application/json" }, body: body ? JSON.stringify(body) : undefined });
  return response.json();
}

async function scenario(overrides) {
  return post("/__mock/scenario", { triage: "live", triageTime: "afternoon", ...overrides });
}

/** Sends one operation and checks its response; `expectError` checks the presented domain error instead. */
async function op(name, variables = {}, { expectError = null } = {}) {
  const entry = operations.get(name);
  if (!entry) throw new Error(`no operation ${name}`);
  exercised.add(name);
  const result = await post("/graphql", { operationName: name, variables });
  if (expectError) {
    const error = result.errors?.[0];
    if (!error) fail(`${name}: expected a ${expectError} error, got data`);
    else if (error.extensions?.code !== expectError) fail(`${name}: expected ${expectError}, got ${error.extensions?.code} (${error.message})`);
    else console.log(`  ✓ ${name} refused: ${error.message}`);
    return null;
  }
  if (result.errors?.length) {
    fail(`${name}: ${result.errors[0].message.split("\n")[0]}`);
    return null;
  }
  const defaults = Object.fromEntries((entry.def.variableDefinitions ?? []).filter((v) => v.defaultValue).map((v) => [v.variable.name.value, v.defaultValue.value === "true" ? true : v.defaultValue.value === "false" ? false : v.defaultValue.value]));
  const before = failures.length;
  const rootType = entry.def.operation === "mutation" ? schema.getMutationType() : schema.getQueryType();
  check(result.data, rootType, [entry.def], name, { variables, defaults });
  if (failures.length === before) console.log(`  ✓ ${name}`);
  return result.data;
}

function expect(condition, message) {
  if (condition) console.log(`    · ${message}`);
  else fail(message);
}

const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms));

// --- the live scenario: every query, then every write and what it changes

console.log("live, afternoon");
const { triageAnchor } = await scenario({});
const now = () => triageAnchor + 30;

const counts0 = (await op("InboundMessageCounts")).inboundMessageCounts;
expect(counts0.waiting === 8 && counts0.overdue === 3, `8 waiting, 3 over target (got ${counts0.waiting}, ${counts0.overdue})`);
await op("InboundMailboxes");
const waiting = (await op("InboundMessages", { input: { lane: "Waiting", sort: "Due", collapseThreads: true, first: 25 } })).inboundMessages;
expect(waiting.edges[0]?.node.id === "imsg_m3", `Due puts the most overdue first (got ${waiting.edges[0]?.node.id})`);
const page1 = (await op("InboundMessages", { input: { lane: "Everything", first: 10 } })).inboundMessages;
const page2 = (await op("InboundMessages", { input: { lane: "Everything", first: 10, after: page1.pageInfo.endCursor } })).inboundMessages;
expect(page1.pageInfo.hasNextPage && page2.edges[0]?.node.id !== page1.edges[0]?.node.id, "cursor paging moves to the next page");
const acme = (await op("InboundMessages", { input: { lane: "Everything", query: "acme" } })).inboundMessages;
expect(acme.edges.length > 0 && acme.edges.every((e) => /acme/i.test(e.node.fromAddress + e.node.subject)), "search narrows to Acme");
const collapsed = (await op("InboundMessages", { input: { lane: "Everything", collapseThreads: true, first: 200 } })).inboundMessages;
expect(!collapsed.edges.some((e) => e.node.id === "imsg_m10"), "collapsing threads keeps the newest of t-harb only");
const harb = (await op("InboundMessages", { input: { threadKey: "t-harb" } })).inboundMessages;
expect(harb.edges.length === 2, "a thread key lists its thread");
const detail = (await op("InboundMessage", { id: "imsg_m2" })).inboundMessage;
expect(detail.actions.map((a) => a.kind).join(",") === "CheckStops,Reply", `m2 asks to check the stops, then reply (got ${detail.actions.map((a) => a.kind)})`);
await op("InboundMessage", { id: "imsg_nope" }, { expectError: "NOT_FOUND" });
await op("InboundThread", { threadKey: "t-crest" });
const briefing = (await op("TriageBriefing")).triageBriefing;
expect(briefing.deck.map((c) => `${c.group}${c.count}`).join(" ") === "File2 Draft2 Decide2 Link1 Held1", `deck File2 Draft2 Decide2 Link1 Held1 (got ${briefing.deck.map((c) => c.group + c.count).join(" ")})`);
expect(briefing.handledByNovaToday === 29 && briefing.estimateMinutes === 17, `Nova handled 29, about 17 minutes left (got ${briefing.handledByNovaToday}, ${briefing.estimateMinutes})`);
expect(briefing.suggestion?.sender === "blueridgebrokerage.example" && briefing.suggestion.handledByHand === 11 && briefing.suggestion.total === 11, "suggests a rule for 11 of 11 Blue Ridge rate cons");
expect(briefing.unhealthySources.map((mb) => mb.name).join() === "Claims", "Claims is quiet");
expect(briefing.onCall?.name === "Marcus Lee" && briefing.carryOver.length === 0 && briefing.phase === "Day", "Marcus is on call; nothing carries over in the afternoon");
await op("MyTriageViews");
await op("TriageResponseTargets");
const settings = (await op("TriageSettings")).triageSettings;
await op("TriageAssignees");
await op("TriageRules");
const preview = (await op("TriageRulePreview", { input: { classification: "RateConfirmation", senderScope: "Domain", sender: "blueridgebrokerage.example", mailboxId: "imb_intake", action: "FileOnShipment" } })).triageRulePreview;
expect(preview.matched >= 12 && preview.matches.length <= 12, `the preview counts the 30 days and lists the newest (matched ${preview.matched})`);
await op("TriageRuleMatches", { id: "trule_r1" });
await op("TriageSenderDomains", { query: "example" });
const profile = (await op("InboundSenderProfile", { messageId: "imsg_m5" })).inboundSenderProfile;
expect(profile.role === "Broker", "Blue Ridge reads as a broker");
await op("TriageReplyTemplates");
const rendered = (await op("RenderTriageReplyTemplate", { templateId: "ttpl_t1", messageId: "imsg_m7" })).renderTriageReplyTemplate;
expect(rendered.startsWith("Hi Crestline — S-104198 is running about two hours behind. Current ETA is 16:10 · 2h late."), `ETA template fills in (got ${rendered.split("\n")[0]})`);
const stops = (await op("TriageStopTimes", { messageId: "imsg_m2" })).triageStopTimes;
expect(stops?.lateMinutes === 40, "the dispute's stop times say 40 minutes late");
const noStops = (await op("TriageStopTimes", { messageId: "imsg_m7" })).triageStopTimes;
expect(noStops === null, "a status request has no stop times");
await op("TriageShipmentPeek", { shipmentId: "shp_104198" });
await op("TriageCustomerPeek", { customerId: "cus_crestline" });

await op("AssignInboundMessages", { selection: { ids: ["imsg_m1"] }, ownerId: "usr_priya" });
await op("AssignInboundMessages", { selection: { ids: ["imsg_m1"] }, ownerId: null });
await op("AssignInboundMessages", { selection: { ids: ["imsg_m1"] }, ownerId: "usr_nobody" }, { expectError: "INVALID" });
await op("SnoozeInboundMessages", { selection: { ids: ["imsg_m8"] }, input: { until: now() + 3600 } });
expect((await op("InboundMessageCounts")).inboundMessageCounts.snoozed === 1, "snoozing moves m8 to Snoozed");
await op("WakeInboundMessage", { id: "imsg_m8" });
await op("SnoozeInboundMessages", { selection: { ids: ["imsg_m1"] }, input: { event: "ShipmentDelivered" } }, { expectError: "INVALID" });
const evented = (await op("SnoozeInboundMessages", { selection: { ids: ["imsg_m7"] }, input: { event: "DocumentAttached", documentTypeId: "dt_proof_of_delivery" } })).snoozeInboundMessages;
expect(evented.messages[0].snooze?.watchShipment?.proNumber === "S-104198", "an event snooze watches the shipment");
await op("WakeInboundMessage", { id: "imsg_m7" });

await op("FlagInboundFact", { id: "imsg_m6", key: "exception" });
await op("ResolveInboundConflict", { id: "imsg_m5", conflictId: "dup-ratecon", resolution: "SkipDuplicate" }, { expectError: "INVALID" });
const resolved = (await op("ResolveInboundConflict", { id: "imsg_m5", conflictId: "dup-ratecon", resolution: "KeepBoth" })).resolveInboundConflict;
expect(resolved.conflicts[0].resolution === "KeepBoth", "the conflict is settled");
await op("ReopenInboundConflict", { id: "imsg_m5", conflictId: "dup-ratecon" });

const drafted = (await op("DraftInboundReply", { messageId: "imsg_m7" })).draftInboundReply;
expect(drafted.draftedByNova && drafted.body.includes("16:10"), "Nova drafts Crestline's ETA");
await op("SaveInboundReplyDraft", { messageId: "imsg_m7", subject: "Re: Where is PO 88213?", body: `${drafted.body}\n` });
const sent = (await op("SendInboundReply", { input: { messageId: "imsg_m7", subject: "Re: Where is PO 88213?", body: drafted.body, draftedByNova: true } })).sendInboundReply;
expect(sent.status === "Sent", "the reply is sent");
await op("WaitForInboundReply", { id: "imsg_m7", until: now() + 86400 });
expect((await op("InboundMessageCounts")).inboundMessageCounts.waitingOnThem === 1, "m7 is waiting on them");
await op("StopWaitingForInboundReply", { id: "imsg_m7" });

const refused = (await op("SendInboundReply", { input: { messageId: "imsg_m3", subject: "Re: status file", body: "Which load is this?" } })).sendInboundReply;
expect(refused.status === "Failed" && refused.suggestedAddress === "ops@partnercarrier.example", "the EDI sender refuses with a 550 and an alternate");
const resent = (await op("SendInboundReply", { input: { messageId: "imsg_m3", subject: "Re: status file", body: "Which load is this?", toAddress: refused.suggestedAddress } })).sendInboundReply;
expect(resent.status === "Sent" && resent.toAddress === "ops@partnercarrier.example", "a resend to the alternate goes");
await op("SendInboundReply", { input: { messageId: "imsg_m3", subject: "Re", body: "  " } }, { expectError: "REQUIRED" });

const decided = (await op("DecideInboundStops", { messageId: "imsg_m2", decision: "Waive" })).decideInboundStops;
expect(decided.replyDraft?.body.includes("$185.00") && decided.actions[0].done, "waiving records the decision and drafts the reply");
await op("DiscardInboundReplyDraft", { messageId: "imsg_m2" });
expect((await op("DraftInboundMany", { messageIds: ["imsg_m1", "imsg_m6"] })).draftInboundMany === 1, "only m1's next step is Nova's to draft");

await op("DraftInboundLoad", { messageId: "imsg_m8" });
await op("DiscardInboundLoad", { messageId: "imsg_m8" });
await sleep(1400);
const m1 = (await op("InboundMessage", { id: "imsg_m1" })).inboundMessage;
expect(m1.loadDraft?.draftedByNova, "Draft all drafted m1's load");
const created = (await op("CreateInboundLoad", { messageId: "imsg_m1", fields: m1.loadDraft.fields })).createInboundLoad;
expect(created.actions.find((a) => a.kind === "BuildLoad")?.createdShipment?.proNumber === "S-104260", "the load becomes S-104260");

const m5 = (await op("InboundMessage", { id: "imsg_m5" })).inboundMessage;
const split = (await op("FileInboundAttachment", { input: { attachmentId: m5.attachments[0].id, shipmentId: "shp_104251", pages: [{ fromPage: 1, toPage: 1, documentTypeId: "dt_rate_confirmation" }, { fromPage: 2, toPage: 2, documentTypeId: "dt_bill_of_lading" }] } })).fileInboundAttachment;
expect(split.attachments[0].splitDocuments.length === 2 && split.actions[0].done, "the rate con splits into two documents");
await op("FileInboundAttachment", { input: { attachmentId: m5.attachments[0].id, shipmentId: "shp_104251", pages: [{ fromPage: 1, toPage: 3, documentTypeId: "dt_other" }] } }, { expectError: "INVALID" });
await op("UnfileInboundAttachment", { attachmentId: m5.attachments[0].id });
const filedMany = (await op("FileInboundMany", { messageIds: ["imsg_m5", "imsg_m6"] })).fileInboundMany;
expect(filedMany.affected === 2, "File all files both");
await op("LogInboundException", { messageId: "imsg_m6", input: { kind: "Shortage", quantity: 2, unit: "pallets" } });

await op("LinkInboundMessage", { id: "imsg_m3", input: { shipmentId: "shp_104251", reason: "The 214 names Columbus" } });
await op("LinkInboundMessage", { id: "imsg_m3", input: { reason: "" } }, { expectError: "REQUIRED" });
await op("UnlinkInboundMessage", { id: "imsg_m3", record: "Shipment" });
await op("CorrectInboundClassification", { id: "imsg_m3", classification: "Other" });
const corrections = (await op("TriageBriefing")).triageBriefing.corrections;
expect(corrections.thisWeek === 5, `a correction and a flag count (got ${corrections.thisWeek})`);

await op("BlockInboundSender", { messageId: "imsg_m4" });
await op("UnblockInboundSender", { mailboxId: "imb_intake", sender: "unknownsender.example" });
await op("ReopenInboundMessages", { selection: { ids: ["imsg_m4"] } });
await op("ReviewInboundMessage", { id: "imsg_m5", input: { status: "Actioned", note: "Signed and returned." } });
await op("ReviewInboundMessage", { id: "imsg_m6", input: { status: "InReview" } }, { expectError: "INVALID" });
const taught = (await op("ReviewInboundMessages", { selection: { ids: ["imsg_m6"] }, input: { status: "Actioned", teach: "FileOnShipment" } })).reviewInboundMessages;
expect(!!taught.ruleId && taught.affected === 1, "teaching on settle turns on a rule");
await op("ReviewInboundMessages", { selection: { folder: { lane: "Waiting", classification: "Tender" } }, input: { status: "Ignored" } });

const viewCreated = (await op("CreateMyTriageView", { name: "Crestline", query: "crestline" })).createMyTriageView;
await op("RenameMyTriageView", { id: viewCreated.id, name: "Crestline Retail" });
await op("DeleteMyTriageView", { id: viewCreated.id });
await op("UpdateTriageResponseTargets", { input: [{ classification: "StatusRequest", minutes: 30 }, { classification: "Tender", minutes: null }] });
await op("UpdateTriageResponseTargets", { input: [{ classification: "StatusRequest", minutes: 1 }] }, { expectError: "INVALID" });
await op("UpdateTriageSettings", { input: { onCallUserId: "usr_dana", endOfDayMinute: 1050, version: settings.version } });
await op("UpdateTriageSettings", { input: { onCallUserId: "usr_dana", endOfDayMinute: 1050, version: settings.version } }, { expectError: "VERSION_MISMATCH" });

const ruleCreated = (await op("CreateTriageRule", { input: { classification: "RateConfirmation", senderScope: "Domain", sender: "blueridgebrokerage.example", mailboxId: "imb_intake", action: "FileOnShipment" } })).createTriageRule;
expect((await op("TriageBriefing")).triageBriefing.suggestion === null, "turning the rule on retires the suggestion");
await op("CreateTriageRule", { input: { classification: "Invoice", senderScope: "Domain", sender: "not a domain", action: "Ignore" } }, { expectError: "INVALID" });
const ruleUpdated = (await op("UpdateTriageRule", { id: ruleCreated.id, version: ruleCreated.version, input: { classification: "RateConfirmation", senderScope: "Domain", sender: "blueridgebrokerage.example", mailboxId: null, action: "FileOnShipment" } })).updateTriageRule;
await op("SetTriageRuleEnabled", { id: ruleUpdated.id, enabled: false });
await op("DeleteTriageRule", { id: ruleUpdated.id });

const tpl = (await op("CreateTriageReplyTemplate", { input: { name: "Lane check", body: "Hi {first} — confirming {lane} for {customer}, PO {po}." } })).createTriageReplyTemplate;
await op("UpdateTriageReplyTemplate", { id: tpl.id, version: tpl.version, input: { name: "Lane check", body: "Hi {first} — confirming {lane}." } });
await op("DeleteTriageReplyTemplate", { id: tpl.id });

const mailboxCreated = (await op("CreateInboundMailbox", { input: { name: "Dispatch", address: "dispatch@dev.trenova.app", provider: "Resend", purpose: "Dispatch", reviewPolicy: "ReviewBelowConfidence", minConfidence: 0.8, status: "Active" }, signingSecret: "whsec_x", apiKey: null })).createInboundMailbox;
expect(mailboxCreated.webhookPath.startsWith("/api/v1/webhooks/inbound-mail/"), "a new source gets its webhook path once");
await op("CreateInboundMailbox", { input: { name: "Again", address: "dispatch@dev.trenova.app", provider: "Resend", reviewPolicy: "AlwaysReview", minConfidence: 0.8, status: "Active" } }, { expectError: "DUPLICATE" });
const updatedMailbox = (await op("UpdateInboundMailbox", { id: mailboxCreated.mailbox.id, version: mailboxCreated.mailbox.version, input: { name: "Dispatch", address: "dispatch@dev.trenova.app", provider: "Postmark", purpose: "Dispatch", reviewPolicy: "AlwaysReview", minConfidence: 0.8, status: "Active" } })).updateInboundMailbox;
expect(!updatedMailbox.hasSigningSecret, "changing the provider clears the secret");
await op("RotateInboundMailboxToken", { id: mailboxCreated.mailbox.id });
await op("SetInboundMailboxSigningSecret", { id: mailboxCreated.mailbox.id, secret: "user:password" });
await op("SetInboundMailboxApiKey", { id: mailboxCreated.mailbox.id, apiKey: "re_mock" });
const testing = (await op("SendInboundMailboxTest", { id: "imb_claims" })).sendInboundMailboxTest;
expect(testing.testSentAt && testing.health === "Quiet", "the test is on its way; Claims is still quiet");
await sleep(1800);
const claims = (await op("InboundMailboxes")).inboundMailboxes.find((mb) => mb.id === "imb_claims");
expect(claims.health === "Healthy" && claims.testReceivedAt >= claims.testSentAt, "the test arrived and Claims is healthy");

const handed = (await op("HandOffTriage", { toUserId: "usr_marcus" })).handOffTriage;
expect(handed.to.name === "Marcus Lee" && handed.affected > 0, `handed ${handed.affected} to Marcus`);
await op("MyAgents", { input: { first: 20 } });
await op("SelectOptions", { input: { resource: "DOCUMENT_TYPE" } });

// --- realtime: the stream says it is ready, carries invalidations, and presence answers on join

console.log("realtime");
{
  const controller = new AbortController();
  const response = await fetch(`${MOCK}/api/v1/realtime/stream/`, { signal: controller.signal });
  const reader = response.body.getReader();
  let text = "";
  const read = (async () => {
    try {
      for (;;) {
        const { value, done } = await reader.read();
        if (done) break;
        text += new TextDecoder().decode(value);
      }
    } catch {
      // aborted
    }
  })();
  await sleep(200);
  const connectionId = /"connectionId":"([^"]+)"/.exec(text)?.[1];
  expect(text.startsWith("event: ready") && connectionId, "the stream opens with ready");
  const snapshot = await post("/api/v1/inbox/presence/", { connectionId, messageId: "imsg_m1", activity: "viewing" });
  expect(snapshot.scope === "inbox:messages" && snapshot.members.some((m) => m.name === "Dana Whitfield" && m.messageId === "imsg_m2") && snapshot.members.some((m) => m.name === "Priya Shah" && m.activity === "writing"), "joining returns Dana viewing m2 and Priya writing on m7");
  await op("AssignInboundMessages", { selection: { ids: ["imsg_m2"] }, ownerId: "usr_mock" });
  await sleep(200);
  expect(/event: resource\.invalidation\ndata: \{[^\n]*"resource":"inbound_message"[^\n]*"recordId":"imsg_m2"/.test(text), "a write reaches the stream as an inbound_message invalidation");
  expect(/event: presence\ndata: \{"scope":"inbox:messages","action":"enter"/.test(text), "joining is announced as a presence event");
  await post(`/api/v1/inbox/presence/?connectionId=${connectionId}`, undefined, "DELETE");
  controller.abort();
  await read;
}

// --- the other scenarios

console.log("desk off");
await scenario({ triage: "deskOff" });
const off = (await op("TriageBriefing")).triageBriefing;
expect(!off.novaEnabled, "Nova is off");
const unread = (await op("InboundMessage", { id: "imsg_m7" })).inboundMessage;
expect(!unread.novaRead && unread.classification === null && unread.actions.length === 0 && unread.facts.length === 0, "a waiting message is unread: no kind, facts or actions");
expect((await op("ReadUnreadInboundMessages")).readUnreadInboundMessages === 8, "Nova reads the 8 it never read");
await sleep(1800);
const read = (await op("InboundMessage", { id: "imsg_m7" })).inboundMessage;
expect(read.novaRead && read.classification === "StatusRequest" && read.actions.length === 1, "after reading, m7 is a status request again");

console.log("first day");
await scenario({ triage: "firstDay" });
const first = (await op("TriageBriefing")).triageBriefing;
expect(first.firstDay && first.waiting === 0, "the first briefing is the first-day one");
await op("CreateInboundMailbox", { input: { name: "Dispatch", address: "dispatch@trenovalogistics.com", provider: "Postmark", purpose: "", reviewPolicy: "ReviewBelowConfidence", minConfidence: 0.75, status: "Active" } });
await sleep(1500);
const reading = (await op("InboundMessageCounts")).inboundMessageCounts;
expect(reading.total > 0 && reading.total < 57, `the backlog arrives over time (${reading.total} so far, ${reading.processing} being read)`);
await sleep(2600);
const readAll = (await op("InboundMessageCounts")).inboundMessageCounts;
expect(readAll.total === 57 && readAll.processing === 0 && readAll.waiting === 8, `all 57 read, 8 waiting (got ${readAll.total}, ${readAll.processing}, ${readAll.waiting})`);
expect(!(await op("TriageBriefing")).triageBriefing.firstDay, "then the briefing is the normal one");

console.log("no mailbox");
await scenario({ triage: "noMailbox" });
expect((await op("InboundMailboxes")).inboundMailboxes.length === 0, "no sources");
expect((await op("TriageBriefing")).triageBriefing.firstDay, "no source is a first day");

console.log("end of day");
await scenario({ triage: "live", triageTime: "endOfDay" });
const eod = (await op("TriageBriefing")).triageBriefing;
expect(eod.phase === "EndOfDay" && eod.carryOver.length === 8, `the end of the day carries 8 over (got ${eod.phase}, ${eod.carryOver.length})`);
await op("HandOffTriage", { toUserId: "usr_marcus" });
expect((await op("TriageBriefing")).triageBriefing.carryOver.every((item) => item.owner?.name === "Marcus Lee"), "after the hand-off, Marcus owns what carries over");

await scenario({});

const missing = contractOperations.filter((name) => !exercised.has(name));
if (missing.length) fail(`operations never sent: ${missing.join(", ")}`);
console.log(`\n${contractOperations.length} contract operations, ${exercised.size} exercised, ${failures.length} failure(s)`);
if (failures.length) process.exitCode = 1;
