import {
  SettingsFormSkeleton,
  type SettingsSectionSpec,
} from "@/components/settings-form-skeleton";

const sections: readonly SettingsSectionSpec[] = [
  { title: "lg", fields: ["switch"] },
  { title: "lg", fields: ["input", { kind: "switch", count: 3 }] },
  { title: "lg", fields: ["switch"] },
  { fields: ["switch"] },
  { fields: [{ kind: "switch", count: 2 }] },
];

export function ShipmentControlSkeleton() {
  return <SettingsFormSkeleton sections={sections} />;
}
