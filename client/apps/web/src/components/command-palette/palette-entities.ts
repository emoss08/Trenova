import type { GlobalSearchEntityType } from "@/services/global-search";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import {
  Building03Icon,
  File06Icon,
  Truck01Icon,
  User01Icon,
} from "@trenova/shared/components/icons";
import type { PaletteIcon } from "./palette-model";

/**
 * How each searchable record type is drawn. The palette is neutral: a tile
 * says what a row is by its icon, never by a colour, so colour is left for
 * the few states that need someone.
 */
export const PALETTE_ENTITY_ICONS: Record<GlobalSearchEntityType, PaletteIcon> = {
  shipment: Truck01Icon,
  customer: Building03Icon,
  worker: User01Icon,
  document: File06Icon,
};

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
  }
}

/** The order record sections and tabs appear in. */
export const PALETTE_RECORD_TYPES: readonly GlobalSearchEntityType[] = [
  "shipment",
  "customer",
  "worker",
  "document",
];

export const NEUTRAL_TILE_CLASS = "bg-sunken text-foreground-muted ring-border-subtle";
export const DANGER_TILE_CLASS = "bg-sunken text-danger ring-border-subtle";
