// The Formula Studio's and the formula libraries' data on Tariff: the team, the shipments the
// preview can rate (as overlays on the rules module's sample record), the standard templates,
// three saved copies of "Regional per mile + fuel" (a draft, one in review and one active) and
// three organisation libraries (approved, in review and a draft whose test fails), with their
// versions, reviews and usage. Every source checks under the module the client ships, and the
// mock asks that module for every check, test, bill and scaffold. Timestamps are relative to the
// board's anchor.
import { ORG, USER } from "./session.mjs";

const MOCK_ORIGIN =
  process.env.MOCK_API_PUBLIC_URL ?? `http://localhost:${process.env.MOCK_API_PORT ?? 8080}`;

export const DRAFT_TEMPLATE_ID = "fmt_studio_draft";
export const REVIEW_TEMPLATE_ID = "fmt_studio_review";
export const ACTIVE_TEMPLATE_ID = "fmt_studio_active";

export const ACCESSORIALS_LIBRARY_ID = "fl_studio_accessorials";
export const FUEL_LIBRARY_ID = "fl_studio_fuel";
export const ZONES_LIBRARY_ID = "fl_studio_zones";

const DAY = 86400;

// --- People ---------------------------------------------------------------------------------

/** The team the other fixtures use (Triage's), each a full user as the server serves one. */
export const TEAM = [
  {
    id: "usr_priya",
    name: "Priya Shah",
    username: "priya.shah",
    emailAddress: "priya.shah@trenova.app",
    jobTitle: "Pricing lead",
  },
  {
    id: "usr_marcus",
    name: "Marcus Lee",
    username: "marcus.lee",
    emailAddress: "marcus.lee@trenova.app",
    jobTitle: "Controller",
  },
  {
    id: "usr_dana",
    name: "Dana Whitfield",
    username: "dana.whitfield",
    emailAddress: "dana.whitfield@trenova.app",
    jobTitle: "Billing",
  },
];

export function avatarUrl(id) {
  return `${MOCK_ORIGIN}/__mock/avatar/${id}.svg`;
}

export function userRecord(id) {
  const person =
    id === USER.id ? { ...USER, jobTitle: "Administrator" } : TEAM.find((u) => u.id === id);
  if (!person) return null;
  const picture = id === USER.id ? "" : avatarUrl(id);
  return {
    id: person.id,
    version: 1,
    createdAt: 1700000000,
    updatedAt: 1700000000,
    businessUnitId: USER.businessUnitId,
    currentOrganizationId: ORG.id,
    status: "Active",
    name: person.name,
    username: person.username,
    emailAddress: person.emailAddress,
    profilePicUrl: picture,
    thumbnailUrl: picture,
    timezone: "America/Chicago",
    timeFormat: "12-hour",
    locale: "en",
    isLocked: false,
    mustChangePassword: false,
    jobTitle: person.jobTitle ?? "",
  };
}

// --- The organisation's context ---------------------------------------------------------------

/** The accessorial charge codes a tariff may bill with `charge … as CODE`; an uncoded charge is freight. */
export const CODES = [
  { code: "FSC", description: "Fuel surcharge", active: true },
  { code: "STOP", description: "Extra stop", active: true },
  { code: "HAZ", description: "Hazmat handling", active: true },
  { code: "WKND", description: "Weekend pickup", active: true },
  { code: "DET", description: "Detention", active: true },
];

export const SETTINGS = { distanceUnit: "mi" };

// --- Shipments --------------------------------------------------------------------------------

const place = (city, state, zip) => ({ city, state, zip, zip3: zip.slice(0, 3), country: "US" });
const miles = (n) => ({ $q: String(n), u: "mi" });
const pounds = (n) => ({ $q: String(n), u: "lb" });
const at = (iso) => ({ $date: iso });

/**
 * The real-shipment picker's loads, each an overlay on the module's sample record, and how the
 * picker and the backtest name it. `stops` lists the kinds after the first pickup.
 */
