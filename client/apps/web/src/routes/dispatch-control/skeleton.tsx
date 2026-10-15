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

const tenderSections: readonly SettingsSectionSpec[] = [
  {
    title: "lg",
    description: 2,
    fields: ["switch", { kind: "input", narrow: true }, "switch", { kind: "input", count: 5, narrow: true }],
  },
];

/**
 * The form's cards; assignment and DOT compliance only for an organization running a fleet,
 * carrier tenders only for a brokerage.
 */
export function DispatchControlSkeleton() {
  const capabilities = useOrgCapabilities();
  const runsAssets = hasOrganizationCapability(
    capabilities,
    OrganizationCapability.AssetOperations,
  );
  const brokers = hasOrganizationCapability(capabilities, OrganizationCapability.Brokerage);

  return (
    <SettingsFormSkeleton
      sections={[
        ...(runsAssets ? assignmentSections : []),
        ...monitoringSections,
        ...(brokers ? tenderSections : []),
        ...(runsAssets ? complianceSections : []),
      ]}
    />
  );
}
