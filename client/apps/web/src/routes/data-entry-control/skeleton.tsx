import {
  SettingsFormSkeleton,
  type SettingsSectionSpec,
} from "@/components/settings-form-skeleton";

const sections: readonly SettingsSectionSpec[] = [
  { cols: 2, fields: [{ kind: "input", count: 4 }] },
];

export function DataEntryControlSkeleton() {
  return <SettingsFormSkeleton sections={sections} />;
}
