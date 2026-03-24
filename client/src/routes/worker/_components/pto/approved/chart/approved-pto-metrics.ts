import { format, parseISO } from "date-fns";
import type { PTOChartDataPoint } from "@/types/worker";

export type PeakOccupancy = {
  dateLabel: string | null;
  occupancy: number;
};

export type ApprovedPTOMetrics = {
  approvedPtoDays: number;
  workersWithApprovedPTO: number;
  peakDay: PeakOccupancy;
};

function getDailyOccupancy(day: PTOChartDataPoint): number {
  return (
    day.vacation +
    day.sick +
    day.holiday +
    day.bereavement +
    day.maternity +
    day.paternity +
    day.personal
  );
}

function formatPeakDate(date: string): string {
  return format(parseISO(date), "MMM dd");
}

export function buildApprovedPTOMetrics(
  chartData: PTOChartDataPoint[],
): ApprovedPTOMetrics {
  const workerIDs = new Set<string>();

  let approvedPtoDays = 0;
  let peakDayOccupancy = 0;
  let peakDayDate: string | null = null;

  for (const day of chartData) {
    approvedPtoDays += getDailyOccupancy(day);

    for (const workers of Object.values(day.workers ?? {})) {
      for (const worker of workers) {
        workerIDs.add(worker.id);
      }
    }

    const currentDayOccupancy = getDailyOccupancy(day);
    if (currentDayOccupancy > peakDayOccupancy) {
      peakDayOccupancy = currentDayOccupancy;
      peakDayDate = day.date;
    }
  }

  return {
    approvedPtoDays,
    workersWithApprovedPTO: workerIDs.size,
    peakDay: {
      dateLabel: peakDayDate ? formatPeakDate(peakDayDate) : null,
      occupancy: peakDayOccupancy,
    },
  };
}
