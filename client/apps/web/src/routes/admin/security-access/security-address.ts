import { appPath } from "@/lib/app-path";

/** Where the security and access page lives. */
export const SECURITY_ACCESS_PATH = appPath("/admin/security-access");

/** The page's tabs, in order; the open one is kept in the address under `tab`. */
export const securityTabValues = [
  "overview",
  "sign-in",
  "provisioning",
  "policies",
  "activity",
] as const;

export type SecurityTabValue = (typeof securityTabValues)[number];

/** The address of a tab of the security and access page, for a link from anywhere. */
export function securityTabHref(tab: SecurityTabValue): string {
  return tab === "overview" ? SECURITY_ACCESS_PATH : `${SECURITY_ACCESS_PATH}?tab=${tab}`;
}
