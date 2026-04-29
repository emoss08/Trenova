import type { LocationCodeStrategy, SequenceConfig, SequenceType } from "@/types/sequence-config";
import {
  BookOpenIcon,
  ClipboardEditIcon,
  FileTextIcon,
  LayersIcon,
  MapPinIcon,
  ReceiptIcon,
  TruckIcon,
  WrenchIcon,
  type LucideIcon,
} from "lucide-react";

export const sequenceTitles: Record<SequenceType, string> = {
  pro_number: "Pro Number",
  consolidation: "Consolidation Number",
  invoice: "Invoice Number",
  work_order: "Work Order Number",
  journal_batch: "Journal Batch Number",
  journal_entry: "Journal Entry Number",
  manual_journal_request: "Manual Journal Request",
  location_code: "Location Code",
};

export const sequenceDescriptions: Record<SequenceType, string> = {
  pro_number: "Controls generated PRO numbers for shipment creation and tracking.",
  consolidation: "Controls consolidation number generation for grouped shipment operations.",
  invoice: "Controls invoice identifier generation for billing documents.",
  work_order: "Controls work order identifier generation for operational workflows.",
  journal_batch: "Controls journal batch numbering for accounting posting groups.",
  journal_entry: "Controls journal entry numbering for posted ledger entries.",
  manual_journal_request: "Controls manual journal request numbering before approval and posting.",
  location_code: "Controls generated location codes assigned during location creation.",
};

export const sequenceIcons: Record<SequenceType, LucideIcon> = {
  pro_number: TruckIcon,
  consolidation: LayersIcon,
  work_order: WrenchIcon,
  invoice: ReceiptIcon,
  journal_batch: BookOpenIcon,
  journal_entry: FileTextIcon,
  manual_journal_request: ClipboardEditIcon,
  location_code: MapPinIcon,
};

export type SidebarGroup = {
  label: string;
  items: SequenceType[];
};

export const sidebarGroups: SidebarGroup[] = [
  { label: "Operations", items: ["pro_number", "consolidation", "work_order"] },
  { label: "Billing", items: ["invoice"] },
  { label: "Accounting", items: ["journal_batch", "journal_entry", "manual_journal_request"] },
  { label: "Locations", items: ["location_code"] },
];

export const separatorOptions = [
  { label: "None", value: "" },
  { label: "Hyphen (-)", value: "-" },
  { label: "Underscore (_)", value: "_" },
  { label: "Slash (/)", value: "/" },
  { label: "Period (.)", value: "." },
];

export const casingOptions = [
  { label: "Uppercase", value: "upper" },
  { label: "Lowercase", value: "lower" },
];

export const yearDigitsOptions = [
  { label: "2-digit year", value: 2, example: "e.g. 26" },
  { label: "4-digit year", value: 4, example: "e.g. 2026" },
] as const;

export const tokenLegend = [
  { token: "{P}", label: "Prefix" },
  { token: "{Y}", label: "Year" },
  { token: "{M}", label: "Month" },
  { token: "{W}", label: "Week" },
  { token: "{D}", label: "Day" },
  { token: "{L}", label: "Location" },
  { token: "{B}", label: "Business Unit" },
  { token: "{S}", label: "Sequence" },
  { token: "{R}", label: "Random" },
  { token: "{C}", label: "Check Digit" },
];

const defaultLocationCodeStrategy: LocationCodeStrategy = {
  components: ["name", "city", "state"],
  componentWidth: 3,
  sequenceDigits: 3,
  separator: "-",
  casing: "upper",
  fallbackPrefix: "LOC",
};

const defaultPrefixes: Record<SequenceType, string> = {
  pro_number: "PRO",
  consolidation: "CON",
  invoice: "INV",
  work_order: "WO",
  journal_batch: "JB",
  journal_entry: "JE",
  manual_journal_request: "MJR",
  location_code: "LOC",
};

export function defaultConfigForType(
  type: SequenceType,
  base: Pick<SequenceConfig, "id" | "organizationId" | "businessUnitId" | "version" | "createdAt" | "updatedAt">,
): SequenceConfig {
  return {
    ...base,
    sequenceType: type,
    prefix: defaultPrefixes[type],
    includeYear: type !== "location_code",
    yearDigits: 2,
    includeMonth: false,
    includeWeekNumber: false,
    includeDay: false,
    sequenceDigits: type === "location_code" ? 3 : 6,
    includeLocationCode: false,
    includeRandomDigits: false,
    randomDigitsCount: 0,
    includeCheckDigit: false,
    includeBusinessUnitCode: false,
    useSeparators: false,
    separatorChar: "-",
    allowCustomFormat: false,
    customFormat: "",
    locationCodeStrategy: type === "location_code" ? { ...defaultLocationCodeStrategy } : null,
  };
}
