import { useT } from "@trenova/shared/i18n/use-t";
import entraLogo from "@/assets/integrations/logos/entra.svg";
import { LazyImage } from "@/components/image";

export function EntraLogo({ className }: { className?: string }) {
  const t = useT();

  return <LazyImage src={entraLogo} alt={t("Microsoft Entra ID")} className={className || "size-6"} />;
}
