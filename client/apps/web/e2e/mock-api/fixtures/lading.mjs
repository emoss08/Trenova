// The Lading studio's fixture: Acme Retail Group, a retailer that tenders loads from its
// distribution centres to its stores, with a project three versions deep, and Northwind Grocers,
// a partner with no project yet. Every project is the module's own starter with the changes a
// planner would have made, so its manifest, readiness, tests, runs and replays are the module's
// answers. A month of Acme's tenders is built here too, as the messages replays and the shadow
// read: from two weeks ago Acme puts a PO number on most tenders, and some store deliveries come
// without a closing time.
import { createHash } from "node:crypto";
import { edi } from "../edi.mjs";
import { ORG } from "./session.mjs";

const DAY = 86_400;
const ZONE = "America/Chicago";

export const ACME_PARTNER_ID = "edip_lading_acme";
export const NORTHWIND_PARTNER_ID = "edip_lading_northwind";
export const ACME_PROJECT_ID = "ldp_acme";

/** What a project is checked against, as `GET /edi/lading/context/` sends it. */
export const LADING_CONTEXT = {
  org: { name: ORG.name, scac: "TRNV", zone: ZONE },
  customFields: {
    shipment: [
      { name: "po_number", type: "Text", label: "PO number" },
      { name: "store_number", type: "Text", label: "Store number" },
      { name: "dock_door", type: "Text", label: "Dock door" },
    ],
  },
};

function partnerRecord({ id, code, name, description, contactName, contactEmail }) {
  return {
    id,
    businessUnitId: "bu_mock",
    organizationId: ORG.id,
    kind: "External",
    status: "Active",
    code,
    name,
    description,
    timezone: ZONE,
    country: "US",
    contactName,
    contactEmail,
    contactPhone: null,
    enabledForInbound: true,
    enabledForOutbound: true,
    settings: {},
    version: 3,
    updatedAt: null,
  };
}

export const LADING_PARTNERS = [
  partnerRecord({
    id: ACME_PARTNER_ID,
    code: "ACME",
    name: "Acme Retail Group",
    description: "Store replenishment from the Dallas and McDonough distribution centres",
    contactName: "Dana Whitfield",
    contactEmail: "edi@acmeretail.example",
  }),
  partnerRecord({
    id: NORTHWIND_PARTNER_ID,
    code: "NWG",
    name: "Northwind Grocers",
    description: "Grocery distribution, onboarding",
    contactName: "Priya Raman",
    contactEmail: "edi@northwindgrocers.example",
  }),
];

/** Acme's codes as the organisation has mapped them; a lookup answers from these. */
export const LADING_LOOKUPS = {
  Customer: { "ACME-BT": { id: "cus_acme", label: "Acme Retail Group" } },
  Location: {
    "DC-DAL": { id: "loc_acme_dal", label: "Acme DC Dallas", zone: ZONE },
    "DC-MCD": { id: "loc_acme_mcd", label: "Acme DC McDonough", zone: "America/New_York" },
    "ST-0412": { id: "loc_acme_0412", label: "Acme Store 0412 Chicago", zone: ZONE },
    "ST-0187": { id: "loc_acme_0187", label: "Acme Store 0187 Naperville", zone: ZONE },
    "ST-0233": {
      id: "loc_acme_0233",
      label: "Acme Store 0233 Indianapolis",
      zone: "America/Indiana/Indianapolis",
    },
    "ST-0951": { id: "loc_acme_0951", label: "Acme Store 0951 Memphis", zone: ZONE },
    "ST-0318": { id: "loc_acme_0318", label: "Acme Store 0318 Columbus", zone: "America/New_York" },
  },
};

const ORIGINS = [
  {
    code: "DC-DAL",
    name: "ACME DC DALLAS",
    street: "4100 DISTRIBUTION WAY",
    city: "DALLAS",
    state: "TX",
    postal: "75212",
  },
  {
    code: "DC-MCD",
    name: "ACME DC MCDONOUGH",
    street: "250 INDUSTRIAL BLVD",
    city: "MCDONOUGH",
    state: "GA",
    postal: "30253",
  },
];
const STORES = [
  {
    code: "ST-0412",
    name: "ACME STORE 0412 CHICAGO",
    street: "1550 N CLYBOURN AVE",
    city: "CHICAGO",
    state: "IL",
    postal: "60642",
  },
  {
    code: "ST-0187",
    name: "ACME STORE 0187 NAPERVILLE",
    street: "2860 FOREST ST",
    city: "NAPERVILLE",
    state: "IL",
    postal: "60563",
  },
  {
    code: "ST-0233",
    name: "ACME STORE 0233 INDIANAPOLIS",
    street: "8700 E 96TH ST",
    city: "INDIANAPOLIS",
    state: "IN",
    postal: "46256",
  },
  {
    code: "ST-0951",
    name: "ACME STORE 0951 MEMPHIS",
    street: "4510 POPLAR AVE",
    city: "MEMPHIS",
    state: "TN",
    postal: "38117",
  },
  {
    code: "ST-0318",
    name: "ACME STORE 0318 COLUMBUS",
    street: "3900 MORSE CROSSING",
    city: "COLUMBUS",
    state: "OH",
    postal: "43219",
  },
];

