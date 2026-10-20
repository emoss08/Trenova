// The fuel purchases and fuel cards the Desk cards thread names, as the
// DeskFuelPurchaseCard and DeskFuelCardCard queries answer for them. Every time
// is relative to the scenario's anchor. The Pilot fill in Amarillo came in on
// the Comdata feed and was taxed at the pump; the yard tank fill was entered by
// hand with no odometer and a total off from quantity × price; the Petro-Canada
// fill was bought in litres in Ontario. Rosa Diaz's Comdata card is active;
// the WEX card ending 7720 was found by the feed and nobody holds it yet, with
// rows from two imports waiting on it; Jamal Reyes's card was cancelled when he
// lost it and replaced by the one ending 5306.

const DAY = 86_400;
const HOUR = 3_600;

/** The first second of the anchor's month, in UTC. */
function monthStart(anchor) {
  const date = new Date(anchor * 1000);
  return Date.UTC(date.getUTCFullYear(), date.getUTCMonth(), 1) / 1000;
}

function quarterOf(anchor) {
  const date = new Date(anchor * 1000);
  return { year: date.getUTCFullYear(), quarter: Math.floor(date.getUTCMonth() / 3) + 1 };
}

const ROSA = { id: "wrk_rosa", wholeName: "Rosa Diaz", firstName: "Rosa", lastName: "Diaz" };
const MARCUS = {
  id: "wrk_marcus",
  wholeName: "Marcus Hale",
  firstName: "Marcus",
  lastName: "Hale",
};
const SAM = { id: "wrk_sam", wholeName: "Samuel Okafor", firstName: "Samuel", lastName: "Okafor" };
const JAMAL = { id: "wrk_jamal", wholeName: "Jamal Reyes", firstName: "Jamal", lastName: "Reyes" };
const TEXAS = { id: "jur_tx", countryCode: "US", code: "TX", name: "Texas" };
const OKLAHOMA = { id: "jur_ok", countryCode: "US", code: "OK", name: "Oklahoma" };
const ARKANSAS = { id: "jur_ar", countryCode: "US", code: "AR", name: "Arkansas" };
const ONTARIO = { id: "jur_on", countryCode: "CA", code: "ON", name: "Ontario" };

function card(anchor, overrides) {
  return {
    id: "fcard_4412",
    businessUnitId: "bu_mock",
    organizationId: "org_mock",
    provider: "Comdata",
    lastFour: "4412",
    label: "Comdata •••• 4412 — Unit T-101",
    externalCardId: "CMD-7781-4412",
    assignedWorkerId: ROSA.id,
    assignedTractorId: "tr_101",
    status: "Active",
    expiresAt: anchor + 1040 * DAY,
    cancelledAt: null,
    cancelReason: null,
    notes: "Primary over-the-road card; diesel and DEF only.",
    discoveredAt: null,
    version: 3,
    createdAt: anchor - 365 * DAY,
    updatedAt: anchor - 40 * DAY,
    assignedWorker: ROSA,
    assignedTractor: { id: "tr_101", code: "T-101", make: "Freightliner", model: "Cascadia" },
    ...overrides,
  };
}

/** A card as a purchase or another card names it, with only its tractor's code. */
function shortCard(full) {
  return {
    ...full,
    assignedTractor: full.assignedTractor
      ? { id: full.assignedTractor.id, code: full.assignedTractor.code }
      : null,
  };
}

function purchase(anchor, overrides) {
  const purchasedAt = overrides.purchasedAt ?? anchor - 13 * HOUR;
  return {
    id: "fpur_pilot",
    businessUnitId: "bu_mock",
    organizationId: "org_mock",
    tractorId: "tr_101",
    workerId: ROSA.id,
    jurisdictionId: TEXAS.id,
    fuelCardId: "fcard_4412",
    cardLastFour: "4412",
    purchasedAt,
    vendor: "Pilot Travel Center",
    vendorCity: "Amarillo",
    fuelType: "Diesel",
    quantity: "118.42",
    quantityUnit: "Gallon",
    gallons: "118.42",
    unitPrice: "3.899",
    totalAmount: "461.72",
    currencyCode: "USD",
    odometer: 181604,
    transactionReference: "TX-20261019-004812",
    source: "CardImport",
    importBatchId: "fpib_comdata_0700",
    taxPaid: true,
    notes: null,
    createdById: null,
    version: 1,
    createdAt: purchasedAt + 23 * 60,
    updatedAt: purchasedAt + 23 * 60,
    tractor: { id: "tr_101", code: "T-101", make: "Freightliner", model: "Cascadia" },
    worker: ROSA,
    jurisdiction: TEXAS,
    fuelCard: shortCard(card(anchor, {})),
    ...overrides,
  };
}

