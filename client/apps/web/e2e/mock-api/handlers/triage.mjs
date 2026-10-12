// Triage's GraphQL operations (client/packages/graphql/src/operations/inbox/{inbox,triage}.graphql)
// and REST routes. Every response is shaped to exactly its operation's selection; check-triage.mjs
// holds the mock to that.
import { DOCUMENT_TYPES } from "../fixtures/triage.mjs";
import { ORG, USER } from "../fixtures/session.mjs";
import { publish } from "../realtime.mjs";
import * as model from "./triage-model.mjs";

const MOCK_ORIGIN = process.env.MOCK_API_PUBLIC_URL ?? `http://localhost:${process.env.MOCK_API_PORT ?? 8080}`;

const tri = (state) => {
  model.wakeDue(state.triage);
  return state.triage;
};

// --- shared pieces

function user(t, id) {
  if (!id) return null;
  const u = t.users.get(id);
  if (!u) return null;
  return { id: u.id, name: u.name, emailAddress: u.emailAddress, thumbnailUrl: `${MOCK_ORIGIN}/__mock/avatar/${u.id}.svg` };
}

function shipmentRef(t, id) {
  if (!id) return null;
  const s = t.shipments.get(id);
  if (!s) return null;
  return {
    id: s.id,
    proNumber: s.proNumber,
    bol: s.bol,
    poNumber: s.poNumber,
    originCity: s.originCity,
    originState: s.originState,
    destinationCity: s.destinationCity,
    destinationState: s.destinationState,
  };
}

function mailbox(t, mb) {
  if (!mb) return null;
  return {
    __typename: "InboundMailbox",
    id: mb.id,
    name: mb.name,
    address: mb.address,
    provider: mb.provider,
    purpose: mb.purpose,
    reviewPolicy: mb.reviewPolicy,
    minConfidence: mb.minConfidence,
    status: mb.status,
    hasSigningSecret: mb.hasSigningSecret,
    hasApiKey: mb.hasApiKey,
    lastReceivedAt: mb.lastReceivedAt,
    health: model.healthOf(t, mb),
    testSentAt: mb.testSentAt,
    testReceivedAt: mb.testReceivedAt,
    blockedSenders: [...mb.blockedSenders],
    version: mb.version,
    createdAt: mb.createdAt,
    updatedAt: mb.updatedAt,
  };
}

function credentials(t, { mailbox: mb, token }) {
  return { token, webhookPath: `/api/v1/webhooks/inbound-mail/${encodeURIComponent(token)}/`, mailbox: mailbox(t, mb) };
}

function attachmentBase(a) {
  return {
    id: a.id,
    fileName: a.fileName,
    contentType: a.contentType,
    byteSize: a.byteSize,
    kind: a.kind,
    documentId: a.documentId,
    failureText: a.failureText,
    filedShipmentId: a.filedShipmentId,
    filedAs: a.filedAs,
    filedAt: a.filedAt,
    pageCount: a.pageCount,
    createdAt: a.createdAt,
  };
}

function attachmentDetail(t, a) {
  return {
    ...attachmentBase(a),
    filedShipment: shipmentRef(t, a.filedShipmentId),
    pages: a.pages.map((p) => ({ number: p.number, label: p.label, suggestedType: p.suggestedType })),
    splitDocuments: a.splitDocuments.map((d) => ({ fromPage: d.fromPage, toPage: d.toPage, documentId: d.documentId, documentType: d.documentType })),
    reading: a.reading
      ? {
          documentType: a.reading.documentType,
          confidence: a.reading.confidence,
          pageCount: a.reading.pageCount,
          fields: a.reading.fields.map((f) => ({ label: f.label, value: f.value, tone: f.tone })),
        }
      : null,
  };
}

function snooze(t, m) {
  if (!model.isSnoozed(m)) return null;
  const type = m.snoozeDocumentTypeId ? DOCUMENT_TYPES.find((d) => d.id === m.snoozeDocumentTypeId) : null;
  return {
    until: m.snoozedUntil,
    event: m.snoozeEvent,
    watchShipment: shipmentRef(t, m.snoozeWatchId),
    documentTypeId: type?.id ?? null,
    documentTypeName: type?.name ?? null,
    snoozedAt: m.snoozedAt,
    snoozedBy: user(t, m.snoozedBy),
  };
}

function replyShape(t, r) {
  if (!r) return null;
  return {
    __typename: "InboundReply",
    id: r.id,
    messageId: r.messageId,
    status: r.status,
    fromAddress: r.fromAddress,
    toAddress: r.toAddress,
    ccAddresses: [...r.ccAddresses],
    subject: r.subject,
    body: r.body,
    draftedByNova: r.draftedByNova,
    templateId: r.templateId,
    failureText: r.failureText,
    suggestedAddress: r.suggestedAddress,
    sentBy: user(t, r.sentById),
    sentAt: r.sentAt,
    createdAt: r.createdAt,
    updatedAt: r.updatedAt,
  };
}