const PURPOSE_CODE = { Original: "00", Change: "04", Cancel: "01" };

const ymd = (unix) => new Date(unix * 1000).toISOString().slice(0, 10).replaceAll("-", "");
const hm = (unix) => new Date(unix * 1000).toISOString().slice(11, 16).replace(":", "");

/**
 * One 204 in Acme's envelope: a pickup at a distribution centre and a delivery to a store, with
 * the PO number and the store's closing time when Acme sent them.
 */
export function acmeTender({
  control,
  key,
  purpose = "Original",
  bol,
  po,
  origin,
  store,
  pickupDate,
  pickupTime,
  deliveryDate,
  deliveryTime,
  closeTime,
  weight,
  total,
  sentAt,
  usage = "P",
}) {
  const isa = String(control).padStart(9, "0");
  const st = String(control % 10_000).padStart(4, "0");
  const party = (role, site) => [
    `N1*${role}*${site.name}*93*${site.code}`,
    `N3*${site.street}`,
    `N4*${site.city}*${site.state}*${site.postal}*US`,
  ];
  const body = [
    `ST*204*${st}`,
    `B2**TRNV**${key}**PP`,
    `B2A*${PURPOSE_CODE[purpose]}*LT`,
    `L11*${bol}*BM`,
    ...(po ? [`L11*${po}*PO`] : []),
    "N1*BT*ACME RETAIL GROUP*93*ACME-BT",
    "S5*1*LD",
    `G62*69*${pickupDate}*U*${pickupTime}`,
    ...party("SH", origin),
    "S5*2*UL",
    `G62*70*${deliveryDate}*X*${deliveryTime}`,
    ...(closeTime ? [`G62*54*${deliveryDate}*Z*${closeTime}`] : []),
    ...party("CN", store),
    `L3*${weight}*G***${Math.round(total * 100)}*******L`,
  ];
  const sent = sentAt ?? 0;
  return [
    `ISA*00*          *00*          *ZZ*ACMERETAIL     *ZZ*TRENOVA        *${ymd(sent).slice(2)}*${hm(sent)}*^*00401*${isa}*0*${usage}*>`,
    `GS*SM*ACMERETAIL*TRENOVA*${ymd(sent)}*${hm(sent)}*${control}*X*004010`,
    ...body,
    `SE*${body.length + 1}*${st}`,
    `GE*1*${control}`,
    `IEA*1*${isa}`,
  ]
    .map((segment) => `${segment}~\n`)
    .join("");
}

/** The samples a version keeps: the flow tests' three messages and a tender with a PO number. */
function acmeSamples({ closeTimes }) {
  const sentAt = Date.UTC(2026, 0, 9, 12) / 1000;
  const base = {
    key: "L-1001",
    bol: "BOL-55",
    origin: ORIGINS[0],
    store: STORES[0],
    pickupDate: "20260110",
    pickupTime: "0800",
    deliveryDate: "20260111",
    deliveryTime: "1000",
    closeTime: closeTimes ? "1400" : undefined,
    weight: 42_000,
    total: 1_850,
    sentAt,
  };
  return [
    { path: "samples/tender-original.x12", content: acmeTender({ ...base, control: 101 }) },
    {
      path: "samples/tender-change.x12",
      content: acmeTender({ ...base, control: 102, purpose: "Change", total: 1_950 }),
    },
    {
      path: "samples/tender-cancel.x12",
      content: acmeTender({ ...base, control: 103, purpose: "Cancel" }),
    },
    {
      path: "samples/tender-with-po.x12",
      content: acmeTender({
        ...base,
        control: 104,
        key: "L-1002",
        bol: "BOL-56",
        po: "4500018842",
        store: STORES[1],
      }),
    },
  ];
}

