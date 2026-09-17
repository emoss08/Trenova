const LOGO_CDN = "https://cdn.brandfetch.io";

const DOMAIN_PATTERN =
  /^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?(?:\.[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?)+$/;

export type BrandfetchLogoOptions = {
  width: number;
  height?: number;
  /** Defaults to VITE_BRANDFETCH_CLIENT_ID. The client id is public; no API key is ever used here. */
  clientId?: string;
};

export function brandfetchClientId(): string | undefined {
  const value = import.meta.env.VITE_BRANDFETCH_CLIENT_ID as string | undefined;
  return value?.trim() || undefined;
}

export function normalizeBrandDomain(input: string): string | null {
  const trimmed = input.trim().toLowerCase();
  if (trimmed === "") {
    return null;
  }

  const withoutScheme = trimmed.replace(/^[a-z][a-z0-9+.-]*:\/\//, "");
  const host = withoutScheme.split(/[/?#]/, 1)[0] ?? "";

  return DOMAIN_PATTERN.test(host) ? host : null;
}

export function brandfetchLogoUrl(
  domain: string,
  { width, height = width, clientId = brandfetchClientId() }: BrandfetchLogoOptions,
): string | null {
  const id = clientId?.trim();
  if (!id) {
    return null;
  }

  const host = normalizeBrandDomain(domain);
  if (!host) {
    return null;
  }

  const w = Math.max(1, Math.round(width));
  const h = Math.max(1, Math.round(height));

  return `${LOGO_CDN}/${host}/w/${w}/h/${h}?c=${encodeURIComponent(id)}`;
}