function action(t, m, a) {
  if (!a) return null;
  const attachment = a.attachmentId ? m.attachments.find((x) => x.id === a.attachmentId) : null;
  const reply = a.replyId ? m.replies.find((r) => r.id === a.replyId) : null;
  return {
    id: a.id,
    kind: a.kind,
    done: a.done,
    novaCanDo: a.novaCanDo,
    attachment: attachment
      ? { id: attachment.id, fileName: attachment.fileName, kind: attachment.kind, documentId: attachment.documentId, filedShipmentId: attachment.filedShipmentId, filedAs: attachment.filedAs, pageCount: attachment.pageCount }
      : null,
    shipment: shipmentRef(t, a.shipmentId),
    replyPurpose: a.replyPurpose,
    refusedFileName: a.refusedFileName,
    reply: reply ? { id: reply.id, status: reply.status, sentAt: reply.sentAt, draftedByNova: reply.draftedByNova } : null,
    loadDrafted: a.loadDrafted,
    createdShipment: shipmentRef(t, a.createdShipmentId),
    stopsDecision: a.stopsDecision,
    exceptionKind: a.exceptionKind,
    completedAt: a.completedAt || null,
  };
}

function preview(text) {
  const kept = [];
  for (const line of text.split("\n")) {
    if (/^\s*>/.test(line) || /^On .+ wrote:\s*$/.test(line.trim())) break;
    kept.push(line);
  }
  const folded = kept.join(" ").replace(/\s+/g, " ").trim();
  return folded.length > 180 ? `${folded.slice(0, 177)}…` : folded;
}

function listFields(t, m) {
  const customer = m.matchedCustomerId ? t.customers.get(m.matchedCustomerId) : null;
  return {
    __typename: "InboundMessage",
    id: m.id,
    mailboxId: m.mailboxId,
    fromAddress: m.fromAddress,
    fromName: m.fromName,
    subject: m.subject,
    receivedAt: m.receivedAt,
    classification: m.classification,
    confidence: m.confidence,
    status: m.status,
    matchedShipmentId: m.matchedShipmentId,
    matchedCustomerId: m.matchedCustomerId,
    matchReason: m.matchReason,
    needsReview: model.needsReview(m),
    failureText: m.failureText,
    preview: preview(m.textBody),
    attachmentCount: m.attachments.length,
    matchedShipment: shipmentRef(t, m.matchedShipmentId),
    matchedCustomer: customer ? { id: customer.id, code: customer.code, name: customer.name } : null,
    threadKey: m.threadKey,
    threadCount: model.threadCount(t, m.threadKey),
    senderDomain: m.senderDomain,
    ownerId: m.ownerId,
    owner: user(t, m.ownerId),
    dueAt: m.dueAt,
    responseTargetMinutes: model.targetMinutes(t, m.classification),
    snooze: snooze(t, m),
    awaitingReply: model.isAwaitingReply(m) ? { since: m.awaitingSince, until: m.awaitingUntil } : null,
    novaRead: m.novaRead,
    handledByNova: m.handledByNova,
    actions: model.actionsOf(m).map((a) => action(t, m, a)),
    version: m.version,
  };
}

function activityLine(t, line) {
  return {
    id: line.id,
    kind: line.kind,
    actorKind: line.actorKind,
    actor: user(t, line.actorId),
    subject: line.subject,
    detail: line.detail,
    until: line.until,
    fromClassification: line.fromClassification,
    toClassification: line.toClassification,
    recordId: line.recordId,
    createdAt: line.createdAt,
  };
}

