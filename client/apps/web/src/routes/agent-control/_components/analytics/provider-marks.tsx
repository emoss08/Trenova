import type { AIAnalyticsProvider } from "@/lib/graphql/ai-analytics";
import { cn } from "@trenova/shared/lib/utils";
import { Mark } from "../kit/marks";

/** Marks this large sit side by side, as in a sheet's head; smaller ones overlap in a row. */
const SPREAD_SIZE = 24;

type ProviderMarksProps = {
  providers: readonly AIAnalyticsProvider[];
  s?: number;
  /** At most this many marks; the busiest come first. */
  limit?: number;
};

/** The logos of the providers that served a series, busiest first. */
export function ProviderMarks({ providers, s = 16, limit = 3 }: ProviderMarksProps) {
  return (
    <span
      className={cn("an-marks", s >= SPREAD_SIZE && "spread")}
      title={providers.map((provider) => provider.name).join(", ")}
    >
      {providers.slice(0, limit).map((provider, index) => (
        <Mark
          key={provider.id ?? `${provider.name}-${index}`}
          provider={{
            name: provider.name,
            kind: provider.kind ?? undefined,
            baseUrl: provider.baseUrl,
          }}
          s={s}
        />
      ))}
    </span>
  );
}
