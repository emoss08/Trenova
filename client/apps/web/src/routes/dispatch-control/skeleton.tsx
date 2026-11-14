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

/**
 * Each tab's sections as the form draws them. A section whose fields wait on a switch is
 * drawn as it stands with the switch off, since the outline cannot know the saved value.
 */
const TAB_SECTIONS: Record<DispatchControlTab, readonly SettingsSectionSpec[]> = {
  assignment: [{ title: "lg", description: 4, fields: ["switch"] }],
  costs: [
    { title: "md", description: 4, cols: 2, fields: [{ kind: "input", count: 10 }] },
    { title: "sm", description: 3, cols: 2, fields: [{ kind: "input", count: 7 }] },
    { title: "lg", description: 4, cols: 2, fields: [{ kind: "input", count: 4 }] },
  ],
  learning: [
    { title: "lg", description: 5, fields: ["switch", { kind: "input", count: 2 }] },
    { title: "sm", description: 2, fields: [{ kind: "switch", count: 2 }] },
    { title: "md", description: 5, fields: [{ kind: "input", count: 2 }] },
  ],
  monitoring: [
    { title: "lg", description: 4, cols: 2, fields: ["input"] },
    { title: "md", description: 5, fields: ["switch"] },
    {
      title: "md",
      description: 6,
      groups: [{ fields: ["switch"] }, { cols: 2, fields: [{ kind: "input", count: 2 }] }],
    },
  ],
  exceptions: [
    { title: "sm", description: 3, cols: 2, fields: [{ kind: "input", count: 3 }] },
    { title: "sm", description: 3, cols: 2, fields: [{ kind: "input", count: 3 }] },
    { title: "sm", description: 2, cols: 2, fields: ["input"] },
    { title: "md", description: 3, cols: 2, fields: [{ kind: "input", count: 2 }] },
  ],
  tenders: [
    { title: "md", description: 2, fields: ["switch"] },
    { title: "md", description: 2, cols: 2, fields: ["input"] },
  ],
  compliance: [
    { title: "lg", description: 4, fields: ["switch"] },
    { title: "md", description: 3, fields: [{ kind: "switch", count: 4 }] },
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
