import { useNowSeconds } from "@/hooks/use-now-seconds";
import type { AccountingConnection } from "@/lib/graphql/accounting-sync";
import {
  accountingConnectionPhase,
  hasReconnectDeadline,
  reconnectDeadlineNear,
  reconnectWarningSeconds,
} from "@/lib/accounting-sync";
import { formatPreciseTimeAgo } from "@/lib/time-utils";
import type { AccountingConnectionStatus } from "@trenova/graphql/generated/graphql";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@trenova/shared/components/ui/alert-dialog";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  DescriptionEmpty,
  DescriptionItem,
  DescriptionList,
} from "@trenova/shared/components/ui/description-list";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { phaseTone } from "@trenova/shared/lib/status-phase";
import { useState } from "react";
import type {
  AccountingAppSettings,
  AccountingPreparedCertificate,
  AccountingProviderProfile,
  AccountingWebhookSubscriptionSummary,
} from "@/lib/graphql/accounting-sync";
import { AccountingAppKeys } from "./accounting-app-keys";
import { AccountingCertificateRenewDialog } from "./accounting-certificate";
import { AccountingCompanyFacts } from "./accounting-company-facts";
import { AccountingWebhookSubscriptions } from "./accounting-webhook-subscriptions";
import { AccountingSyncSettings } from "./accounting-sync-settings";
import type { AccountingVendor } from "./accounting-vendors";

type AccountingConnectionPanelProps = {
  vendor: AccountingVendor;
  app: AccountingAppSettings;
  profile: AccountingProviderProfile;
  connection: AccountingConnection;
  webhookSubscriptions: AccountingWebhookSubscriptionSummary | null;
  pendingCertificate: AccountingPreparedCertificate | null;
  canUpdate: boolean;
  canManage: boolean;
  isChecking: boolean;
  isConnecting: boolean;
  isDisconnecting: boolean;
  onCheck: () => void;
  onReconnect: () => void;
  onDisconnect: () => Promise<unknown>;
};

