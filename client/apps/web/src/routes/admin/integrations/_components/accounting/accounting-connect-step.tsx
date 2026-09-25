import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { ExternalLinkIcon, InfoIcon, LockIcon } from "lucide-react";
import type { AccountingAppSettings, AccountingConnection } from "@/lib/graphql/accounting-sync";
import { AccountingAppKeys } from "./accounting-app-keys";
import type { AccountingVendor } from "./accounting-vendors";

type AccountingConnectStepProps = {
  vendor: AccountingVendor;
  available: boolean;
  app: AccountingAppSettings;
  connection: AccountingConnection | null;
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
  connection,
  canManage,
  previousCompanyName,
  isConnecting,
  onConnect,
  onCancel,
}: AccountingConnectStepProps) {
  const t = useT();

  return (
    <>
      <div className="space-y-2">
        <h3 className="text-base font-semibold">{t("Connect {0}", vendor.name)}</h3>
        <p className="text-foreground-muted text-sm">
          {t(
            "You sign in on {0}'s own page and choose the company to connect. Trenova never sees your password; it keeps an encrypted authorization that you can revoke at any time.",
            vendor.name,
          )}
        </p>
      </div>
      <ul className="text-foreground-muted space-y-1.5 text-sm">
        <li className="flex gap-2">
          <LockIcon aria-hidden className="text-foreground-subtle mt-0.5 size-3.5 shrink-0" />
          <span>
            {t(
              "Right away, Trenova reads the company's name, home currency, multicurrency setting and closing date, and checks the connection every fifteen minutes.",
            )}
          </span>
        </li>
        <li className="flex gap-2">
          <InfoIcon aria-hidden className="text-foreground-subtle mt-0.5 size-3.5 shrink-0" />
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
      <AccountingAppKeys vendor={vendor} app={app} connection={connection} canManage={canManage} />
      {available && !canManage ? (
        <Alert size="sm" variant="warning">
          <AlertDescription>
            {t("Connecting {0} needs manage access to the accounting integration.", vendor.name)}
          </AlertDescription>
        </Alert>
      ) : null}
      <div className="flex justify-end gap-2 border-t pt-4">
        <Button type="button" variant="outline" onClick={onCancel}>
          {t("Cancel")}
        </Button>
        <Button
          type="button"
          onClick={onConnect}
          disabled={!available || !canManage}
          isLoading={isConnecting}
          loadingText={t("Opening {0}...", vendor.name)}
        >
          {t("Connect to {0}", vendor.name)}
          <ExternalLinkIcon aria-hidden className="size-3.5" />
        </Button>
      </div>
    </>
  );
}
