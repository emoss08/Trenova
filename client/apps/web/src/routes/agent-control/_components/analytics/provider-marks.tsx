import type { AIAnalyticsProvider } from "@/lib/graphql/ai-analytics";
import { Mark } from "../kit/marks";

type ProviderMarksProps = {
  providers: readonly AIAnalyticsProvider[];
  s?: number;
  /** At most this many marks; the busiest come first. */
  limit?: number;
};

/** The logos of the providers that served a series, busiest first. */
export function ProviderMarks({ providers, s = 16, limit = 3 }: ProviderMarksProps) {
  return (
    <span className="an-marks" title={providers.map((provider) => provider.name).join(", ")}>
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