function detailFields(t, m) {
  return {
    ...listFields(t, m),
    toAddresses: [...m.toAddresses],
    ccAddresses: [...m.ccAddresses],
    textBody: m.textBody,
    spamScore: m.spamScore,
    matchedCarrierId: m.matchedCarrierId,
    runId: m.runId,
    reviewedBy: m.reviewedBy,
    reviewedAt: m.reviewedAt,
    reviewNote: m.reviewNote,
    failureCode: m.failureCode,
    createdAt: m.createdAt,
    updatedAt: m.updatedAt,
    handledByRuleId: m.handledByRuleId,
    handledAction: m.handledAction,
    classificationCorrected: m.classificationCorrected,
    reviewer: user(t, m.reviewedBy),
    mailbox: mailbox(t, model.mailboxOf(t, m.mailboxId)),
    attachments: m.attachments.map((a) => attachmentDetail(t, a)),
    facts: m.facts.map((f) => ({ key: f.key, label: f.label, value: f.value, quote: f.quote, tone: f.tone, flagged: f.flagged })),
    conflicts: m.conflicts.map((c) => ({
      id: c.id,
      kind: c.kind,
      attachmentId: c.attachmentId,
      existingDocumentId: c.existingDocumentId,
      existingShipment: shipmentRef(t, c.existingShipmentId),
      existingFiledAt: c.existingFiledAt,
      existingAmount: c.existingAmount,
      newAmount: c.newAmount,
      loadIndex: c.loadIndex,
      resolution: c.resolution,
      resolvedAt: c.resolvedAt,
      options: [...c.options],
    })),
    activity: [...m.activity].sort((a, b) => a.createdAt - b.createdAt).map((line) => activityLine(t, line)),
    replies: m.replies.map((r) => replyShape(t, r)),
    replyDraft: replyShape(t, model.replyDraftOf(m)),
    loadDraft: m.loadDraft
      ? { source: m.loadDraft.source, draftedAt: m.loadDraft.draftedAt, draftedByNova: m.loadDraft.draftedByNova, fields: m.loadDraft.fields.map((f) => ({ key: f.key, value: f.value })) }
      : null,
  };
}

function threadFields(t, m) {
  return { ...listFields(t, m), textBody: m.textBody, reviewNote: m.reviewNote, reviewer: user(t, m.reviewedBy) };
}

function sample(t, { m, reference }) {
  return { messageId: m.id, fromName: m.fromName, fromAddress: m.fromAddress, subject: m.subject, classification: m.classification, receivedAt: m.receivedAt, reference };
}

function riverPoint({ m, done }) {
  return { messageId: m.id, fromName: m.fromName, fromAddress: m.fromAddress, subject: m.subject, classification: m.classification, receivedAt: m.receivedAt, dueAt: m.dueAt, done };
}

function deckItem(t, item) {
  if (!item) return null;
  const { m } = item;
  return {
    messageId: m.id,
    fromName: m.fromName,
    fromAddress: m.fromAddress,
    subject: m.subject,
    classification: m.classification,
    receivedAt: m.receivedAt,
    dueAt: m.dueAt,
    action: action(t, m, item.action),
    owner: user(t, m.ownerId),
  };
}

function rule(t, r) {
  const last = model.ruleLastMatch(t, r);
  return {
    __typename: "TriageRule",
    id: r.id,
    classification: r.classification,
    senderScope: r.senderScope,
    sender: r.sender,
    mailboxId: r.mailboxId,
    action: r.action,
    assigneeId: r.assigneeId,
    assignee: user(t, r.assigneeId),
    enabled: r.enabled,
    createdBy: user(t, r.createdById),
    matchesThisMonth: r.baseThisMonth,
    matchCount: r.baseTotal,
    lastMatchedAt: last?.reviewedAt ?? null,
    lastMatch: last ? sample(t, { m: last, reference: model.referenceOf(t, last) }) : null,
    version: r.version,
    createdAt: r.createdAt,
  };
}

function rulePreview(p) {
  return {
    matched: p.matched,
    off: p.off,
    since: p.since,
    matches: p.matches.map((x) => ({ messageId: x.messageId, fromName: x.fromName, fromAddress: x.fromAddress, subject: x.subject, receivedAt: x.receivedAt, reference: x.reference, mismatch: x.mismatch })),
  };
}

function view(t, v) {
  return { id: v.id, name: v.name, query: v.query, position: v.position, waiting: model.viewWaiting(t, v) };
}

function template(t, x) {
  return { id: x.id, name: x.name, body: x.body, version: x.version, createdAt: x.createdAt, createdBy: user(t, x.createdById) };
}

function shipmentPeek(p) {
  return {
    id: p.id,
    proNumber: p.proNumber,
    bol: p.bol,
    poNumber: p.poNumber,
    status: p.status,
    customerName: p.customerName,
    equipment: p.equipment,
    driverName: p.driverName,
    progress: p.progress,
    late: p.late,
    eta: p.eta,
    deliveredAt: p.deliveredAt,
    stops: p.stops.map((s) => ({ kind: s.kind, city: s.city, state: s.state, scheduledAt: s.scheduledAt, arrivedAt: s.arrivedAt, departedAt: s.departedAt, done: s.done })),
    documentsOnFile: [...p.documentsOnFile],
  };
}

function settings(t) {
  return {
    onCallUserId: t.settings.onCallUserId,
    onCallUser: user(t, t.settings.onCallUserId),
    endOfDayMinute: t.settings.endOfDayMinute,
    timezone: t.settings.timezone,
    version: t.settings.version,
  };
}

