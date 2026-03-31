import oktaDarkLogo from "@/assets/integrations/logos/okta_dark_logo.svg";
import oktaLightLogo from "@/assets/integrations/logos/okta_light_logo.svg";
import { LazyImage } from "@/components/image";
import { useTheme } from "@/components/theme-provider";

export function OktaLogo({ className }: { className?: string }) {
  const { theme } = useTheme();
  const src = theme === "dark" ? oktaDarkLogo : oktaLightLogo;

  return <LazyImage src={src} alt="Okta Logo" className={className || "size-5"} />;
}
