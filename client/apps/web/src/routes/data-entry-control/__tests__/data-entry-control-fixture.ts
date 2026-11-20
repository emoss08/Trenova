import type { DataEntryControl } from "@/types/data-entry-control";

/**
 * A saved data entry control as the server sends it (the Go entity's JSON), holding a new
 * organization's defaults (`NewDefaultDataEntryControl`).
 */
export const savedDataEntryControl: DataEntryControl = {
  id: "dec_01",
  version: 2,
  createdAt: 1_793_577_600,
  updatedAt: 1_793_577_600,
  organizationId: "org_01",
  businessUnitId: "bu_01",
  codeCase: "Upper",
  nameCase: "TitleCase",
  emailCase: "Lower",
  cityCase: "TitleCase",
};
