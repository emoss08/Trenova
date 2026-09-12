import { useT } from "@trenova/shared/i18n/use-t";
import oktaDarkLogo from "@/assets/integrations/logos/okta_dark_logo.svg";
import oktaLightLogo from "@/assets/integrations/logos/okta_light_logo.svg";
import { LazyImage } from "@/components/image";
import { useTheme } from "@trenova/shared/components/theme-provider";

export function OktaLogo({ className }: { className?: string }) {
  const t = useT();

  const { theme } = useTheme();
  const src = theme === "dark" ? oktaDarkLogo : oktaLightLogo;

  return <LazyImage src={src} alt={t("Okta Logo")} className={className || "size-5"} />;
}