export const SHIPMENTS = [
  {
    id: "shp_fs_240811",
    pro: "S-240811",
    lane: "Dallas, TX → Atlanta, GA",
    meta: "781 mi · 2 stops · dry van",
    customer: "Acme Foods",
    from: place("Dallas", "TX", "75207"),
    to: place("Atlanta", "GA", "30318"),
    distance: 781,
    weight: 41200,
    stops: ["Delivery"],
    pickupAt: "2026-10-12T08:00:00-05:00",
  },
  {
    id: "shp_fs_240794",
    pro: "S-240794",
    lane: "Fresno, CA → Denver, CO",
    meta: "1,146 mi · 3 stops · reefer",
    customer: "Harborline Grocers",
    from: place("Fresno", "CA", "93706"),
    to: place("Denver", "CO", "80216"),
    distance: 1146,
    weight: 39800,
    stops: ["Delivery", "Delivery"],
    reefer: true,
    pickupAt: "2026-10-17T06:30:00-07:00",
  },
  {
    id: "shp_fs_240772",
    pro: "S-240772",
    lane: "Houston, TX → Baton Rouge, LA",
    meta: "271 mi · 4 stops · hazmat",
    customer: "Gulf Coast Chemical",
    from: place("Houston", "TX", "77029"),
    to: place("Baton Rouge", "LA", "70805"),
    distance: 271,
    weight: 22600,
    stops: ["Pickup", "Delivery", "Delivery"],
    hazmat: true,
    pickupAt: "2026-10-14T09:00:00-05:00",
  },
  {
    id: "shp_fs_240758",
    pro: "S-240758",
    lane: "Dallas, TX → Fort Worth, TX",
    meta: "38 mi · 2 stops · dry van",
    customer: "Crestline Retail",
    from: place("Dallas", "TX", "75207"),
    to: place("Fort Worth", "TX", "76102"),
    distance: 38,
    weight: 12400,
    stops: ["Delivery"],
    pickupAt: "2026-10-15T10:00:00-05:00",
  },
  {
    id: "shp_fs_240741",
    pro: "S-240741",
    lane: "Memphis, TN → Chicago, IL",
    meta: "532 mi · 5 stops · weekend",
    customer: "Acme Foods",
    from: place("Memphis", "TN", "38118"),
    to: place("Chicago", "IL", "60609"),
    distance: 532,
    weight: 30500,
    stops: ["Delivery", "Delivery", "Delivery", "Delivery"],
    pickupAt: "2026-10-18T07:00:00-05:00",
  },
  {
    id: "shp_fs_240733",
    pro: "S-240733",
    lane: "Phoenix, AZ → Los Angeles, CA",
    meta: "372 mi · 2 stops · reefer",
    customer: "Harborline Grocers",
    from: place("Phoenix", "AZ", "85043"),
    to: place("Los Angeles", "CA", "90021"),
    distance: 372,
    weight: 36100,
    stops: ["Delivery"],
    reefer: true,
    pickupAt: "2026-10-16T05:00:00-07:00",
  },
  {
    id: "shp_fs_240719",
    pro: "S-240719",
    lane: "Denver, CO → Salt Lake City, UT",
    meta: "525 mi · 3 stops · hazmat",
    customer: "Gulf Coast Chemical",
    from: place("Denver", "CO", "80216"),
    to: place("Salt Lake City", "UT", "84104"),
    distance: 525,
    weight: 28800,
    stops: ["Delivery", "Delivery"],
    hazmat: true,
    pickupAt: "2026-10-13T08:00:00-06:00",
  },
];

/**
 * A shipment's overlay on the sample record: its distance, weight, flags, lane and stops. The
 * mock's rules runtime (`recordFrom`) lays it over the sample, so every field it leaves out is
 * the sample's.
 */
export function shipmentOverlay(shipment, stop) {
  const location = (p) => ({
    city: p.city,
    state: p.state,
    zip: p.zip,
    zip3: p.zip3,
    country: p.country,
    timezone: "America/Chicago",
  });
  const stops = [
    stop("Pickup", 1, {
      location: location(shipment.from),
      window: { start: at(shipment.pickupAt), end: null },
    }),
    ...shipment.stops.map((kind, index) =>
      stop(kind, index + 2, {
        location: location(index === shipment.stops.length - 1 ? shipment.to : shipment.from),
      }),
    ),
  ];
  return {
    shipment: {
      proNumber: shipment.pro,
      status: "Completed",
      distance: miles(shipment.distance),
      weight: pounds(shipment.weight),
      hazmat: Boolean(shipment.hazmat),
      reefer: Boolean(shipment.reefer),
      stops,
    },
    lane: {
      origin: { ...shipment.from, zones: [] },
      destination: { ...shipment.to, zones: [] },
      interstate: shipment.from.state !== shipment.to.state,
    },
    customer: { name: shipment.customer, code: shipment.customer.split(" ")[0].toUpperCase() },
  };
}