function bulk(t, result) {
  return { affected: result.affected, ruleId: result.ruleId, messages: result.messages.map((m) => listFields(t, m)) };
}

function briefingShape(t) {
  const b = model.briefing(t);
  return {
    generatedAt: b.generatedAt,
    dayStart: b.dayStart,
    phase: b.phase,
    firstDay: b.firstDay,
    novaEnabled: b.novaEnabled,
    activeSources: b.activeSources,
    waiting: b.waiting,
    overdue: b.overdue,
    closedByPeopleToday: b.closedByPeopleToday,
    handledByNovaToday: b.handledByNovaToday,
    handledByNova: b.handledByNova.map((g) => ({ action: g.action, classification: g.classification, count: g.count, samples: g.samples.map((s) => sample(t, s)) })),
    arrivals: b.arrivals.map((a) => ({ hourStart: a.hourStart, handledByNova: a.handledByNova, leftForPerson: a.leftForPerson })),
    novaPoints: b.novaPoints.map(riverPoint),
    yours: b.yours.map(riverPoint),
    deck: b.deck.map((card) => ({ group: card.group, count: card.count, estimateMinutes: card.estimateMinutes, messageIds: card.messageIds, items: card.items.map((item) => deckItem(t, item)) })),
    longestWaiting: deckItem(t, b.longestWaiting),
    estimateMinutes: b.estimateMinutes,
    corrections: { ...b.corrections },
    comingBack: b.comingBack,
    unhealthySources: b.unhealthySources.map((mb) => mailbox(t, mb)),
    suggestion: b.suggestion ? { ...b.suggestion } : null,
    onCall: user(t, b.onCallUserId),
    carryOver: b.carryOver.map((item) => deckItem(t, item)),
  };
}

// --- operations

