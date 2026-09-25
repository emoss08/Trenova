import { LazyImage } from "@/components/image";
import { ExternalLink } from "@/components/link";
import { usePermission } from "@/hooks/use-permission";
import { useAccountingMappingSummary } from "@/hooks/use-accounting-mapping-summary";
import {
  hasLiveAccountingConnection,
  needsAccountingMappings,
  needsAccountingStartDate,
} from "@/lib/accounting-sync";
import type { IntegrationSetupStep } from "@/lib/integration-setup";
import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import { useTheme } from "@trenova/shared/components/theme-provider";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { IntegrationSetupWizard } from "../shared/integration-setup-wizard";
import { AccountingCompanyFacts } from "./accounting-company-facts";
import { AccountingConnectStep } from "./accounting-connect-step";
import { AccountingConnectionPanel } from "./accounting-connection-panel";
import { AccountingMapStep } from "./accounting-map-step";
import { AccountingStartDateStep } from "./accounting-start-date-step";
import { quickBooksVendor, type AccountingVendor } from "./accounting-vendors";
import { useAccountingConnectionActions } from "./use-accounting-connection";

type AccountingIntegrationModalProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  justConnected: boolean;
  onReviewed: () => void;
};

export function QuickBooksIntegrationModal(props: AccountingIntegrationModalProps) {
  return <AccountingIntegrationModal vendor={quickBooksVendor} {...props} />;
}

function AccountingIntegrationModal({
  vendor,
  open,
  onOpenChange,
  justConnected,
  onReviewed,
}: AccountingIntegrationModalProps & { vendor: AccountingVendor }) {
  const t = useT();
  const { theme } = useTheme();

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="lg" className="flex max-h-[90vh] flex-col overflow-hidden">
        <DialogHeader className="flex flex-row items-center gap-3 pr-8">
          <LazyImage
            src={theme === "dark" ? vendor.logoDark : vendor.logoLight}
            alt={t("{0} logo", vendor.name)}
            className="h-7 max-w-28 object-contain"
          />
          <div className="min-w-0 space-y-0.5">
            <DialogTitle>{t("Accounting sync")}</DialogTitle>
            <DialogDescription className="flex flex-wrap items-center gap-1 text-xs">
              <span>{t("Keep {0} in step with what Trenova posts.", vendor.name)}</span>
              <ExternalLink href={vendor.docsUrl} className="text-xs">
                {t("{0} help", vendor.name)}
              </ExternalLink>
            </DialogDescription>
          </div>
        </DialogHeader>
        <div className="min-h-0 flex-1 overflow-y-auto px-1 py-2">
          <AccountingIntegrationBody
            vendor={vendor}
            open={open}
            onClose={() => onOpenChange(false)}
            justConnected={justConnected}
            onReviewed={onReviewed}
          />
        </div>
      </DialogContent>
    </Dialog>
  );
}

