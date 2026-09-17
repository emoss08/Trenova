import { usePermission } from "@/hooks/use-permission";
import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import { useTheme } from "@trenova/shared/components/theme-provider";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Tabs, TabsList, TabsPanel, TabsTab } from "@trenova/shared/components/ui/tabs";
import { useT } from "@trenova/shared/i18n/use-t";
import { Operation, Resource } from "@trenova/shared/types/permission";
import type { TableSheetProps } from "@trenova/shared/types/data-table";
import { LazyImage } from "@/components/image";
import { ExternalLink } from "@/components/link";
import { useEffect, useState } from "react";
import type { CarrierIntelSettingsTab } from "./carrier-intel-settings-schema";
import { CarrierIntelSettingsForm } from "./carrier-intel-settings-form";
import {
  carrierOKVendor,
  fmcsaQCMobileVendor,
  type CarrierIntelVendor,
} from "./carrier-intelligence-vendors";
import { CarrierIntelConnectionTab } from "./connection-tab";

type ModalTab = "connection" | CarrierIntelSettingsTab;

const settingsTabs: { value: CarrierIntelSettingsTab; label: string }[] = [
  { value: "rules", label: "Rules" },
  { value: "monitoring", label: "Monitoring" },
  { value: "spend", label: "Spend" },
];

export function CarrierOKIntegrationModal({ open, onOpenChange }: TableSheetProps) {
  return (
    <CarrierIntelligenceModal vendor={carrierOKVendor} open={open} onOpenChange={onOpenChange} />
  );
}

export function FMCSAQCMobileIntegrationModal({ open, onOpenChange }: TableSheetProps) {
  return (
    <CarrierIntelligenceModal
      vendor={fmcsaQCMobileVendor}
      open={open}
      onOpenChange={onOpenChange}
    />
  );
}

export function CarrierIntelligenceModal({
  vendor,
  open,
  onOpenChange,
}: TableSheetProps & { vendor: CarrierIntelVendor }) {
  const t = useT();
  const { theme } = useTheme();
  const [activeTab, setActiveTab] = useState<ModalTab>("connection");

  const { allowed: canRead, isLoading: permissionsLoading } = usePermission(
    Resource.CarrierIntelligence,
    Operation.Read,
  );
  const { allowed: canManage } = usePermission(Resource.CarrierIntelligence, Operation.Manage);

  const settingsQuery = useQuery({
    ...queries.carrierIntelSettings.settings(),
    enabled: open && canRead,
  });

  useEffect(() => {
    if (!open) {
      setActiveTab("connection");
    }
  }, [open]);

  const close = () => onOpenChange(false);
  const logo = theme === "dark" ? (vendor.logoDark ?? vendor.logoLight) : vendor.logoLight;
  const isSettingsTab = activeTab !== "connection";

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="flex max-h-[90vh] flex-col overflow-hidden sm:max-w-4xl">
        <DialogHeader className="flex flex-row items-center gap-3 pr-8">
          {logo ? (
            <LazyImage
              src={logo}
              alt={`${vendor.name} Logo`}
              className="h-8 max-w-24 object-contain"
            />
          ) : null}
          <div className="min-w-0 space-y-0.5">
            <DialogTitle>{t(vendor.headline)}</DialogTitle>
            <DialogDescription className="flex flex-wrap items-center gap-1 text-xs">
              <span>{t(vendor.blurb)}</span>
              <ExternalLink href={vendor.docsUrl} className="text-xs">
                {t(vendor.docsLabel)}
              </ExternalLink>
            </DialogDescription>
          </div>
        </DialogHeader>
        <Tabs
          value={activeTab}
          onValueChange={(value) => setActiveTab(value as ModalTab)}
          className="flex min-h-0 flex-1 flex-col"
        >
          <TabsList className="w-full">
            <TabsTab value="connection">{t("Connection")}</TabsTab>
            {settingsTabs.map((tab) => (
              <TabsTab key={tab.value} value={tab.value} disabled={!canRead}>
                {t(tab.label)}
              </TabsTab>
            ))}
          </TabsList>
          <TabsPanel value="connection" className="min-h-0 flex-1 overflow-y-auto px-1 py-2">
            <CarrierIntelConnectionTab
              vendor={vendor}
              open={open}
              onClose={close}
              settings={settingsQuery.data}
              settingsLoading={canRead && settingsQuery.isLoading}
              canRead={canRead}
              canManage={canManage}
            />
          </TabsPanel>
          {isSettingsTab ? (
            <SettingsTabsBody
              canRead={canRead}
              permissionsLoading={permissionsLoading}
              isLoading={settingsQuery.isLoading}
              isError={settingsQuery.isError && !settingsQuery.data}
              onRetry={() => void settingsQuery.refetch()}
            />
          ) : null}
          {settingsQuery.data && canRead ? (
            <CarrierIntelSettingsForm
              settings={settingsQuery.data}
              open={open}
              activeTab={activeTab}
              canManage={canManage}
              onTabChange={setActiveTab}
              onClose={close}
            />
          ) : null}
        </Tabs>
      </DialogContent>
    </Dialog>
  );
}

function SettingsTabsBody({
  canRead,
  permissionsLoading,
  isLoading,
  isError,
  onRetry,
}: {
  canRead: boolean;
  permissionsLoading: boolean;
  isLoading: boolean;
  isError: boolean;
  onRetry: () => void;
}) {
  const t = useT();

  if (permissionsLoading || (canRead && isLoading)) {
    return (
      <div className="space-y-3 py-2">
        <Skeleton className="h-8 w-1/3" />
        <Skeleton className="h-40 w-full" />
        <Skeleton className="h-40 w-full" />
      </div>
    );
  }

  if (!canRead) {
    return (
      <div className="border-border bg-muted/20 text-muted-foreground rounded-md border p-4 text-sm">
        {t("You do not have permission to view carrier intelligence settings.")}
      </div>
    );
  }

  if (isError) {
    return (
      <div className="border-border flex items-center justify-between gap-3 rounded-md border p-4 text-sm">
        <span className="text-destructive">
          {t("Carrier intelligence settings could not be loaded.")}
        </span>
        <Button type="button" size="sm" variant="outline" onClick={onRetry}>
          {t("Retry")}
        </Button>
      </div>
    );
  }

  return null;
}