/** Each fuel purchase the cards thread names, by id. */
export function deskFuelPurchases(anchor) {
  const { year, quarter } = quarterOf(anchor);
  const pilot = purchase(anchor, {});
  const yard = purchase(anchor, {
    id: "fpur_yard",
    tractorId: "tr_205",
    workerId: MARCUS.id,
    fuelCardId: null,
    cardLastFour: null,
    purchasedAt: anchor - 2 * DAY - 5 * HOUR,
    vendor: "Yard bulk tank",
    vendorCity: "Houston",
    quantity: "160",
    gallons: "160",
    unitPrice: "3.489",
    totalAmount: "562.54",
    odometer: null,
    transactionReference: "YARD-HOU-1017",
    source: "Manual",
    importBatchId: null,
    taxPaid: false,
    createdById: "usr_dana",
    tractor: { id: "tr_205", code: "T-205", make: "Kenworth", model: "T680" },
    worker: MARCUS,
    fuelCard: null,
  });
  const petro = purchase(anchor, {
    id: "fpur_petro",
    tractorId: "tr_118",
    workerId: SAM.id,
    jurisdictionId: ONTARIO.id,
    fuelCardId: "fcard_0931",
    cardLastFour: "0931",
    purchasedAt: anchor - DAY - 2 * HOUR,
    vendor: "Petro-Canada",
    vendorCity: "Windsor",
    quantity: "420",
    quantityUnit: "Litre",
    gallons: "110.954",
    unitPrice: "1.559",
    totalAmount: "654.78",
    currencyCode: "CAD",
    odometer: 302447,
    transactionReference: "ON-1018-77213",
    importBatchId: "fpib_wex_upload",
    tractor: { id: "tr_118", code: "T-118", make: "Volvo", model: "VNL 760" },
    worker: SAM,
    jurisdiction: ONTARIO,
    fuelCard: shortCard(
      card(anchor, {
        id: "fcard_0931",
        provider: "WEX",
        lastFour: "0931",
        label: "WEX •••• 0931 — Unit T-118",
        externalCardId: null,
        assignedWorkerId: null,
        assignedWorker: null,
        assignedTractorId: "tr_118",
        assignedTractor: { id: "tr_118", code: "T-118", make: "Volvo", model: "VNL 760" },
      }),
    ),
  });

  return {
    fpur_pilot: {
      purchase: pilot,
      ifta: {
        year,
        quarter,
        countsForIfta: true,
        taxPaidGallons: "118.42",
        ratePerGallon: "0.2",
        taxPaidCredit: "23.68",
        tractorGallons: "612.84",
        tractorPurchaseCount: 5,
      },
      previousFill: {
        id: "fpur_loves",
        purchasedAt: pilot.purchasedAt - 14 * HOUR - 24 * 60,
        vendor: "Love's",
        vendorCity: "Oklahoma City",
        odometer: 180868,
      },
      provenance: {
        rowNumber: 37,
        resolutionNotes: [
          "Card found by its last four",
          "Tractor matched by unit code T101",
          "Driver taken from the card",
        ],
        batch: {
          id: "fpib_comdata_0700",
          provider: "Comdata",
          origin: "Feed",
          fileName: null,
          rowCount: 112,
          createdAt: pilot.createdAt,
        },
      },
      createdBy: null,
    },
    fpur_yard: {
      purchase: yard,
      ifta: {
        year,
        quarter,
        countsForIfta: true,
        taxPaidGallons: "0",
        ratePerGallon: "0.2",
        taxPaidCredit: null,
        tractorGallons: "1204.5",
        tractorPurchaseCount: 8,
      },
      previousFill: null,
      provenance: null,
      createdBy: { id: "usr_dana", name: "Dana Park" },
    },
    fpur_petro: {
      purchase: petro,
      ifta: {
        year,
        quarter,
        countsForIfta: true,
        taxPaidGallons: "110.954",
        ratePerGallon: "0.1",
        taxPaidCredit: "11.10",
        tractorGallons: "331.27",
        tractorPurchaseCount: 3,
      },
      previousFill: {
        id: "fpur_flyingj",
        purchasedAt: petro.purchasedAt - 9 * HOUR,
        vendor: "Flying J",
        vendorCity: "London",
        odometer: 301836,
      },
      provenance: {
        rowNumber: 4,
        resolutionNotes: ["Card found by its last four", "Litres converted to US gallons"],
        batch: {
          id: "fpib_wex_upload",
          provider: "WEX",
          origin: "Upload",
          fileName: "wex-statement-week-42.csv",
          rowCount: 38,
          createdAt: petro.createdAt,
        },
      },
      createdBy: null,
    },
  };
}

