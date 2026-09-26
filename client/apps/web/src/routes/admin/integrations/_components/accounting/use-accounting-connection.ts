import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  checkAccountingConnection,
  disconnectAccountingSystem,
  removeAccountingApp,
  saveAccountingApp,
  startAccountingAuthorization,
  type AccountingSyncStatus,
} from "@/lib/graphql/accounting-sync";
import { isTrustedAuthorizeUrl } from "@/lib/accounting-sync";
import { queries } from "@/lib/queries";
import { useQueryClient } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import { useCallback } from "react";
import { toast } from "sonner";
import type { UseFormReturn } from "react-hook-form";
import {
  enableAccountingSync,
  updateAccountingSyncSettings,
} from "@/lib/graphql/accounting-sync-ledger";
import type { AccountingAppFormValues } from "./accounting-app-schema";
import type {
  AccountingStartDateValues,
  AccountingSyncSettingsValues,
} from "./accounting-start-date-schema";
import type { AccountingVendor } from "./accounting-vendors";

export function useAccountingConnectionActions(vendor: AccountingVendor) {
  const t = useT();
  const queryClient = useQueryClient();

  const refresh = useCallback(
    () =>
      Promise.all([
        queryClient.invalidateQueries({
          queryKey: queries.accountingSync.status(vendor.system).queryKey,
        }),
        queryClient.invalidateQueries({ queryKey: queries.integration.catalog().queryKey }),
      ]),
    [queryClient, vendor.system],
  );

  const connect = useApiMutation({
    mutationFn: async () => {
      const start = await startAccountingAuthorization(vendor.system);
      if (!isTrustedAuthorizeUrl(vendor.system, start.authorizeUrl)) {
        throw new Error(
          t(
            "{0} sent back an address Trenova does not recognize, so nothing was opened.",
            vendor.name,
          ),
        );
      }
      return start;
    },
    resourceName: vendor.name,
    onSuccess: (start) => window.location.assign(start.authorizeUrl),
  });

  const check = useApiMutation({
    mutationFn: () => checkAccountingConnection(vendor.system),
    resourceName: vendor.name,
    onSuccess: async (connection) => {
      await refresh();
      if (connection.status === "Connected") {
        toast.success(t("{0} answered", vendor.name));
        return;
      }
      toast.warning(t("{0} did not answer", vendor.name), {
        description: connection.lastErrorMessage || undefined,
      });
    },
  });

  const disconnect = useApiMutation({
    mutationFn: () => disconnectAccountingSystem(vendor.system),
    resourceName: vendor.name,
    onSuccess: async () => {
      await refresh();
      toast.success(t("{0} disconnected", vendor.name));
    },
  });

  return { connect, check, disconnect };
}

export function useAccountingAppActions(
  vendor: AccountingVendor,
  form?: UseFormReturn<AccountingAppFormValues>,
) {
  const t = useT();
  const queryClient = useQueryClient();

  const applyStatus = useCallback(
    async (status: AccountingSyncStatus) => {
      queryClient.setQueryData(queries.accountingSync.status(vendor.system).queryKey, status);
      await queryClient.invalidateQueries({ queryKey: queries.integration.catalog().queryKey });
    },
    [queryClient, vendor.system],
  );

  const save = useApiMutation({
    mutationFn: (values: AccountingAppFormValues) =>
      saveAccountingApp({
        integrationType: vendor.system,
        environment: values.environment,
        clientId: values.clientId,
        clientSecret: values.clientSecret || null,
        webhookVerifierToken: values.webhookVerifierToken || null,
        clearWebhookVerifierToken: values.clearWebhookVerifierToken,
      }),
    form,
    resourceName: t("{0} keys", vendor.appName),
    onSuccess: async (status) => {
      await applyStatus(status);
      toast.success(t("{0} keys saved", vendor.appName), {
        description: t("{0} accepted the client ID and secret.", vendor.name),
      });
    },
  });

  const remove = useApiMutation({
    mutationFn: () => removeAccountingApp(vendor.system),
    resourceName: t("{0} keys", vendor.appName),
    onSuccess: async (status) => {
      await applyStatus(status);
      toast.success(t("{0} keys removed", vendor.appName));
    },
  });

  return { save, remove };
}

export function useAccountingSyncSetupActions(
  vendor: AccountingVendor,
  form?: UseFormReturn<AccountingStartDateValues>,
) {
  const t = useT();
  const queryClient = useQueryClient();

  const enable = useApiMutation({
    mutationFn: (values: AccountingStartDateValues) =>
      enableAccountingSync({
        integrationType: vendor.system,
        startDate: values.startDate,
        autoSync: values.autoSync,
        driverSettlements: values.driverSettlements,
        backfill: values.backfill,
      }),
    form,
    resourceName: vendor.name,
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: queries.accountingSync.status(vendor.system).queryKey,
        }),
        queryClient.invalidateQueries({
          queryKey: queries.accountingSync.syncSummary(vendor.system).queryKey,
        }),
        queryClient.invalidateQueries({ queryKey: queries.integration.catalog().queryKey }),
      ]);
      toast.success(t("Sending to {0} is on", vendor.name));
    },
  });

  return { enable };
}

export function useAccountingSyncSettingsAction(
  vendor: AccountingVendor,
  settingsForm: UseFormReturn<AccountingSyncSettingsValues>,
) {
  const t = useT();
  const queryClient = useQueryClient();

  return useApiMutation({
    mutationFn: (values: AccountingSyncSettingsValues) =>
      updateAccountingSyncSettings({
        integrationType: vendor.system,
        autoSync: values.autoSync,
        driverSettlements: values.driverSettlements,
        inboundPayments: values.inboundPayments,
      }),
    form: settingsForm,
    resourceName: vendor.name,
    onSuccess: async (connection) => {
      settingsForm.reset({
        autoSync: connection.autoSync,
        driverSettlements: connection.syncsDriverSettlements,
        inboundPayments: connection.inboundPaymentPolicy,
      });
      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: queries.accountingSync.status(vendor.system).queryKey,
        }),
        queryClient.invalidateQueries({
          queryKey: queries.accountingSync.syncSummary(vendor.system).queryKey,
        }),
        queryClient.invalidateQueries({
          queryKey: queries.accountingSync.inboundOverview(vendor.system).queryKey,
        }),
      ]);
      toast.success(t("Sync settings saved"));
    },
  });
}
