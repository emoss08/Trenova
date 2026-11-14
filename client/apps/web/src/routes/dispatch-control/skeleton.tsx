import { useSettingsTab } from "@/components/settings/settings-tab";
import {
  SettingsTabsSkeleton,
  type SettingsSectionSpec,
} from "@/components/settings-form-skeleton";
import { useOrgCapabilities } from "@trenova/shared/hooks/use-org-capabilities";
import {
  hasOrganizationCapability,
  OrganizationCapability,
} from "@trenova/shared/types/organization-capability";
import { dispatchControlTabs, type DispatchControlTab } from "./dispatch-control-tabs";

const figures = (count: number) => ({ kind: "input", count, narrow: true }) as const;
const choice = { kind: "input", narrow: false } as const;

/**
 * Each tab's sections as the form draws them: one row to a setting, a figure's row with a
 * compact field (`narrow`) and a choice's with a wider one. A section whose rows wait on a
 * switch is drawn as it stands with the switch off, since the outline cannot know the
 * saved value.
 */
const TAB_SECTIONS: Record<DispatchControlTab, readonly SettingsSectionSpec[]> = {
  assignment: [{ title: "lg", description: 2, fields: ["switch"] }],
  costs: [
    { title: "md", description: 2, fields: [figures(10)] },
    { title: "sm", description: 1, fields: [figures(7)] },
    { title: "lg", description: 2, fields: [figures(4)] },
  ],
  learning: [
    { title: "lg", description: 2, fields: ["switch", figures(2)] },
    { title: "sm", description: 1, fields: [{ kind: "switch", count: 2 }] },
    { title: "md", description: 2, fields: [figures(2)] },
  ],
  monitoring: [
    { title: "lg", description: 2, fields: [choice] },
    { title: "md", description: 2, fields: ["switch"] },
    { title: "md", description: 2, fields: ["switch", figures(2)] },
  ],
  exceptions: [
    { title: "sm", description: 1, fields: [figures(3)] },
    { title: "sm", description: 1, fields: [figures(2), choice] },
    { title: "sm", description: 1, fields: [choice] },
    { title: "md", description: 1, fields: [figures(2)] },
  ],
  tenders: [
    { title: "md", description: 1, fields: ["switch"] },
    { title: "md", description: 1, fields: [choice] },
  ],
  compliance: [
    { title: "lg", description: 2, fields: ["switch"] },
    { title: "md", description: 1, fields: [{ kind: "switch", count: 4 }] },
  ],
};

/**
 * The form's tab row and the open tab's sections. Which tabs there are follows the
 * organization, as in the form: a fleet's planning tabs, a brokerage's carrier tenders.
 */
export function DispatchControlSkeleton() {
  const capabilities = useOrgCapabilities();
  const tabs = dispatchControlTabs({
    runsAssets: hasOrganizationCapability(capabilities, OrganizationCapability.AssetOperations),
    brokers: hasOrganizationCapability(capabilities, OrganizationCapability.Brokerage),
  });
  const [openTab] = useSettingsTab(tabs);

  return (
    <SettingsTabsSkeleton
      tabs={tabs.length}
      sections={TAB_SECTIONS[openTab as DispatchControlTab] ?? []}
    />
  );
}