export const TRIAGE_HANDLERS = {
  InboundMessages: (state, v) => {
    const t = tri(state);
    const page = model.pageMessages(t, v.input ?? {});
    return {
      inboundMessages: {
        edges: page.edges.map((edge) => ({ cursor: edge.cursor, node: listFields(t, edge.node) })),
        pageInfo: { hasNextPage: page.hasNextPage, endCursor: page.endCursor },
      },
    };
  },
  InboundMessage: (state, v) => {
    const t = tri(state);
    return { inboundMessage: detailFields(t, model.messageOf(t, v.id)) };
  },
  InboundMessageCounts: (state) => ({ inboundMessageCounts: model.counts(tri(state)) }),
  InboundMailboxes: (state) => {
    const t = tri(state);
    return { inboundMailboxes: t.mailboxes.map((mb) => mailbox(t, mb)) };
  },
  InboundThread: (state, v) => {
    const t = tri(state);
    const thread = t.messages.filter((m) => m.threadKey === v.threadKey).sort((a, b) => a.receivedAt - b.receivedAt);
    return { inboundThread: thread.map((m) => threadFields(t, m)) };
  },
  TriageBriefing: (state) => ({ triageBriefing: briefingShape(tri(state)) }),
  MyTriageViews: (state) => {
    const t = tri(state);
    return { myTriageViews: [...t.views].sort((a, b) => a.position - b.position).map((v) => view(t, v)) };
  },
  TriageResponseTargets: (state) => ({ triageResponseTargets: model.responseTargets(tri(state)) }),
  TriageSettings: (state) => ({ triageSettings: settings(tri(state)) }),
  TriageAssignees: (state) => {
    const t = tri(state);
    return {
      triageAssignees: [...t.users.values()].map((u) => ({
        waiting: t.messages.filter((m) => model.isWaiting(m) && m.ownerId === u.id).length,
        user: user(t, u.id),
      })),
    };
  },
  TriageRules: (state) => {
    const t = tri(state);
    return { triageRules: t.rules.map((r) => rule(t, r)) };
  },
  TriageRulePreview: (state, v) => ({ triageRulePreview: rulePreview(model.rulePreview(tri(state), v.input)) }),
  TriageRuleMatches: (state, v) => {
    const t = tri(state);
    return { triageRuleMatches: rulePreview(model.rulePreview(t, model.ruleOf(t, v.id))) };
  },
  TriageSenderDomains: (state, v) => {
    const t = tri(state);
    const frequency = new Map();
    for (const m of t.messages) frequency.set(m.senderDomain, (frequency.get(m.senderDomain) ?? 0) + 1);
    const query = (v.query ?? "").trim().toLowerCase();
    return {
      triageSenderDomains: [...frequency.entries()]
        .filter(([domain]) => !query || domain.includes(query))
        .sort((a, b) => b[1] - a[1] || a[0].localeCompare(b[0]))
        .slice(0, 20)
        .map(([domain]) => domain),
    };
  },
  InboundSenderProfile: (state, v) => {
    const t = tri(state);
    const p = model.senderProfile(t, v.messageId);
    return {
      inboundSenderProfile: {
        address: p.address,
        name: p.name,
        domain: p.domain,
        role: p.role,
        since: p.since,
        messagesLast90Days: p.messagesLast90Days,
        usualReplyMinutes: p.usualReplyMinutes,
        kinds: p.kinds,
        rules: p.rules.map((r) => rule(t, r)),
        recent: p.recent.map((s) => sample(t, s)),
      },
    };
  },
  TriageReplyTemplates: (state) => {
    const t = tri(state);
    return { triageReplyTemplates: t.templates.map((x) => template(t, x)) };
  },
  RenderTriageReplyTemplate: (state, v) => ({ renderTriageReplyTemplate: model.renderTemplate(tri(state), v.templateId, v.messageId) }),
  TriageStopTimes: (state, v) => {
    const t = tri(state);
    const stops = model.stopTimes(t, v.messageId);
    if (!stops) return { triageStopTimes: null };
    const { shipmentId, ...rest } = stops;
    return { triageStopTimes: { shipment: shipmentRef(t, shipmentId), ...rest } };
  },
  TriageShipmentPeek: (state, v) => ({ triageShipmentPeek: shipmentPeek(model.shipmentPeek(tri(state), v.shipmentId)) }),
  TriageCustomerPeek: (state, v) => {
    const t = tri(state);
    const p = model.customerPeek(t, v.customerId);
    return {
      triageCustomerPeek: {
        id: p.id,
        code: p.code,
        name: p.name,
        since: p.since,
        shipmentsThisWeek: p.shipmentsThisWeek,
        waitingInTriage: p.waitingInTriage,
        shipments: p.shipments.map(shipmentPeek),
        messages: p.messages.map((s) => sample(t, s)),
      },
    };
  },

  ReviewInboundMessage: (state, v) => {
    const t = tri(state);
    return { reviewInboundMessage: detailFields(t, model.review(t, v.id, v.input)) };
  },
  LinkInboundMessage: (state, v) => {
    const t = tri(state);
    return { linkInboundMessage: detailFields(t, model.link(t, v.id, v.input)) };
  },
  CreateInboundMailbox: (state, v) => {
    const t = tri(state);
    return { createInboundMailbox: credentials(t, model.createMailbox(t, v.input, v.signingSecret, v.apiKey)) };
  },
  UpdateInboundMailbox: (state, v) => {
    const t = tri(state);
    return { updateInboundMailbox: mailbox(t, model.updateMailbox(t, v.id, v.version, v.input)) };
  },
  RotateInboundMailboxToken: (state, v) => {
    const t = tri(state);
    return { rotateInboundMailboxToken: credentials(t, model.rotateToken(t, v.id)) };
  },
  SetInboundMailboxSigningSecret: (state, v) => {
    const t = tri(state);
    return { setInboundMailboxSigningSecret: mailbox(t, model.setSecret(t, v.id, v.secret, "hasSigningSecret")) };
  },
  SetInboundMailboxApiKey: (state, v) => {
    const t = tri(state);
    return { setInboundMailboxApiKey: mailbox(t, model.setSecret(t, v.id, v.apiKey, "hasApiKey")) };
  },
  ReviewInboundMessages: (state, v) => {
    const t = tri(state);
    return { reviewInboundMessages: bulk(t, model.reviewMany(t, v.selection, v.input)) };
  },
  ReopenInboundMessages: (state, v) => {
    const t = tri(state);
    return { reopenInboundMessages: bulk(t, model.reopenMany(t, v.selection)) };
  },
  AssignInboundMessages: (state, v) => {
    const t = tri(state);
    return { assignInboundMessages: bulk(t, model.assignMany(t, v.selection, v.ownerId ?? null)) };
  },
  SnoozeInboundMessages: (state, v) => {
    const t = tri(state);
    return { snoozeInboundMessages: bulk(t, model.snoozeMany(t, v.selection, v.input)) };
  },
  WakeInboundMessage: (state, v) => {
    const t = tri(state);
    return { wakeInboundMessage: detailFields(t, model.wake(t, v.id)) };
  },
  WaitForInboundReply: (state, v) => {
    const t = tri(state);
    return { waitForInboundReply: detailFields(t, model.waitForReply(t, v.id, v.until)) };
  },
  StopWaitingForInboundReply: (state, v) => {
    const t = tri(state);
    return { stopWaitingForInboundReply: detailFields(t, model.stopWaiting(t, v.id)) };
  },
  UnlinkInboundMessage: (state, v) => {
    const t = tri(state);
    return { unlinkInboundMessage: detailFields(t, model.unlink(t, v.id, v.record)) };
  },
  SendInboundMailboxTest: (state, v) => {
    const t = tri(state);
    return { sendInboundMailboxTest: mailbox(t, model.sendMailboxTest(t, v.id)) };
  },
  HandOffTriage: (state, v) => {
    const t = tri(state);
    const result = model.handOff(t, v.toUserId);
    return { handOffTriage: { affected: result.affected, to: user(t, result.user.id) } };
  },
  ReadUnreadInboundMessages: (state) => ({ readUnreadInboundMessages: model.readUnread(tri(state)) }),
  CreateMyTriageView: (state, v) => {
    const t = tri(state);
    return { createMyTriageView: view(t, model.createView(t, v.name, v.query)) };
  },
  RenameMyTriageView: (state, v) => {
    const t = tri(state);
    return { renameMyTriageView: view(t, model.renameView(t, v.id, v.name)) };
  },
  DeleteMyTriageView: (state, v) => ({ deleteMyTriageView: model.deleteView(tri(state), v.id) }),
  UpdateTriageResponseTargets: (state, v) => ({ updateTriageResponseTargets: model.updateTargets(tri(state), v.input ?? []) }),
  UpdateTriageSettings: (state, v) => {
    const t = tri(state);
    model.updateSettings(t, v.input);
    return { updateTriageSettings: settings(t) };
  },
  CreateTriageRule: (state, v) => {
    const t = tri(state);
    return { createTriageRule: rule(t, model.createRule(t, v.input)) };
  },
  UpdateTriageRule: (state, v) => {
    const t = tri(state);
    return { updateTriageRule: rule(t, model.updateRule(t, v.id, v.version, v.input)) };
  },
  SetTriageRuleEnabled: (state, v) => {
    const t = tri(state);
    return { setTriageRuleEnabled: rule(t, model.setRuleEnabled(t, v.id, v.enabled)) };
  },
  DeleteTriageRule: (state, v) => ({ deleteTriageRule: model.deleteRule(tri(state), v.id) }),
  CorrectInboundClassification: (state, v) => {
    const t = tri(state);
    return { correctInboundClassification: detailFields(t, model.correct(t, v.id, v.classification)) };
  },
  FlagInboundFact: (state, v) => {
    const t = tri(state);
    return { flagInboundFact: detailFields(t, model.flagFact(t, v.id, v.key)) };
  },
  ResolveInboundConflict: (state, v) => {
    const t = tri(state);
    return { resolveInboundConflict: detailFields(t, model.resolveConflict(t, v.id, v.conflictId, v.resolution)) };
  },
  ReopenInboundConflict: (state, v) => {
    const t = tri(state);
    return { reopenInboundConflict: detailFields(t, model.reopenConflict(t, v.id, v.conflictId)) };
  },
  DraftInboundReply: async (state, v) => {
    const t = tri(state);
    return { draftInboundReply: replyShape(t, await model.draftReply(t, v.messageId)) };
  },
  SaveInboundReplyDraft: (state, v) => {
    const t = tri(state);
    return { saveInboundReplyDraft: replyShape(t, model.saveDraft(t, v.messageId, v.subject, v.body)) };
  },
  DiscardInboundReplyDraft: (state, v) => {
    const t = tri(state);
    return { discardInboundReplyDraft: detailFields(t, model.discardDraft(t, v.messageId)) };
  },
  SendInboundReply: async (state, v) => {
    const t = tri(state);
    return { sendInboundReply: replyShape(t, await model.sendReply(t, v.input)) };
  },
  DraftInboundMany: (state, v) => ({ draftInboundMany: model.draftMany(tri(state), v.messageIds ?? []) }),
  CreateTriageReplyTemplate: (state, v) => {
    const t = tri(state);
    return { createTriageReplyTemplate: template(t, model.createTemplate(t, v.input)) };
  },
  UpdateTriageReplyTemplate: (state, v) => {
    const t = tri(state);
    return { updateTriageReplyTemplate: template(t, model.updateTemplate(t, v.id, v.version, v.input)) };
  },
  DeleteTriageReplyTemplate: (state, v) => ({ deleteTriageReplyTemplate: model.deleteTemplate(tri(state), v.id) }),
  DraftInboundLoad: async (state, v) => {
    const t = tri(state);
    return { draftInboundLoad: detailFields(t, await model.draftLoad(t, v.messageId)) };
  },
  DiscardInboundLoad: (state, v) => {
    const t = tri(state);
    return { discardInboundLoad: detailFields(t, model.discardLoad(t, v.messageId)) };
  },
  CreateInboundLoad: (state, v) => {
    const t = tri(state);
    return { createInboundLoad: detailFields(t, model.createLoad(t, v.messageId, v.fields ?? [])) };
  },
  FileInboundAttachment: (state, v) => {
    const t = tri(state);
    return { fileInboundAttachment: detailFields(t, model.fileAttachment(t, v.input)) };
  },
  FileInboundMany: (state, v) => {
    const t = tri(state);
    return { fileInboundMany: bulk(t, model.fileMany(t, v.messageIds ?? [])) };
  },
  UnfileInboundAttachment: (state, v) => {
    const t = tri(state);
    return { unfileInboundAttachment: detailFields(t, model.unfile(t, v.attachmentId)) };
  },
  LogInboundException: (state, v) => {
    const t = tri(state);
    return { logInboundException: detailFields(t, model.logException(t, v.messageId, v.input)) };
  },
  BlockInboundSender: (state, v) => {
    const t = tri(state);
    return { blockInboundSender: detailFields(t, model.blockSender(t, v.messageId)) };
  },
  UnblockInboundSender: (state, v) => {
    const t = tri(state);
    return { unblockInboundSender: mailbox(t, model.unblockSender(t, v.mailboxId, v.sender)) };
  },
  DecideInboundStops: (state, v) => {
    const t = tri(state);
    return { decideInboundStops: detailFields(t, model.decideStops(t, v.messageId, v.decision)) };
  },
};

