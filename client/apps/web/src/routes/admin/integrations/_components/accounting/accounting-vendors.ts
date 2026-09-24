import type { AccountingSystem } from "@trenova/graphql/generated/graphql";

export type AccountingVendor = {
  system: AccountingSystem;
  name: string;
  logoLight: string;
  logoDark: string;
  docsUrl: string;
};

export const quickBooksVendor: AccountingVendor = {
  system: "QuickBooksOnline",
  name: "QuickBooks Online",
  logoLight: "/integrations/logos/quickbooks-light.svg",
  logoDark: "/integrations/logos/quickbooks-dark.svg",
  docsUrl: "https://quickbooks.intuit.com/learn-support/",
};