/**
 * The shipments a backtest or approval impact re-rates: the picker's seven, then recent freight
 * drawn from a seeded generator, 64 in all.
 */
export function backtestShipments() {
  let seed = 7;
  const rnd = () => (seed = (seed * 16807) % 2147483647) / 2147483647;
  const rows = [...SHIPMENTS];
  const lanes = [
    [place("Dallas", "TX", "75207"), place("Oklahoma City", "OK", "73108")],
    [place("Atlanta", "GA", "30318"), place("Charlotte", "NC", "28206")],
    [place("Denver", "CO", "80216"), place("Albuquerque", "NM", "87102")],
    [place("Houston", "TX", "77029"), place("San Antonio", "TX", "78219")],
  ];
  for (let i = rows.length; i < 64; i++) {
    const [from, to] = lanes[i % lanes.length];
    const extra = Math.floor(rnd() * rnd() * 4);
    rows.push({
      id: `shp_fs_bt_${239400 + i * 7}`,
      pro: `S-${239400 + i * 7}`,
      lane: `${from.city}, ${from.state} → ${to.city}, ${to.state}`,
      meta: "",
      customer: i % 3 === 0 ? "Acme Foods" : "Crestline Retail",
      from,
      to,
      distance: Math.round(40 + rnd() * 1100),
      weight: Math.round(8000 + rnd() * 36000),
      stops: [...Array.from({ length: extra }, () => "Delivery"), "Delivery"],
      hazmat: rnd() < 0.1,
      reefer: rnd() < 0.2,
      pickupAt: "2026-10-12T08:00:00-05:00",
    });
  }
  return rows;
}

// --- Libraries --------------------------------------------------------------------------------

const ACCESSORIALS_V1 = `library "org.accessorials"

/// Accessorial charges our tariffs share.

/// Extra stops: \`fee\` for every stop beyond the first \`included\`.
fn extraStops(stops: Int, included: Int = 2, fee: Money = $50) -> Money {
  if stops <= included {
    return $0
  }
  return fee * (stops - included)
}

test "extra stops bill each stop beyond those included" {
  expect extraStops(2) == $0
  expect extraStops(3) == $50
}
`;

const ACCESSORIALS_V3 = `library "org.accessorials"

/// Accessorial charges our tariffs share.

/// The hazmat handling fee, per shipment.
const HAZMAT_FEE: Money = $175

/// Extra stops: \`fee\` for every stop beyond the first \`included\`.
fn extraStops(stops: Int, included: Int = 2, fee: Money = $75) -> Money {
  if stops <= included {
    return $0
  }
  return fee * (stops - included)
}

/// The hazmat fee when the shipment carries hazardous goods.
fn hazmat(carries: Bool) -> Money {
  if carries {
    return HAZMAT_FEE
  }
  return $0
}

test "extra stops bill each stop beyond those included" {
  expect extraStops(2) == $0
  expect extraStops(3) == $75
  expect extraStops(5, fee: $50) == $150
}

test "hazmat bills only hazardous loads" {
  expect hazmat(true) == $175
  expect hazmat(false) == $0
}
`;

const FUEL_V1 = `library "org.fuel"

/// Fuel surcharges by the DOE diesel price.

/// The surcharge for a diesel price: 1% for every whole 6¢ above the $1.20 peg.
fn surcharge(price: Money/gal) -> Percent {
  if price <= $1.20/gal {
    return 0%
  }
  return 1% * ((price - $1.20/gal) / $0.06/gal).floor()
}

test "a step for every 6 cents above the peg" {
  expect surcharge($1.20/gal) == 0%
  expect surcharge($2.40/gal) == 20%
}
`;

const FUEL_V2 = `library "org.fuel"

/// Fuel surcharges by the DOE diesel price.

/// The price the surcharge starts above.
const PEG: Money/gal = $1.25/gal

/// The surcharge for a diesel price: 1% for every whole 6¢ above the peg.
fn surcharge(price: Money/gal) -> Percent {
  if price <= PEG {
    return 0%
  }
  return 1% * ((price - PEG) / $0.06/gal).floor()
}

test "a step for every 6 cents above the peg" {
  expect surcharge($1.25/gal) == 0%
  expect surcharge($2.45/gal) == 20%
  expect surcharge($3.85/gal) == 43%
}
`;

