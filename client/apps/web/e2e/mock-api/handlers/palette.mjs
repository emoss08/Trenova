// The command palette's endpoints: global search with the server's enriched
// metadata, the previews' GraphQL and REST reads, pinned pages, and a quick
// question to Nova streamed the way the assistant streams a turn — tool steps
// that name the records they read, then the answer.

import { PALETTE_CUSTOMERS, PALETTE_WORKERS } from "../fixtures/palette.mjs";

const SHIPMENT_HREF = (id) =>
  `/shipment-management/shipments?expanded=${id}&panelEntityId=${id}&panelType=edit`;
const CUSTOMER_HREF = (id) =>
  `/billing/configuration-files/customers?panelEntityId=${id}&panelType=edit`;
const WORKER_HREF = (id) => `/hr/workers?panelEntityId=${id}&panelType=edit`;
const MINOR = (value) => (value / 100).toFixed(2);
const DAY = 86400;

const palette = (state) => state.palette;
const workerName = (w) => `${w.first} ${w.last}`;
const customerById = (id) => PALETTE_CUSTOMERS.find((c) => c.id === id);
const workerById = (id) => PALETTE_WORKERS.find((w) => w.id === id);

function matches(query, ...fields) {
  const q = query.trim().toLowerCase();
  if (q === "") return true;
  const haystack = fields.filter(Boolean).join(" ").toLowerCase();
  return q.split(/\s+/).every((term) => haystack.includes(term));
}

function currentLoadOf(state, workerId) {
  return palette(state).shipments.find(
    (s) =>
      s.palette.driver?.id === workerId &&
      ["InTransit", "Delayed", "Assigned", "New"].includes(s.status),
  );
}

function shipmentHit(s) {
  const customer = customerById(s.customerId);
  return {
    id: s.id,
    entityType: "shipment",
    title: s.proNumber,
    href: SHIPMENT_HREF(s.id),
    metadata: {
      proNumber: s.proNumber,
      status: s.status,
      bol: s.bol,
      customerId: customer.id,
      customerName: customer.name,
      customerCode: customer.code,
      origin: `${s.palette.origin[0]}, ${s.palette.origin[1]}`,
      destination: `${s.palette.dest[0]}, ${s.palette.dest[1]}`,
      ...(s.palette.driver
        ? { driverName: workerName(s.palette.driver), driverId: s.palette.driver.id }
        : {}),
    },
  };
}

function customerHit(c) {
  const open = c.buckets.reduce((sum, v) => sum + v, 0);
  const overdue = open - c.buckets[0];
  return {
    id: c.id,
    entityType: "customer",
    title: c.name,
    subtitle: c.code,
    href: CUSTOMER_HREF(c.id),
    metadata: {
      status: c.status ?? "Active",
      code: c.code,
      city: `${c.city}, ${c.state}`,
      openBalance: MINOR(open),
      overdueBalance: MINOR(overdue),
      currencyCode: "USD",
    },
  };
}

function workerHit(state, w) {
  const load = currentLoadOf(state, w.id);
  return {
    id: w.id,
    entityType: "worker",
    title: workerName(w),
    subtitle: w.type,
    href: WORKER_HREF(w.id),
    metadata: {
      status: "Active",
      externalId: w.externalId,
      workerType: w.type,
      driverType: w.driverType,
      city: `${w.city}, ${w.state}`,
      availableForDispatch: String(w.available),
      ...(load
        ? { currentLoad: load.proNumber, currentLoadId: load.id, currentLoadStatus: load.status }
        : {}),
    },
  };
}

function documentHit(d) {
  return {
    id: d.id,
    entityType: "document",
    title: d.originalName,
    subtitle: d.resourceType,
    href: SHIPMENT_HREF(d.resourceId),
    metadata: {
      resourceType: d.resourceType,
      resourceId: d.resourceId,
      status: d.status,
      detectedKind: d.detectedKind,
      receivedAt: String(d.createdAt),
      attachedLabel: d.palette.pro,
    },
  };
}

