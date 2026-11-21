import { SettingsPageSkeleton } from "@/components/settings/settings-page";
import type { SettingsSectionSpec } from "@/components/settings-form-skeleton";
import { sequenceConfigTabs, type SequenceConfigTab } from "./sequence-config-tabs";

const choice = { kind: "input", narrow: false } as const;
const figure = { kind: "input", narrow: true } as const;

/**
 * A sequence on the server's default format: its sample, prefix and own-pattern switch,
 * the year with its digits, the month, week, day and separators, then the counter, random
 * digits and check digit. The sample is drawn as a row of its height.
 */
const sequence: SettingsSectionSpec = {
  title: "md",
  description: 1,
  fields: [choice, choice, "switch", "switch", choice, "switch", "switch", "switch", "switch", figure, "switch", "switch"],
};

/** A PRO number adds the business unit and location codes, and its random digits are on. */
const proNumber: SettingsSectionSpec = {
  title: "md",
  description: 1,
  fields: [
    choice,
    choice,
    "switch",
    "switch",
    "switch",
    choice,
    "switch",
    "switch",
    "switch",
    "switch",
    "switch",
    figure,
    "switch",
    figure,
    "switch",
  ],
};

const locationCode: SettingsSectionSpec = {
  title: "md",
  description: 1,
  fields: [choice, choice, figure, figure, choice, choice, choice],
};

/** Each tab's sequences as the form draws them on a new organization's formats. */
const TAB_SECTIONS: Record<SequenceConfigTab, readonly SettingsSectionSpec[]> = {
  shipments: [proNumber, sequence],
  billing: [sequence, sequence, sequence],
  accounting: [sequence, sequence, sequence],
  settlements: [sequence, sequence],
  locations: [locationCode],
};

/** The form's tab row and the open tab's sequences. */
export function SequenceConfigSkeleton() {
  return <SettingsPageSkeleton tabs={sequenceConfigTabs()} sections={TAB_SECTIONS} />;
}
