import { useT } from "@trenova/shared/i18n/use-t";
import { generateDateTimeStringFromUnixTimestamp } from "@trenova/shared/lib/date";
import { SectionPanel } from "@/components/section-panel";
import { toneVar } from "@/components/kpi/tone";
import type { AIUsageSummary } from "@/lib/graphql/ai-usage";
import { CircleAlertIcon } from "lucide-react";

export type UsageFailure = AIUsageSummary["recentFailures"][number];

/**
 * The provider's own words for each recent failure.
 *
 * A count of failed calls beside a count of calls says something is wrong and
 * nothing about what. The difference between a provider that is down and a
 * provider that refuses every request is the difference between waiting and
 * fixing the configuration, and the provider says which in its message. The
 * panel puts that message where the count is, so an administrator reads it
 * here rather than in the server log.
 */
export function RecentFailures({ failures }: { failures: readonly UsageFailure[] }) {
  const t = useT();

  if (failures.length === 0) {
    return null;
  }

  return (
    <SectionPanel
      title={t("Recent model failures")}
      help={t("The newest failed calls, with the provider's own message.")}
    >
      <ul className="divide-border/70 divide-y">
        {failures.map((failure) => (
          <li key={`${failure.at}-${failure.providerId}-${failure.model}`} className="px-3 py-2">
            <div className="flex items-center gap-2 text-xs">
              <CircleAlertIcon
                aria-hidden
                className="size-3.5 shrink-0"
                style={{ color: toneVar("danger") }}
              />
              <span className="min-w-0 truncate font-medium">
                {failure.providerName || t("Unknown provider")}
              </span>
              <span className="text-muted-foreground min-w-0 truncate">{failure.model}</span>
              <span className="text-muted-foreground ml-auto shrink-0 tabular-nums">
                {generateDateTimeStringFromUnixTimestamp(failure.at)}
              </span>
            </div>
            <p className="text-muted-foreground mt-1 text-xs">
              <span className="text-foreground">{describeFailureClass(failure.errorClass, t)}</span>
              {failure.message !== "" && (
                <>
                  {" · "}
                  <span className="break-words">{failure.message}</span>
                </>
              )}
            </p>
          </li>
        ))}
      </ul>
    </SectionPanel>
  );
}

/** The failure class in words. The token is for the log; the panel is for a person. */
export function describeFailureClass(errorClass: string, t: (key: string) => string): string {
  switch (errorClass) {
    case "provider_unavailable":
      return t("Provider unavailable");
    case "provider_error":
      return t("Request rejected");
    case "timeout":
      return t("Timed out");
    case "network":
      return t("Could not connect");
    case "refused":
      return t("Model declined");
    case "cancelled":
      return t("Cancelled");
    default:
      return t("Failed");
  }
}