const ZONES_V1 = `library "org.zones"

/// Delivery zones by the destination's zip3.

/// The zone a destination zip3 falls in, 1 to 3.
fn zone(zip3: Text) -> Int {
  if zip3 in ["750", "751", "752", "760", "761"] {
    return 1
  }
  if zip3 in ["770", "773", "787"] {
    return 2
  }
  return 3
}

test "Dallas and Fort Worth are zone 1" {
  expect zone("752") == 1
  expect zone("761") == 1
}

test "Austin is zone 2" {
  expect zone("787") == 2
  expect zone("786") == 2
}
`;

/** One saved library: its row, versions and review history. */
function savedLibrary(anchor, spec) {
  const versions = spec.versions.map((version, index) => ({
    versionNumber: index + 1,
    status: version.status,
    createdById: version.by,
    createdAt: anchor - version.daysAgo * DAY,
    changeMessage: version.message ?? "",
    content: { name: spec.name, description: spec.description, source: version.source },
  }));
  const current = versions.at(-1);
  const approved = [...versions].reverse().find((version) => version.status === "Active");
  return {
    id: spec.id,
    name: spec.name,
    description: spec.description,
    source: current.content.source,
    status: spec.status,
    submittedById: spec.submittedById ?? null,
    submittedAt: spec.submittedDaysAgo != null ? anchor - spec.submittedDaysAgo * DAY : null,
    approvedById: approved ? (spec.approvedById ?? null) : null,
    approvedAt: approved ? approved.createdAt : null,
    reviewComment: spec.reviewComment ?? "",
    version: versions.length + 1,
    sourceLibraryId: null,
    sourceVersionNumber: null,
    currentVersionNumber: current.versionNumber,
    createdAt: versions[0].createdAt,
    updatedAt: current.createdAt + 600,
    versions,
    reviews: (spec.reviews ?? []).map((review, index) => ({
      id: `${spec.id.replace("fl_", "flr_")}_${index + 1}`,
      libraryId: spec.id,
      round: review.round,
      decision: review.decision,
      actorId: review.by,
      comment: review.comment ?? "",
      baseVersionNumber: review.base,
      createdAt: anchor - review.daysAgo * DAY,
      requestedReviewerId: review.reviewerId ?? null,
      reviewerNote: review.reviewerNote ?? "",
    })),
  };
}

/** The organisation's three libraries, built fresh for each scenario. */
export function createFormulaLibraries(anchor) {
  return [
    savedLibrary(anchor, {
      id: ACCESSORIALS_LIBRARY_ID,
      name: "accessorials",
      description:
        "Accessorial charges the regional tariffs share: extra stops and hazmat handling.",
      status: "Active",
      approvedById: "usr_priya",
      reviewComment: "Good for every regional tariff.",
      versions: [
        {
          source: ACCESSORIALS_V1,
          status: "Draft",
          by: "usr_dana",
          daysAgo: 45,
          message: "Initial version",
        },
        {
          source: ACCESSORIALS_V1,
          status: "Active",
          by: "usr_priya",
          daysAgo: 44,
          message: "Approved",
        },
        {
          source: ACCESSORIALS_V3,
          status: "Active",
          by: "usr_priya",
          daysAgo: 31,
          message: "Hazmat handling, and stops to $75",
        },
      ],
      reviews: [
        {
          round: 1,
          decision: "Submitted",
          by: "usr_dana",
          base: 0,
          daysAgo: 45,
          comment: "Extra stops, shared.",
        },
        { round: 1, decision: "Approved", by: "usr_priya", base: 0, daysAgo: 44 },
        {
          round: 2,
          decision: "Submitted",
          by: "usr_dana",
          base: 2,
          daysAgo: 32,
          comment: "Adds hazmat handling at $175 and raises stops to $75.",
        },
        {
          round: 2,
          decision: "Approved",
          by: "usr_priya",
          base: 2,
          daysAgo: 31,
          comment: "Good for every regional tariff.",
        },
      ],
    }),
    savedLibrary(anchor, {
      id: FUEL_LIBRARY_ID,
      name: "fuel",
      description: "The DOE-indexed fuel surcharge our dry van contracts quote.",
      status: "InReview",
      approvedById: "usr_marcus",
      submittedById: "usr_dana",
      submittedDaysAgo: 1,
      versions: [
        {
          source: FUEL_V1,
          status: "Draft",
          by: "usr_dana",
          daysAgo: 20,
          message: "Initial version",
        },
        { source: FUEL_V1, status: "Active", by: "usr_marcus", daysAgo: 19, message: "Approved" },
        {
          source: FUEL_V2,
          status: "Draft",
          by: "usr_dana",
          daysAgo: 1,
          message: "The peg moves to $1.25",
        },
      ],
      reviews: [
        { round: 1, decision: "Submitted", by: "usr_dana", base: 0, daysAgo: 20 },
        { round: 1, decision: "Approved", by: "usr_marcus", base: 0, daysAgo: 19 },
        {
          round: 2,
          decision: "Submitted",
          by: "usr_dana",
          base: 2,
          daysAgo: 1,
          comment: "The peg moves to $1.25 with the new contracts.",
          reviewerId: USER.id,
          reviewerNote: "Moves the peg from $1.20 to $1.25 and names it `PEG`. Every test passes.",
        },
      ],
    }),
    savedLibrary(anchor, {
      id: ZONES_LIBRARY_ID,
      name: "zones",
      description: "Delivery zones for the Texas lanes, by destination zip3.",
      status: "Draft",
      versions: [
        { source: ZONES_V1, status: "Draft", by: USER.id, daysAgo: 2, message: "Initial version" },
      ],
    }),
  ];
}

