import {
  SettingsFormSkeleton,
  type SettingsSectionSpec,
} from "@/components/settings-form-skeleton";

const sections: readonly SettingsSectionSpec[] = [
  { narrow: true, fields: [{ kind: "switch", count: 3 }, "input"] },
  { fields: ["switch"] },
  { title: "lg", width: "full", cols: 2, fields: [{ kind: "input", count: 8 }] },
];

export function DistanceControlSkeleton() {
  return <SettingsFormSkeleton sections={sections} />;
}
