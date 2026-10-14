// Customer tracking: the public page a customer opens from a link
// (`/shipment-tracking/:token`) and the shipment panel's Customer tracking tab.
//
// The public page answers by token, one per state the page draws:
//   in-transit, delayed, delivered, planned, canceled, minimal (route only),
//   pod-ready (delivered with the POD on file); any other token is an invalid link.
// Every timestamp is relative to the real clock, so "12 minutes ago" reads true.
import { ORG } from "../fixtures/session.mjs";

const HOUR = 3600;
const MIN = 60;
const now = () => Math.floor(Date.now() / 1000);

const ALL_FIELDS = [
  "Status",
  "LocationNames",
  "Windows",
  "Actuals",
  "Eta",
  "References",
  "Carrier",
  "Equipment",
  "Freight",
  "LastLocation",
  "Holds",
  "Notes",
  "ProofOfDelivery",
];

function baseView(t) {
  return {
    companyName: ORG.name,
    shipmentProNumber: "S2610-0398",
    fields: ALL_FIELDS,
    expiresAt: t + 27 * 24 * HOUR,
    closed: false,
    bol: "BOL-88398",
    customerReference: "PO-55120",
    carrierName: "Prairie Line Freight",
    equipment: "53' Reefer",
    pieces: 22,
    weight: 38400,
    holds: [],
    notes: [],
    proofOfDelivery: null,
  };
}

function stops(t, shape) {
  const pickup = {
    sequence: 1,
    type: "Pickup",
    city: "Omaha",
    stateCode: "NE",
    locationName: "Heartland Cold Storage",
    addressLine: "4410 S 72nd St",
    windowStart: t - 10 * HOUR,
    windowEnd: t - 8 * HOUR,
  };
  const drop = {
    sequence: 2,
    type: "SplitDelivery",
    city: "Des Moines",
    stateCode: "IA",
    locationName: "Hy-Vee Distribution",
    addressLine: "5820 Westown Pkwy",
    windowStart: t - 4 * HOUR,
    windowEnd: t - 3 * HOUR,
  };
  const delivery = {
    sequence: 3,
    type: "Delivery",
    city: "Minneapolis",
    stateCode: "MN",
    locationName: "North Star Grocers DC",
    addressLine: "1200 Industrial Blvd NE",
    windowStart: t + 4.5 * HOUR,
    windowEnd: t + 5.5 * HOUR,
  };
  switch (shape) {
    case "planned":
      return [
        { ...pickup, windowStart: t + 18 * HOUR, windowEnd: t + 20 * HOUR },
        { ...drop, windowStart: t + 24 * HOUR, windowEnd: t + 25 * HOUR },
        { ...delivery, windowStart: t + 30 * HOUR, windowEnd: t + 31 * HOUR, estimatedArrival: t + 30.2 * HOUR, running: "OnTime" },
      ];
    case "delivered":
      return [
        { ...pickup, windowStart: t - 34 * HOUR, windowEnd: t - 32 * HOUR, arrivedAt: t - 34.2 * HOUR, departedAt: t - 33.4 * HOUR },
        { ...drop, windowStart: t - 28 * HOUR, windowEnd: t - 27 * HOUR, arrivedAt: t - 28.1 * HOUR, departedAt: t - 27.3 * HOUR },
        { ...delivery, windowStart: t - 22 * HOUR, windowEnd: t - 21 * HOUR, arrivedAt: t - 22.3 * HOUR, departedAt: t - 21.4 * HOUR },
      ];
    case "delayed":
      return [
        { ...pickup, arrivedAt: t - 10.1 * HOUR, departedAt: t - 9.2 * HOUR },
        { ...drop, arrivedAt: t - 3.6 * HOUR, departedAt: t - 2.9 * HOUR },
        { ...delivery, estimatedArrival: t + 7.1 * HOUR, running: "Late" },
      ];
    default:
      return [
        { ...pickup, arrivedAt: t - 10.1 * HOUR, departedAt: t - 9.2 * HOUR },
        { ...drop, arrivedAt: t - 4.1 * HOUR, departedAt: t - 3.3 * HOUR },
        { ...delivery, estimatedArrival: t + 4.2 * HOUR, running: "OnTime" },
      ];
  }
}

