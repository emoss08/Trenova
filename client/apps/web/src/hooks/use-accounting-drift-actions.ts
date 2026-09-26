import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  checkAccountingDrift,
  dismissAccountingDrift,
  resolveAccountingDrift,
} from "@/lib/graphql/accounting-drift";
import { ACCOUNTING_DRIFT_TABLE_KEY } from "@/lib/graphql/accounting-drift-table";
import { ACCOUNTING_SYNC_LEDGER_KEY } from "@/lib/graphql/accounting-sync-ledger-table";
import { queries } from "@/lib/queries";
import type {
  AccountingDriftDirection,
  AccountingSystem,
} from "@trenova/graphql/generated/graphql";
import { useQueryClient } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import { useCallback } from "react";
import { toast } from "sonner";

export function useAccountingDriftActions() {
  const t = useT();
  const queryClient = useQueryClient();

  const refresh = useCallback(
    () =>
      Promise.all([
        queryClient.invalidateQueries({ queryKey: queries.accountingSync._def }),
        queryClient.invalidateQueries({ queryKey: [ACCOUNTING_DRIFT_TABLE_KEY] }),
        queryClient.invalidateQueries({ queryKey: [ACCOUNTING_SYNC_LEDGER_KEY] }),
      ]),
    [queryClient],
  );

  const resolve = useApiMutation({
    mutationFn: (params: { id: string; direction: AccountingDriftDirection }) =>
      resolveAccountingDrift(params),
    resourceName: t("Difference"),
    onSuccess: async (finding) => {
      await refresh();
      toast.success(
        finding.pushed
          ? t("Trenova's value is on its way to the books")
          : t("Trenova adjusted to match the books"),
      );
    },
  });

  const dismiss = useApiMutation({
    mutationFn: (params: { id: string; note: string }) => dismissAccountingDrift(params),
    resourceName: t("Difference"),
    onSuccess: async () => {
      await refresh();
      toast.success(t("Difference dismissed"));
    },
  });

  const check = useApiMutation({
    mutationFn: (system: AccountingSystem) => checkAccountingDrift(system),
    resourceName: t("Drift check"),
    onSuccess: async () => {
      await refresh();
      toast.success(t("Checking the books now"));
    },
  });

  return { resolve, dismiss, check };
}
