import type { GlobalSearchEntityType } from "@/services/global-search";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import {
  Building03Icon,
  File06Icon,
  InvoiceIcon,
  Truck01Icon,
  User01Icon,
} from "@trenova/shared/components/icons";
import type { RecordKind } from "@trenova/shared/types/generated/record-kinds";
import { Resource, type ResourceType } from "@trenova/shared/types/permission";
import type { PaletteIcon } from "./palette-model";

/**
 * The kinds of record the palette draws, and how each is drawn: every kind it
 * searches, and any kind an answer may cite that it can preview without
 * searching it. This table owns which kinds those are; a kind with a preview
 * is added here. The palette is neutral: a tile says what a row
 * is by its icon, never by a colour, so colour is left for the few states that
 * need someone.
 */
export const PALETTE_ENTITY_ICONS = {
  shipment: Truck01Icon,
  customer: Building03Icon,
  worker: User01Icon,
  document: File06Icon,
  invoice: InvoiceIcon,
} as const satisfies Record<GlobalSearchEntityType, PaletteIcon> &
  Partial<Record<RecordKind, PaletteIcon>>;

/** A kind of record the palette draws and previews. */
export type PaletteRecordType = keyof typeof PALETTE_ENTITY_ICONS;

export function isPaletteRecordType(value: string): value is PaletteRecordType {
  return Object.hasOwn(PALETTE_ENTITY_ICONS, value);
}

export interface PaletteEntityText {
  /** Singular, for a source line: "shipment". */
  label: string;
  /** Plural, for a tab and a group heading. */
  plural: string;
  /** The heading over every record of the type, when its scope is browsed. */
  all: string;
  /** What the search box asks for while the scope is set. */
  placeholder: string;
}

/** Whole phrases per type: a noun spliced into a sentence does not translate. */
export function paletteEntityText(type: GlobalSearchEntityType, t: TranslateFn): PaletteEntityText {
  switch (type) {
    case "shipment":
      return {
        label: t("Shipment"),
        plural: t("Shipments"),
        all: t("All shipments"),
        placeholder: t("PRO, BOL, customer or lane…"),
      };
    case "customer":
      return {
        label: t("Customer"),
        plural: t("Customers"),
        all: t("All customers"),
        placeholder: t("Name, code or city…"),
      };
    case "worker":
      return {
        label: t("Worker"),
        plural: t("Workers"),
        all: t("All workers"),
        placeholder: t("Name, role or home terminal…"),
      };
    case "document":
      return {
        label: t("Document"),
        plural: t("Documents"),
        all: t("All documents"),
        placeholder: t("File name, type or record…"),
      };
    case "invoice":
      return {
        label: t("Invoice"),
        plural: t("Invoices"),
        all: t("All invoices"),
        placeholder: t("Number, customer, PRO or PO…"),
      };
  }
}

/** The order record sections and tabs appear in. */
export const PALETTE_RECORD_TYPES: readonly GlobalSearchEntityType[] = [
  "shipment",
  "customer",
  "invoice",
  "worker",
  "document",
];

/**
 * The permission each kind the palette searches is read under, as the server's
 * search definition guards it. A kind the person may not read is never offered:
 * no tab, no `@` filter, and the server returns none of its records either.
 */
export const PALETTE_SEARCH_RESOURCES: Record<GlobalSearchEntityType, ResourceType> = {
  shipment: Resource.Shipment,
  customer: Resource.Customer,
  worker: Resource.Worker,
  document: Resource.Document,
  invoice: Resource.Invoice,
};

export const NEUTRAL_TILE_CLASS = "bg-sunken text-foreground-muted ring-border-subtle";
export const DANGER_TILE_CLASS = "bg-sunken text-danger ring-border-subtle";
