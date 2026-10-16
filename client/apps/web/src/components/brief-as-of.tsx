import { NovaBadge } from "@trenova/shared/components/ui/nova-badge";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatToUserTimezone } from "@trenova/shared/lib/date";

/**
 * The small print under a page brief: when it was written and, when Nova wrote the wording,
 * Nova's badge and name, so a brief a model narrated is never mistaken for one the records
 * spelled out. A brief nobody narrated reads only its time.
 */
export function BriefAsOf({ generatedAt, narrated }: { generatedAt: number; narrated: boolean }) {
  const t = useT();
  const asOf = formatToUserTimezone(generatedAt, {
    showDate: false,
    showTimeZone: false,
    showSeconds: false,
  });

  return (
    <span className="text-muted-foreground flex items-center gap-1.5 text-xs">
      {narrated ? (
        <>
          <NovaBadge size="xs" />
          {t("Written by Nova · as of {0}", asOf)}
        </>
      ) : (
        t("Brief as of {0}", asOf)
      )}
    </span>
  );
}
