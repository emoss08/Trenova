import { useT } from "@trenova/shared/i18n/use-t";
import type { CarrierIntelProfile } from "@/lib/graphql/carrier-intelligence";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { HistoryIcon } from "lucide-react";
import { IntelSectionCard } from "./intel-section-card";

export type ChangeHistoryCardProps = {
  profile: CarrierIntelProfile;
  provider: string | null | undefined;
  className?: string;
};

const FREQUENT_CHANGE_THRESHOLD = 2;

export function ChangeHistoryCard({ profile, provider, className }: ChangeHistoryCardProps) {
  const t = useT();
  const history = profile.changeHistory;

  const counters = history
    ? [
        {
          key: "name",
          label: t("Name"),
          count: history.nameChanges,
          at: history.nameLastChangedAt,
        },
        {
          key: "email",
          label: t("Email"),
          count: history.emailChanges,
          at: history.emailLastChangedAt,
        },
        {
          key: "phone",
          label: t("Phone"),
          count: history.phoneChanges,
          at: history.phoneLastChangedAt,
        },
        {
          key: "address",
          label: t("Address"),
          count: history.addressChanges,
          at: history.addressLastChangedAt,
        },
        {
          key: "contact",
          label: t("Contact"),
          count: history.contactChanges,
          at: history.contactLastChangedAt,
        },
      ]
    : [];
  const frequent = counters.some((counter) => (counter.count ?? 0) >= FREQUENT_CHANGE_THRESHOLD);

  return (
    <IntelSectionCard
      title={t("Change history")}
      icon={HistoryIcon}
      coverage={profile.coverage}
      provider={provider}
      emphasis={frequent ? "warning" : "none"}
      className={className}
      parts={[
        {
          section: "ChangeHistory",
          hasData: history !== null,
          content: (
            <div className="flex flex-col gap-2">
              <ul className="grid grid-cols-2 gap-2 sm:grid-cols-5">
                {counters.map((counter) => {
                  const count = counter.count ?? 0;
                  const flagged = count >= FREQUENT_CHANGE_THRESHOLD;
                  return (
                    <li
                      key={counter.key}
                      className={cn(
                        "flex flex-col gap-0.5 rounded-md border px-2.5 py-2",
                        flagged && "border-yellow-600/40 bg-yellow-600/10",
                      )}
                    >
                      <span className="text-muted-foreground text-xs">
                        {t("{0} changes", counter.label)}
                      </span>
                      <span className="text-lg font-semibold tabular-nums">
                        {counter.count ?? "-"}
                      </span>
                      <span className="text-muted-foreground text-2xs">
                        {counter.at
                          ? t("last {0}", formatUnixDateMedium(counter.at))
                          : t("never changed")}
                      </span>
                    </li>
                  );
                })}
              </ul>
              {frequent ? (
                <p className="text-muted-foreground text-xs">
                  {t(
                    "Frequent identity changes are a common trait of carriers reinventing themselves after a bad record.",
                  )}
                </p>
              ) : null}
            </div>
          ),
        },
      ]}
    />
  );
}
