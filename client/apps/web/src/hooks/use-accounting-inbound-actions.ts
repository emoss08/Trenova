import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  applyAccountingInboundChange,
  ignoreAccountingInboundChange,
} from "@/lib/graphql/accounting-inbound";
import { ACCOUNTING_INBOUND_TABLE_KEY } from "@/lib/graphql/accounting-inbound-table";
import { ACCOUNTING_SYNC_LEDGER_KEY } from "@/lib/graphql/accounting-sync-ledger-table";
import { accountingSync } from "@/lib/queries/accounting-sync";
import { useQueryClient } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import { useCallback } from "react";

export function useAccountingInboundActions() {
  const t = useT();
  const queryClient = useQueryClient();

  const refresh = useCallback(
    () =>
      Promise.all([
        queryClient.invalidateQueries({ queryKey: accountingSync._def }),
        queryClient.invalidateQueries({ queryKey: [ACCOUNTING_INBOUND_TABLE_KEY] }),
        queryClient.invalidateQueries({ queryKey: [ACCOUNTING_SYNC_LEDGER_KEY] }),
      ]),
    [queryClient],
  );

  const apply = useApiMutation({
    mutationFn: (id: string) => applyAccountingInboundChange(id),
    resourceName: t("Payment"),
    onSuccess: async () => {
      await refresh();
    },
  });

  const ignore = useApiMutation({
    mutationFn: (params: { id: string; note: string }) => ignoreAccountingInboundChange(params),
    resourceName: t("Payment"),
    onSuccess: async () => {
      await refresh();
    },
  });

  return { apply, ignore };
}
