import { referenceRefreshRunning } from "@/lib/accounting-sync";
import type { AccountingConnection } from "@/lib/graphql/accounting-sync";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTime } from "@trenova/shared/lib/date";
import { RefreshCwIcon } from "lucide-react";

export function ReferenceRefreshStatus({
  connection,
  providerName,
  canUpdate,
  isRequesting,
  onRefresh,
}: {
  connection: AccountingConnection;
  providerName: string;
  canUpdate: boolean;
  isRequesting: boolean;
  onRefresh: () => void;
}) {
  const t = useT();
  const running = referenceRefreshRunning(connection);

  const refreshButton = canUpdate ? (
    <Button
      type="button"
      size="sm"
      variant="outline"
      isLoading={isRequesting}
      disabled={running}
      onClick={onRefresh}
    >
      <RefreshCwIcon className="size-3.5" />
      {t("Read {0} again", providerName)}
    </Button>
  ) : null;

  if (connection.referenceRefreshError && !running) {
    return (
      <Alert size="sm" variant="destructive">
        <AlertDescription className="flex flex-wrap items-center justify-between gap-3">
          <span>
            {t("The last read of {0} failed: {1}", providerName, connection.referenceRefreshError)}
          </span>
          {refreshButton}
        </AlertDescription>
      </Alert>
    );
  }

  return (
    <div className="flex flex-wrap items-center justify-between gap-3 text-sm">
      <span className="text-foreground-muted">
        {running
          ? t(
              "Reading {0}'s accounts, items, customers and vendors. Proposals update when it finishes.",
              providerName,
            )
          : connection.referenceRefreshedAt
            ? t(
                "{0} records read {1}",
                providerName,
                formatUnixDateTime(connection.referenceRefreshedAt),
              )
            : t("{0}'s records have not been read yet.", providerName)}
      </span>
      {refreshButton}
    </div>
  );
}
