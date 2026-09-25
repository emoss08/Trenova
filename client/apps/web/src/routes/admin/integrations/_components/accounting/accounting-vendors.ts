import type { AccountingSystem } from "@trenova/graphql/generated/graphql";

export type AccountingVendor = {
  system: AccountingSystem;
  name: string;
  logoLight: string;
  logoDark: string;
  docsUrl: string;
  appName: string;
  developerPortalUrl: string;
  appKeysHelpUrl: string;
};

export const quickBooksVendor: AccountingVendor = {
  system: "QuickBooksOnline",
  name: "QuickBooks Online",
  logoLight: "/integrations/logos/quickbooks-light.svg",
  logoDark: "/integrations/logos/quickbooks-dark.svg",
  docsUrl: "https://quickbooks.intuit.com/learn-support/",
  appName: "Intuit app",
  developerPortalUrl: "https://developer.intuit.com/app/developer/dashboard",
  appKeysHelpUrl:
    "https://developer.intuit.com/app/developer/qbo/docs/get-started/get-client-id-and-client-secret",
};
