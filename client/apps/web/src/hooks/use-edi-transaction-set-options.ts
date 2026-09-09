import { toEDITransactionSetFilterOptions } from "@/lib/edi-transaction-set-options";
import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import type { SelectOption } from "@trenova/shared/types/fields";
import { useMemo } from "react";

// The transaction set catalog is seeded by migration and changes with releases,
// not with user activity, so a long stale window avoids refetching it on every
// filter popover open.
const TRANSACTION_SET_OPTIONS_STALE_TIME = 5 * 60 * 1000;

export function useEDITransactionSetOptions(fallback: readonly SelectOption[]): SelectOption[] {
  const { data } = useQuery({
    ...queries.edi.transactionSetOptions(),
    staleTime: TRANSACTION_SET_OPTIONS_STALE_TIME,
  });

  return useMemo(
    () => toEDITransactionSetFilterOptions(data?.results ?? [], fallback),
    [data, fallback],
  );
}
