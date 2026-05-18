import { dateToUnixTimestamp } from "@/lib/date";

export function dateInputToUnix(value: string, endOfDay: boolean) {
  if (!value.trim()) return 0;

  const date = new Date(`${value}T${endOfDay ? "23:59:59" : "00:00:00"}`);
  if (Number.isNaN(date.getTime())) return 0;

  return dateToUnixTimestamp(date);
}
