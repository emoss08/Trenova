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
