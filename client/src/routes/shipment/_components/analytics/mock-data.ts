export type Tone = "success" | "danger" | "warning" | "brand" | "info" | "muted";

export type DeepPartial<T> = {
  [K in keyof T]?: T[K] extends Array<unknown>
    ? T[K]
    : T[K] extends object
      ? DeepPartial<T[K]>
      : T[K];
};

export type ShipmentAnalyticsData = {
  revenueToday: {
    total: number;
    sparkline: { hour: string; value: number }[];
    deltaPct: number;
    rpm: number;
    marginPct: number;
  };
  activeShipments: {
    count: number;
    changeFromYesterday: number;
    sparkline: { hour: string; value: number }[];
    breakdown: {
      inTransit: number;
      atRisk: number;
      loading: number;
      done: number;
    };
  };
  onTimePercent: {
    percent: number;
    onTimeCount: number;
    totalCount: number;
    target: number;
    deltaPp: number;
    sevenDayPercent: number;
  };
  emptyMilePercent: {
    percent: number;
    emptyMiles: number;
    totalMiles: number;
    target: number;
    deltaPp: number;
  };
  tenderAccept: {
    percent: number;
    accepted: number;
    declined: number;
    target: number;
    deltaPp: number;
  };
  atRisk: {
    count: number;
    delta: number;
    etaSlip: number;
    weather: number;
    reefer: number;
  };
  unassigned: {
    count: number;
    delta: number;
    revenueWaiting: number;
  };
  readyToDispatch: {
    count: number;
    delta: number;
    unassigned: number;
    driverReady: number;
  };
  hosNearLimit: {
    items: { driverId: string; name: string; hoursLeftLabel: string; tone: Tone }[];
  };
  detentionWatchlist: {
    items: { shipmentId: string; customer: string; dwellLabel: string; tone: Tone }[];
  };
  customerMix: {
    windowDays: number;
    entries: {
      customerId: string;
      name: string;
      revenue: number;
      share: number;
      loads: number;
      trend: number;
    }[];
  };
  tomorrowsPickups: {
    date: string;
    pickups: {
      shipmentId: string;
      proNumber: string;
      pickupWindowStart: number;
      customer: string;
      origin: string;
      destination: string;
      driver: string;
      status: "scheduled" | "confirmed" | "tentative" | "unassigned";
    }[];
  };
  laneHeatmap: {
    windowDays: number;
    cells: {
      origin: Region;
      destination: Region;
      count: number;
    }[];
    total: number;
  };
};

export type Region = "West" | "Midwest" | "South" | "Northeast";

const REVENUE_SERIES = [
  1.2, 1.8, 3.1, 4.4, 5.6, 6.8, 8.1, 9.5, 10.9, 12.4, 14.1, 15.8, 17.4, 19.0, 20.8, 22.7, 24.6,
  26.5, 28.4, 30.2, 31.9, 33.5, 35.0, 36.4,
];
const ACTIVE_SERIES = [
  38, 42, 40, 45, 48, 52, 49, 54, 58, 55, 60, 62, 58, 55, 52, 50, 48, 45, 47, 49, 51, 53, 55, 58,
];

function asSparkline(values: number[]): { hour: string; value: number }[] {
  return values.map((value, index) => ({
    hour: `${String(index).padStart(2, "0")}:00`,
    value,
  }));
}

