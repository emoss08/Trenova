import entraLogo from "@/assets/integrations/logos/entra.svg";
import { LazyImage } from "@/components/image";

export function EntraLogo({ className }: { className?: string }) {
  return <LazyImage src={entraLogo} alt="Microsoft Entra ID" className={className || "size-6"} />;
}
