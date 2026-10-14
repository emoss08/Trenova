import {
  SettingsFormSkeleton,
  type SettingsSectionSpec,
} from "@/components/settings-form-skeleton";
import { useOrgCapabilities } from "@trenova/shared/hooks/use-org-capabilities";
import {
  hasOrganizationCapability,
  OrganizationCapability,
} from "@trenova/shared/types/organization-capability";

const assignmentSections: readonly SettingsSectionSpec[] = [
  { title: "lg", fields: ["switch"] },
  { title: "lg", description: 3, cols: 2, fields: [{ kind: "input", count: 13 }] },
];

const monitoringSections: readonly SettingsSectionSpec[] = [
  { title: "lg", fields: [{ kind: "input", narrow: false }] },
  { fields: ["switch"] },
  { narrow: true, fields: ["switch", { kind: "input", count: 2 }] },
  { narrow: true, fields: [{ kind: "input", count: 9 }] },
];

const complianceSections: readonly SettingsSectionSpec[] = [
  { title: "lg", fields: [{ kind: "switch", count: 5 }] },
];

const fleetSections: readonly SettingsSectionSpec[] = [
  ...assignmentSections,
  ...monitoringSections,
  ...complianceSections,
];

/** The form's cards; assignment and DOT compliance only for an organization running a fleet. */
export function DispatchControlSkeleton() {
  const capabilities = useOrgCapabilities();
  const runsAssets = hasOrganizationCapability(
    capabilities,
    OrganizationCapability.AssetOperations,
  );

  return <SettingsFormSkeleton sections={runsAssets ? fleetSections : monitoringSections} />;
}
