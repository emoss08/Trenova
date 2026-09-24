import { useNowSeconds } from "@/hooks/use-now-seconds";
import type { AccountingConnection } from "@/lib/graphql/accounting-sync";
import { accountingConnectionPhase, reconnectDeadlineNear } from "@/lib/accounting-sync";
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
import { AccountingCompanyFacts } from "./accounting-company-facts";
import type { AccountingVendor } from "./accounting-vendors";

type AccountingConnectionPanelProps = {
  vendor: AccountingVendor;
  connection: AccountingConnection;
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
  connection,
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
    !revoked && reconnectDeadlineNear(connection.refreshTokenAbsoluteExpiresAt, nowSeconds);

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
            {t(
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
            {t(
              "{0} ends this authorization on {1}. Reconnect before then to keep syncing without a gap.",
              vendor.name,
              formatUnixDateMedium(connection.refreshTokenAbsoluteExpiresAt),
            )}
          </AlertDescription>
        </Alert>
      ) : null}

      <AccountingCompanyFacts connection={connection} />

      <DescriptionList columns={2} className="border-t pt-4">
        <DescriptionItem label={t("Connected on")} numeric>
          {formatUnixDateMedium(connection.connectedAt)}
        </DescriptionItem>
        <DescriptionItem label={t("Reconnect by")} numeric>
          {formatUnixDateMedium(connection.refreshTokenAbsoluteExpiresAt)}
        </DescriptionItem>
        <DescriptionItem label={t("Last checked")} numeric>
          {since(connection.lastCheckedAt) ?? <DescriptionEmpty />}
        </DescriptionItem>
        <DescriptionItem label={t("Last successful call")} numeric>
          {since(connection.lastSuccessAt) ?? <DescriptionEmpty />}
        </DescriptionItem>
        <DescriptionItem label={t("Last failure")} numeric>
          {since(connection.lastFailureAt) ?? <DescriptionEmpty />}
        </DescriptionItem>
        <DescriptionItem label={t("Last notice from {0}", vendor.name)} numeric>
          {since(connection.lastWebhookAt) ?? <DescriptionEmpty />}
        </DescriptionItem>
      </DescriptionList>

      {canManage ? (
        <div className="flex justify-end gap-2 border-t pt-4">
          <Button type="button" variant="outline" onClick={() => setConfirmOpen(true)}>
            {t("Disconnect")}
          </Button>
          {revoked || deadlineNear ? (
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

      <AlertDialog open={confirmOpen} onOpenChange={setConfirmOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t("Disconnect {0}?", vendor.name)}</AlertDialogTitle>
            <AlertDialogDescription>
              {t(
                "Trenova revokes its access to {0} and stops reading from and writing to it. Nothing already in {1} is changed or deleted. You can connect again later.",
                connection.externalCompanyName || vendor.name,
                vendor.name,
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