// --- Desk hand-off: the agents a message can be handed to, and select options the Triage forms read

const AGENTS = [
  { id: "agdef_dispatch", name: "Dispatch", description: "Loads, tenders, carriers", template: "DispatchAssistant", icon: "truck", accent: "blue", toolNames: ["create_shipment", "tender_shipment", "search_carriers"], starters: [{ label: "Build a load", prompt: "Build the load this email describes." }] },
  { id: "agdef_billing", name: "Billing", description: "Invoices, detention, pay", template: "BillingAssistant", icon: "receipt", accent: "teal", toolNames: ["file_document", "waive_detention", "queue_carrier_pay"], starters: [{ label: "Check detention", prompt: "Check the stop times and tell me if the detention stands." }] },
  { id: "agdef_cs", name: "Customer service", description: "Status, ETAs, replies", template: "CustomerAssistant", icon: "message", accent: "amber", toolNames: ["shipment_eta", "draft_reply"], starters: [{ label: "Send an ETA", prompt: "Draft the ETA reply for this customer." }] },
];

export const DESK_HANDLERS = {
  MyAgents: () => ({
    myAgents: {
      edges: AGENTS.map((agent) => ({
        node: {
          ...agent,
          systemKey: "",
          delegates: AGENTS.filter((other) => other.id !== agent.id).map((other) => ({ id: other.id, name: other.name, icon: other.icon, accent: other.accent, template: other.template })),
        },
      })),
      pageInfo: { hasNextPage: false, endCursor: null },
    },
  }),
  SelectOptions: (state, v) => {
    const t = state.triage;
    const input = v.input ?? {};
    const query = (input.query ?? "").toLowerCase();
    const options = {
      DOCUMENT_TYPE: () => DOCUMENT_TYPES.map((d) => ({ id: d.id, label: d.code, description: d.name, meta: { name: d.name } })),
      SHIPMENT: () => [...t.shipments.values()].map((s) => ({ id: s.id, label: s.proNumber, description: `${s.originCity}, ${s.originState} → ${s.destinationCity}, ${s.destinationState}`, meta: { bol: s.bol, poNumber: s.poNumber } })),
      CUSTOMER: () => [...t.customers.values()].map((c) => ({ id: c.id, label: c.code, description: c.name, meta: null })),
      CARRIER: () => [...t.carriers.values()].map((c) => ({ id: c.id, label: c.name, description: c.domain, meta: null })),
      USER: () => [...t.users.values()].map((u) => ({ id: u.id, label: u.name, description: u.emailAddress, meta: null })),
    }[input.resource];
    let rows = options ? options() : [];
    if (input.ids?.length) rows = rows.filter((row) => input.ids.includes(row.id));
    if (query) rows = rows.filter((row) => `${row.label} ${row.description ?? ""}`.toLowerCase().includes(query));
    const offset = input.offset ?? 0;
    const first = input.first ?? 20;
    const page = rows.slice(offset, offset + first);
    return {
      selectOptions: {
        edges: page.map((node, i) => ({ node, cursor: Buffer.from(`offset:${offset + i + 1}`).toString("base64url") })),
        pageInfo: { hasNextPage: offset + first < rows.length, endCursor: page.length ? Buffer.from(`offset:${offset + page.length}`).toString("base64url") : null },
        totalCount: rows.length,
      },
    };
  },
};

