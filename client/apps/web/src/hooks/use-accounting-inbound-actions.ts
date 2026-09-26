import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  applyAccountingInboundChange,
  ignoreAccountingInboundChange,
} from "@/lib/graphql/accounting-inbound";
import { ACCOUNTING_INBOUND_TABLE_KEY } from "@/lib/graphql/accounting-inbound-table";
import { ACCOUNTING_SYNC_LEDGER_KEY } from "@/lib/graphql/accounting-sync-ledger-table";
import { queries } from "@/lib/queries";
import { useQueryClient } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import { useCallback } from "react";
import { toast } from "sonner";

export function useAccountingInboundActions() {
  const t = useT();
  const queryClient = useQueryClient();

  const refresh = useCallback(
    () =>
      Promise.all([
        queryClient.invalidateQueries({ queryKey: queries.accountingSync._def }),
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
      toast.success(t("Payment applied in Trenova"));
    },
  });

  const ignore = useApiMutation({
    mutationFn: (params: { id: string; note: string }) => ignoreAccountingInboundChange(params),
    resourceName: t("Payment"),
    onSuccess: async () => {
      await refresh();
      toast.success(t("Payment ignored"));
    },
  });

  return { apply, ignore };
}