const KIND_WORDS = {
  BillOfLading: "bill of lading bol",
  ProofOfDelivery: "proof of delivery pod",
  RateConfirmation: "rate confirmation ratecon",
  Invoice: "invoice",
};

function globalSearch(state, _path, _body, query) {
  const q = (query.get("query") ?? "").trim();
  const limit = Math.min(Number(query.get("limit") ?? 8), 25);
  const types = (query.get("entityTypes") ?? "shipment,customer,worker,document").split(",");
  const browse = q === "" && types.length === 1;
  if (q === "" && !browse) return { query: "", groups: [] };

  const p = palette(state);
  const candidates = {
    shipment: () =>
      p.shipments
        .filter((s) =>
          matches(
            q,
            s.proNumber,
            s.bol,
            customerById(s.customerId).name,
            s.palette.driver && workerName(s.palette.driver),
            s.palette.origin[0],
            s.palette.dest[0],
          ),
        )
        .map(shipmentHit),
    customer: () =>
      PALETTE_CUSTOMERS.filter((c) => matches(q, c.name, c.code, c.city)).map(customerHit),
    worker: () =>
      PALETTE_WORKERS.filter((w) => matches(q, workerName(w), w.externalId, w.city, w.type)).map(
        (w) => workerHit(state, w),
      ),
    document: () =>
      p.documents
        .filter((d) => matches(q, d.originalName, d.palette.pro, KIND_WORDS[d.detectedKind]))
        .map(documentHit),
  };
  const labels = {
    shipment: "Shipments",
    customer: "Customers",
    worker: "Workers",
    document: "Documents",
  };
  const groups = types
    .filter((type) => candidates[type])
    .map((type) => {
      const hits = candidates[type]();
      return {
        entityType: type,
        label: labels[type],
        total: hits.length,
        hits: hits.slice(0, limit),
      };
    })
    .filter((group) => group.hits.length > 0);
  return { query: q, groups };
}

function documentById(state, path) {
  const id = /doc_cp_\d+/.exec(path)?.[0];
  return palette(state).documents.find((d) => d.id === id) ?? null;
}

function strip(document) {
  const { palette: _palette, ...rest } = document;
  return rest;
}

function getDocument(state, path) {
  const document = documentById(state, path);
  return document ? strip(document) : { __status: 404, body: { title: "Not found", status: 404 } };
}

function shipmentDraft(state, path) {
  const document = documentById(state, path);
  if (!document) return { __status: 404, body: { title: "Not found", status: 404 } };
  const fields = Object.fromEntries(
    document.palette.fields.map(([key, label, value, confidence, review]) => [
      key,
      { label, value, confidence, reviewRequired: Boolean(review), conflict: false },
    ]),
  );
  return {
    id: `dsd_${document.id}`,
    documentId: document.id,
    organizationId: "org_mock",
    businessUnitId: "bu_mock",
    status: "Ready",
    documentKind: document.detectedKind,
    confidence: document.palette.confidence,
    draftData: {
      kind: document.detectedKind,
      overallConfidence: document.palette.confidence,
      missingFields: [],
      signals: [],
      conflicts: [],
      fields,
      stops: [],
    },
    failureCode: null,
    failureMessage: null,
    attachedShipmentId: document.resourceId,
    attachedAt: document.createdAt,
    attachedById: null,
    version: 1,
    createdAt: document.createdAt,
    updatedAt: document.createdAt,
  };
}

