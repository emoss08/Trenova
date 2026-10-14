import {
  SettingsFormSkeleton,
  type SettingsSectionSpec,
} from "@/components/settings-form-skeleton";

const sections: readonly SettingsSectionSpec[] = [
  { narrow: true, fields: [{ kind: "input", count: 3 }] },
  {
    title: "sm",
    narrow: true,
    fields: [{ kind: "input", count: 2 }, "switch", { kind: "input", count: 10 }],
  },
  { title: "lg", narrow: true, fields: [{ kind: "input", count: 6 }] },
  {
    title: "lg",
    narrow: true,
    fields: ["input", "switch", { kind: "input", count: 3 }, { kind: "switch", count: 2 }],
  },
  { narrow: true, fields: [{ kind: "input", count: 2 }] },
];

export function AccountingControlSkeleton() {
  return <SettingsFormSkeleton sections={sections} />;
}
