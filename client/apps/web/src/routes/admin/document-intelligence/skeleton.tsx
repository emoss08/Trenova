import {
  SettingsFormSkeleton,
  type SettingsSectionSpec,
} from "@/components/settings-form-skeleton";

const dependentSwitch = { kind: "switch", nested: true } as const;

const sections: readonly SettingsSectionSpec[] = [
  { fields: ["switch", dependentSwitch] },
  { title: "lg", fields: ["switch", dependentSwitch, "switch", dependentSwitch, "switch"] },
  {
    title: "lg",
    fields: ["switch", { kind: "checkbox", nested: true, outlined: true, count: 4 }],
  },
  { fields: ["switch"] },
];

export function DocumentIntelligenceSkeleton() {
  return <SettingsFormSkeleton sections={sections} />;
}
