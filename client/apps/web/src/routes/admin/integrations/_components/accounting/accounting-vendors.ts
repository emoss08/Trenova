import type { AccountingSystem } from "@trenova/graphql/generated/graphql";

export type AccountingVendor = {
  system: AccountingSystem;
  name: string;
  logoLight: string;
  logoDark: string;
  docsUrl: string;
  appName: string;
  developer: string;
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
  developer: "Intuit",
  developerPortalUrl: "https://developer.intuit.com/app/developer/dashboard",
  appKeysHelpUrl:
    "https://developer.intuit.com/app/developer/qbo/docs/get-started/get-client-id-and-client-secret",
};

export const xeroVendor: AccountingVendor = {
  system: "Xero",
  name: "Xero",
  logoLight: "/integrations/logos/xero-light.svg",
  logoDark: "/integrations/logos/xero-dark.svg",
  docsUrl: "https://central.xero.com/",
  appName: "Xero app",
  developer: "Xero",
  developerPortalUrl: "https://developer.xero.com/app/manage",
  appKeysHelpUrl: "https://developer.xero.com/documentation/guides/oauth2/auth-flow/",
};

export const businessCentralVendor: AccountingVendor = {
  system: "BusinessCentral",
  name: "Business Central",
  logoLight: "/integrations/logos/business-central-light.svg",
  logoDark: "/integrations/logos/business-central-dark.svg",
  docsUrl: "https://learn.microsoft.com/en-us/dynamics365/business-central/",
  appName: "Microsoft Entra app",
  developer: "Microsoft Entra",
  developerPortalUrl:
    "https://entra.microsoft.com/#view/Microsoft_AAD_RegisteredApps/ApplicationsListBlade",
  appKeysHelpUrl:
    "https://learn.microsoft.com/en-us/dynamics365/business-central/dev-itpro/developer/devenv-develop-connect-apps",
};

const ACCOUNTING_VENDORS: Record<AccountingSystem, AccountingVendor> = {
  QuickBooksOnline: quickBooksVendor,
  Xero: xeroVendor,
  BusinessCentral: businessCentralVendor,
};

export const accountingVendors: readonly AccountingVendor[] = Object.values(ACCOUNTING_VENDORS);

export function accountingVendor(system: AccountingSystem): AccountingVendor {
  return ACCOUNTING_VENDORS[system];
}
