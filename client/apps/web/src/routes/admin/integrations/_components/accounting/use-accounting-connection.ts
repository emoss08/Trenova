import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  checkAccountingConnection,
  connectAccountingWithCertificate,
  disconnectAccountingSystem,
  prepareAccountingCertificate,
  removeAccountingApp,
  renewAccountingCertificate,
  saveAccountingApp,
  startAccountingAuthorization,
  type AccountingConnection,
  type AccountingSyncStatus,
} from "@/lib/graphql/accounting-sync";
import { isTrustedAuthorizeUrl, normalizeNetSuiteAccountId } from "@/lib/accounting-sync";
import { accountingSync } from "@/lib/queries/accounting-sync";
import { integration } from "@/lib/queries/integration";
import { useQueryClient } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import { useCallback } from "react";
import { toast } from "@trenova/shared/components/ui/toast";
import type { UseFormReturn } from "react-hook-form";
import {
  chooseAccountingSyncMode,
  enableAccountingSync,
  updateAccountingSyncSettings,
} from "@/lib/graphql/accounting-sync-ledger";
import type { AccountingAppFormValues } from "./accounting-app-schema";
import type {
  AccountingCertificateConnectValues,
  AccountingCertificateRenewValues,
} from "./accounting-certificate-schema";
import {
  accountingModeInput,
  type AccountingModeValues,
  type AccountingStartDateValues,
  type AccountingSyncSettingsValues,
} from "./accounting-start-date-schema";
import type { AccountingVendor } from "./accounting-vendors";

export function useAccountingConnectionActions(vendor: AccountingVendor) {
  const t = useT();
  const queryClient = useQueryClient();

  const refresh = useCallback(
    () =>
      Promise.all([
        queryClient.invalidateQueries({
          queryKey: accountingSync.status(vendor.system).queryKey,
        }),
        queryClient.invalidateQueries({ queryKey: integration.catalog().queryKey }),
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
      queryClient.setQueryData(accountingSync.status(vendor.system).queryKey, status);
      await queryClient.invalidateQueries({ queryKey: integration.catalog().queryKey });
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
    },
  });

  const remove = useApiMutation({
    mutationFn: () => removeAccountingApp(vendor.system),
    resourceName: t("{0} keys", vendor.appName),
    onSuccess: async (status) => {
      await applyStatus(status);
    },
  });

  return { save, remove };
}

export function useAccountingSyncSetupActions(
  vendor: AccountingVendor,
  form?: UseFormReturn<AccountingStartDateValues>,
) {
  const queryClient = useQueryClient();

  const enable = useApiMutation({
    mutationFn: (values: AccountingStartDateValues) =>
      enableAccountingSync({
        integrationType: vendor.system,
        startDate: values.startDate,
        autoSync: values.autoSync,
        driverSettlements: values.driverSettlements,
        backfill: values.backfill,
        openingBalances: values.openingBalances,
      }),
    form,
    resourceName: vendor.name,
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: accountingSync.status(vendor.system).queryKey,
        }),
        queryClient.invalidateQueries({
          queryKey: accountingSync.syncSummary(vendor.system).queryKey,
        }),
        queryClient.invalidateQueries({ queryKey: integration.catalog().queryKey }),
      ]);
    },
  });

  return { enable };
}

export function useAccountingModeAction(
  vendor: AccountingVendor,
  ledgerAvailable: boolean,
  form?: UseFormReturn<AccountingModeValues>,
) {
  const queryClient = useQueryClient();

  return useApiMutation({
    mutationFn: (values: AccountingModeValues) =>
      chooseAccountingSyncMode({
        integrationType: vendor.system,
        ...accountingModeInput(values, ledgerAvailable),
      }),
    form,
    resourceName: vendor.name,
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: accountingSync.status(vendor.system).queryKey,
        }),
        queryClient.invalidateQueries({
          queryKey: accountingSync.mappingSummary(vendor.system).queryKey,
        }),
      ]);
    },
  });
}

export function useAccountingSyncSettingsAction(
  vendor: AccountingVendor,
  settingsForm: UseFormReturn<AccountingSyncSettingsValues>,
) {
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
          queryKey: accountingSync.status(vendor.system).queryKey,
        }),
        queryClient.invalidateQueries({
          queryKey: accountingSync.syncSummary(vendor.system).queryKey,
        }),
        queryClient.invalidateQueries({
          queryKey: accountingSync.inboundOverview(vendor.system).queryKey,
        }),
      ]);
    },
  });
}

export function useAccountingCertificateActions(
  vendor: AccountingVendor,
  forms?: {
    connect?: UseFormReturn<AccountingCertificateConnectValues>;
    renew?: UseFormReturn<AccountingCertificateRenewValues>;
  },
) {
  const t = useT();
  const queryClient = useQueryClient();
  const connectForm = forms?.connect;
  const renewForm = forms?.renew;

  const refresh = useCallback(
    () =>
      Promise.all([
        queryClient.invalidateQueries({
          queryKey: accountingSync.status(vendor.system).queryKey,
        }),
        queryClient.invalidateQueries({ queryKey: integration.catalog().queryKey }),
      ]),
    [queryClient, vendor.system],
  );

  const prepare = useApiMutation({
    mutationFn: () => prepareAccountingCertificate(vendor.system),
    resourceName: t("{0} certificate", vendor.name),
    onSuccess: async () => {
      await refresh();
    },
  });

  const connect = useApiMutation({
    mutationFn: (values: AccountingCertificateConnectValues) =>
      connectAccountingWithCertificate({
        integrationType: vendor.system,
        accountId: normalizeNetSuiteAccountId(values.accountId) ?? values.accountId.trim(),
        certificateId: values.certificateId.trim(),
      }),
    form: connectForm,
    resourceName: vendor.name,
  });

  const renew = useApiMutation({
    mutationFn: ({
      connection,
      values,
    }: {
      connection: AccountingConnection;
      values: AccountingCertificateRenewValues;
    }) =>
      renewAccountingCertificate({
        connectionId: connection.id,
        certificateId: values.certificateId.trim(),
      }),
    form: renewForm,
    resourceName: t("{0} certificate", vendor.name),
    onSuccess: async () => {
      await refresh();
    },
  });

  return { prepare, connect, renew, refresh };
}
