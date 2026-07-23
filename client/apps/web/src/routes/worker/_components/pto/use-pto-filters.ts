import { dateToUnixTimestamp } from "@/lib/date";
import type { PTOType } from "@/types/worker";
import { useMemo } from "react";

export const ptoTypeOptions = [
  { value: "All", label: "All Types" },
  { value: "Vacation", label: "Vacation" },
  { value: "Sick", label: "Sick" },
  { value: "Holiday", label: "Holiday" },
  { value: "Bereavement", label: "Bereavement" },
  { value: "Maternity", label: "Maternity" },
  { value: "Paternity", label: "Paternity" },
  { value: "Personal", label: "Personal" },
];

export function usePTOFilters() {
  const now = useMemo(() => new Date(), []);

  const defaultValues = useMemo(
    () => ({
      startDate: dateToUnixTimestamp(
        new Date(now.getFullYear(), now.getMonth(), 1),
      ),
      endDate: dateToUnixTimestamp(
        new Date(now.getFullYear(), now.getMonth() + 1, 0),
      ),
      type: undefined as PTOType | undefined,
      workerId: undefined as string | undefined,
    }),
    [now],
  );

  return {
    defaultValues,
    ptoTypeOptions,
  };
}