// --- Templates --------------------------------------------------------------------------------

const DESCRIPTION =
  "Regional dry van lanes. Per-mile linehaul with a percentage fuel surcharge, a fee for each stop past the second, and a hazmat fee.";

/** The regional tariff, at a fuel percent and with or without the library's accessorials. */
function regional({ name, fuel, testFuel = fuel, accessorials = true }) {
  const head = `tariff "${name}"

/// Linehaul by the mile and fuel on it${accessorials ? ", extra stops and hazmat" : ""}, with a minimum for short hauls.
${accessorials ? "\nuse org.accessorials.{ extraStops, hazmat }\n" : ""}
/// The rate per loaded mile.
input rate: Money/mi = $2.85/mi
/// The fuel surcharge, as a share of the linehaul.
input fuelPercent: Percent = ${fuel}%
/// The least a shipment pays.
input minimumCharge: Money = $350

let linehaul = rate * shipment.distance

charge "Linehaul" (quantity: shipment.distance, rate: rate)
charge "Fuel surcharge" as FSC = linehaul * fuelPercent
`;
  const extras = accessorials
    ? `
let stops = extraStops(shipment.stops.len)
if stops > $0 {
  charge "Extra stops" as STOP = stops
}
if shipment.hazmat {
  charge "Hazmat" as HAZ = hazmat(true)
}
`
    : "";
  const fuelOn742 = (2114.7 * testFuel) / 100;
  const total = (2114.7 + fuelOn742).toFixed(2);
  return `${head}${extras}
minimum minimumCharge

test "a 742 mile load bills linehaul and fuel" {
  given shipment.distance = 742 mi
  expect charge("Linehaul") == $2_114.70
  expect total == $${total.replace(/^(\d)(\d{3})/, "$1_$2")}
}

test "a short haul bills the minimum" {
  given shipment.distance = 38 mi
  expect total == $350
}
${
  accessorials
    ? `
test "stops past the second bill $75 each" {
  given shipment.distance = 500 mi
  given shipment.stops = [pickup(), delivery(), delivery(), delivery()]
  expect charge(STOP) == $150
}
`
    : ""
}`;
}

/** Template usage, by the kinds the server counts. */
export const USAGE = [
  { type: "rate_agreement", count: 6 },
  { type: "rate_agreement_rule", count: 2 },
  { type: "shipment", count: 214 },
];