// --- REST: presence on the inbox, the Desk thread a hand-off opens, and avatar images

function presenceEvent(action, member) {
  publish("presence", { scope: model.PRESENCE_SCOPE, action, ...member }, { resumable: false });
}

function joinPresence(state, _path, body) {
  const t = state.triage;
  if (!body?.connectionId) return { __status: 422, body: { title: "Invalid", status: 422, errors: [{ field: "connectionId", message: "Connection is required", code: "REQUIRED" }] } };
  const member = {
    userId: USER.id,
    connectionId: body.connectionId,
    name: USER.name,
    messageId: body.messageId ?? null,
    activity: body.activity === "writing" ? "writing" : "viewing",
  };
  t.presence.set(member.connectionId, member);
  presenceEvent("enter", member);
  return model.presenceSnapshot(t);
}

function leavePresence(state, _path, _body, query) {
  const t = state.triage;
  const connectionId = query?.get("connectionId");
  const member = connectionId ? t.presence.get(connectionId) : null;
  if (member) {
    t.presence.delete(connectionId);
    presenceEvent("leave", member);
  }
  return { left: !!member };
}

function typing(state, _path, body) {
  const t = state.triage;
  const member = body?.connectionId ? t.presence.get(body.connectionId) : null;
  if (!member) return { __status: 404, body: { title: "Not found", status: 404 } };
  member.activity = body.stop ? "viewing" : "writing";
  if (body.messageId) member.messageId = body.messageId;
  publish("typing", { scope: model.PRESENCE_SCOPE, userId: member.userId, name: member.name, messageId: member.messageId, stop: !!body.stop }, { resumable: false });
  presenceEvent("enter", member);
  return { ok: true };
}

