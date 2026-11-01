import { ExternalLink } from "@/components/link";
import { usePermission } from "@/hooks/use-permission";
import { useAccountingMappingSummary } from "@/hooks/use-accounting-mapping-summary";
import { useConnectedAccountingSystem } from "@/hooks/use-connected-accounting-system";
import {
  hasLiveAccountingConnection,
  needsAccountingMappings,
  needsAccountingMode,
  needsAccountingStartDate,
} from "@/lib/accounting-sync";
import type { IntegrationSetupStep } from "@/lib/integration-setup";
import { accountingSync } from "@/lib/queries/accounting-sync";
import { useQuery } from "@tanstack/react-query";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { IntegrationSetupFooter, IntegrationSetupWizard } from "../shared/integration-setup-wizard";
import { AccountingCompanyFacts } from "./accounting-company-facts";
import { AccountingConnectStep } from "./accounting-connect-step";
import { AccountingConnectionPanel } from "./accounting-connection-panel";
import { AccountingMapStep } from "./accounting-map-step";
import { AccountingModeStep } from "./accounting-mode-step";
import { AccountingStartDateStep } from "./accounting-start-date-step";
import { accountingVendor, type AccountingVendor } from "./accounting-vendors";
import { useAccountingConnectionActions } from "./use-accounting-connection";
import { IntegrationLogoMark } from "../shared/integration-connect-header";
import { IntegrationDialogContent } from "../shared/integration-dialog-content";

type AccountingIntegrationModalProps = {
  vendor: AccountingVendor;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  justConnected: boolean;
  onReviewed: () => void;
};

export function AccountingIntegrationModal({
  vendor,
  open,
  onOpenChange,
  justConnected,
  onReviewed,
}: AccountingIntegrationModalProps) {
  const t = useT();

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      {/* A fixed frame, so moving from one step to the next never resizes the dialog. */}
      <IntegrationDialogContent
        size="xl"
        className="flex h-[min(46rem,90vh)] flex-col gap-0 overflow-hidden p-0"
      >
        <DialogHeader className="flex-row items-center gap-3 border-b px-5 py-4 pr-12">
          <IntegrationLogoMark
            name={vendor.name}
            logo={{ light: vendor.logoLight, dark: vendor.logoDark }}
            className="max-sm:max-w-24"
          />
          <div className="flex min-w-0 flex-col gap-1">
            <span className="text-muted-foreground text-xs">{t("Accounting sync")}</span>
            <DialogTitle className="truncate">{vendor.name}</DialogTitle>
            <DialogDescription className="flex flex-wrap items-center gap-x-1 text-xs">
              <span>{t("Keep {0} in step with what Trenova posts.", vendor.name)}</span>
              <ExternalLink href={vendor.docsUrl} className="text-xs">
                {t("{0} help", vendor.name)}
              </ExternalLink>
            </DialogDescription>
          </div>
        </DialogHeader>
        <div className="flex min-h-0 flex-1 flex-col">
          <AccountingIntegrationBody
            vendor={vendor}
            open={open}
            onClose={() => onOpenChange(false)}
            justConnected={justConnected}
            onReviewed={onReviewed}
          />
        </div>
      </IntegrationDialogContent>
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
    ...accountingSync.status(vendor.system),
    enabled: open && canRead,
  });
  const tenantSystem = useConnectedAccountingSystem("status", open && canRead);
  const { connect, check, disconnect } = useAccountingConnectionActions(vendor);
  const mapping = needsAccountingMappings(statusQuery.data?.connection);
  const summaryQuery = useAccountingMappingSummary(vendor.system, open && canRead && mapping);

  const steps: IntegrationSetupStep[] = [
    {
      id: "connect",
      label: t("Connect"),
      detail: statusQuery.data?.profile.clientCredentials
        ? t("Sign in or use a certificate")
        : t("Sign in to {0}", vendor.name),
    },
    { id: "review", label: t("Review company"), detail: t("Confirm what was connected") },
    { id: "mode", label: t("What is sent"), detail: t("Documents or journal entries") },
    { id: "map", label: t("Match records"), detail: t("Confirm what each record is sent as") },
    { id: "start", label: t("Start date"), detail: t("Choose the first day sent") },
  ];

  if (permissionsLoading || (canRead && statusQuery.isLoading)) {
    return (
      <div aria-busy="true" aria-label={t("Loading")} className="space-y-3 p-5">
        <Skeleton className="h-8 w-1/2" />
        <Skeleton className="h-32 w-full" />
      </div>
    );
  }

  if (!canRead) {
    return (
      <div className="p-5">
        <Alert size="sm" variant="warning">
          <AlertDescription>
            {t("You do not have permission to view the accounting integration.")}
          </AlertDescription>
        </Alert>
      </div>
    );
  }

  const status = statusQuery.data;
  if (statusQuery.isError || !status) {
    return (
      <div className="p-5">
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
      </div>
    );
  }

  const connection = status.connection;
  if (!connection || !hasLiveAccountingConnection(connection)) {
    const connectedElsewhere =
      tenantSystem.connected &&
      tenantSystem.system !== null &&
      tenantSystem.system !== vendor.system
        ? accountingVendor(tenantSystem.system)
        : null;
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
          profile={status.profile}
          connection={connection ?? null}
          pendingCertificate={status.pendingCertificate}
          connectedElsewhere={connectedElsewhere}
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
        <div className="space-y-1.5">
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
        <div className="rounded-lg border p-4">
          <AccountingCompanyFacts connection={connection} />
        </div>
        <IntegrationSetupFooter>
          <Button type="button" onClick={onReviewed}>
            {needsAccountingMode(connection) ||
            needsAccountingMappings(connection) ||
            needsAccountingStartDate(connection)
              ? t("Continue")
              : t("Done")}
          </Button>
        </IntegrationSetupFooter>
      </IntegrationSetupWizard>
    );
  }

  if (needsAccountingMode(connection)) {
    return (
      <IntegrationSetupWizard steps={steps} activeStepId="mode" label={t("{0} setup", vendor.name)}>
        <AccountingModeStep
          vendor={vendor}
          profile={status.profile}
          connection={connection}
          canManage={canManage}
        />
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
      profile={status.profile}
      connection={connection}
      webhookSubscriptions={status.webhookSubscriptions}
      pendingCertificate={status.pendingCertificate}
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
