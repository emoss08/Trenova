import type { GlobalSearchEntityType } from "@/services/global-search";
import { FileTextIcon, TruckIcon, UserRoundIcon, BuildingIcon } from "lucide-react";
import type { PaletteIcon } from "./palette-model";

export interface PaletteEntityPresentation {
  icon: PaletteIcon;
  /** Singular, for a row's type and an action's object: "Copy shipment link". */
  label: string;
  /** Plural, for a tab and a group heading. */
  pluralLabel: string;
  /** The categorical accent the type is drawn in. Record types have no severity. */
  tileClass: string;
}

/**
 * How each searchable record type is drawn. The four types are categories,
 * not severities, so each takes a categorical accent: a shipment is teal
 * everywhere in the palette, and never because something is wrong with it.
 */
export const PALETTE_ENTITIES: Record<GlobalSearchEntityType, PaletteEntityPresentation> = {
  shipment: {
    icon: TruckIcon,
    label: "Shipment",
    pluralLabel: "Shipments",
    tileClass: "bg-accent-teal-subtle text-accent-teal-on-subtle ring-accent-teal-border",
  },
  customer: {
    icon: BuildingIcon,
    label: "Customer",
    pluralLabel: "Customers",
    tileClass: "bg-accent-violet-subtle text-accent-violet-on-subtle ring-accent-violet-border",
  },
  worker: {
    icon: UserRoundIcon,
    label: "Worker",
    pluralLabel: "Workers",
    tileClass: "bg-accent-sky-subtle text-accent-sky-on-subtle ring-accent-sky-border",
  },
  document: {
    icon: FileTextIcon,
    label: "Document",
    pluralLabel: "Documents",
    tileClass: "bg-accent-amber-subtle text-accent-amber-on-subtle ring-accent-amber-border",
  },
};

export const NEUTRAL_TILE_CLASS = "bg-sunken text-foreground-muted ring-border-subtle";
export const BRAND_TILE_CLASS = "bg-brand-subtle text-brand-subtle-foreground ring-brand-border";
