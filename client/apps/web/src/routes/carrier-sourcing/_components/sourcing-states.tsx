import { useT } from "@trenova/shared/i18n/use-t";
import { carrierIntelProviderLabel } from "@/lib/carrier-intelligence";
import { CARRIER_INTEL_INTEGRATIONS_PATH } from "@/lib/carrier-links";
import { Button } from "@trenova/shared/components/ui/button";
import { EmptySheet, GhostLine } from "@trenova/shared/components/ui/empty-sheet";
import { ArrowRightIcon, PlugZapIcon } from "lucide-react";
import { Link } from "react-router";

export function SourcingListSketch() {
  return (
    <div className="flex flex-col overflow-hidden rounded-lg border text-left">
      {[
        ["w-40", "w-56", "w-14"],
        ["w-28", "w-48", "w-10"],
        ["w-36", "w-52", "w-16"],
        ["w-24", "w-44", "w-12"],
      ].map(([name, meta, fleet]) => (
        <div
          key={`${name}-${meta}`}
          className="flex h-12 items-center justify-between gap-4 border-b px-3 last:border-b-0"
        >
          <div className="flex flex-col gap-1.5">
            <GhostLine className={name} />
            <GhostLine className={meta} />
          </div>
          <div className="flex items-center gap-4">
            <GhostLine className={fleet} />
            <span className="bg-muted size-2 rounded-full" />
          </div>
        </div>
      ))}
    </div>
  );
}

export type SourcingExample = {
  id: string;
  label: string;
  onSelect: () => void;
};

export function SourcingIntro({
  examples,
  searchSupported,
}: {
  examples: readonly SourcingExample[];
  searchSupported: boolean;
}) {
  const t = useT();
  return (
    <EmptySheet
      title={t("Find a carrier")}
      description={
        searchSupported
          ? t(
              "Search the market by name, or enter a USDOT or MC number to pull one carrier. Every result is checked against your vetting rules before you import it.",
            )
          : t(
              "Enter a USDOT or MC number to pull a carrier and check it against your vetting rules before you import it.",
            )
      }
      sketch={<SourcingListSketch />}
      action={
        examples.length > 0 ? (
          <div className="flex flex-wrap items-center justify-center gap-2">
            {examples.map((example) => (
              <Button
                key={example.id}
                type="button"
                variant="outline"
                size="sm"
                onClick={example.onSelect}
              >
                {example.label}
                <ArrowRightIcon className="size-3" aria-hidden />
              </Button>
            ))}
          </div>
        ) : null
      }
    />
  );
}

export function SourcingNoMatches({
  filtered,
  onClearFilters,
}: {
  filtered: boolean;
  onClearFilters: () => void;
}) {
  const t = useT();
  return (
    <EmptySheet
      title={t("Nothing matches")}
      description={
        filtered
          ? t("No carrier on this page passed your filters. Loosen them or try another search.")
          : t("No carrier fits this search. Try a shorter name, a USDOT number or a home state.")
      }
      sketch={<SourcingListSketch />}
      action={
        filtered ? (
          <Button type="button" variant="outline" size="sm" onClick={onClearFilters}>
            {t("Clear filters")}
          </Button>
        ) : null
      }
    />
  );
}

export function SourcingNotConfigured() {
  const t = useT();
  return (
    <EmptySheet
      title={t("Connect a carrier data provider")}
      description={t(
        "Connect CarrierOk to search the carrier market, or the free FMCSA QCMobile service to look carriers up by USDOT number.",
      )}
      sketch={<SourcingListSketch />}
      action={
        <Button
          size="sm"
          variant="outline"
          nativeButton={false}
          render={<Link to={CARRIER_INTEL_INTEGRATIONS_PATH} />}
        >
          <PlugZapIcon className="size-3.5" aria-hidden />
          {t("Open integrations")}
        </Button>
      }
    />
  );
}

export function SourcingSearchUnsupported({ provider }: { provider: string | null }) {
  const t = useT();
  const providerName = carrierIntelProviderLabel(provider);
  return (
    <div data-testid="sourcing-search-unsupported">
      <EmptySheet
        title={t("{0} can't search by name", providerName)}
        description={t(
          "{0} looks carriers up by USDOT or MC number only. Enter a number above, or connect a provider that searches the market.",
          providerName,
        )}
        sketch={<SourcingListSketch />}
        action={
          <Button
            size="sm"
            variant="outline"
            nativeButton={false}
            render={<Link to={CARRIER_INTEL_INTEGRATIONS_PATH} />}
          >
            {t("Change provider")}
          </Button>
        }
      />
    </div>
  );
}
