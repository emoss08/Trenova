export type ShipmentAnalyticsData = {
  activeShipments: {
    count: number;
    changeFromYesterday: number;
  };
  onTimePercent: {
    percent: number;
    onTimeCount: number;
    totalCount: number;
  };
  revenueToday: {
    total: number;
    sparkline: { hour: string; value: number }[];
  };
  emptyMilePercent: {
    percent: number;
    emptyMiles: number;
    totalMiles: number;
  };
  readyToDispatch: {
    count: number;
  };
  detentionAlerts: {
    count: number;
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

export const defaultAnalyticsData: ShipmentAnalyticsData = {
  activeShipments: {
    count: 0,
    changeFromYesterday: 0,
  },
  onTimePercent: {
    percent: 0,
    onTimeCount: 0,
    totalCount: 0,
  },
  revenueToday: {
    total: 0,
    sparkline: [],
  },
  emptyMilePercent: {
    percent: 0,
    emptyMiles: 0,
    totalMiles: 0,
  },
  readyToDispatch: {
    count: 0,
  },
  detentionAlerts: {
    count: 0,
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