function documentThumbnail(state, path) {
  const document = documentById(state, path);
  const title = (document?.detectedKind ?? "Document")
    .replace(/([a-z])([A-Z])/g, "$1 $2")
    .toUpperCase();
  const lines = (document?.palette.fields ?? [])
    .map(
      ([, label, value], i) =>
        `<text x="40" y="${150 + i * 34}" font-size="16" fill="#555">${label}</text><text x="230" y="${150 + i * 34}" font-size="16" fill="#111">${String(value).replace(/&/g, "&amp;")}</text>`,
    )
    .join("");
  return {
    __raw: `<svg xmlns="http://www.w3.org/2000/svg" width="612" height="792" viewBox="0 0 612 792"><rect width="612" height="792" fill="#fff"/><text x="40" y="70" font-family="Arial" font-size="26" font-weight="700" fill="#111">${title}</text><text x="40" y="100" font-family="Arial" font-size="14" fill="#777">${document?.originalName ?? ""} · page 1</text><line x1="40" y1="118" x2="572" y2="118" stroke="#ddd"/><g font-family="Arial">${lines}</g>${Array.from({ length: 10 }, (_, i) => `<rect x="40" y="${330 + i * 26}" width="${420 + (i % 3) * 40}" height="8" rx="4" fill="#eee"/>`).join("")}</svg>`,
    contentType: "image/svg+xml",
  };
}

// --- Nova: a quick question answered from the fixture's records ------------------------------

function record(type, id, label) {
  return { type, id, label };
}

function shipmentRecord(s) {
  return record("shipment", s.id, s.proNumber);
}

function answerFor(state, content, mentions) {
  const p = palette(state);
  const pro = (value) => p.shipments.find((s) => s.proNumber === value);
  const subject = mentions?.[0] ?? null;
  const lower = content.toLowerCase();

  if (subject?.type === "shipment") {
    const s = p.shipments.find((x) => x.id === subject.id);
    const customer = customerById(s.customerId);
    return {
      tools: [
        {
          name: "get_shipment",
          summary: s.proNumber,
          records: [shipmentRecord(s), record("customer", customer.id, customer.name)],
        },
        { name: "list_shipment_events", summary: `${s.palette.events.length} events`, records: [] },
      ],
      text: `${s.proNumber} is running late — the latest ETA has it about 20 minutes inside ${customer.name}'s receiving window. Most of the slip came from one event: a 52-minute hold at the Meridian weigh station. The receiver is open until 18:00, so it is still on time for them.`,
    };
  }
  if (subject?.type === "customer") {
    const c = customerById(subject.id);
    const open = c.buckets.reduce((sum, v) => sum + v, 0);
    return {
      tools: [
        {
          name: "get_customer_ar_profile",
          summary: c.name,
          records: [record("customer", c.id, c.name)],
        },
      ],
      text: `${c.name} owes $${(open / 100).toLocaleString("en-US", { minimumFractionDigits: 2 })}, of which $${((open - c.buckets[0]) / 100).toLocaleString("en-US", { minimumFractionDigits: 2 })} is past terms. The oldest item is over 90 days and is a disputed detention charge.`,
    };
  }
  if (subject?.type === "worker") {
    const w = workerById(subject.id);
    const load = currentLoadOf(state, w.id);
    return {
      tools: [
        {
          name: "get_worker_hos",
          summary: workerName(w),
          records: [record("worker", w.id, workerName(w)), ...(load ? [shipmentRecord(load)] : [])],
        },
      ],
      text: `${workerName(w)} has ${Math.floor(w.hos[0] / 3600000)}h ${Math.round((w.hos[0] % 3600000) / 60000)}m of drive time left${load ? ` and is still on ${load.proNumber}` : ""}.`,
    };
  }
  if (/late|behind|delay|risk|eta|on time/.test(lower)) {
    const late = pro("S-240193");
    const risk = pro("S-240201");
    const leon = workerById("wrk_cp_1063");
    const halv = customerById("cus_cp_halv");
    return {
      tools: [
        {
          name: "list_shipments",
          summary: "42 shipments",
          records: [shipmentRecord(late), shipmentRecord(risk)],
        },
        {
          name: "get_worker_hos",
          summary: workerName(leon),
          records: [record("worker", leon.id, workerName(leon))],
        },
        {
          name: "get_customer",
          summary: halv.name,
          records: [record("customer", halv.id, halv.name)],
        },
      ],
      text: `Two loads are behind. ${late.proNumber} is late into Atlanta after a 52-minute weigh station hold, but Acme's receiver is open until 18:00. ${risk.proNumber} is the real risk — ${workerName(leon)} has 2h 10m of drive time for about 3h of road. Warn ${halv.name} now or relay the last leg.`,
    };
  }
  return {
    tools: [{ name: "search_records", summary: "No records", records: [] }],
    text: "I couldn't find anything in your records about that. Continue in the Desk and I'll look wider, including mail in Triage.",
  };
}