export function AccountingConnectionPanel({
  vendor,
  app,
  profile,
  connection,
  webhookSubscriptions,
  pendingCertificate,
  canUpdate,
  canManage,
  isChecking,
  isConnecting,
  isDisconnecting,
  onCheck,
  onReconnect,
  onDisconnect,
}: AccountingConnectionPanelProps) {
  const t = useT();
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [renewOpen, setRenewOpen] = useState(false);
  const usesCertificate = connection.authMethod === "ClientCredentials";
  const receivesNotices = profile.webhookSubscriptions || profile.webhookKeyLabel !== "";

  const statusLabels: Record<AccountingConnectionStatus, string> = {
    Connected: t("Connected"),
    Degraded: t("Retrying"),
    Failing: t("Failing"),
    Revoked: t("Needs reconnecting"),
    Disconnected: t("Disconnected"),
  };
  const nowSeconds = useNowSeconds(30_000);
  const since = (unixSeconds: number | null) =>
    unixSeconds ? formatPreciseTimeAgo(unixSeconds * 1000, nowSeconds * 1000) : null;
  const revoked = connection.status === "Revoked";
  const deadlineNear =
    !revoked &&
    reconnectDeadlineNear(
      connection.refreshTokenAbsoluteExpiresAt,
      reconnectWarningSeconds(connection, profile),
      nowSeconds,
    );

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="flex min-w-0 items-center gap-2">
          <h3 className="truncate text-base font-semibold">
            {connection.externalCompanyName || vendor.name}
          </h3>
          <Badge variant={phaseTone(accountingConnectionPhase(connection.status))}>
            {statusLabels[connection.status]}
          </Badge>
        </div>
        {canUpdate && !revoked ? (
          <Button
            type="button"
            size="sm"
            variant="outline"
            onClick={onCheck}
            isLoading={isChecking}
            loadingText={t("Checking...")}
          >
            {t("Check now")}
          </Button>
        ) : null}
      </div>

      {revoked ? (
        <Alert variant="destructive" size="sm">
          <AlertTitle>{t("{0} no longer accepts Trenova's access", vendor.name)}</AlertTitle>
          <AlertDescription>
            {usesCertificate
              ? t(
                  "{0} refused Trenova's certificate. If the account was refreshed from production, set up the certificate mapping again, then renew the certificate here. Nothing reaches {0} until then.",
                  vendor.name,
                )
              : t(
                  "The authorization was revoked or has expired. Nothing reaches {0} until someone with manage access reconnects the same company.",
                  vendor.name,
                )}
          </AlertDescription>
        </Alert>
      ) : null}
      {connection.status === "Degraded" || connection.status === "Failing" ? (
        <Alert variant={connection.status === "Failing" ? "destructive" : "warning"} size="sm">
          <AlertTitle>
            {connection.status === "Failing"
              ? t("{0} calls in a row have failed", connection.consecutiveFailures)
              : t("The last call to {0} failed; Trenova will keep retrying", vendor.name)}
          </AlertTitle>
          {connection.lastErrorMessage ? (
            <AlertDescription>{connection.lastErrorMessage}</AlertDescription>
          ) : null}
        </Alert>
      ) : null}
      {deadlineNear ? (
        <Alert variant="warning" size="sm">
          <AlertDescription>
            {usesCertificate
              ? t(
                  "The certificate Trenova connects to {0} with expires on {1}. Renew it before then to keep syncing without a gap.",
                  vendor.name,
                  formatUnixDateMedium(connection.refreshTokenAbsoluteExpiresAt),
                )
              : t(
                  "{0} ends this authorization on {1}. Reconnect before then to keep syncing without a gap.",
                  vendor.name,
                  formatUnixDateMedium(connection.refreshTokenAbsoluteExpiresAt),
                )}
          </AlertDescription>
        </Alert>
      ) : null}

      <AccountingCompanyFacts connection={connection} />

      {webhookSubscriptions ? (
        <AccountingWebhookSubscriptions vendor={vendor} summary={webhookSubscriptions} />
      ) : null}

      <DescriptionList columns={2} className="border-t pt-4">
        <DescriptionItem label={t("What is sent")}>
          {connection.syncMode === "Ledger"
            ? connection.ledgerGranularity === "DailySummary"
              ? t("Journal entries, summed by day")
              : t("Journal entries")
            : t("Documents")}
        </DescriptionItem>
        {connection.syncMode === "Ledger" ? (
          <DescriptionItem label={t("Opening balances")}>
            {connection.ledgerOpeningBalancesSentAt != null
              ? t("Sent on {0}", formatUnixDateMedium(connection.ledgerOpeningBalancesSentAt))
              : t("Not sent")}
          </DescriptionItem>
        ) : null}
        <DescriptionItem label={t("Connected on")} numeric>
          {formatUnixDateMedium(connection.connectedAt)}
        </DescriptionItem>
        <DescriptionItem label={t("Connected with")}>
          {usesCertificate ? t("A certificate") : t("A sign-in")}
        </DescriptionItem>
        {usesCertificate && connection.certificateExpiresAt != null ? (
          <DescriptionItem label={t("Certificate expires")} numeric>
            {formatUnixDateMedium(connection.certificateExpiresAt)}
          </DescriptionItem>
        ) : null}
        {!usesCertificate && hasReconnectDeadline(connection.refreshTokenAbsoluteExpiresAt) ? (
          <DescriptionItem label={t("Reconnect by")} numeric>
            {formatUnixDateMedium(connection.refreshTokenAbsoluteExpiresAt)}
          </DescriptionItem>
        ) : null}
        <DescriptionItem label={t("Last checked")} numeric>
          {since(connection.lastCheckedAt) ?? <DescriptionEmpty />}
        </DescriptionItem>
        <DescriptionItem label={t("Last successful call")} numeric>
          {since(connection.lastSuccessAt) ?? <DescriptionEmpty />}
        </DescriptionItem>
        <DescriptionItem label={t("Last failure")} numeric>
          {since(connection.lastFailureAt) ?? <DescriptionEmpty />}
        </DescriptionItem>
        {receivesNotices ? (
          <DescriptionItem label={t("Last notice from {0}", vendor.name)} numeric>
            {since(connection.lastWebhookAt) ?? <DescriptionEmpty />}
          </DescriptionItem>
        ) : null}
      </DescriptionList>

      {connection.setupStep === "Complete" ? (
        <AccountingSyncSettings
          vendor={vendor}
          profile={profile}
          connection={connection}
          canManage={canManage}
        />
      ) : null}

      <AccountingAppKeys
        vendor={vendor}
        app={app}
        profile={profile}
        connection={connection}
        canManage={canManage}
      />

      {canManage ? (
        <div className="flex justify-end gap-2 border-t pt-4">
          <Button type="button" variant="outline" onClick={() => setConfirmOpen(true)}>
            {t("Disconnect")}
          </Button>
          {usesCertificate ? (
            <Button
              type="button"
              variant={revoked || deadlineNear ? "default" : "outline"}
              onClick={() => setRenewOpen(true)}
            >
              {t("Renew certificate")}
            </Button>
          ) : revoked || deadlineNear ? (
            <Button
              type="button"
              onClick={onReconnect}
              isLoading={isConnecting}
              loadingText={t("Opening {0}...", vendor.name)}
            >
              {t("Reconnect")}
            </Button>
          ) : null}
        </div>
      ) : null}

      {usesCertificate ? (
        <AccountingCertificateRenewDialog
          vendor={vendor}
          appName={t(profile.appName)}
          connection={connection}
          pending={pendingCertificate}
          canManage={canManage}
          open={renewOpen}
          onOpenChange={setRenewOpen}
        />
      ) : null}

      <AlertDialog open={confirmOpen} onOpenChange={setConfirmOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t("Disconnect {0}?", vendor.name)}</AlertDialogTitle>
            <AlertDialogDescription>
              {profile.revokesTokens
                ? t(
                    "Trenova revokes its access to {0} and stops reading from and writing to it. Nothing already in {1} is changed or deleted. You can connect again later.",
                    connection.externalCompanyName || vendor.name,
                    vendor.name,
                  )
                : t(
                    "Trenova deletes its sign-in to {0}, removes its webhook subscriptions and stops reading from and writing to it. {1} offers no way for Trenova to withdraw the consent itself: to remove it, an administrator removes the {2} from Enterprise applications in Microsoft Entra. Nothing already in {1} is changed or deleted.",
                    connection.externalCompanyName || vendor.name,
                    vendor.name,
                    t(profile.appName),
                  )}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={isDisconnecting}>{t("Cancel")}</AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              isLoading={isDisconnecting}
              loadingText={t("Disconnecting...")}
              onClick={() => {
                void onDisconnect().then(
                  () => setConfirmOpen(false),
                  () => undefined,
                );
              }}
            >
              {t("Disconnect")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
