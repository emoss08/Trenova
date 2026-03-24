import { useAuthStore } from "@/stores/auth-store";

type MetadataProps = {
  title?: string;
  description?: string;
  keywords?: string;
  ogTitle?: string;
  ogDescription?: string;
  ogImage?: string;
  ogType?: string;
  canonical?: string;
  noIndex?: boolean;
};

const FALLBACK_NAME = "Trenova";
const DEFAULT_DESCRIPTION = "Transportation Management System";

function useAppName(): string {
  const { user, isLoading } = useAuthStore();

  if (isLoading) return "Loading...";

  if (user) {
    const membership = user.memberships?.find(
      (m) => m.organizationId === user.currentOrganizationId,
    );
    if (membership?.organization?.name) {
      return membership.organization.name;
    }
  }

  return FALLBACK_NAME;
}

export function Metadata({
  title,
  description = DEFAULT_DESCRIPTION,
  keywords,
  ogTitle,
  ogDescription,
  ogImage,
  ogType = "website",
  canonical,
  noIndex = false,
}: MetadataProps) {
  const appName = useAppName();
  const fullTitle = title ? `${title} | ${appName}` : appName;
  const resolvedOgTitle = ogTitle || fullTitle;
  const resolvedOgDescription = ogDescription || description;

  return (
    <>
      <title>{fullTitle}</title>
      <meta name="description" content={description} />
      {keywords && <meta name="keywords" content={keywords} />}
      {noIndex && <meta name="robots" content="noindex, nofollow" />}
      {canonical && <link rel="canonical" href={canonical} />}

      <meta property="og:title" content={resolvedOgTitle} />
      <meta property="og:description" content={resolvedOgDescription} />
      <meta property="og:type" content={ogType} />
      {ogImage && <meta property="og:image" content={ogImage} />}

      <meta name="twitter:card" content="summary_large_image" />
      <meta name="twitter:title" content={resolvedOgTitle} />
      <meta name="twitter:description" content={resolvedOgDescription} />
      {ogImage && <meta name="twitter:image" content={ogImage} />}
    </>
  );
}
