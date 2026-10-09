import { LegalLinks, hasLegalUrls } from "@/components/legal-links";
import { useLegalUrls } from "@/hooks/use-legal-urls";
import { useT } from "@trenova/shared/i18n/use-t";

/**
 * The line at the foot of every sign-in screen that names the documents continuing
 * agrees to. Nothing when the install publishes neither.
 */
export function LegalAgreementNote() {
  const t = useT();
  const urls = useLegalUrls();

  if (!hasLegalUrls(urls)) {
    return null;
  }

  return (
    <p className="text-muted-foreground m-0 max-w-[360px] text-sm text-pretty">
      {t("By continuing you agree to our")}{" "}
      <LegalLinks
        urls={urls}
        linkClassName="text-muted-foreground hover:text-foreground decoration-muted-foreground/40 hover:decoration-foreground underline underline-offset-[3px] transition-colors"
      />
      .
    </p>
  );
}
