import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  checkAccountingConnection,
  disconnectAccountingSystem,
  startAccountingAuthorization,
} from "@/lib/graphql/accounting-sync";
import { isTrustedAuthorizeUrl } from "@/lib/accounting-sync";
import { queries } from "@/lib/queries";
import { useQueryClient } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import { useCallback } from "react";
import { toast } from "sonner";
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
