import { dateToUnixTimestamp } from "@/lib/date";
import { PTOType } from "@/types/worker";
import { useMemo } from "react";

export const ptoTypeOptions = [
  { value: "All", label: "All Types" },
  { value: PTOType.Vacation, label: "Vacation" },
  { value: PTOType.Sick, label: "Sick" },
  { value: PTOType.Holiday, label: "Holiday" },
  { value: PTOType.Bereavement, label: "Bereavement" },
  { value: PTOType.Maternity, label: "Maternity" },
  { value: PTOType.Paternity, label: "Paternity" },
  { value: PTOType.Personal, label: "Personal" },
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
