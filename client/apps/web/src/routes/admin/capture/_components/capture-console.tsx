import { Kept } from "@/components/kept";
import {
  settingsColumnClass,
  settingsFillClass,
  settingsScrollBodyClass,
  settingsScrollClass,
  settingsTabRowClass,
  settingsTabsClass,
} from "@/components/settings/settings-layout";
import { useSettingsTab } from "@/components/settings/settings-tab";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Tabs, TabsList, TabsPanel, TabsTab } from "@trenova/shared/components/ui/tabs";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { Suspense, type ReactNode } from "react";
import { useCaptureTabs, type CaptureTab } from "../capture-tabs";
import { CaptureSettingsSkeleton } from "../skeleton";
import CaptureSettingsForm from "./capture-settings-form";
import { FleetPanel } from "./fleet-panel";
import { ProfilesPanel } from "./profiles-panel";

/**
 * A tab that is left goes off screen at once and only the arriving one slides in, as in
 * `SettingsTabs`: the tab library keeps a left tab mounted until its animations end, and an
 * arrival animation on it showed two tabs at once.
 */
const PANEL_CLASS = cn(
  settingsFillClass,
  "animate-slide-in data-ending-style:hidden data-[activation-direction=left]:animate-slide-in-back data-[activation-direction=none]:animate-none",
);

/** A tab that is not the settings form: its own scrolling region under the tab row. */
function TabScroll({ children }: { children: ReactNode }) {
  return (
    <ScrollArea className={settingsScrollClass}>
      <div className={cn(settingsColumnClass, settingsScrollBodyClass)}>{children}</div>
    </ScrollArea>
  );
}

/**
 * The Scanning and printing page: the capture settings (one record, one save), the scan
 * profiles (records, each saved from its own panel) and the paired computers (a list with a
 * way to stop one). Only the settings tab is a form, so creating a profile or revoking a
 * computer never submits the settings, and a settings save never waits on either.
 *
 * The tab row holds its place at the top of the page and only the open tab scrolls. The
 * settings stay mounted once opened, so changes not yet saved survive a look at another tab.
 */
export default function CaptureConsole() {
  const t = useT();
  const tabs = useCaptureTabs();
  const [open, select] = useSettingsTab(tabs);

  if (tabs.length === 0) {
    return (
      <p className="py-8 text-sm text-muted-foreground">
        {t("You don't have access to any part of scanning and printing.")}
      </p>
    );
  }

  const content = (tab: CaptureTab) => {
    switch (tab) {
      case "settings":
        return (
          <Kept shown={open === "settings"}>
            <Suspense fallback={<CaptureSettingsSkeleton />}>
              <CaptureSettingsForm />
            </Suspense>
          </Kept>
        );
      case "profiles":
        return (
          <TabScroll>
            <ProfilesPanel />
          </TabScroll>
        );
      case "computers":
        return (
          <TabScroll>
            <FleetPanel />
          </TabScroll>
        );
    }
  };

  return (
    <Tabs
      value={open}
      onValueChange={(next) => select(String(next))}
      className={settingsTabsClass}
    >
      {tabs.length > 1 ? (
        <div className={cn(settingsColumnClass, settingsTabRowClass)}>
          <TabsList variant="underline" aria-label={t("Scanning and printing")}>
            {tabs.map((tab) => (
              <TabsTab key={tab.value} value={tab.value}>
                {tab.label}
              </TabsTab>
            ))}
          </TabsList>
        </div>
      ) : null}
      {tabs.map((tab) => (
        <TabsPanel
          key={tab.value}
          value={tab.value}
          keepMounted={tab.value === "settings"}
          className={PANEL_CLASS}
        >
          {content(tab.value)}
        </TabsPanel>
      ))}
    </Tabs>
  );
}
