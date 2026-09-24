import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { FileIcon } from "lucide-react";
import type { PaletteCommand, PalettePage, PaletteRecord } from "../palette-model";
import type { ActionContext } from "../palette-actions";

/** Passes English through and shows arguments, so a test can see what was interpolated. */
export const t: TranslateFn = (message, ...args) =>
  args.length === 0 ? (message ?? "") : `${message ?? ""}|${args.join(",")}`;

export function page(overrides: Partial<PalettePage> = {}): PalettePage {
  return {
    id: "billing:invoices",
    title: "Invoices",
    trail: "Billing > Invoices",
    href: "/billing/invoices",
    icon: FileIcon,
    keywords: ["Invoices", "Billing"],
    module: "Billing",
    ...overrides,
  };
}

export function command(overrides: Partial<PaletteCommand> = {}): PaletteCommand {
  return {
    id: "quick:create-shipment",
    label: "New shipment",
    description: "Add a new shipment",
    group: "create",
    icon: FileIcon,
    keywords: ["shipment"],
    intent: { type: "navigate", href: "/shipment-management/shipments?panelType=create" },
    ...overrides,
  };
}

export function record(overrides: Partial<PaletteRecord> = {}): PaletteRecord {
  return {
    entityType: "shipment",
    id: "shp_1",
    title: "PRO-1001",
    href: "/shipment-management/shipments?expanded=shp_1&panelType=edit&panelEntityId=shp_1",
    metadata: {},
    ...overrides,
  };
}

export function actionContext(overrides: Partial<ActionContext> = {}): ActionContext {
  return {
    mac: true,
    canReach: () => true,
    canUseAssistant: true,
    pinnedUrls: new Set(),
    ...overrides,
  };
}
