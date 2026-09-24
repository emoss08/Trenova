import { describeApiError } from "@/lib/api-error-message";
import { accountingSetupPath, readAccountingCallback } from "@/lib/accounting-sync";
import { completeAccountingAuthorization } from "@/lib/graphql/accounting-sync";
import type { CompleteAccountingAuthorizationInput } from "@trenova/graphql/generated/graphql";
import { useMutation } from "@tanstack/react-query";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import { useT } from "@trenova/shared/i18n/use-t";
import { useEffect, useRef, useState } from "react";
import { Link, useLocation, useNavigate } from "react-router";
import { toast } from "sonner";
import type { AccountingVendor } from "./accounting-vendors";

export function AccountingAuthorizationCallback({ vendor }: { vendor: AccountingVendor }) {
  const t = useT();
  const navigate = useNavigate();
  const location = useLocation();
  const [callback] = useState(() => readAccountingCallback(new URLSearchParams(location.search)));
  const started = useRef(false);
  const setupPath = accountingSetupPath(vendor.system);

  const complete = useMutation({
    mutationFn: (input: CompleteAccountingAuthorizationInput) =>
      completeAccountingAuthorization(input),
    onSuccess: (connection) => {
      toast.success(t("{0} connected", connection.externalCompanyName || vendor.name));
      void navigate(`${setupPath}&setup=connected`, { replace: true });
    },
  });
  const { mutate } = complete;

  useEffect(() => {
    if (started.current) {
      return;
    }
    started.current = true;
    if (location.search !== "") {
      void navigate(location.pathname, { replace: true });
    }
    if (callback.kind === "authorized") {
      mutate({
        integrationType: vendor.system,
        code: callback.code,
        state: callback.state,
        realmId: callback.realmId,
      });
    }
  }, [callback, location.pathname, location.search, mutate, navigate, vendor.system]);

  let failure: string | null = null;
  switch (callback.kind) {
    case "denied":
      failure = t("Access to {0} was not granted, so nothing was connected.", vendor.name);
      break;
    case "provider-error":
      failure = t(
        "{0} returned an error ({1}). Nothing was connected.",
        vendor.name,
        callback.error,
      );
      break;
    case "incomplete":
      failure = t(
        "This page was opened without everything {0} sends back. Start the connection again from Integrations.",
        vendor.name,
      );
      break;
    case "authorized":
      failure = complete.isError ? describeApiError(complete.error) : null;
      break;
  }

  if (failure !== null) {
    return (
      <div className="mx-auto w-full max-w-lg space-y-4">
        <Alert variant="destructive">
          <AlertTitle>{t("{0} was not connected", vendor.name)}</AlertTitle>
          <AlertDescription>{failure}</AlertDescription>
        </Alert>
        <div className="flex justify-end gap-2">
          <Button variant="outline" render={<Link to="/admin/integrations" replace />}>
            {t("Back to integrations")}
          </Button>
          <Button render={<Link to={setupPath} replace />}>{t("Try again")}</Button>
        </div>
      </div>
    );
  }

  return (
    <div
      role="status"
      className="text-foreground-muted mx-auto flex w-full max-w-lg items-center gap-2 text-sm"
    >
      <Spinner className="size-4" />
      {t("Finishing the connection to {0}...", vendor.name)}
    </div>
  );
}
