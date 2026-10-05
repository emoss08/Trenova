import { LegalLinks, hasLegalUrls } from "@/components/legal-links";
import { useLegalUrls } from "@/hooks/use-legal-urls";
import { useT } from "@trenova/shared/i18n/use-t";

/**
 * The line under the sign-in and reset cards that names the documents continuing
 * agrees to. Nothing when the install publishes neither.
 */
export function LegalAgreementNote() {
  const t = useT();
  const urls = useLegalUrls();

  if (!hasLegalUrls(urls)) {
    return null;
  }

  return (
    <p className="text-subtle-foreground m-0 text-center text-xs text-balance">
      {t("By continuing you agree to our")}{" "}
      <LegalLinks
        urls={urls}
        linkClassName="text-muted-foreground hover:text-foreground underline underline-offset-[3px]"
      />
      .
    </p>
  );
}