function thread(id, anchor, title) {
  return {
    id,
    businessUnitId: "bu_mock",
    organizationId: "org_mock",
    userId: "usr_mock",
    agentDefinitionId: "agd_cp_nova",
    title,
    status: "Active",
    lastMessageAt: anchor,
    lastReadAt: anchor,
    origin: "Ask",
    pinned: false,
    pinnedFacts: [],
    subjectType: "",
    subjectId: null,
    version: 1,
    createdAt: anchor,
    updatedAt: anchor,
  };
}

function startAsk(state, _path, body) {
  const asks = palette(state).asks;
  const n = asks.size + 1;
  const turnId = `aturn_cp_${n}`;
  const threadId = `athr_cp_${n}`;
  const created = thread(threadId, state.now(), body?.content ?? "");
  asks.set(turnId, {
    threadId,
    thread: created,
    content: body?.content ?? "",
    mentions: body?.mentions ?? [],
  });
  return {
    turnId,
    threadId,
    streamUrl: `/assistant/turns/${turnId}/stream/`,
    status: "Running",
    thread: created,
  };
}

function followAsk(state, path) {
  const turnId = /aturn_cp_\d+/.exec(path)[0];
  const ask = palette(state).asks.get(turnId);
  if (!ask) return { __status: 404, body: { title: "Not found", status: 404 } };
  const answer = answerFor(state, ask.content, ask.mentions);
  const frames = [
    { event: "accepted", data: { content: ask.content }, delay: 250 },
    { event: "thread", data: ask.thread, delay: 50 },
  ];
  answer.tools.forEach((tool, index) => {
    const callId = `call_cp_${turnId}_${index}`;
    frames.push({
      event: "tool_started",
      data: { callId, name: tool.name, arguments: {}, effect: "lookup" },
      delay: 350,
    });
    frames.push({
      event: "tool_finished",
      data: {
        callId,
        name: tool.name,
        failed: false,
        proposed: false,
        content: JSON.stringify({ records: tool.records }),
        effect: "lookup",
        summary: tool.summary,
        records: tool.records,
      },
      delay: 700,
    });
  });
  const words = answer.text.split(/(?<= )/);
  for (let i = 0; i < words.length; i += 2) {
    frames.push({
      event: "delta",
      data: { text: words.slice(i, i + 2).join("") },
      delay: i === 0 ? 300 : 45,
    });
  }
  const at = state.now();
  const messages = [
    {
      id: `amsg_${turnId}_1`,
      threadId: ask.threadId,
      sequence: 1,
      role: "User",
      kind: "Message",
      content: ask.content,
      mentions: ask.mentions,
      model: "",
      inputTokens: 0,
      outputTokens: 0,
      createdAt: at,
    },
    {
      id: `amsg_${turnId}_2`,
      threadId: ask.threadId,
      sequence: 2,
      role: "Assistant",
      kind: "Message",
      content: answer.text,
      mentions: null,
      model: "mock",
      inputTokens: 0,
      outputTokens: 0,
      createdAt: at,
    },
  ];
  frames.push({ event: "message", data: { content: answer.text, model: "mock" }, delay: 40 });
  frames.push({
    event: "done",
    data: { thread: ask.thread, messages, reply: answer.text, refused: false },
    delay: 40,
  });
  return { __sse: frames };
}

function keepThread(state, path) {
  const threadId = /athr_cp_\d+/.exec(path)[0];
  const found = [...palette(state).asks.values()].find((ask) => ask.threadId === threadId);
  return found ? found.thread : { __status: 404, body: { title: "Not found", status: 404 } };
}

