import { ChoiceButton } from "@/components/choice-button";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { InfoCircleIcon, LinkExternal01Icon, Lock01Icon } from "@trenova/shared/components/icons";
import { useState } from "react";
import type {
  AccountingAppSettings,
  AccountingConnection,
  AccountingPreparedCertificate,
  AccountingProviderProfile,
} from "@/lib/graphql/accounting-sync";
import { AccountingAppKeys } from "./accounting-app-keys";
import { AccountingCertificateConnect } from "./accounting-certificate";
import type { AccountingVendor } from "./accounting-vendors";

type AccountingConnectStepProps = {
  vendor: AccountingVendor;
  available: boolean;
  app: AccountingAppSettings;
  profile: AccountingProviderProfile;
  connection: AccountingConnection | null;
  pendingCertificate: AccountingPreparedCertificate | null;
  connectedElsewhere: AccountingVendor | null;
  canManage: boolean;
  previousCompanyName: string;
  isConnecting: boolean;
  onConnect: () => void;
  onCancel: () => void;
};

export function AccountingConnectStep({
  vendor,
  available,
  app,
  profile,
  connection,
  pendingCertificate,
  connectedElsewhere,
  canManage,
  previousCompanyName,
  isConnecting,
  onConnect,
  onCancel,
}: AccountingConnectStepProps) {
  const t = useT();
  const [method, setMethod] = useState<ConnectMethod>(() =>
    pendingCertificate !== null || connection?.authMethod === "ClientCredentials"
      ? "certificate"
      : "signIn",
  );
  const certificate = profile.clientCredentials && method === "certificate";
  const blocked = !available || !canManage || connectedElsewhere !== null;

  return (
    <>
      <div className="space-y-2">
        <h3 className="text-base font-semibold">{t("Connect {0}", vendor.name)}</h3>
        <p className="text-foreground-muted text-sm">
          {certificate
            ? t(
                "Trenova signs every request to {0} with a certificate you upload there once, so no one has to sign in again until the certificate expires.",
                vendor.name,
              )
            : t(
                "You sign in on {0}'s own page and choose the company to connect. Trenova never sees your password; it keeps an encrypted authorization that you can revoke at any time.",
                vendor.name,
              )}
        </p>
      </div>
      {profile.clientCredentials ? (
        <div
          role="radiogroup"
          aria-label={t("How to connect {0}", vendor.name)}
          className="flex flex-col gap-2 sm:flex-row"
        >
          <ChoiceButton selected={method === "signIn"} onClick={() => setMethod("signIn")}>
            <span className="font-medium">{t("Sign in to {0}", vendor.name)}</span>
            <span className="text-foreground-muted text-xs">
              {t(
                "Quick to set up. {0} ends each sign-in after a week, so someone signs in again every week.",
                vendor.name,
              )}
            </span>
          </ChoiceButton>
          <ChoiceButton
            selected={method === "certificate"}
            onClick={() => setMethod("certificate")}
          >
            <span className="font-medium">{t("Connect with a certificate")}</span>
            <span className="text-foreground-muted text-xs">
              {t(
                "An administrator uploads a certificate once. It lasts two years and needs no sign-in.",
              )}
            </span>
          </ChoiceButton>
        </div>
      ) : null}
      <ul className="text-foreground-muted space-y-1.5 text-sm">
        <li className="flex gap-2">
          <Lock01Icon aria-hidden className="text-foreground-subtle mt-0.5 size-3.5 shrink-0" />
          <span>
            {t(
              "Right away, Trenova reads the company's name, home currency, multicurrency setting and closing date, and checks the connection every fifteen minutes.",
            )}
          </span>
        </li>
        <li className="flex gap-2">
          <InfoCircleIcon aria-hidden className="text-foreground-subtle mt-0.5 size-3.5 shrink-0" />
          <span>
            {t(
              "Nothing is sent to {0} until you map your accounts and turn syncing on.",
              vendor.name,
            )}
          </span>
        </li>
      </ul>
      {previousCompanyName ? (
        <Alert size="sm" variant="info">
          <AlertDescription>
            {t(
              "This organization was connected to {0} before. Connecting the same company picks up where it left off.",
              previousCompanyName,
            )}
          </AlertDescription>
        </Alert>
      ) : null}
      {connectedElsewhere ? (
        <Alert size="sm" variant="warning">
          <AlertDescription>
            {t(
              "This organization is connected to {0}. Disconnect {0} before connecting {1}, so the same documents never reach two sets of books.",
              connectedElsewhere.name,
              vendor.name,
            )}
          </AlertDescription>
        </Alert>
      ) : null}
      <AccountingAppKeys
        vendor={vendor}
        app={app}
        profile={profile}
        connection={connection}
        canManage={canManage}
      />
      {available && !canManage ? (
        <Alert size="sm" variant="warning">
          <AlertDescription>
            {t("Connecting {0} needs manage access to the accounting integration.", vendor.name)}
          </AlertDescription>
        </Alert>
      ) : null}
      {certificate ? (
        <AccountingCertificateConnect
          vendor={vendor}
          appName={t(profile.appName)}
          pending={pendingCertificate}
          canManage={canManage}
          disabled={blocked}
          onCancel={onCancel}
        />
      ) : (
        <div className="flex justify-end gap-2 border-t pt-4">
          <Button type="button" variant="outline" onClick={onCancel}>
            {t("Cancel")}
          </Button>
          <Button
            type="button"
            onClick={onConnect}
            disabled={blocked}
            isLoading={isConnecting}
            loadingText={t("Opening {0}...", vendor.name)}
          >
            {t("Connect to {0}", vendor.name)}
            <LinkExternal01Icon aria-hidden className="size-3.5" />
          </Button>
        </div>
      )}
    </>
  );
}

type ConnectMethod = "signIn" | "certificate";
