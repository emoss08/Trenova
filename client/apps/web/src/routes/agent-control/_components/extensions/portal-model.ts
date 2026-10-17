import type {
  PortalSite,
  PortalSiteFormValues,
  SavePortalSiteRequest,
} from "@/types/agent-extension";

/** The form's values for a site, or for a new one. A stored sign-in is never sent back. */
export function siteFormDefaults(site: PortalSite | null): PortalSiteFormValues {
  return {
    name: site?.name ?? "",
    purpose: site?.purpose ?? "",
    domains: site?.domains ?? [],
    startUrl: site?.startUrl ?? "",
    username: "",
    password: "",
    authenticatorKey: "",
    clearSignIn: false,
    clearAuthenticator: false,
    browserProfileId: site?.browserProfileId ?? "",
    allowActions: site?.allowActions ?? false,
    enabled: site?.enabled ?? true,
  };
}

/**
 * What the form saves. Addresses are hostnames, so a pasted URL keeps only its host; a
 * sign-in or authenticator key left blank keeps the stored one, and clearing the sign-in
 * clears the key with it.
 */
export function siteRequestOf(
  values: PortalSiteFormValues,
  site: PortalSite | null,
): SavePortalSiteRequest {
  return {
    name: values.name.trim(),
    purpose: values.purpose.trim(),
    domains: normalizeDomains(values.domains),
    startUrl: values.startUrl.trim(),
    username: values.clearSignIn ? "" : values.username.trim(),
    password: values.clearSignIn ? "" : values.password,
    authenticatorKey:
      values.clearSignIn || values.clearAuthenticator ? "" : values.authenticatorKey.trim(),
    clearSignIn: values.clearSignIn,
    clearAuthenticator: values.clearAuthenticator && !values.clearSignIn,
    browserProfileId: values.browserProfileId.trim(),
    allowActions: values.allowActions,
    enabled: values.enabled,
    version: site?.version ?? 0,
  };
}

/** Hostnames from what was typed: lower case, no scheme, sign-in, path or port, no repeats. */
export function normalizeDomains(entries: readonly string[]): string[] {
  const hosts: string[] = [];
  for (const entry of entries) {
    const host = hostOf(entry);
    if (host !== "" && !hosts.includes(host)) {
      hosts.push(host);
    }
  }
  return hosts;
}

function hostOf(entry: string): string {
  const trimmed = entry.trim().toLowerCase();
  if (trimmed === "") {
    return "";
  }
  const withoutScheme = trimmed.replace(/^[a-z]+:\/\//, "");
  const authority = withoutScheme.split(/[/?#]/, 1)[0] ?? "";
  const host = authority.slice(authority.lastIndexOf("@") + 1);
  return host.replace(/:\d+$/, "").replace(/\.$/, "");
}