function starter(usage) {
  return edi("lading.starter", {
    name: "acme",
    label: "Acme Retail Group",
    role: "carrier",
    them: ["ZZ", "ACMERETAIL"],
    us: ["ZZ", "TRENOVA"],
    zone: ZONE,
    sets: ["204", "990", "214", "210"],
    usage,
    acknowledge: "997",
    ackWithin: 30,
  });
}

/** `files` with `from` replaced by `to` in `path`; a change that no longer applies fails loudly. */
function patch(files, path, from, to) {
  return files.map((file) => {
    if (file.path !== path) return file;
    if (!file.source.includes(from)) {
      throw new Error(`lading fixture: ${path} no longer holds ${JSON.stringify(from)}`);
    }
    return { ...file, source: file.source.replace(from, to) };
  });
}

const PO_ASSIGNMENT = 'poNumber <- L11[02 == "PO"].01\n';

/** Version 3: Acme's PO number saved on the load, and store deliveries held without a closing time. */
function draftFiles(files) {
  let next = patch(
    files,
    "maps/tender-in.lading",
    PO_ASSIGNMENT,
    `${PO_ASSIGNMENT}custom.po_number <- L11[02 == "PO"].01\n`,
  );
  next = next.map((file) =>
    file.path === "maps/tender-in.lading"
      ? {
          ...file,
          source: `${file.source}
test "the PO number is saved on the load" {
  given file "samples/tender-with-po.x12"
  expect custom.po_number == "4500018842"
}
`,
        }
      : file.path === "rules/tender-checks.lading"
        ? {
            ...file,
            source: `${file.source}
rule "stores receive inside a window" {
  require exists stops.last.window.end
    else hold "Delivery to {stops.last.name} has no closing time"
}
`,
          }
        : file,
  );
  return next;
}

export const digestOf = (content) => createHash("sha256").update(content).digest("hex");

/** The manifest of a version, checked with its samples, which its tests read. */
export function checkOf(files, samples) {
  return edi("lading.check", { project: { files, samples }, context: LADING_CONTEXT });
}

function readinessOf(files, samples, checkedAt) {
  const report = edi("lading.test", { project: { files, samples }, context: LADING_CONTEXT });
  if (!report.readiness.ready) {
    throw new Error(
      `lading fixture: a released version is not ready: ${JSON.stringify(report.readiness)}`,
    );
  }
  return { ...report.readiness, checkedAt };
}

/** A stored version from its files and samples, checked by the module. */
export function versionRecord({
  id,
  number,
  status,
  files,
  samples,
  notes,
  times,
  parentVersionId,
}) {
  const manifest = checkOf(files, samples);
  return {
    id,
    projectId: ACME_PROJECT_ID,
    number,
    status,
    parentVersionId: parentVersionId ?? null,
    files,
    samples: samples.map((sample) => ({
      path: sample.path,
      digest: digestOf(sample.content),
      size: sample.content.length,
    })),
    manifest,
    readiness: times.readiness ? readinessOf(files, samples, times.readiness) : null,
    notes,
    errorCount: manifest.errors,
    warningCount: manifest.warnings,
    shadowUntil: times.shadowUntil ?? null,
    certifiedAt: times.certified ?? null,
    activatedAt: times.activated ?? null,
    supersededAt: times.superseded ?? null,
    archivedAt: null,
    version: times.version ?? 1,
    createdAt: times.created,
    updatedAt: times.updated ?? times.created,
  };
}

