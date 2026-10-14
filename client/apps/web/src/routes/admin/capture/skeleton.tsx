import {
  SettingsFormSkeleton,
  type SettingsSectionSpec,
} from "@/components/settings-form-skeleton";

const sections: readonly SettingsSectionSpec[] = [
  { title: "sm", fields: ["switch"] },
  { title: "sm", fields: ["switch", "input"] },
  { title: "lg", fields: ["input", "switch"] },
];

export function CaptureSettingsSkeleton() {
  return <SettingsFormSkeleton sections={sections} />;
}