export const defaultAnalyticsData: ShipmentAnalyticsData = {
  revenueToday: {
    total: 36400,
    sparkline: asSparkline(REVENUE_SERIES),
    deltaPct: 8.1,
    rpm: 2.18,
    marginPct: 22.4,
  },
  activeShipments: {
    count: 142,
    changeFromYesterday: 5,
    sparkline: asSparkline(ACTIVE_SERIES),
    breakdown: {
      inTransit: 58,
      atRisk: 9,
      loading: 6,
      done: 64,
    },
  },
  onTimePercent: {
    percent: 94.2,
    onTimeCount: 0,
    totalCount: 0,
    target: 96,
    deltaPp: -1.2,
    sevenDayPercent: 95.4,
  },
  emptyMilePercent: {
    percent: 11.8,
    emptyMiles: 1840,
    totalMiles: 0,
    target: 10,
    deltaPp: -0.4,
  },
  tenderAccept: {
    percent: 94.1,
    accepted: 23,
    declined: 1,
    target: 95,
    deltaPp: 0.4,
  },
  atRisk: {
    count: 9,
    delta: 1,
    etaSlip: 4,
    weather: 3,
    reefer: 2,
  },
  unassigned: {
    count: 5,
    delta: -2,
    revenueWaiting: 8650,
  },
  readyToDispatch: {
    count: 12,
    delta: 3,
    unassigned: 5,
    driverReady: 7,
  },
  hosNearLimit: {
    items: [
      { driverId: "D-211", name: "J. Park", hoursLeftLabel: "02:15 left", tone: "danger" },
      {
        driverId: "D-176",
        name: "K. Whitehorse",
        hoursLeftLabel: "04:30 left",
        tone: "warning",
      },
      { driverId: "D-189", name: "L. Mendez", hoursLeftLabel: "07:48 left", tone: "muted" },
    ],
  },
  detentionWatchlist: {
    items: [
      {
        shipmentId: "SHP-1040",
        customer: "GlobalTrade",
        dwellLabel: "3h 38m",
        tone: "danger",
      },
      {
        shipmentId: "SHP-1041",
        customer: "FreshHaul",
        dwellLabel: "2h 22m",
        tone: "warning",
      },
      { shipmentId: "SHP-1037", customer: "Peak", dwellLabel: "2h 04m", tone: "warning" },
    ],
  },
  customerMix: {
    windowDays: 30,
    entries: [],
  },
  tomorrowsPickups: {
    date: "",
    pickups: [],
  },
  laneHeatmap: {
    windowDays: 7,
    cells: [],
    total: 0,
  },
};

export function mergeShipmentAnalyticsWithDefaults(
  data: DeepPartial<ShipmentAnalyticsData>,
): ShipmentAnalyticsData {
  return {
    revenueToday: { ...defaultAnalyticsData.revenueToday, ...data.revenueToday },
    activeShipments: {
      ...defaultAnalyticsData.activeShipments,
      ...data.activeShipments,
      breakdown: {
        ...defaultAnalyticsData.activeShipments.breakdown,
        ...data.activeShipments?.breakdown,
      },
      sparkline:
        (data.activeShipments?.sparkline as
          | ShipmentAnalyticsData["activeShipments"]["sparkline"]
          | undefined) ?? defaultAnalyticsData.activeShipments.sparkline,
    },
    onTimePercent: { ...defaultAnalyticsData.onTimePercent, ...data.onTimePercent },
    emptyMilePercent: { ...defaultAnalyticsData.emptyMilePercent, ...data.emptyMilePercent },
    tenderAccept: { ...defaultAnalyticsData.tenderAccept, ...data.tenderAccept },
    atRisk: { ...defaultAnalyticsData.atRisk, ...data.atRisk },
    unassigned: { ...defaultAnalyticsData.unassigned, ...data.unassigned },
    readyToDispatch: { ...defaultAnalyticsData.readyToDispatch, ...data.readyToDispatch },
    hosNearLimit:
      (data.hosNearLimit as ShipmentAnalyticsData["hosNearLimit"] | undefined) ??
      defaultAnalyticsData.hosNearLimit,
    detentionWatchlist:
      (data.detentionWatchlist as ShipmentAnalyticsData["detentionWatchlist"] | undefined) ??
      defaultAnalyticsData.detentionWatchlist,
    customerMix:
      (data.customerMix as ShipmentAnalyticsData["customerMix"] | undefined) ??
      defaultAnalyticsData.customerMix,
    tomorrowsPickups:
      (data.tomorrowsPickups as ShipmentAnalyticsData["tomorrowsPickups"] | undefined) ??
      defaultAnalyticsData.tomorrowsPickups,
    laneHeatmap:
      (data.laneHeatmap as ShipmentAnalyticsData["laneHeatmap"] | undefined) ??
      defaultAnalyticsData.laneHeatmap,
  };
}
