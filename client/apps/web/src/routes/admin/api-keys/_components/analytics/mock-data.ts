export type ApiKeyAnalyticsData = {
  totalKeys: {
    count: number;
    newThisMonth: number;
  };
  activeKeys: {
    count: number;
    percentOfTotal: number;
  };
  revokedKeys: {
    count: number;
    percentOfTotal: number;
  };
  requests30d: {
    total: number;
    sparkline: { day: string; value: number }[];
  };
};

export const defaultApiKeyAnalyticsData: ApiKeyAnalyticsData = {
  totalKeys: {
    count: 0,
    newThisMonth: 0,
  },
  activeKeys: {
    count: 0,
    percentOfTotal: 0,
  },
  revokedKeys: {
    count: 0,
    percentOfTotal: 0,
  },
  requests30d: {
    total: 0,
    sparkline: [],
  },
};

export const mockApiKeyAnalyticsData: ApiKeyAnalyticsData = {
  totalKeys: {
    count: 24,
    newThisMonth: 3,
  },
  activeKeys: {
    count: 18,
    percentOfTotal: 75,
  },
  revokedKeys: {
    count: 6,
    percentOfTotal: 25,
  },
  requests30d: {
    total: 48_320,
    sparkline: [
      { day: "Mar 1", value: 1200 },
      { day: "Mar 3", value: 1450 },
      { day: "Mar 5", value: 980 },
      { day: "Mar 7", value: 1680 },
      { day: "Mar 9", value: 2100 },
      { day: "Mar 11", value: 1900 },
      { day: "Mar 13", value: 1750 },
      { day: "Mar 15", value: 2300 },
      { day: "Mar 17", value: 1400 },
      { day: "Mar 19", value: 1620 },
      { day: "Mar 21", value: 1850 },
      { day: "Mar 23", value: 2050 },
      { day: "Mar 25", value: 1700 },
      { day: "Mar 27", value: 1950 },
      { day: "Mar 29", value: 2200 },
    ],
  },
};
