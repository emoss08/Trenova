import { PageLayout } from "@/components/navigation/sidebar-layout";
import { usePermission } from "@/hooks/use-permission";
import { SuspenseLoader } from "@trenova/shared/components/component-loader";
import { Tabs, TabsContent, TabsList, TabsTab } from "@trenova/shared/components/ui/tabs";
import { useT } from "@trenova/shared/i18n/use-t";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { MonitorIcon, SettingsIcon, SlidersHorizontalIcon } from "lucide-react";
import { parseAsStringLiteral, useQueryState } from "nuqs";
import { lazy } from "react";
import { FleetPanel } from "./_components/fleet-panel";
import { ProfilesPanel } from "./_components/profiles-panel";

const CaptureSettingsForm = lazy(() => import("./_components/capture-settings-form"));

const TABS = ["settings", "profiles", "computers"] as const;
type CaptureTab = (typeof TABS)[number];

/**
 * Scanning and printing, for the people who run it: whether it is on, the
 * scan profiles people pick from, and every computer paired to scan. Each part
 * is shown to whoever may see it.
 */
export function CaptureAdminPage() {
  const t = useT();
  const { allowed: canSettings } = usePermission(Resource.DocumentControl, Operation.Read);
  const { allowed: canProfiles } = usePermission(Resource.CaptureProfile, Operation.Read);
  const { allowed: canComputers } = usePermission(Resource.CaptureDevice, Operation.Read);

  const visible = TABS.filter((tab) =>
    tab === "settings" ? canSettings : tab === "profiles" ? canProfiles : canComputers,
  );
  const [requested, setTab] = useQueryState(
    "tab",
    parseAsStringLiteral(TABS).withDefault(visible[0] ?? "profiles"),
  );
  const tab: CaptureTab = visible.includes(requested) ? requested : (visible[0] ?? "profiles");

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Scanning and printing"),
        description: t(
          "Turn on Trenova Capture, set the scan profiles people pick from, and see every computer paired to scan",
        ),
      }}
    >
      <Tabs value={tab} onValueChange={(value) => void setTab(value as CaptureTab)}>
        <TabsList variant="underline">
          {canSettings && (
            <TabsTab value="settings">
              <SettingsIcon size={16} />
              {t("Settings")}
            </TabsTab>
          )}
          {canProfiles && (
            <TabsTab value="profiles">
              <SlidersHorizontalIcon size={16} />
              {t("Scan profiles")}
            </TabsTab>
          )}
          {canComputers && (
            <TabsTab value="computers">
              <MonitorIcon size={16} />
              {t("Computers")}
            </TabsTab>
          )}
        </TabsList>
        {canSettings && (
          <TabsContent value="settings" className="pt-4">
            <SuspenseLoader>
              <CaptureSettingsForm />
            </SuspenseLoader>
          </TabsContent>
        )}
        {canProfiles && (
          <TabsContent value="profiles" className="pt-4">
            <ProfilesPanel />
          </TabsContent>
        )}
        {canComputers && (
          <TabsContent value="computers" className="pt-4">
            <FleetPanel />
          </TabsContent>
        )}
      </Tabs>
    </PageLayout>
  );
}
