import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  clearAccountingMapping,
  completeAccountingSetup,
  confirmAccountingMappings,
  createAccountingReferenceRecord,
  refreshAccountingReferenceData,
  rejectAccountingMapping,
  setAccountingMapping,
} from "@/lib/graphql/accounting-sync";
import { queries } from "@/lib/queries";
import type { AccountingSystem } from "@trenova/graphql/generated/graphql";
import { useQueryClient } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import { useCallback } from "react";
import { toast } from "sonner";

export function useAccountingMappingActions(system: AccountingSystem, providerName: string) {
  const t = useT();
  const queryClient = useQueryClient();

  const refresh = useCallback(
    () =>
      Promise.all([
        queryClient.invalidateQueries({ queryKey: queries.accountingSync._def }),
        queryClient.invalidateQueries({ queryKey: queries.integration.catalog().queryKey }),
      ]),
    [queryClient],
  );

  const confirm = useApiMutation({
    mutationFn: (items: { id: string; externalId: string }[]) => confirmAccountingMappings(items),
    resourceName: t("Mapping"),
    onSuccess: async (rows) => {
      await refresh();
      toast.success(
        rows.length === 1 ? t("Confirmed 1 mapping") : t("Confirmed {0} mappings", rows.length),
      );
    },
  });

  const reject = useApiMutation({
    mutationFn: (id: string) => rejectAccountingMapping(id),
    resourceName: t("Mapping"),
    onSuccess: async () => {
      await refresh();
      toast.success(t("Proposal turned down"));
    },
  });

  const set = useApiMutation({
    mutationFn: (input: { mappingId: string; externalId: string; reason?: string }) =>
      setAccountingMapping(input),
    resourceName: t("Mapping"),
    onSuccess: async (row) => {
      await refresh();
      toast.success(t("{0} is mapped to {1}", row.targetLabel, row.externalName));
    },
  });

  const clear = useApiMutation({
    mutationFn: (id: string) => clearAccountingMapping(id),
    resourceName: t("Mapping"),
    onSuccess: async (row) => {
      await refresh();
      toast.success(t("{0} is unmatched", row.targetLabel));
    },
  });

  const create = useApiMutation({
    mutationFn: (input: { mappingId: string; name?: string }) =>
      createAccountingReferenceRecord(input),
    resourceName: providerName,
    onSuccess: async (row) => {
      await refresh();
      toast.success(t("Created {0} in {1}", row.externalName, providerName));
    },
  });

  const refreshReference = useApiMutation({
    mutationFn: () => refreshAccountingReferenceData(system),
    resourceName: providerName,
    onSuccess: async () => {
      await refresh();
      toast.success(t("Reading {0} again", providerName), {
        description: t("Proposals update when it finishes."),
      });
    },
  });

  const completeSetup = useApiMutation({
    mutationFn: () => completeAccountingSetup(system),
    resourceName: providerName,
    onSuccess: async () => {
      await refresh();
      toast.success(t("{0} setup is finished", providerName));
    },
  });

  return { confirm, reject, set, clear, create, refreshReference, completeSetup };
}