function publicView(token) {
  const t = now();
  const view = baseView(t);
  switch (token) {
    case "in-transit":
      return {
        ...view,
        status: "InTransit",
        stops: stops(t, "transit"),
        lastLocation: { city: "Albert Lea", stateCode: "MN", reportedAt: t - 12 * MIN },
        notes: [
          { text: "Driver checked in after the Des Moines drop. Running ahead of schedule.", postedAt: t - 3 * HOUR },
          { text: "Reefer set to 34°F per your instructions.", postedAt: t - 9 * HOUR },
        ],
      };
    case "delayed":
      return {
        ...view,
        status: "Delayed",
        stops: stops(t, "delayed"),
        lastLocation: { city: "Clear Lake", stateCode: "IA", reportedAt: t - 26 * MIN },
        holds: [{ type: "OperationalHold", label: "Weather on I-35", startedAt: t - 2 * HOUR }],
        notes: [
          { text: "Snow north of Clear Lake has slowed traffic on I-35. We'll update you as soon as the road clears.", postedAt: t - 40 * MIN },
          { text: "Driver checked in after the Des Moines drop.", postedAt: t - 2.8 * HOUR },
        ],
      };
    case "delivered":
      return {
        ...view,
        status: "Delivered",
        closed: true,
        stops: stops(t, "delivered"),
        notes: [{ text: "Delivered complete, no exceptions noted by the receiver.", postedAt: t - 21 * HOUR }],
      };
    case "pod-ready":
      return {
        ...view,
        status: "Delivered",
        closed: true,
        stops: stops(t, "delivered"),
        proofOfDelivery: { documentId: "doc_pod_0398", fileName: "POD_S2610-0398.pdf", filedAt: t - 20 * HOUR },
        notes: [{ text: "Delivered complete, no exceptions noted by the receiver.", postedAt: t - 21 * HOUR }],
      };
    case "planned":
      return { ...view, status: "Planned", stops: stops(t, "planned"), carrierName: undefined };
    case "canceled":
      return { ...view, status: "Canceled", closed: true, stops: stops(t, "planned").map((s) => ({ ...s, estimatedArrival: undefined, running: undefined })) };
    case "minimal":
      return {
        companyName: ORG.name,
        shipmentProNumber: "S2610-0398",
        fields: ["Status"],
        expiresAt: t + 6 * 24 * HOUR,
        closed: false,
        status: "InTransit",
        stops: stops(t, "transit").map(({ sequence, type, city, stateCode }) => ({ sequence, type, city, stateCode })),
      };
    default:
      return null;
  }
}

const INVALID = {
  __status: 400,
  body: {
    type: "validation-error",
    title: "Validation failed",
    status: 400,
    errors: [{ field: "token", code: "INVALID", message: "This tracking link is no longer valid" }],
  },
};

export const CUSTOMER_TRACKING_ROUTES = [
  [
    "GET",
    /^\/api\/v1\/shipments\/[^/]+\/customer-tracking\/preview\/?$/,
    () => {
      const view = publicView("in-transit");
      return { ...view, fields: view.fields.filter((f) => !["Equipment", "Freight", "Holds"].includes(f)), equipment: undefined, pieces: undefined, weight: undefined };
    },
  ],
  [
    "GET",
    /^\/api\/v1\/shipment-tracking-links\/[^/]+\/?$/,
    (_state, path) => {
      const token = decodeURIComponent(path.split("/")[4] ?? "");
      return publicView(token) ?? INVALID;
    },
  ],
  [
    "GET",
    /^\/api\/v1\/shipment-tracking-links\/[^/]+\/proof-of-delivery\/?$/,
    () => ({ __raw: "%PDF-1.4 mock", contentType: "application/pdf" }),
  ],
];

// The shipment tab. State lives per scenario, keyed on the state object.
const TABS = new WeakMap();

function tabState(state, shipmentId) {
  let byShipment = TABS.get(state);
  if (!byShipment) {
    byShipment = new Map();
    TABS.set(state, byShipment);
  }
  if (!byShipment.has(shipmentId)) {
    byShipment.set(shipmentId, seedTab(shipmentId));
  }
  return byShipment.get(shipmentId);
}

function link(shipmentId, overrides) {
  return {
    id: `ctl_${Math.random().toString(36).slice(2, 10)}`,
    shipmentId,
    customerId: "cus_cargillprote",
    purpose: "Shared",
    recipientEmail: null,
    expiresAt: now() + 30 * 24 * HOUR,
    revokedAt: null,
    lastViewedAt: null,
    viewCount: 0,
    createdById: "usr_mock",
    createdAt: now(),
    ...overrides,
  };
}

