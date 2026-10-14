import {
  SettingsFormSkeleton,
  type SettingsSectionSpec,
} from "@/components/settings-form-skeleton";

const sections: readonly SettingsSectionSpec[] = [
  { narrow: true, fields: [{ kind: "input", count: 7 }] },
  { title: "lg", narrow: true, fields: ["input"] },
  { narrow: true, fields: [{ kind: "input", count: 2 }, "switch"] },
  { narrow: true, fields: [{ kind: "input", count: 3 }] },
];

export function InvoiceAdjustmentControlSkeleton() {
  return <SettingsFormSkeleton sections={sections} />;
}