function savedTemplate(anchor, spec) {
  const versions = spec.versions.map((version, index) => ({
    versionNumber: index + 1,
    status: version.status,
    createdById: version.by,
    createdAt: anchor - version.daysAgo * DAY,
    changeMessage: version.message ?? "",
    tags: version.tags ?? [],
    content: {
      name: spec.name,
      description: spec.description ?? DESCRIPTION,
      source: version.source,
    },
  }));
  const current = versions.at(-1);
  const approved = [...versions].reverse().find((version) => version.status === "Active");
  return {
    id: spec.id,
    name: spec.name,
    description: spec.description ?? DESCRIPTION,
    source: current.content.source,
    status: spec.status,
    submittedById: spec.submittedById ?? null,
    submittedAt: spec.submittedDaysAgo != null ? anchor - spec.submittedDaysAgo * DAY : null,
    approvedById: spec.status === "Active" ? (spec.approvedById ?? null) : null,
    approvedAt: spec.status === "Active" && approved ? approved.createdAt : null,
    reviewComment: spec.reviewComment ?? "",
    version: versions.length + 2,
    sourceTemplateId: spec.sourceTemplateId ?? null,
    sourceVersionNumber: spec.sourceVersionNumber ?? null,
    currentVersionNumber: current.versionNumber,
    createdAt: versions[0].createdAt,
    updatedAt: current.createdAt + 600,
    versions,
    usages: spec.usages ?? [],
    reviews: spec.reviews.map((review, index) => ({
      id: `${spec.id.replace("fmt_", "ftr_")}_${index + 1}`,
      templateId: spec.id,
      round: review.round,
      decision: review.decision,
      actorId: review.by,
      comment: review.comment ?? "",
      baseVersionNumber: review.base,
      createdAt: anchor - review.daysAgo * DAY,
      requestedReviewerId: review.reviewerId ?? null,
      reviewerNote: review.reviewerNote ?? "",
    })),
  };
}

/** The three saved templates, built fresh for each scenario. */
export function createFormulaTemplates(anchor) {
  const draftName = "Regional per mile + fuel";
  const draft = savedTemplate(anchor, {
    id: DRAFT_TEMPLATE_ID,
    name: draftName,
    status: "Draft",
    reviewComment: "Approved for the regional lanes.",
    versions: [
      {
        source: regional({ name: draftName, fuel: 14, accessorials: false }),
        status: "Draft",
        by: "usr_dana",
        daysAgo: 40,
        message: "Initial version",
      },
      {
        source: regional({ name: draftName, fuel: 14 }),
        status: "Draft",
        by: "usr_dana",
        daysAgo: 33,
      },
      {
        source: regional({ name: draftName, fuel: 14 }),
        status: "Active",
        by: "usr_priya",
        daysAgo: 30,
        message: "Approved",
        tags: ["Production"],
      },
      // Fuel moved to 18%, but the first test still expects the 14% total: it fails until fixed.
      {
        source: regional({ name: draftName, fuel: 18, testFuel: 14 }),
        status: "Draft",
        by: USER.id,
        daysAgo: 1,
      },
    ],
    usages: USAGE,
    reviews: [
      {
        round: 1,
        decision: "Submitted",
        by: "usr_dana",
        base: 0,
        daysAgo: 38,
        comment: "First cut of the regional per-mile method.",
      },
      {
        round: 1,
        decision: "ChangesRequested",
        by: "usr_marcus",
        base: 0,
        daysAgo: 36,
        comment:
          "Hazmat loads on these lanes carry a handling fee. Import it from org.accessorials before this goes live.",
      },
      {
        round: 1,
        decision: "Submitted",
        by: "usr_dana",
        base: 0,
        daysAgo: 33,
        comment: "Imports extra stops and hazmat from org.accessorials.",
      },
      {
        round: 1,
        decision: "Approved",
        by: "usr_priya",
        base: 0,
        daysAgo: 30,
        comment: "Approved for the regional lanes.",
      },
    ],
  });

  const reviewName = "Regional per mile + fuel · Southeast";
  const review = savedTemplate(anchor, {
    id: REVIEW_TEMPLATE_ID,
    name: reviewName,
    status: "InReview",
    submittedById: "usr_dana",
    submittedDaysAgo: 2,
    reviewComment: "Fuel to 18% to match the DOE index this quarter.",
    sourceTemplateId: DRAFT_TEMPLATE_ID,
    sourceVersionNumber: 3,
    versions: [
      {
        source: regional({ name: reviewName, fuel: 16 }),
        status: "Draft",
        by: "usr_dana",
        daysAgo: 26,
        message: "Forked from Regional per mile + fuel v3",
      },
      {
        source: regional({ name: reviewName, fuel: 16 }),
        status: "Active",
        by: "usr_marcus",
        daysAgo: 24,
        message: "Approved",
        tags: ["Production"],
      },
      {
        source: regional({ name: reviewName, fuel: 18 }),
        status: "Draft",
        by: "usr_dana",
        daysAgo: 2,
      },
    ],
    usages: [
      { type: "rate_agreement", count: 3 },
      { type: "shipment", count: 88 },
    ],
    reviews: [
      {
        round: 1,
        decision: "Submitted",
        by: "usr_dana",
        base: 0,
        daysAgo: 25,
        comment: "Southeast copy of the regional method.",
      },
      { round: 1, decision: "Approved", by: "usr_marcus", base: 0, daysAgo: 24 },
      {
        round: 2,
        decision: "Submitted",
        by: "usr_dana",
        base: 2,
        daysAgo: 2,
        comment: "Fuel to 18% to match the DOE index this quarter.",
        reviewerId: USER.id,
        reviewerNote: "Raises `fuelPercent` from 16% to 18%. Every test passes on this version.",
      },
    ],
  });

  const activeName = "Regional per mile + fuel · Mountain West";
  const active = savedTemplate(anchor, {
    id: ACTIVE_TEMPLATE_ID,
    name: activeName,
    status: "Active",
    approvedById: "usr_priya",
    reviewComment: "Good to go.",
    sourceTemplateId: DRAFT_TEMPLATE_ID,
    sourceVersionNumber: 3,
    versions: [
      {
        source: regional({ name: activeName, fuel: 18 }),
        status: "Draft",
        by: "usr_marcus",
        daysAgo: 16,
        message: "Forked from Regional per mile + fuel v3",
      },
      {
        source: regional({ name: activeName, fuel: 18 }),
        status: "Active",
        by: "usr_priya",
        daysAgo: 14,
        message: "Approved",
        tags: ["Production", "Stable"],
      },
    ],
    usages: USAGE,
    reviews: [
      {
        round: 1,
        decision: "Submitted",
        by: "usr_marcus",
        base: 0,
        daysAgo: 15,
        comment: "Mountain West lanes, same method at 18% fuel.",
      },
      {
        round: 1,
        decision: "Approved",
        by: "usr_priya",
        base: 0,
        daysAgo: 14,
        comment: "Good to go.",
      },
    ],
  });

  return [draft, review, active];
}

