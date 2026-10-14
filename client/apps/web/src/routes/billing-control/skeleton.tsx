import {
  SettingsFormSkeleton,
  type SettingsSectionSpec,
} from "@/components/settings-form-skeleton";

const sections: readonly SettingsSectionSpec[] = [
  {
    cols: 2,
    fields: [
      { kind: "switch", count: 2 },
      { kind: "input", full: true },
      { kind: "textarea", full: true, count: 2 },
    ],
  },
  { narrow: true, fields: [{ kind: "input", count: 4 }] },
  { narrow: true, fields: [{ kind: "input", count: 2 }, "switch", { kind: "input", count: 2 }] },
  { title: "sm", narrow: true, fields: ["input", { kind: "switch", count: 2 }] },
  { title: "sm", narrow: true, fields: [{ kind: "input", count: 2 }] },
  { narrow: true, fields: ["switch"] },
  { narrow: true, fields: ["switch"] },
];

export function BillingControlSkeleton() {
  return <SettingsFormSkeleton sections={sections} />;
}
