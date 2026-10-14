import {
  SettingsFormSkeleton,
  type SettingsSectionSpec,
} from "@/components/settings-form-skeleton";

const sections: readonly SettingsSectionSpec[] = [
  { width: "full", fields: ["input", "callout"] },
  { title: "lg", width: "full", cols: 3, fields: [{ kind: "input", count: 3 }] },
  { title: "lg", width: "full", cols: 3, fields: [{ kind: "input", count: 3 }] },
];

export function LeaveControlSkeleton() {
  return <SettingsFormSkeleton sections={sections} />;
}
