import {
  SettingsFormSkeleton,
  type SettingsSectionSpec,
} from "@/components/settings-form-skeleton";

const sections: readonly SettingsSectionSpec[] = [
  { title: "sm", width: "full", cols: 2, fields: [{ kind: "input", count: 4 }] },
  { width: "full", switches: "right", cols: 2, fields: [{ kind: "switch", count: 5 }] },
  { width: "full", cols: 2, fields: [{ kind: "input", count: 2 }] },
  { width: "full", cols: 2, fields: [{ kind: "input", count: 2 }] },
];

export function SettlementControlSkeleton() {
  return <SettingsFormSkeleton sections={sections} />;
}
