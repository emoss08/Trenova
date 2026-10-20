import type { DeskFuelCard, DeskFuelPurchaseCard } from "@/lib/graphql/fuel-desk-cards";

/** Mon, Oct 19, 2026 at 6:42 AM in the test's time zone (UTC). */
export const PURCHASED_AT = Date.UTC(2026, 9, 19, 6, 42) / 1000;

export function fuelCardOnFile(overrides: Partial<DeskFuelCard> = {}): DeskFuelCard {
  return {
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
    notes: null,
    discoveredAt: null,
    version: 3,
    createdAt: Date.UTC(2024, 9, 20) / 1000,
    updatedAt: Date.UTC(2024, 9, 20) / 1000,
    assignedWorker: { id: "wrk_rosa", wholeName: "Rosa Diaz", firstName: "Rosa", lastName: "Diaz" },
    assignedTractor: { id: "tr_101", code: "T-101" },
    ...overrides,
  };
}

/** A diesel fill read from the Comdata feed: taxed in Texas, its odometer moved on from the last fill. */
export function importedPurchase(): DeskFuelPurchaseCard {
  return {
    purchase: {
      id: "fpur_1019",
      businessUnitId: "bu_1",
      organizationId: "org_1",
      tractorId: "tr_101",
      workerId: "wrk_rosa",
      jurisdictionId: "jur_tx",
      fuelCardId: "fcard_4412",
      cardLastFour: "4412",
      purchasedAt: PURCHASED_AT,
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
      importBatchId: "fpib_0700",
      taxPaid: true,
      notes: null,
      createdById: null,
      version: 1,
      createdAt: PURCHASED_AT + 1380,
      updatedAt: PURCHASED_AT + 1380,
      tractor: { id: "tr_101", code: "T-101", make: "Freightliner", model: "Cascadia" },
      worker: { id: "wrk_rosa", wholeName: "Rosa Diaz", firstName: "Rosa", lastName: "Diaz" },
      jurisdiction: { id: "jur_tx", countryCode: "US", code: "TX", name: "Texas" },
      fuelCard: fuelCardOnFile(),
    },
    ifta: {
      year: 2026,
      quarter: 4,
      countsForIfta: true,
      taxPaidGallons: "118.42",
      ratePerGallon: "0.2",
      taxPaidCredit: "23.68",
      tractorGallons: "612.84",
      tractorPurchaseCount: 5,
    },
    previousFill: {
      id: "fpur_1018",
      purchasedAt: Date.UTC(2026, 9, 18, 16, 18) / 1000,
      vendor: "Love's",
      vendorCity: "Oklahoma City",
      odometer: 180868,
    },
    provenance: {
      rowNumber: 37,
      resolutionNotes: ["Card found by its last four", "Tractor matched by unit code"],
      batch: {
        id: "fpib_0700",
        provider: "Comdata",
        origin: "Feed",
        fileName: null,
        rowCount: 112,
        createdAt: PURCHASED_AT + 1380,
      },
    },
    createdBy: null,
  };
}

/** The yard tank entered by hand: untaxed, no card, no odometer, and a total that is off from quantity × price. */
export function manualPurchase(): DeskFuelPurchaseCard {
  const card = importedPurchase();
  return {
    ...card,
    purchase: {
      ...card.purchase,
      id: "fpur_yard",
      fuelCardId: null,
      cardLastFour: null,
      fuelCard: null,
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
    },
    ifta: {
      ...card.ifta,
      taxPaidGallons: "0",
      taxPaidCredit: null,
      tractorGallons: "1204.5",
      tractorPurchaseCount: 8,
    },
    previousFill: null,
    provenance: null,
    createdBy: { id: "usr_dana", name: "Dana Park" },
  };
}
