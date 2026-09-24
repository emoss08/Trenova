import { Badge } from "@trenova/shared/components/ui/badge";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";

/**
 * Marks a run or a conversation that has read text written outside the
 * organization — a customer's document, an email, a web page. Once it has,
 * every change it proposes waits for a person. `title` says so where the
 * badge stands alone.
 */
export function OutsideContentBadge({
  t,
  title,
  className,
}: {
  t: TranslateFn;
  title?: string;
  className?: string;
}) {
  return (
    <Badge variant="warning" className={className} title={title}>
      {t("Read outside content")}
    </Badge>
  );
}
