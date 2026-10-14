import type { ModuleId } from "@/config/navigation.types";

/**
 * The letter that follows G to jump to a module from anywhere: G then S opens
 * Shipments. Only modules with a letter people would guess carry one; the
 * rest are a click on the rail or a name typed into the palette.
 */
export const MODULE_GO_KEYS: Partial<Record<ModuleId, string>> = {
  home: "H",
  desk: "D",
  inbox: "T",
  intake: "I",
  shipment: "S",
  dispatch: "P",
  fleet: "E",
  fuel: "F",
  billing: "B",
  "carrier-settlements": "C",
  accounting: "A",
  reports: "R",
  admin: ",",
};

export const GO_CHORD_KEY = "G";
export const GO_CHORD_WINDOW_MS = 1500;