/** Acme's tenders over the last 35 days, about three a day, in the order they arrived. */
function acmeTraffic(anchor) {
  let seed = 20_261_027;
  const random = () => {
    seed = (seed * 1_103_515_245 + 12_345) % 2_147_483_648;
    return seed / 2_147_483_648;
  };
  const messages = [];
  let control = 4_100;
  let load = 48_210;
  for (let day = 35; day >= 1; day -= 1) {
    const count = 2 + Math.floor(random() * 3);
    for (let index = 0; index < count; index += 1) {
      const sentAt = anchor - day * DAY + Math.floor((0.25 + random() * 0.4) * DAY);
      const origin = ORIGINS[random() < 0.7 ? 0 : 1];
      const store = STORES[Math.floor(random() * STORES.length)];
      const pickup = sentAt + DAY;
      const delivery = pickup + DAY;
      const key = `L-${load}`;
      load += 1;
      control += 1;
      const tender = {
        control,
        key,
        bol: `BOL-${load + 7_000}`,
        po:
          day <= 14 && random() < 0.7
            ? `45000${String(18_000 + control).padStart(5, "0")}`
            : undefined,
        origin,
        store,
        pickupDate: ymd(pickup),
        pickupTime: ["0600", "0800", "1000"][Math.floor(random() * 3)],
        deliveryDate: ymd(delivery),
        deliveryTime: ["0700", "0900", "1100"][Math.floor(random() * 3)],
        closeTime: random() < 0.86 ? "1500" : undefined,
        weight: 18_000 + Math.floor(random() * 26) * 1_000,
        total: 900 + Math.floor(random() * 140) * 10,
        sentAt,
      };
      messages.push({
        id: `edim_acme_${control}`,
        set: "204",
        generatedAt: sentAt,
        x12: acmeTender(tender),
      });
      if (random() < 0.12) {
        control += 1;
        messages.push({
          id: `edim_acme_${control}`,
          set: "204",
          generatedAt: sentAt + 3_600,
          x12: acmeTender({
            ...tender,
            control,
            purpose: "Change",
            total: tender.total + 75,
            sentAt: sentAt + 3_600,
          }),
        });
      }
    }
  }
  return messages;
}

/**
 * The Lading state for one scenario. `stage` is `draft` (version 3 is the draft being written,
 * replayed once against version 2) or `shadow` (version 3 was certified and has run in shadow for
 * six days).
 */
export function createLadingState(anchor, stage) {
  const testStarter = starter("test");
  const live = starter("production");
  const v2Samples = acmeSamples({ closeTimes: false });
  const v3Samples = acmeSamples({ closeTimes: true });
  const v3Files = draftFiles(live.files);
  const shadowing = stage === "shadow";

  const versions = [
    versionRecord({
      id: "ldv_acme_1",
      number: 1,
      status: "Superseded",
      files: testStarter.files,
      samples: testStarter.samples,
      notes: "Started from Trenova's carrier starter",
      times: {
        created: anchor - 41 * DAY,
        readiness: anchor - 40 * DAY,
        certified: anchor - 40 * DAY,
        activated: anchor - 39 * DAY,
        superseded: anchor - 26 * DAY,
        version: 4,
      },
    }),
    versionRecord({
      id: "ldv_acme_2",
      number: 2,
      status: "Active",
      parentVersionId: "ldv_acme_1",
      files: live.files,
      samples: v2Samples,
      notes:
        "Production envelope after Acme signed off the test interchange. Samples are Acme's own tenders from the test week.",
      times: {
        created: anchor - 29 * DAY,
        readiness: anchor - 27 * DAY,
        certified: anchor - 27 * DAY,
        activated: anchor - 26 * DAY,
        version: 4,
      },
    }),
    versionRecord({
      id: "ldv_acme_3",
      number: 3,
      status: shadowing ? "Shadow" : "Draft",
      parentVersionId: "ldv_acme_2",
      files: v3Files,
      samples: v3Samples,
      notes:
        "Acme's guide 4.1 puts the PO number on every tender: save it to the PO number field. Stores turn trucks away after closing, so a delivery without a closing time waits for a planner.",
      times: {
        created: anchor - 6 * DAY - 6 * 3600,
        updated: anchor - 6 * DAY - 4 * 3600,
        readiness: shadowing ? anchor - 6 * DAY - 3 * 3600 : undefined,
        certified: shadowing ? anchor - 6 * DAY - 2 * 3600 : undefined,
        shadowUntil: shadowing ? anchor + DAY : undefined,
        version: shadowing ? 5 : 2,
      },
    }),
  ];

  const samples = new Map();
  for (const sample of [...testStarter.samples, ...v2Samples, ...v3Samples]) {
    samples.set(digestOf(sample.content), sample.content);
  }

  const project = {
    id: ACME_PROJECT_ID,
    ediPartnerId: ACME_PARTNER_ID,
    activeVersionId: "ldv_acme_2",
    shadowVersionId: shadowing ? "ldv_acme_3" : null,
    settings: { autoAccept: true },
    nextVersionNumber: 4,
    version: 7,
    createdAt: anchor - 41 * DAY,
    updatedAt: anchor - 26 * DAY,
    partner: { id: ACME_PARTNER_ID, code: "ACME", name: "Acme Retail Group" },
  };

  return {
    projects: new Map([[project.id, project]]),
    versions: new Map(versions.map((version) => [version.id, version])),
    samples,
    replays: new Map(),
    shadowResults: new Map(),
    traffic: acmeTraffic(anchor),
    nextId: 1,
  };
}
