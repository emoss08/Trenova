import type {
  DeskFuelCardCard,
  DeskFuelCardPurchase,
  DeskFuelHeldRow,
} from "@/lib/graphql/fuel-desk-cards";

export const MONTH_START = Date.UTC(2026, 9, 1) / 1000;
export const LAST_POLL = Date.UTC(2026, 9, 20, 7) / 1000;

function purchase(
  id: string,
  purchasedAt: number,
  overrides: Partial<DeskFuelCardPurchase> = {},
): DeskFuelCardPurchase {
  return {
    id,
    purchasedAt,
    vendor: "Pilot Travel Center",
    vendorCity: "Amarillo",
    fuelType: "Diesel",
    gallons: "118.42",
    totalAmount: "461.72",
    currencyCode: "USD",
    taxPaid: true,
    jurisdiction: { id: "jur_tx", code: "TX" },
    ...overrides,
  };
}

/** Rosa Diaz's Comdata card on T-101: active, nine purchases this month, three jurisdictions this quarter. */
export function activeCard(): DeskFuelCardCard {
  return {
    card: {
      id: "fcard_4412",
      businessUnitId: "bu_1",
      organizationId: "org_1",
      provider: "Comdata",
      lastFour: "4412",
      label: "Comdata •••• 4412 — Unit T-101",
      externalCardId: "CMD-7781-4412",
      assignedWorkerId: "wrk_rosa",
      assignedTractorId: "tr_101",
      status: "Active",
      expiresAt: Date.UTC(2029, 7, 31) / 1000,
      cancelledAt: null,
      cancelReason: null,
      notes: "Primary over-the-road card; diesel and DEF only.",
      discoveredAt: null,
      version: 3,
      createdAt: Date.UTC(2024, 9, 20) / 1000,
      updatedAt: Date.UTC(2024, 9, 20) / 1000,
      assignedWorker: {
        id: "wrk_rosa",
        wholeName: "Rosa Diaz",
        firstName: "Rosa",
        lastName: "Diaz",
      },
      assignedTractor: { id: "tr_101", code: "T-101", make: "Freightliner", model: "Cascadia" },
    },
    monthStart: MONTH_START,
    year: 2026,
    quarter: 4,
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
      {
        jurisdictionId: "jur_tx",
        jurisdiction: { id: "jur_tx", code: "TX", name: "Texas", countryCode: "US" },
        gallons: "612.84",
        purchaseCount: 5,
      },
      {
        jurisdictionId: "jur_ok",
        jurisdiction: { id: "jur_ok", code: "OK", name: "Oklahoma", countryCode: "US" },
        gallons: "306.42",
        purchaseCount: 3,
      },
    ],
    purchaseCount: 23,
    lastPurchasedAt: Date.UTC(2026, 9, 19, 6, 51) / 1000,
    recent: [
      purchase("fpur_1019a", Date.UTC(2026, 9, 19, 6, 42) / 1000),
      purchase("fpur_1019b", Date.UTC(2026, 9, 19, 6, 51) / 1000, {
        fuelType: "DEF",
        gallons: "15",
        totalAmount: "49.49",
        taxPaid: false,
      }),
      purchase("fpur_1018", Date.UTC(2026, 9, 18, 16, 18) / 1000, {
        vendor: "Love's Travel Stop",
        vendorCity: "Oklahoma City",
        jurisdiction: { id: "jur_ok", code: "OK" },
        taxPaid: false,
      }),
    ],
    held: [],
    heldCount: 0,
    heldSpend: [],
    heldBatches: [],
    feed: { lastPolledAt: LAST_POLL, lastSuccessAt: LAST_POLL, failureCount: 0, lastError: null },
    successors: [],
  };
}

function heldRow(id: string, overrides: Partial<DeskFuelHeldRow> = {}): DeskFuelHeldRow {
  return {
    id,
    rowNumber: 12,
    status: "Error",
    error: "No tractor could be determined for card ending 7720",
    importBatchId: "fpib_feed",
    resolutionNotes: [],
    parsed: {
      purchasedAt: Date.UTC(2026, 9, 19, 20, 55) / 1000,
      vendor: "Love's Travel Stop",
      vendorCity: "Joplin",
      jurisdictionCode: "MO",
      fuelType: "Diesel",
      gallons: "132.1",
      totalAmount: "498.73",
      currencyCode: "USD",
      tractorCode: null,
    },
    ...overrides,
  };
}

/** A WEX card the feed found two days ago: nobody holds it, it is suspended, and twelve rows wait on it in two imports. */
export function discoveredCard(): DeskFuelCardCard {
  const card = activeCard();
  return {
    ...card,
    card: {
      ...card.card,
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
      discoveredAt: Date.UTC(2026, 9, 18, 7) / 1000,
      version: 1,
    },
    month: [],
    quarterByJurisdiction: [],
    purchaseCount: 0,
    lastPurchasedAt: null,
    recent: [],
    held: [
      heldRow("fpir_1"),
      heldRow("fpir_2", {
        status: "New",
        error: null,
        importBatchId: "fpib_upload",
        parsed: {
          purchasedAt: Date.UTC(2026, 9, 19, 11, 20) / 1000,
          vendor: "TA",
          vendorCity: "Tulsa",
          jurisdictionCode: "OK",
          fuelType: "Diesel",
          gallons: "104.7",
          totalAmount: "402.05",
          currencyCode: "USD",
          tractorCode: "T-130",
        },
      }),
    ],
    heldCount: 12,
    heldSpend: [{ currencyCode: "USD", gallons: "418.6", totalAmount: "1612.45" }],
    heldBatches: [
      { id: "fpib_feed", version: 4, provider: "WEX", origin: "Feed" },
      { id: "fpib_upload", version: 2, provider: "WEX", origin: "Upload" },
    ],
    feed: { lastPolledAt: LAST_POLL, lastSuccessAt: LAST_POLL, failureCount: 0, lastError: null },
  };
}

/** Jamal Reyes's lost card, cancelled on Oct 3 and replaced. */
export function cancelledCard(): DeskFuelCardCard {
  const card = activeCard();
  return {
    ...card,
    card: {
      ...card.card,
      id: "fcard_2290",
      lastFour: "2290",
      label: "Comdata •••• 2290 — Unit T-087",
      status: "Cancelled",
      cancelledAt: Date.UTC(2026, 9, 3, 14, 14) / 1000,
      cancelReason: "Reported lost by the driver at Love's, Oklahoma City.",
      assignedWorker: {
        id: "wrk_jamal",
        wholeName: "Jamal Reyes",
        firstName: "Jamal",
        lastName: "Reyes",
      },
      assignedWorkerId: "wrk_jamal",
      assignedTractor: { id: "tr_087", code: "T-087", make: "Kenworth", model: "T680" },
      assignedTractorId: "tr_087",
    },
    month: [],
    purchaseCount: 2,
    successors: [
      {
        ...card.card,
        id: "fcard_5306",
        lastFour: "5306",
        label: "Comdata •••• 5306 — Unit T-087",
        assignedTractor: { id: "tr_087", code: "T-087" },
      },
    ],
  };
}
