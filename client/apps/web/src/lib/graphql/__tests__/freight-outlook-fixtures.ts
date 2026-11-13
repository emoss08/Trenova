import type { DispatchForecastQuality } from "@/lib/graphql/dispatch-console";
import type {
  FreightArea,
  FreightMove,
  FreightOutlook,
  FreightSlot,
} from "@/lib/graphql/freight-outlook";

/** Fixtures for the freight outlook, written from dispatch_console.graphqls. */
export const OUTLOOK_START = 4_102_444_800;
export const SLOT = 4 * 3_600;

export const readyQuality: DispatchForecastQuality = {
  status: "Ready",
  historyWeeks: 35,
  historyLoads: 22_400,
  weeksTested: 8,
  loadsTested: 9_800,
  intervalCoverage: 0.85,
  meanAbsoluteError: 0.9,
  naiveMeanAbsoluteError: 1.22,
  skill: 0.26,
  fittedAt: OUTLOOK_START - 12 * 3_600,
  refitting: false,
};

export const slotStarts = Array.from({ length: 12 }, (_, index) => OUTLOOK_START + index * SLOT);

export function slots(overrides: Record<number, Partial<FreightSlot>> = {}): FreightSlot[] {
  return slotStarts.map((start, index) => ({
    start,
    likelyLow: 1,
    likelyHigh: 3,
    booked: 1,
    need: 2,
    drivers: 2,
    short: 0,
    spare: 0,
    ...overrides[index],
  }));
}

export function area(
  cell: number,
  label: string,
  slotOverrides: Record<number, Partial<FreightSlot>> = {},
  extra: Partial<FreightArea> = {},
): FreightArea {
  return {
    cell,
    label,
    latitude: 41.5,
    longitude: -81.7,
    valueCents: 104_000,
    slots: slots(slotOverrides),
    customers: [],
    ...extra,
  };
}

export const cleveland = area(
  12987,
  "Cleveland, OH",
  { 1: { short: 2.8, need: 4, drivers: 1, likelyLow: 3, likelyHigh: 6 } },
  { customers: [{ customerId: "cus_northstar", name: "Northstar Meats", share: 0.58 }] },
);

export const kansasCity = area(
  11602,
  "Kansas City, MO",
  { 0: { spare: 3.3 }, 1: { spare: 2.7 }, 2: { spare: 1.8 } },
  { valueCents: 29_000 },
);

export function move(overrides: Partial<FreightMove>): FreightMove {
  return {
    kind: "Send",
    workerId: "wrk_ana",
    workerName: "Ana Ruiz",
    fromCell: kansasCity.cell,
    fromLabel: "Kansas City, MO",
    toCell: cleveland.cell,
    toLabel: "Cleveland, OH",
    slotStart: slotStarts[1],
    short: 2.8,
    freeAt: OUTLOOK_START + 1_800,
    arriveAt: OUTLOOK_START + 4 * 3_600,
    miles: 52.4,
    emptyCostCents: 10_480,
    dropCents: 104_000,
    stayCents: 9_000,
    netCents: 84_520,
    savedCents: 0,
    moveId: null,
    proNumber: "",
    laneMoveIds: [],
    partnerId: null,
    partnerName: "",
    partnerCell: 0,
    partnerLabel: "",
    partnerFreeAt: null,
    partnerMiles: 0,
    customerId: null,
    customerName: "",
    usual: 0,
    byNow: 0,
    tendered: 0,
    loads: 0,
    ...overrides,
  };
}

export function outlook(overrides: Partial<FreightOutlook> = {}): FreightOutlook {
  return {
    quality: readyQuality,
    generatedAt: OUTLOOK_START + 1_800,
    slotStarts,
    network: slotStarts.map((start) => ({
      start,
      likelyLow: 20,
      likelyHigh: 30,
      needLow: 18,
      needHigh: 28,
      need: 25,
      drivers: 26,
      short: 0,
      uncovered: 3,
      idle: 4,
    })),
    areas: [cleveland, kansasCity],
    areaCount: 13,
    moves: [move({})],
    pace: [],
    record: [],
    positionings: [],
    ...overrides,
  };
}