// --- Standard templates -----------------------------------------------------------------------

/** standard_templates.yaml's first four, as Tariff source with their own tests. */
export const STANDARDS = [
  {
    name: "Flat Rate",
    description:
      "One flat amount for the whole shipment, whatever it weighs or however far it goes.",
    source: `tariff "Flat Rate"

/// Bills one flat amount for the shipment.

/// The amount the shipment pays.
input rate: Money
/// The least the shipment pays.
input minimumCharge: Money = $0

charge "Flat Rate" = rate

minimum minimumCharge

test "bills the flat amount" {
  given input.rate = $850
  expect charge("Flat Rate") == $850
  expect total == $850
}
`,
  },
  {
    name: "Per Mile",
    description: "A rate for every mile the shipment travels.",
    source: `tariff "Per Mile"

/// Bills the rate for every mile of the shipment's distance.

/// The rate per mile.
input rate: Money/mi
/// The least the shipment pays.
input minimumCharge: Money = $0

charge "Per Mile" (quantity: shipment.distance, rate: rate)

minimum minimumCharge

test "bills the rate for every mile" {
  given input.rate = $2.35/mi
  given shipment.distance = 400 mi
  expect charge("Per Mile") == $940
  expect total == $940
}
`,
  },
  {
    name: "Per Stop",
    description: "A rate for every stop on the shipment.",
    source: `tariff "Per Stop"

/// Bills the rate for every stop, pickups and deliveries alike.

/// The rate per stop.
input rate: Money/stop
/// The least the shipment pays.
input minimumCharge: Money = $0

charge "Per Stop" (quantity: shipment.stops.len, rate: rate)

minimum minimumCharge

test "bills the rate for every stop" {
  given input.rate = $75/stop
  given shipment.stops = [pickup(), delivery(), delivery()]
  expect charge("Per Stop") == $225
}
`,
  },
  {
    name: "Per Pound",
    description: "A rate for every pound the shipment weighs.",
    source: `tariff "Per Pound"

/// Bills the rate for every pound of the shipment's weight.

/// The rate per pound.
input rate: Money/lb
/// The least the shipment pays.
input minimumCharge: Money = $0

require let weight = shipment.weight else "This rate needs the shipment's weight"

charge "Per Pound" (quantity: weight, rate: rate)

minimum minimumCharge

test "bills the rate for every pound" {
  given input.rate = $0.12/lb
  given shipment.weight = 2_000 lb
  expect charge("Per Pound") == $240
}
`,
  },
];

export { USER, ORG };