function recent(anchor, id, hoursAgo, overrides) {
  return {
    id,
    purchasedAt: anchor - hoursAgo * HOUR,
    vendor: "Pilot Travel Center",
    vendorCity: "Amarillo",
    fuelType: "Diesel",
    gallons: "118.42",
    totalAmount: "461.72",
    currencyCode: "USD",
    taxPaid: true,
    jurisdiction: { id: TEXAS.id, code: TEXAS.code },
    ...overrides,
  };
}

function jurisdictionRow(jurisdiction, gallons, purchaseCount) {
  return {
    jurisdictionId: jurisdiction.id,
    jurisdiction: {
      id: jurisdiction.id,
      code: jurisdiction.code,
      name: jurisdiction.name,
      countryCode: jurisdiction.countryCode,
    },
    gallons,
    purchaseCount,
  };
}

function heldRow(anchor, id, hoursAgo, overrides) {
  return {
    id,
    rowNumber: 12,
    status: "Error",
    error: "No tractor could be determined for card ending 7720",
    importBatchId: "fpib_wex_feed",
    resolutionNotes: [],
    ...overrides,
    parsed: {
      purchasedAt: anchor - hoursAgo * HOUR,
      vendor: "Love's Travel Stop",
      vendorCity: "Joplin",
      jurisdictionCode: "MO",
      fuelType: "Diesel",
      gallons: "132.1",
      totalAmount: "498.73",
      currencyCode: "USD",
      tractorCode: null,
      ...overrides.parsed,
    },
  };
}

