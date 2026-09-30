import { ChoiceButton } from "@/components/choice-button";
import { useNowSeconds } from "@/hooks/use-now-seconds";
import { describeApiError } from "@/lib/api-error-message";
import { accountingSetupPath, readAccountingCallback } from "@/lib/accounting-sync";
import {
  chooseAccountingCompany,
  finishAccountingAuthorization,
  type AccountingAuthorizationResult,
  type AccountingConnection,
} from "@/lib/graphql/accounting-sync";
import { queries } from "@/lib/queries";
import type { CompleteAccountingAuthorizationInput } from "@trenova/graphql/generated/graphql";
import { useMutation, useQuery } from "@tanstack/react-query";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixTime } from "@trenova/shared/lib/date";
import { useEffect, useRef, useState } from "react";
import { Link, useLocation, useNavigate } from "react-router";
import { toast } from "sonner";
import type { AccountingVendor } from "./accounting-vendors";

type CompanyChoice = Extract<AccountingAuthorizationResult, { kind: "choose" }>;

export function AccountingAuthorizationCallback({ vendor }: { vendor: AccountingVendor }) {
  const t = useT();
  const navigate = useNavigate();
  const location = useLocation();
  const [params] = useState(() => new URLSearchParams(location.search));
  const started = useRef(false);
  const setupPath = accountingSetupPath(vendor.system);

  const returned = readAccountingCallback(params, { requireCompany: false });
  const statusQuery = useQuery({
    ...queries.accountingSync.status(vendor.system),
    enabled: returned.kind === "authorized",
  });
  const profile = statusQuery.data?.profile;
  const callback = profile
    ? readAccountingCallback(params, { requireCompany: profile.callbackCarriesCompany })
    : returned;

  const connected = (connection: AccountingConnection) => {
    toast.success(t("{0} connected", connection.externalCompanyName || vendor.name));
    void navigate(`${setupPath}&setup=connected`, { replace: true });
  };

  const finish = useMutation({
    mutationFn: (input: CompleteAccountingAuthorizationInput) =>
      finishAccountingAuthorization(input),
    onSuccess: (result) => {
      if (result.kind === "connected") {
        connected(result.connection);
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
      <AccountingCompanyChoiceStep
        vendor={vendor}
        choice={result}
        setupPath={setupPath}
        onConnected={connected}
      />
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

function AccountingCompanyChoiceStep({
  vendor,
  choice,
  setupPath,
  onConnected,
}: {
  vendor: AccountingVendor;
  choice: CompanyChoice;
  setupPath: string;
  onConnected: (connection: AccountingConnection) => void;
}) {
  const t = useT();
  const nowSeconds = useNowSeconds(15_000);
  const [companyId, setCompanyId] = useState<string | null>(null);
  const choose = useMutation({
    mutationFn: (id: string) =>
      chooseAccountingCompany({
        integrationType: vendor.system,
        choiceToken: choice.choiceToken,
        companyId: id,
      }),
    onSuccess: onConnected,
  });

  const expired = choice.choiceExpiresAt !== null && nowSeconds >= choice.choiceExpiresAt;
  if (expired && !choose.isPending && !choose.isSuccess) {
    return (
      <CallbackFailure
        vendor={vendor}
        setupPath={setupPath}
        message={t(
          "The time to choose an organisation ran out, so nothing was connected. Start the connection again from Integrations.",
        )}
      />
    );
  }

  return (
    <div className="mx-auto w-full max-w-lg space-y-4">
      <div className="space-y-1">
        <h2 className="text-base font-semibold">{t("Choose the organisation")}</h2>
        <p className="text-foreground-muted text-sm">
          {t(
            "{0} gave Trenova access to more than one organisation. Choose the one whose books this Trenova organization keeps; Trenova gives up its access to the others.",
            vendor.name,
          )}
        </p>
        {choice.choiceExpiresAt !== null ? (
          <p className="text-foreground-subtle text-xs">
            {t(
              "Choose by {0}. After that, start the connection again.",
              formatUnixTime(choice.choiceExpiresAt),
            )}
          </p>
        ) : null}
      </div>
      <div
        role="radiogroup"
        aria-label={t("Organisations in {0}", vendor.name)}
        className="flex flex-col gap-2"
      >
        {choice.companies.map((company) => (
          <ChoiceButton
            key={company.id}
            selected={companyId === company.id}
            onClick={() => setCompanyId(company.id)}
            disabled={choose.isPending || choose.isSuccess}
          >
            <span className="font-medium">{company.name || company.id}</span>
          </ChoiceButton>
        ))}
      </div>
      {choose.isError ? (
        <Alert size="sm" variant="destructive">
          <AlertDescription>{describeApiError(choose.error)}</AlertDescription>
        </Alert>
      ) : null}
      <div className="flex justify-end gap-2 border-t pt-4">
        <Button variant="outline" render={<Link to="/admin/integrations" replace />}>
          {t("Back to integrations")}
        </Button>
        <Button
          type="button"
          disabled={companyId === null || choose.isSuccess}
          isLoading={choose.isPending}
          loadingText={t("Connecting...")}
          onClick={() => {
            if (companyId !== null) {
              choose.mutate(companyId);
            }
          }}
        >
          {t("Connect this organisation")}
        </Button>
      </div>
    </div>
  );
}