function thread(state, threadId, body = {}) {
  const now = state.triage.now();
  return {
    id: threadId,
    businessUnitId: USER.businessUnitId,
    organizationId: ORG.id,
    userId: USER.id,
    agentDefinitionId: body.agentDefinitionId ?? AGENTS[0].id,
    title: body.title ?? "",
    status: "Active",
    lastMessageAt: now,
    lastReadAt: now,
    origin: body.origin ?? "Desk",
    pinned: false,
    pinnedFacts: [],
    subjectType: body.subjectType ?? "",
    subjectId: body.subjectId ?? null,
  };
}

function startThread(state, _path, body) {
  const t = state.triage;
  const id = `athr_mock_${t.assistantThreads.size + 1}`;
  const created = thread(state, id, body ?? {});
  t.assistantThreads.set(id, created);
  return created;
}

function avatar(state, path) {
  const id = /\/__mock\/avatar\/([a-z_]+)\.svg$/.exec(path)[1];
  const u = state.triage.users.get(id);
  const label = u ? u.name.split(" ").map((w) => w[0]).slice(0, 2).join("") : "?";
  const hue = [...id].reduce((h, c) => (h * 31 + c.charCodeAt(0)) % 360, 7);
  return {
    __raw: `<svg xmlns="http://www.w3.org/2000/svg" width="64" height="64" viewBox="0 0 64 64"><rect width="64" height="64" rx="32" fill="hsl(${hue} 32% 42%)"/><text x="32" y="40" text-anchor="middle" font-family="Inter, Arial, sans-serif" font-size="24" font-weight="600" fill="#fff">${label}</text></svg>`,
    contentType: "image/svg+xml",
  };
}

export const TRIAGE_ROUTES = [
  ["POST", /^\/api\/v1\/inbox\/presence(\/[^/]+)?\/?$/, joinPresence],
  ["PUT", /^\/api\/v1\/inbox\/presence(\/[^/]+)?\/?$/, joinPresence],
  ["PATCH", /^\/api\/v1\/inbox\/presence(\/[^/]+)?\/?$/, joinPresence],
  ["DELETE", /^\/api\/v1\/inbox\/presence(\/[^/]+)?\/?$/, leavePresence],
  ["POST", /^\/api\/v1\/inbox\/typing\/?$/, typing],
  ["POST", /^\/api\/v1\/assistant\/threads\/?$/, startThread],
  ["GET", /^\/api\/v1\/assistant\/threads\/athr_mock_\d+\/?$/, (state, path) => {
    const id = /athr_mock_\d+/.exec(path)[0];
    return state.triage.assistantThreads.get(id) ?? { __status: 404, body: { title: "Not found", status: 404 } };
  }],
  ["GET", /^\/__mock\/avatar\/[a-z_]+\.svg$/, avatar],
];

