import {
  SettingsFormSkeleton,
  type SettingsSectionSpec,
} from "@/components/settings-form-skeleton";

const sections: readonly SettingsSectionSpec[] = [
  { title: "sm", width: "full", cols: 2, fields: [{ kind: "input", count: 4 }] },
  { width: "full", switches: "right", cols: 2, fields: [{ kind: "switch", count: 2 }] },
  {
    width: "full",
    switches: "right",
    groups: [
      { cols: 2, fields: ["input"] },
      { cols: 2, fields: [{ kind: "switch", count: 3 }] },
    ],
  },
  { width: "full", cols: 2, fields: [{ kind: "input", count: 2 }] },
];

export function CarrierSettlementControlSkeleton() {
  return <SettingsFormSkeleton sections={sections} />;
}
