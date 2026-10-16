import { useT } from "@trenova/shared/i18n/use-t";
import { Badge } from "@trenova/shared/components/ui/badge";
import { cn } from "@trenova/shared/lib/utils";

export function BetaTag({ className }: { className?: string }) {
  const t = useT();

  return (
    <Badge tabIndex={0} variant="neutral" className={cn("uppercase", className)}>
      {t("Beta")}
    </Badge>
  );
}