export const PALETTE_ROUTES = [
  ["GET", /^\/api\/v1\/search\/global\/?$/, globalSearch],
  ["GET", /^\/api\/v1\/documents\/doc_cp_\d+\/?$/, getDocument],
  ["GET", /^\/api\/v1\/documents\/doc_cp_\d+\/shipment-draft\/?$/, shipmentDraft],
  ["GET", /^\/api\/v1\/documents\/doc_cp_\d+\/preview\/?$/, documentThumbnail],
  ["GET", /^\/api\/v1\/page-favorites\/?$/, (state) => palette(state).favorites],
  ["POST", /^\/api\/v1\/assistant\/ask\/?$/, startAsk],
  ["GET", /^\/api\/v1\/assistant\/turns\/aturn_cp_\d+\/stream\/?$/, followAsk],
  ["POST", /^\/api\/v1\/assistant\/turns\/aturn_cp_\d+\/stop\/?$/, () => ({})],
  ["PATCH", /^\/api\/v1\/assistant\/threads\/athr_cp_\d+\/?$/, keepThread],
];

// --- GraphQL ---------------------------------------------------------------------------------

function nodeOf(s) {
  const { palette: _palette, ...node } = s;
  return node;
}

function customerNode(c) {
  return {
    id: c.id,
    status: c.status ?? "Active",
    code: c.code,
    name: c.name,
    addressLine1: "100 Commerce St",
    city: c.city,
    postalCode: "00000",
    updatedAt: 0,
    state: { abbreviation: c.state },
    billingProfile: {
      paymentTerm: c.terms,
      creditStatus: c.status === "Inactive" ? "Hold" : "Active",
      creditLimit: MINOR(c.limit),
      creditBalance: MINOR(c.buckets.reduce((sum, v) => sum + v, 0)),
      invoiceDelivery: "Email",
      billingCycle: "Immediate",
    },
    emailProfile: { toRecipients: c.contact },
  };
}

function arProfile(state, customerId) {
  const c = customerById(customerId);
  if (!c) return null;
  const [current, d30, d60, d90, over] = c.buckets;
  const total = current + d30 + d60 + d90 + over;
  return {
    snapshot: {
      customerId: c.id,
      customerName: c.name,
      totalOpenMinor: total,
      overdueMinor: total - current,
      unappliedCashMinor: 0,
      creditLimitMinor: c.limit,
      hasCreditLimit: true,
      openInvoiceCount: 7,
      oldestOpenInvoiceDate: state.anchor - 96 * DAY,
      oldestDaysPastDue: over > 0 ? 96 : d90 > 0 ? 70 : 0,
      lastPaymentDate: state.anchor - 9 * DAY,
      lastPaymentMinor: 412000,
      avgDaysToPay: 34,
      billedTrailing91Minor: total * 2,
      buckets: {
        currentMinor: current,
        days1To30Minor: d30,
        days31To60Minor: d60,
        days61To90Minor: d90,
        daysOver90Minor: over,
        totalOpenMinor: total,
      },
      monthlyCollections: [],
    },
    dsoDays: 38,
    creditUtilization: total / c.limit,
    delinquencyScore: 0.2,
  };
}

function recordShipments(state, input, includeTotalCount) {
  const filters = input.fieldFilters ?? [];
  const customer = filters.find((f) => f.field === "customerId")?.value;
  const closed = filters.find((f) => f.field === "status" && f.operator === "notin")?.value ?? [];
  const worker = input.workerIds?.[0];
  const rows = palette(state)
    .shipments.filter((s) => (customer ? s.customerId === customer : true))
    .filter((s) => (worker ? s.palette.driver?.id === worker : true))
    .filter((s) => !closed.includes(s.status))
    .slice(0, input.first ?? 10);
  const all = palette(state).shipments.filter(
    (s) =>
      (customer ? s.customerId === customer : true) &&
      (worker ? s.palette.driver?.id === worker : true) &&
      !closed.includes(s.status),
  );
  return {
    shipments: {
      edges: rows.map((s) => ({
        node: {
          id: s.id,
          proNumber: s.proNumber,
          status: s.status,
          totalChargeAmount: s.totalChargeAmount,
          moves: s.moves.map((m) => ({
            sequence: m.sequence,
            stops: m.stops.map((stop) => ({
              sequence: stop.sequence,
              type: stop.type,
              scheduledWindowStart: stop.scheduledWindowStart,
              location: {
                city: stop.location.city,
                state: { abbreviation: stop.location.state.abbreviation },
              },
            })),
          })),
        },
      })),
      pageInfo: { hasNextPage: false, endCursor: null },
      totalCount: includeTotalCount ? all.length : undefined,
    },
  };
}

