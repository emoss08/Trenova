import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  changeAccountingBackfill,
  pauseAccountingSync,
  redateAccountingSync,
  releaseAccountingSync,
  requestAccountingBackfill,
  resumeAccountingSync,
  retryAccountingSync,
  skipAccountingSync,
} from "@/lib/graphql/accounting-sync-ledger";
import { ACCOUNTING_SYNC_LEDGER_KEY } from "@/lib/graphql/accounting-sync-ledger-table";
import { queries } from "@/lib/queries";
import type {
  AccountingBackfillAction,
  AccountingSyncErrorCategory,
  AccountingSyncObjectType,
  AccountingSystem,
} from "@trenova/graphql/generated/graphql";
import { useQueryClient } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDate } from "@trenova/shared/lib/date";
import { useCallback } from "react";
import { toast } from "sonner";

export type AccountingBackfillRequest = {
  rangeStart: number | null;
  rangeEnd: number | null;
  objectTypes: AccountingSyncObjectType[];
};

export function useAccountingSyncActions(system: AccountingSystem, providerName: string) {
  const t = useT();
  const queryClient = useQueryClient();

  const refresh = useCallback(
    () =>
      Promise.all([
        queryClient.invalidateQueries({ queryKey: queries.accountingSync._def }),
        queryClient.invalidateQueries({ queryKey: [ACCOUNTING_SYNC_LEDGER_KEY] }),
      ]),
    [queryClient],
  );

  const pause = useApiMutation({
    mutationFn: (reason: string) => pauseAccountingSync({ integrationType: system, reason }),
    resourceName: providerName,
    onSuccess: async () => {
      await refresh();
      toast.success(t("Sending to {0} is paused", providerName), {
        description: t("Posted documents keep queueing and go out when you resume."),
      });
    },
  });

  const resume = useApiMutation({
    mutationFn: () => resumeAccountingSync(system),
    resourceName: providerName,
    onSuccess: async () => {
      await refresh();
      toast.success(t("Sending to {0} resumed", providerName));
    },
  });

  const retry = useApiMutation({
    mutationFn: (params: { ids?: string[]; errorCategories?: AccountingSyncErrorCategory[] }) =>
      retryAccountingSync({
        integrationType: system,
        ids: params.ids ?? null,
        errorCategories: params.errorCategories ?? null,
      }),
    resourceName: t("Sync records"),
    onSuccess: async (affected) => {
      await refresh();
      toast.success(
        t("{0, plural, one {# record queued again} other {# records queued again}}", affected),
      );
    },
  });

  const release = useApiMutation({
    mutationFn: (ids?: string[]) =>
      releaseAccountingSync({ integrationType: system, ids: ids ?? null }),
    resourceName: t("Sync records"),
    onSuccess: async (affected) => {
      await refresh();
      toast.success(t("{0, plural, one {# record released} other {# records released}}", affected));
    },
  });

  const skip = useApiMutation({
    mutationFn: (params: { id: string; reason: string }) => skipAccountingSync(params),
    resourceName: t("Sync record"),
    onSuccess: async () => {
      await refresh();
      toast.success(t("Record skipped"));
    },
  });

  const redate = useApiMutation({
    mutationFn: (id: string) => redateAccountingSync(id),
    resourceName: t("Sync record"),
    onSuccess: async (record) => {
      await refresh();
      toast.success(
        record.redatedTo
          ? t("Queued again, dated {0}", formatUnixDate(record.redatedTo))
          : t("Queued again"),
      );
    },
  });

  const backfill = useApiMutation({
    mutationFn: (request: AccountingBackfillRequest) =>
      requestAccountingBackfill({
        integrationType: system,
        rangeStart: request.rangeStart,
        rangeEnd: request.rangeEnd,
        objectTypes: request.objectTypes.length > 0 ? request.objectTypes : null,
      }),
    resourceName: t("Backfill"),
    onSuccess: async () => {
      await refresh();
      toast.success(t("Backfill started"));
    },
  });

  const changeBackfill = useApiMutation({
    mutationFn: (params: { id: string; action: AccountingBackfillAction }) =>
      changeAccountingBackfill(params),
    resourceName: t("Backfill"),
    onSuccess: refresh,
  });

  return { pause, resume, retry, release, skip, redate, backfill, changeBackfill, refresh };
}
