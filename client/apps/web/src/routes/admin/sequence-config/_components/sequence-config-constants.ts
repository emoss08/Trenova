import type {
  LocationCodeComponent,
  LocationCodeStrategy,
  SequenceConfig,
  SequenceType,
} from "@/types/sequence-config";
import { defineLabels } from "@trenova/shared/i18n/labels";

/** Each sequence by the records it numbers, as a section of the page names it. */
export const sequenceTitles: Record<SequenceType, string> = defineLabels({
  pro_number: "PRO numbers",
  consolidation: "Consolidation numbers",
  order: "Order numbers",
  invoice: "Invoice numbers",
  work_order: "Work order numbers",
  journal_batch: "Journal batch numbers",
  journal_entry: "Journal entry numbers",
  manual_journal_request: "Manual journal request numbers",
  location_code: "Location codes",
  driver_settlement: "Driver settlement numbers",
  carrier_settlement: "Carrier settlement numbers",
  invoice_run: "Invoice run numbers",
  factoring_submission: "Factoring schedule numbers",
});

export const sequenceDescriptions: Record<SequenceType, string> = defineLabels({
  pro_number: "The number each new shipment is given, on its paperwork and in tracking.",
  consolidation: "The number a group of consolidated shipments is given.",
  order: "The number each commercial order is given, whether entered or created with a shipment.",
  invoice: "The number printed on each invoice you send.",
  work_order: "The number each maintenance work order is given.",
  journal_batch: "The number each batch of journal entries is posted under.",
  journal_entry: "The number each posted journal entry is given.",
  manual_journal_request: "The number a manual journal entry carries while it waits for approval.",
  location_code: "The code each new location is given, built from its name and address.",
  driver_settlement: "The number each driver pay settlement is given.",
  carrier_settlement: "The number each settlement paying a carrier for a brokered load is given.",
  invoice_run: "The number each batch of invoices billed together is given.",
  factoring_submission: "The number each schedule of invoices sent to a factoring company is given.",
});

/**
 * The only sequence whose numbers are drawn with a location and a business unit at hand:
 * a PRO number comes from a shipment, whose first stop has a location. Every other number
 * is drawn without them, so asking for either there stops the number being issued at all.
 */
export function drawsRecordCodes(type: SequenceType): boolean {
  return type === "pro_number";
}

export const separatorChoices = [
  { label: "Hyphen ( - )", value: "-" },
  { label: "Underscore ( _ )", value: "_" },
  { label: "Slash ( / )", value: "/" },
  { label: "Period ( . )", value: "." },
];

/** A location code may join its parts with nothing at all. */
export const locationSeparatorChoices = [{ label: "None", value: "" }, ...separatorChoices];

export const casingLabels = defineLabels({ upper: "Uppercase", lower: "Lowercase" });

export const yearDigitOptions = [
  { label: "26", value: 2 },
  { label: "2026", value: 4 },
] as const;

export const locationComponentLabels: Record<LocationCodeComponent, string> = defineLabels({
  name: "Name",
  city: "City",
  state: "State",
  postal_code: "ZIP code",
});

/** The pattern's placeholders, by the letter between the braces, with what each stands for. */
export const patternTokenLabels = defineLabels({
  P: "Prefix",
  Y: "Year",
  M: "Month",
  W: "Week",
  D: "Day",
  S: "Counter",
  R: "Random digits",
  C: "Check digit",
  L: "Location code",
  B: "Business unit code",
});

export type PatternToken = keyof typeof patternTokenLabels;

/** The placeholders every pattern can use. */
export const patternTokens: readonly PatternToken[] = ["P", "Y", "M", "W", "D", "S", "R", "C"];

/** The placeholders a PRO number can also use, since a shipment carries both codes. */
export const recordCodeTokens: readonly PatternToken[] = ["L", "B"];

const defaultLocationCodeStrategy: LocationCodeStrategy = {
  components: ["name", "city", "state"],
  componentWidth: 3,
  sequenceDigits: 3,
  separator: "-",
  casing: "upper",
  fallbackPrefix: "LOC",
};

type DefaultFormat = Pick<SequenceConfig, "prefix" | "sequenceDigits"> &
  Partial<Pick<SequenceConfig, "includeRandomDigits" | "randomDigitsCount">>;

/** The server's own defaults (`DefaultSequenceFormat`), so "Use the default" means the same. */
const defaultFormats: Record<SequenceType, DefaultFormat> = {
  pro_number: { prefix: "S", sequenceDigits: 4, includeRandomDigits: true, randomDigitsCount: 6 },
  consolidation: { prefix: "C", sequenceDigits: 5 },
  order: { prefix: "O", sequenceDigits: 5 },
  invoice: { prefix: "INV", sequenceDigits: 6 },
  work_order: { prefix: "WO", sequenceDigits: 6 },
  journal_batch: { prefix: "JB", sequenceDigits: 6 },
  journal_entry: { prefix: "JE", sequenceDigits: 6 },
  manual_journal_request: { prefix: "MJR", sequenceDigits: 6 },
  invoice_run: { prefix: "IRUN", sequenceDigits: 5 },
  factoring_submission: { prefix: "FS", sequenceDigits: 5 },
  driver_settlement: { prefix: "STL", sequenceDigits: 6 },
  carrier_settlement: { prefix: "CSTL", sequenceDigits: 6 },
  location_code: { prefix: "LOC", sequenceDigits: 3 },
};

/** A sequence put back to the server's default format, keeping its identity and version. */
export function defaultConfigForType(
  type: SequenceType,
  base: Pick<
    SequenceConfig,
    "id" | "organizationId" | "businessUnitId" | "version" | "createdAt" | "updatedAt"
  >,
): SequenceConfig {
  const format = defaultFormats[type];
  const isLocation = type === "location_code";

  return {
    ...base,
    sequenceType: type,
    prefix: format.prefix,
    includeYear: !isLocation,
    yearDigits: 2,
    includeMonth: !isLocation,
    includeWeekNumber: false,
    includeDay: false,
    sequenceDigits: format.sequenceDigits,
    includeLocationCode: false,
    includeRandomDigits: format.includeRandomDigits ?? false,
    randomDigitsCount: format.randomDigitsCount ?? 0,
    includeCheckDigit: false,
    includeBusinessUnitCode: false,
    useSeparators: isLocation,
    separatorChar: isLocation ? "-" : "",
    allowCustomFormat: false,
    customFormat: "",
    locationCodeStrategy: isLocation ? { ...defaultLocationCodeStrategy } : null,
  };
}