function seedTab(shipmentId) {
  const t = now();
  return {
    links: [
      link(shipmentId, {
        id: "ctl_receiving",
        recipientEmail: "receiving@northstargrocers.com",
        createdAt: t - 9 * HOUR,
        expiresAt: t + 29 * 24 * HOUR,
        viewCount: 4,
        lastViewedAt: t - 22 * MIN,
      }),
      link(shipmentId, {
        id: "ctl_pickedup",
        purpose: "Notification",
        recipientEmail: "shipping@cargillprotein.com",
        createdAt: t - 9.2 * HOUR,
        expiresAt: t + 29 * 24 * HOUR,
        viewCount: 1,
        lastViewedAt: t - 8.5 * HOUR,
      }),
      link(shipmentId, {
        id: "ctl_byhand",
        createdAt: t - 2 * 24 * HOUR,
        revokedAt: t - 30 * HOUR,
        viewCount: 0,
      }),
      link(shipmentId, {
        id: "ctl_old",
        recipientEmail: "ap@cargillprotein.com",
        createdAt: t - 40 * 24 * HOUR,
        expiresAt: t - 10 * 24 * HOUR,
        viewCount: 2,
        lastViewedAt: t - 35 * 24 * HOUR,
      }),
    ],
    notifications: [
      {
        id: "csn_pickedup",
        shipmentId,
        event: "PickedUp",
        stopId: "stp_pickup",
        occurredAt: t - 9.2 * HOUR,
        estimatedAt: null,
        status: "Sent",
        skipReason: null,
        recipients: ["shipping@cargillprotein.com", "ops@cargillprotein.com"],
        attempts: 1,
        lastError: null,
        sentAt: t - 9.15 * HOUR,
        createdAt: t - 9.2 * HOUR,
      },
      {
        id: "csn_delayed",
        shipmentId,
        event: "Delayed",
        stopId: "stp_delivery",
        occurredAt: t - 38 * MIN,
        estimatedAt: t + 7.1 * HOUR,
        status: "Pending",
        skipReason: null,
        recipients: ["shipping@cargillprotein.com", "ops@cargillprotein.com"],
        attempts: 2,
        lastError: "550 5.1.1 ops@cargillprotein.com: mailbox unavailable",
        sentAt: null,
        createdAt: t - 38 * MIN,
      },
    ],
  };
}

function profile(customerId) {
  return {
    id: "ctp_cargill",
    customerId,
    linkEnabled: true,
    visibleFields: ALL_FIELDS.filter((f) => f !== "Equipment" && f !== "Freight" && f !== "Holds"),
    linkTtlDays: 30,
    notifyPickedUp: true,
    notifyDelayed: true,
    notifyDelivered: true,
    notifyPodReady: false,
    includeLink: true,
    recipients: "shipping@cargillprotein.com, ops@cargillprotein.com",
    notificationsEnabledAt: now() - 60 * 24 * HOUR,
    version: 3,
    updatedAt: now() - 6 * 24 * HOUR,
  };
}

const APP_ORIGIN = process.env.MOCK_APP_ORIGIN ?? "http://localhost:5173";

export const CUSTOMER_TRACKING_HANDLERS = {
  CustomerTrackingLinks: (state, v) => ({
    customerTrackingLinks: [...tabState(state, v.shipmentId).links].sort((a, b) => b.createdAt - a.createdAt),
  }),
  CustomerStatusNotifications: (state, v) => ({
    customerStatusNotifications: tabState(state, v.shipmentId).notifications,
  }),
  CustomerTrackingProfile: (_state, v) => ({ customerTrackingProfile: profile(v.customerId) }),
  IssueCustomerTrackingLink: (state, v) => {
    const tab = tabState(state, v.input.shipmentId);
    const recipient = v.input.recipientEmail ?? null;
    const created = link(v.input.shipmentId, { recipientEmail: recipient });
    tab.links.unshift(created);
    return {
      issueCustomerTrackingLink: {
        link: created,
        url: `${APP_ORIGIN}/shipment-tracking/in-transit`,
        emailed: recipient != null,
        emailError: null,
      },
    };
  },
  RevokeCustomerTrackingLink: (state, v) => {
    const tab = tabState(state, v.shipmentId);
    let count = 0;
    for (const row of tab.links) {
      const live = row.revokedAt == null && row.expiresAt > now();
      if (live && (v.linkId == null || row.id === v.linkId)) {
        row.revokedAt = now();
        count += 1;
      }
    }
    return { revokeCustomerTrackingLink: count };
  },
};