/** Each fuel card the cards thread names, by id. */
export function deskFuelCards(anchor) {
  const { year, quarter } = quarterOf(anchor);
  const feed = {
    lastPolledAt: anchor - 18 * 60,
    lastSuccessAt: anchor - 18 * 60,
    failureCount: 0,
    lastError: null,
  };
  const active = {
    card: card(anchor, {}),
    monthStart: monthStart(anchor),
    year,
    quarter,
    month: [
      {
        currencyCode: "USD",
        purchaseCount: 9,
        gallons: "1046.3",
        untaxedGallons: "15",
        totalAmount: "4092.18",
      },
    ],
    quarterByJurisdiction: [
      jurisdictionRow(TEXAS, "612.84", 5),
      jurisdictionRow(OKLAHOMA, "301.18", 3),
      jurisdictionRow(ARKANSAS, "132.31", 1),
    ],
    purchaseCount: 23,
    lastPurchasedAt: anchor - 13 * HOUR + 9 * 60,
    recent: [
      recent(anchor, "fpur_pilot", 13, {}),
      recent(anchor, "fpur_pilot_def", 13 - 0.15, {
        fuelType: "DEF",
        gallons: "15",
        totalAmount: "49.49",
        taxPaid: false,
      }),
      recent(anchor, "fpur_loves", 27, {
        vendor: "Love's Travel Stop",
        vendorCity: "Oklahoma City",
        gallons: "126",
        totalAmount: "473.63",
        jurisdiction: { id: OKLAHOMA.id, code: OKLAHOMA.code },
      }),
      recent(anchor, "fpur_flyingj_dallas", 48, {
        vendor: "Flying J",
        vendorCity: "Dallas",
        gallons: "131.6",
        totalAmount: "480.21",
      }),
      recent(anchor, "fpur_loves_tulsa", 82, {
        vendor: "Love's Travel Stop",
        vendorCity: "Tulsa",
        gallons: "120.8",
        totalAmount: "450.46",
        jurisdiction: { id: OKLAHOMA.id, code: OKLAHOMA.code },
      }),
    ],
    held: [],
    heldCount: 0,
    heldSpend: [],
    heldBatches: [],
    feed,
    successors: [],
  };

  const discovered = {
    ...active,
    card: card(anchor, {
      id: "fcard_7720",
      provider: "WEX",
      lastFour: "7720",
      label: "",
      externalCardId: null,
      assignedWorkerId: null,
      assignedTractorId: null,
      assignedWorker: null,
      assignedTractor: null,
      status: "Suspended",
      expiresAt: null,
      notes: null,
      discoveredAt: anchor - 2 * DAY - HOUR,
      version: 1,
      createdAt: anchor - 2 * DAY - HOUR,
    }),
    month: [],
    quarterByJurisdiction: [],
    purchaseCount: 0,
    lastPurchasedAt: null,
    recent: [],
    held: [
      heldRow(anchor, "fpir_joplin", 10, {}),
      heldRow(anchor, "fpir_tulsa", 20, {
        rowNumber: 5,
        status: "New",
        error: null,
        importBatchId: "fpib_wex_upload",
        resolutionNotes: ["Tractor matched by license plate TX-55120"],
        parsed: {
          vendor: "TA",
          vendorCity: "Tulsa",
          jurisdictionCode: "OK",
          gallons: "104.7",
          totalAmount: "402.05",
          tractorCode: "T-130",
        },
      }),
      heldRow(anchor, "fpir_texarkana", 39, {
        rowNumber: 31,
        parsed: {
          vendor: "Pilot Travel Center",
          vendorCity: "Texarkana",
          jurisdictionCode: "AR",
          gallons: "98.4",
          totalAmount: "377.33",
        },
      }),
      heldRow(anchor, "fpir_fortworth", 51, {
        rowNumber: 8,
        parsed: {
          vendor: "Flying J",
          vendorCity: "Fort Worth",
          jurisdictionCode: "TX",
          gallons: "83.4",
          totalAmount: "334.34",
        },
      }),
    ],
    heldCount: 4,
    heldSpend: [{ currencyCode: "USD", gallons: "418.6", totalAmount: "1612.45" }],
    heldBatches: [
      { id: "fpib_wex_feed", version: 4, provider: "WEX", origin: "Feed" },
      { id: "fpib_wex_upload", version: 2, provider: "WEX", origin: "Upload" },
    ],
  };

  const replacement = card(anchor, {
    id: "fcard_5306",
    lastFour: "5306",
    label: "Comdata •••• 5306 — Unit T-087",
    externalCardId: "CMD-7781-5306",
    assignedWorkerId: JAMAL.id,
    assignedWorker: JAMAL,
    assignedTractorId: "tr_087",
    assignedTractor: { id: "tr_087", code: "T-087", make: "Kenworth", model: "T680" },
    notes: null,
    createdAt: anchor - 17 * DAY,
  });
  const cancelled = {
    ...active,
    card: card(anchor, {
      id: "fcard_2290",
      lastFour: "2290",
      label: "Comdata •••• 2290 — Unit T-087",
      externalCardId: "CMD-7781-2290",
      assignedWorkerId: JAMAL.id,
      assignedWorker: JAMAL,
      assignedTractorId: "tr_087",
      assignedTractor: { id: "tr_087", code: "T-087", make: "Kenworth", model: "T680" },
      status: "Cancelled",
      cancelledAt: anchor - 17 * DAY,
      cancelReason:
        "Reported lost by the driver at Love's, Oklahoma City. Replacement is Comdata •••• 5306.",
      expiresAt: anchor + 520 * DAY,
      notes: null,
    }),
    month: [],
    quarterByJurisdiction: [
      jurisdictionRow(OKLAHOMA, "141.2", 1),
      jurisdictionRow(TEXAS, "122.9", 1),
    ],
    purchaseCount: 2,
    lastPurchasedAt: anchor - 18 * DAY,
    recent: [
      recent(anchor, "fpur_loves_okc", 18 * 24 + 4, {
        vendor: "Love's Travel Stop",
        vendorCity: "Oklahoma City",
        gallons: "141.2",
        totalAmount: "539.39",
        jurisdiction: { id: OKLAHOMA.id, code: OKLAHOMA.code },
      }),
      recent(anchor, "fpur_pilot_ama", 19 * 24 + 2, { gallons: "122.9", totalAmount: "478.98" }),
    ],
    successors: [shortCard(replacement)],
  };

  return {
    fcard_4412: active,
    fcard_7720: discovered,
    fcard_2290: cancelled,
    fcard_5306: {
      ...active,
      card: replacement,
      purchaseCount: 4,
      recent: active.recent.slice(0, 4),
    },
  };
}