function AccountingIntegrationBody({
  vendor,
  open,
  onClose,
  justConnected,
  onReviewed,
}: {
  vendor: AccountingVendor;
  open: boolean;
  onClose: () => void;
  justConnected: boolean;
  onReviewed: () => void;
}) {
  const t = useT();
  const { allowed: canRead, isLoading: permissionsLoading } = usePermission(
    Resource.AccountingIntegration,
    Operation.Read,
  );
  const { allowed: canUpdate } = usePermission(Resource.AccountingIntegration, Operation.Update);
  const { allowed: canManage } = usePermission(Resource.AccountingIntegration, Operation.Manage);

  const statusQuery = useQuery({
    ...queries.accountingSync.status(vendor.system),
    enabled: open && canRead,
  });
  const { connect, check, disconnect } = useAccountingConnectionActions(vendor);
  const mapping = needsAccountingMappings(statusQuery.data?.connection);
  const summaryQuery = useAccountingMappingSummary(vendor.system, open && canRead && mapping);

  const steps: IntegrationSetupStep[] = [
    { id: "connect", label: t("Connect"), detail: t("Sign in to {0}", vendor.name) },
    { id: "review", label: t("Review company"), detail: t("Confirm what was connected") },
    { id: "map", label: t("Match records"), detail: t("Confirm what each record is sent as") },
    { id: "start", label: t("Start date"), detail: t("Choose the first day sent") },
  ];

  if (permissionsLoading || (canRead && statusQuery.isLoading)) {
    return (
      <div className="space-y-3">
        <Skeleton className="h-8 w-1/2" />
        <Skeleton className="h-32 w-full" />
      </div>
    );
  }

  if (!canRead) {
    return (
      <Alert size="sm" variant="warning">
        <AlertDescription>
          {t("You do not have permission to view the accounting integration.")}
        </AlertDescription>
      </Alert>
    );
  }

  const status = statusQuery.data;
  if (statusQuery.isError || !status) {
    return (
      <Alert size="sm" variant="destructive">
        <AlertDescription className="flex items-center justify-between gap-3">
          <span>{t("The {0} connection could not be loaded.", vendor.name)}</span>
          <Button
            type="button"
            size="sm"
            variant="outline"
            onClick={() => void statusQuery.refetch()}
          >
            {t("Retry")}
          </Button>
        </AlertDescription>
      </Alert>
    );
  }

  const connection = status.connection;
  if (!connection || !hasLiveAccountingConnection(connection)) {
    return (
      <IntegrationSetupWizard
        steps={steps}
        activeStepId="connect"
        label={t("{0} setup", vendor.name)}
      >
        <AccountingConnectStep
          vendor={vendor}
          available={status.available}
          app={status.app}
          connection={connection ?? null}
          canManage={canManage}
          previousCompanyName={connection?.externalCompanyName ?? ""}
          isConnecting={connect.isPending || connect.isSuccess}
          onConnect={() => connect.mutate()}
          onCancel={onClose}
        />
      </IntegrationSetupWizard>
    );
  }

  if (justConnected) {
    return (
      <IntegrationSetupWizard
        steps={steps}
        activeStepId="review"
        label={t("{0} setup", vendor.name)}
      >
        <div className="space-y-2">
          <h3 className="text-base font-semibold">
            {t("Connected to {0}", connection.externalCompanyName || vendor.name)}
          </h3>
          <p className="text-foreground-muted text-sm">
            {t(
              "Check this is the company you meant to connect. If it is not, disconnect it and connect again, choosing the right company on {0}'s page.",
              vendor.name,
            )}
          </p>
        </div>
        <AccountingCompanyFacts connection={connection} />
        <div className="flex justify-end gap-2 border-t pt-4">
          <Button type="button" onClick={onReviewed}>
            {needsAccountingMappings(connection) || needsAccountingStartDate(connection)
              ? t("Continue")
              : t("Done")}
          </Button>
        </div>
      </IntegrationSetupWizard>
    );
  }

  if (mapping) {
    return (
      <IntegrationSetupWizard steps={steps} activeStepId="map" label={t("{0} setup", vendor.name)}>
        {summaryQuery.data ? (
          <AccountingMapStep vendor={vendor} summary={summaryQuery.data} canUpdate={canUpdate} />
        ) : summaryQuery.isError ? (
          <Alert size="sm" variant="destructive">
            <AlertDescription className="flex items-center justify-between gap-3">
              <span>{t("The mappings could not be loaded.")}</span>
              <Button
                type="button"
                size="sm"
                variant="outline"
                onClick={() => void summaryQuery.refetch()}
              >
                {t("Retry")}
              </Button>
            </AlertDescription>
          </Alert>
        ) : (
          <div className="space-y-3">
            <Skeleton className="h-8 w-1/2" />
            <Skeleton className="h-32 w-full" />
          </div>
        )}
      </IntegrationSetupWizard>
    );
  }

  if (needsAccountingStartDate(connection)) {
    return (
      <IntegrationSetupWizard
        steps={steps}
        activeStepId="start"
        label={t("{0} setup", vendor.name)}
      >
        <AccountingStartDateStep vendor={vendor} connection={connection} canManage={canManage} />
      </IntegrationSetupWizard>
    );
  }

  return (
    <AccountingConnectionPanel
      vendor={vendor}
      app={status.app}
      connection={connection}
      canUpdate={canUpdate}
      canManage={canManage}
      isChecking={check.isPending}
      isConnecting={connect.isPending || connect.isSuccess}
      isDisconnecting={disconnect.isPending}
      onCheck={() => check.mutate()}
      onReconnect={() => connect.mutate()}
      onDisconnect={() => disconnect.mutateAsync()}
    />
  );
}
