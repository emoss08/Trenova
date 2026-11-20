import { PageLayout } from "@/components/navigation/sidebar-layout";
import {
  SettingsTabsSkeleton,
  type SettingsSectionSpec,
} from "@/components/settings-form-skeleton";
import { LazyComponent } from "@trenova/shared/components/error-boundary";
import type { ReactElement, ReactNode } from "react";
import { useSettingsTab, type SettingsTabSpec } from "./settings-tab";

/**
 * A settings page's frame: its header, then a form that fills the rest of the page so the
 * tab row stays put and only the open tab scrolls (`SettingsTabs`). The form is lazy, and
 * its loading state is drawn in its place.
 */
export function SettingsPage({
  title,
  description,
  skeleton,
  children,
}: {
  title: string;
  description: string;
  skeleton: ReactElement;
  children: ReactNode;
}) {
  return (
    <PageLayout fill className="pb-0" pageHeaderProps={{ title, description }}>
      <LazyComponent fallback={skeleton}>{children}</LazyComponent>
    </PageLayout>
  );
}

/**
 * A tabbed settings page's loading state: its tab row and the open tab's sections, read from
 * the same tab spec as the form so the outline has the form's tabs. A section whose rows wait
 * on a switch is drawn as it stands with the switch off.
 */
export function SettingsPageSkeleton<TTab extends string>({
  tabs,
  sections,
}: {
  tabs: readonly (SettingsTabSpec & { value: TTab })[];
  sections: Readonly<Record<TTab, readonly SettingsSectionSpec[]>>;
}) {
  const [openTab] = useSettingsTab(tabs);

  return <SettingsTabsSkeleton tabs={tabs.length} sections={sections[openTab as TTab] ?? []} />;
}
