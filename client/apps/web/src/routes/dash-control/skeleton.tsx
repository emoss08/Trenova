import {
  SettingsFormSkeleton,
  type SettingsSectionSpec,
} from "@/components/settings-form-skeleton";

const sections: readonly SettingsSectionSpec[] = [
  { width: "full", switches: "right", cols: 2, fields: [{ kind: "switch", count: 5 }] },
  { width: "full", switches: "right", cols: 2, fields: [{ kind: "switch", count: 2 }] },
  { width: "full", switches: "right", cols: 2, fields: [{ kind: "switch", count: 3 }] },
  { width: "full", switches: "right", cols: 2, fields: [{ kind: "switch", count: 4 }] },
  {
    width: "full",
    switches: "right",
    cols: 2,
    fields: ["switch", { kind: "input", count: 2 }, "switch", "input"],
  },
];

export function DashControlSkeleton() {
  return <SettingsFormSkeleton sections={sections} />;
}
