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
};

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
};