function deskWorker(state, id) {
  const w = workerById(id);
  if (!w) return { worker: null };
  return {
    worker: {
      id: w.id,
      status: "Active",
      type: w.type,
      driverType: w.driverType,
      firstName: w.first,
      lastName: w.last,
      phoneNumber: "555-0100",
      email: `${w.first.toLowerCase()}@mock.example`,
      city: w.city,
      externalId: w.externalId,
      assignmentBlocked: false,
      canBeAssigned: true,
      availableForDispatch: w.available,
      createdAt: state.anchor - 400 * DAY,
      state: { abbreviation: w.state },
      fleetCode: null,
      manager: null,
      profile: {
        licenseNumber: `D${w.externalId.slice(2)}000`,
        cdlClass: w.cdl,
        cdlRestrictions: [],
        endorsement: "N",
        licenseExpiry: state.anchor + w.licenseDays * DAY,
        hazmatExpiry: null,
        medicalCardExpiry: state.anchor + w.medicalDays * DAY,
        twicExpiry: null,
        physicalDueDate: null,
        mvrDueDate: null,
        hireDate: state.anchor - 900 * DAY,
        isQualified: true,
        disqualificationReason: "",
        licenseState: { abbreviation: w.state },
      },
    },
  };
}

export function paletteHandlers(base) {
  const isPalette = (id) => typeof id === "string" && id.startsWith("shp_cp_");
  return {
    CommandPaletteCustomerPreview: (_state, v) => {
      const c = customerById(v.id);
      return { customer: c ? customerNode(c) : null };
    },
    ArCustomerProfile: (state, v) => ({ arCustomerProfile: arProfile(state, v.customerId) }),
    DeskRecordShipments: (state, v) =>
      v.input?.invoiceIds?.length ||
      v.input?.driverSettlementIds?.length ||
      v.input?.carrierSettlementIds?.length ||
      v.input?.fieldFilters?.some((filter) => filter.field === "id")
        ? base.DeskRecordShipments(state, v)
        : recordShipments(state, v.input ?? {}, v.includeTotalCount),
    DeskWorkerCard: (state, v) => deskWorker(state, v.id),
    WorkerHosState: (state, v) => ({ workerHosState: palette(state).hos.get(v.workerId) ?? null }),
    ShipmentDetail: (state, v) => {
      if (!isPalette(v.id)) return base.ShipmentDetail(state, v);
      const s = palette(state).shipments.find((x) => x.id === v.id);
      return { shipment: s ? nodeOf(s) : null };
    },
    ShipmentEvents: (state, v) => {
      if (!isPalette(v.input?.shipmentId)) return base.ShipmentEvents(state, v);
      return {
        shipmentEvents: (palette(state).events.get(v.input.shipmentId) ?? []).slice(
          0,
          v.input?.limit ?? 50,
        ),
      };
    },
    AttentionSummary: (state, v) =>
      !state.scenario.palette
        ? base.AttentionSummary(state, v)
        : {
            attentionSummary: {
              billingQueue: 6,
              pendingApprovals: 0,
              reconciliationExceptions: 0,
              serviceFailures: 2,
              ediAttention: 0,
              agentDecisions: 3,
            },
          },
    WorkerRosterAttention: () => ({
      workerRosterAttention: {
        activeWorkers: 64,
        nonCompliant: 0,
        trainingOverdue: 5,
        atRisk: 1,
        expiringSoon: 2,
        reviewsAwaitingSignOff: 0,
        ptoLiabilityDays: 0,
        leaveCertificationsOutstanding: 0,
      },
    }),
  };
}
