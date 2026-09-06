const DAY_SECONDS = 86_400;

export type TenureParts = {
  years: number;
  months: number;
  days: number;
  totalDays: number;
};

const EMPTY: TenureParts = { years: 0, months: 0, days: 0, totalDays: 0 };

/**
 * Whole years, months and days of service between the hire date and either the
 * termination date or now, in UTC calendar terms. A termination in the future
 * is ignored, matching the server's Tenure helper.
 */
export function tenureParts(
  hireDate: number | null | undefined,
  terminationDate: number | null | undefined,
  now: number,
): TenureParts {
  if (!hireDate || hireDate <= 0) return EMPTY;
  let end = now;
  if (terminationDate && terminationDate > 0 && terminationDate < now) {
    end = terminationDate;
  }
  if (end <= hireDate) return EMPTY;

  const start = new Date(hireDate * 1000);
  const stop = new Date(end * 1000);
  let years = stop.getUTCFullYear() - start.getUTCFullYear();
  let months = stop.getUTCMonth() - start.getUTCMonth();
  let days = stop.getUTCDate() - start.getUTCDate();
  if (days < 0) {
    months -= 1;
    const previousMonthEnd = new Date(Date.UTC(stop.getUTCFullYear(), stop.getUTCMonth(), 0));
    days += previousMonthEnd.getUTCDate();
  }
  if (months < 0) {
    years -= 1;
    months += 12;
  }
  return {
    years,
    months,
    days,
    totalDays: Math.floor((end - hireDate) / DAY_SECONDS),
  };
}

/** Compact tenure: "2y 3m", "5m", "13d", "Today", or a dash without a hire date. */
export function formatTenure(
  hireDate: number | null | undefined,
  terminationDate: number | null | undefined,
  now: number,
): string {
  if (!hireDate || hireDate <= 0) return "—";
  const parts = tenureParts(hireDate, terminationDate, now);
  if (parts.years > 0) {
    return parts.months > 0 ? `${parts.years}y ${parts.months}m` : `${parts.years}y`;
  }
  if (parts.months > 0) return `${parts.months}m`;
  if (parts.days > 0) return `${parts.days}d`;
  return "Today";
}
