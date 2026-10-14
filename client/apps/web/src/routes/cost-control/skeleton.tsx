import {
  SettingsFormSkeleton,
  type SettingsSectionSpec,
} from "@/components/settings-form-skeleton";

const categoryRow = {
  cols: 2,
  fields: ["switch", "switch", { kind: "input", full: true }],
} as const;

const sections: readonly SettingsSectionSpec[] = [
  { title: "sm", narrow: true, fields: ["switch", "input", "switch", "input"] },
  { title: "sm", description: 2, width: "full", groups: [{ ...categoryRow, rows: 6 }] },
  { title: "sm", description: 2, width: "full", groups: [{ ...categoryRow, rows: 4 }] },
  { title: "sm", description: 2, narrow: true, fields: ["switch"] },
];

export function CostControlSkeleton() {
  return <SettingsFormSkeleton sections={sections} />;
}
