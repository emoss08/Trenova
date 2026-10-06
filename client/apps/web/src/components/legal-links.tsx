import type { LegalUrls } from "@/hooks/use-legal-urls";
import { useT } from "@trenova/shared/i18n/use-t";
import type { ReactNode } from "react";

/**
 * The terms and privacy documents as a phrase: "Terms of Service and Privacy policy",
 * each a link. A document the install names no address for is left out, so a
 * self-hosted install that publishes neither renders nothing rather than a dead link.
 */
export function LegalLinks({ urls, linkClassName }: { urls: LegalUrls; linkClassName?: string }) {
  const t = useT();
  const links: ReactNode[] = [];

  if (urls.termsUrl) {
    links.push(
      <a
        key="terms"
        href={urls.termsUrl}
        target="_blank"
        rel="noreferrer"
        className={linkClassName}
      >
        {t("Terms of Service")}
      </a>,
    );
  }
  if (urls.privacyUrl) {
    links.push(
      <a
        key="privacy"
        href={urls.privacyUrl}
        target="_blank"
        rel="noreferrer"
        className={linkClassName}
      >
        {t("Privacy policy")}
      </a>,
    );
  }

  if (links.length < 2) {
    return links[0] ?? null;
  }

  return (
    <>
      {links[0]} {t("and")} {links[1]}
    </>
  );
}

export function hasLegalUrls(urls: LegalUrls): boolean {
  return urls.termsUrl !== "" || urls.privacyUrl !== "";
}
