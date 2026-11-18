import { SectionPanel } from "@/components/section-panel";
import type { SupportAccessSettings } from "@/lib/graphql/support-access";
import { useT } from "@trenova/shared/i18n/use-t";
import { scopeDescription, scopeLabel } from "./support-access-labels";

/**
 * What each kind of access lets Trenova staff do, and what no grant ever lets them do. The second
 * list comes from the server's support policy itself, so it says exactly what is enforced.
 */
export function SupportAccessLimits({ settings }: { settings: SupportAccessSettings }) {
  const t = useT();

  return (
    <SectionPanel title={t("What Trenova support can do")}>
      <div className="flex flex-col gap-4 p-4 text-sm">
        <dl className="grid gap-2 sm:grid-cols-[8rem_1fr]">
          <dt className="font-medium text-foreground">{scopeLabel(t, "ReadOnly")}</dt>
          <dd className="text-foreground-muted">{scopeDescription(t, "ReadOnly")}</dd>
          <dt className="font-medium text-foreground">{scopeLabel(t, "Full")}</dt>
          <dd className="text-foreground-muted">{scopeDescription(t, "Full")}</dd>
        </dl>
        {settings.limits.length > 0 ? (
          <div className="flex flex-col gap-1.5">
            <p className="font-medium text-foreground">
              {t("Whatever the access, Trenova support can never:")}
            </p>
            <ul className="flex list-disc flex-col gap-1 pl-5 text-foreground-muted">
              {settings.limits.map((limit) => (
                <li key={limit}>{limit}</li>
              ))}
            </ul>
          </div>
        ) : null}
        <p className="text-xs text-foreground-subtle">
          {t(
            "Staff use the assistant only when you allow it for their access, because each answer is paid for by your organization's AI providers.",
          )}
        </p>
      </div>
    </SectionPanel>
  );
}
