import type { SettingsTabSpec } from "@/components/settings/settings-tab";
import type { SequenceConfigDocument, SequenceType } from "@/types/sequence-config";
import { defineLabels } from "@trenova/shared/i18n/labels";
import type { FieldPath } from "react-hook-form";

export type SequenceConfigTab = "shipments" | "billing" | "accounting" | "settlements" | "locations";

export type SequenceConfigField = FieldPath<SequenceConfigDocument>;

export type SequenceConfigTabSpec = SettingsTabSpec<SequenceConfigField> & {
  value: SequenceConfigTab;
};

const TAB_LABELS = defineLabels({
  shipments: "Shipments",
  billing: "Billing",
  accounting: "Accounting",
  settlements: "Settlements",
  locations: "Locations",
});

/** The sequences each tab holds, in the order their sections stand. */
export const TAB_SEQUENCES: Record<SequenceConfigTab, readonly SequenceType[]> = {
  shipments: ["pro_number", "order"],
  billing: ["invoice", "invoice_run", "factoring_submission"],
  accounting: ["journal_batch", "journal_entry", "manual_journal_request"],
  settlements: ["driver_settlement", "carrier_settlement"],
  locations: ["location_code"],
};

/**
 * Sequences the server keeps but nothing issues a number from: there are no consolidations
 * or work orders yet. They are saved as they are and not offered, since a format for them
 * would change nothing.
 */
export const UNISSUED_SEQUENCES: readonly SequenceType[] = ["consolidation", "work_order"];

const TAB_ORDER: readonly SequenceConfigTab[] = [
  "shipments",
  "billing",
  "accounting",
  "settlements",
  "locations",
];

/** Where each sequence sits in the document the server sent, which is its own order. */
export function indexBySequence(
  configs: readonly { sequenceType: SequenceType }[],
): Partial<Record<SequenceType, number>> {
  return Object.fromEntries(configs.map((config, index) => [config.sequenceType, index]));
}

/**
 * The tabs, in order, each holding the sequences of one part of the business. A tab names
 * the entries of the document's list it holds, so a refused save opens the tab at fault;
 * the loading state, which has no document yet, reads the same tabs without them.
 */
export function sequenceConfigTabs(
  configs: readonly { sequenceType: SequenceType }[] = [],
): SequenceConfigTabSpec[] {
  const positions = indexBySequence(configs);

  return TAB_ORDER.map((tab) => ({
    value: tab,
    label: TAB_LABELS[tab],
    fields: TAB_SEQUENCES[tab].flatMap((type) => {
      const index = positions[type];
      return index === undefined ? [] : [`configs.${index}` as SequenceConfigField];
    }),
  }));
}