/** The cards thread's fuel purchase conversations. */
export const DESK_FUEL_PURCHASE_CONVERSATIONS = {
  "fuel-pilot": {
    recordId: "fpur_pilot",
    title: "Fuel purchase at Pilot Travel Center",
    question: "Did Rosa's Amarillo fill this morning come through?",
    reply:
      "Yes. The Comdata feed brought it in at 7:05: 118.42 gallons of diesel, $461.72, taxed in Texas. [Pilot Travel Center](artifact:art_fuel-pilot) has the receipt and what it does to IFTA.",
  },
  "fuel-yard": {
    recordId: "fpur_yard",
    title: "Fuel purchase at the yard bulk tank",
    question: "Anything off with the fuel entered this week?",
    reply:
      "One: Dana's yard tank fill for T-205 has no odometer, and its total is $4.30 more than quantity × price. [Yard bulk tank](artifact:art_fuel-yard) shows both.",
  },
  "fuel-petro": {
    recordId: "fpur_petro",
    title: "Fuel purchase at Petro-Canada",
    question: "How was Samuel's Windsor fill counted?",
    reply:
      "In US gallons: 420 litres became 110.954 gallons, taxed in Ontario. [Petro-Canada](artifact:art_fuel-petro) has the conversion.",
  },
  "fuel-gone": {
    recordId: "fpur_gone",
    title: "Fuel purchase",
    question: "Pull up the Love's fill from last week.",
    reply:
      "That purchase has been deleted since I looked it up. [Fuel purchase](artifact:art_fuel-gone) is what I found then.",
  },
};

/** The cards thread's fuel card conversations. */
export const DESK_FUEL_CARD_CONVERSATIONS = {
  "fuelcard-active": {
    recordId: "fcard_4412",
    title: "Comdata •••• 4412 — Unit T-101",
    question: "How much has Rosa's card spent this month?",
    reply:
      "$4,092.18 over nine purchases, most of it in Texas and Oklahoma. [Comdata •••• 4412](artifact:art_fuelcard-active) has the purchases and where it fuels.",
  },
  "fuelcard-found": {
    recordId: "fcard_7720",
    title: "WEX •••• 7720",
    question: "Why are some WEX purchases not posting?",
    reply:
      "They are on a card the feed hadn't seen before, ending 7720, and nobody holds it yet. Four purchases, $1,612.45, are waiting on it. [WEX •••• 7720](artifact:art_fuelcard-found) can be assigned from there.",
  },
  "fuelcard-cancelled": {
    recordId: "fcard_2290",
    title: "Comdata •••• 2290 — Unit T-087",
    question: "Is Jamal's old card still live?",
    reply:
      "No. It was cancelled when he reported it lost, and nothing has been charged to it since. [Comdata •••• 2290](artifact:art_fuelcard-cancelled) links to its replacement.",
  },
  "fuelcard-gone": {
    recordId: "fcard_0001",
    title: "Fuel card",
    question: "Pull up the old EFS card.",
    reply:
      "That card has been removed since I looked it up. [Fuel card](artifact:art_fuelcard-gone) is what I found then.",
  },
};

/** The receipts attached to each fuel purchase, as the documents route lists them. */
export const DESK_FUEL_PURCHASE_DOCUMENTS = {
  fpur_pilot: [["Pilot_Amarillo_receipt.jpg", "image/jpeg", 212_400]],
  fpur_yard: [],
  fpur_petro: [["Petro-Canada_Windsor.pdf", "application/pdf", 88_300]],
};
