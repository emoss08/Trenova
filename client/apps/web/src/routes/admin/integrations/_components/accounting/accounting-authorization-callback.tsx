import { describeApiError } from "@/lib/api-error-message";
import { accountingSetupPath, readAccountingCallback } from "@/lib/accounting-sync";
import { finishAccountingAuthorization } from "@/lib/graphql/accounting-sync";
import { accountingSync } from "@/lib/queries/accounting-sync";
import type { CompleteAccountingAuthorizationInput } from "@trenova/graphql/generated/graphql";
import { useMutation, useQuery } from "@tanstack/react-query";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import { useT } from "@trenova/shared/i18n/use-t";
import { useEffect, useRef, useState } from "react";
import { Link, useLocation, useNavigate } from "react-router";
import { AccountingCompanyChoice } from "./accounting-company-choice";
import type { AccountingVendor } from "./accounting-vendors";

export function AccountingAuthorizationCallback({ vendor }: { vendor: AccountingVendor }) {
  const t = useT();
  const navigate = useNavigate();
  const location = useLocation();
  const [params] = useState(() => new URLSearchParams(location.search));
  const started = useRef(false);
  const setupPath = accountingSetupPath(vendor.system);

  const returned = readAccountingCallback(params, { requireCompany: false });
  const statusQuery = useQuery({
    ...accountingSync.status(vendor.system),
    enabled: returned.kind === "authorized",
  });
  const profile = statusQuery.data?.profile;
  const callback = profile
    ? readAccountingCallback(params, { requireCompany: profile.callbackCarriesCompany })
    : returned;

  const connected = () => {
    void navigate(`${setupPath}&setup=connected`, { replace: true });
  };

  const finish = useMutation({
    mutationFn: (input: CompleteAccountingAuthorizationInput) =>
      finishAccountingAuthorization(input),
    onSuccess: (result) => {
      if (result.kind === "connected") {
        connected();
      }
    },
  });
  const { mutate } = finish;

  useEffect(() => {
    if (location.search !== "") {
      void navigate(location.pathname, { replace: true });
    }
  }, [location.pathname, location.search, navigate]);

  useEffect(() => {
    if (started.current || !profile || callback.kind !== "authorized") {
      return;
    }
    started.current = true;
    mutate({
      integrationType: vendor.system,
      code: callback.code,
      state: callback.state,
      realmId: callback.realmId,
    });
  }, [callback, mutate, profile, vendor.system]);

  const result = finish.data;
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
      if (statusQuery.isError) {
        failure = describeApiError(statusQuery.error);
      } else if (finish.isError) {
        failure = describeApiError(finish.error);
      } else if (result?.kind === "none") {
        failure = t(
          "{0} did not give Trenova access to any organisation, so nothing was connected.",
          vendor.name,
        );
      }
      break;
  }

  if (failure !== null) {
    return <CallbackFailure vendor={vendor} setupPath={setupPath} message={failure} />;
  }

  if (result?.kind === "choose") {
    return (
      <div className="mx-auto w-full max-w-lg">
        <AccountingCompanyChoice
          vendor={vendor}
          choice={result}
          onConnected={connected}
          cancelAction={
            <Button variant="outline" render={<Link to="/admin/integrations" replace />}>
              {t("Back to integrations")}
            </Button>
          }
          expired={
            <CallbackFailure
              vendor={vendor}
              setupPath={setupPath}
              message={t(
                "The time to choose an organisation ran out, so nothing was connected. Start the connection again from Integrations.",
              )}
            />
          }
        />
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

function CallbackFailure({
  vendor,
  setupPath,
  message,
}: {
  vendor: AccountingVendor;
  setupPath: string;
  message: string;
}) {
  const t = useT();

  return (
    <div className="mx-auto w-full max-w-lg space-y-4">
      <Alert variant="destructive">
        <AlertTitle>{t("{0} was not connected", vendor.name)}</AlertTitle>
        <AlertDescription>{message}</AlertDescription>
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
