import { useT } from "@trenova/shared/i18n/use-t";
import type { CarrierIntelProfile } from "@/lib/graphql/carrier-intelligence";
import { CARRIER_INTEL_SECTIONS, carrierIntelProviderLabel } from "@/lib/carrier-intelligence";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { formatList } from "@trenova/shared/i18n/format";
import { cn } from "@trenova/shared/lib/utils";
import { SearchXIcon } from "lucide-react";
import type { ComponentType } from "react";
import { AuthorityCard } from "./authority-card";
import { BasicsChartCard } from "./basics-chart-card";
import { BenchmarksCard } from "./benchmarks-card";
import { ChangeHistoryCard } from "./change-history-card";
import { ContactsOperationsCard } from "./contacts-operations-card";
import { FleetEquipmentCard } from "./fleet-equipment-card";
import { IdentityCard } from "./identity-card";
import { InspectionsCrashesCard } from "./inspections-crashes-card";
import { InsuranceCard } from "./insurance-card";
import { LanesCard } from "./lanes-card";
import { NetworkSignalsCard } from "./network-signals-card";
import { SafetyCard } from "./safety-card";
import { useCarrierIntelLabels } from "./use-carrier-intel-labels";

export type CarrierIntelProfileCardId =
  | "identity"
  | "authority"
  | "insurance"
  | "safety"
  | "basics"
  | "inspections"
  | "network"
  | "fleet"
  | "contacts"
  | "changeHistory"
  | "lanes"
  | "benchmarks";

type ProfileCardProps = {
  profile: CarrierIntelProfile;
  provider: string | null | undefined;
  className?: string;
};

type ProfileCardDefinition = {
  id: CarrierIntelProfileCardId;
  component: ComponentType<ProfileCardProps>;
  wide?: boolean;
};

const PROFILE_CARDS: readonly ProfileCardDefinition[] = [
  { id: "identity", component: IdentityCard },
  { id: "authority", component: AuthorityCard },
  { id: "insurance", component: InsuranceCard, wide: true },
  { id: "safety", component: SafetyCard },
  { id: "basics", component: BasicsChartCard },
  { id: "inspections", component: InspectionsCrashesCard, wide: true },
  { id: "network", component: NetworkSignalsCard, wide: true },
  { id: "fleet", component: FleetEquipmentCard, wide: true },
  { id: "contacts", component: ContactsOperationsCard, wide: true },
  { id: "changeHistory", component: ChangeHistoryCard, wide: true },
  { id: "lanes", component: LanesCard },
  { id: "benchmarks", component: BenchmarksCard },
];

export type CarrierIntelProfileViewProps = {
  profile: CarrierIntelProfile;
  provider: string | null | undefined;
  notFound?: boolean;
  cards?: readonly CarrierIntelProfileCardId[];
  columns?: 1 | 2;
  showCoverageSummary?: boolean;
  className?: string;
};

export function CarrierIntelProfileView({
  profile,
  provider,
  notFound = false,
  cards,
  columns = 2,
  showCoverageSummary = true,
  className,
}: CarrierIntelProfileViewProps) {
  const t = useT();
  const labels = useCarrierIntelLabels();
  const providerName = carrierIntelProviderLabel(provider);
  const visible = cards ? PROFILE_CARDS.filter((card) => cards.includes(card.id)) : PROFILE_CARDS;
  const missing = CARRIER_INTEL_SECTIONS.filter((section) => !profile.coverage.includes(section));

  return (
    <div className={cn("flex flex-col gap-3", className)}>
      {notFound ? (
        <Alert variant="warning">
          <SearchXIcon />
          <AlertTitle>{t("Carrier not found at {0}", providerName)}</AlertTitle>
          <AlertDescription>
            {t(
              "The provider has no record for this USDOT number. Check the number on the carrier, or confirm the carrier is registered with the FMCSA.",
            )}
          </AlertDescription>
        </Alert>
      ) : null}
      {showCoverageSummary ? (
        <p className="text-muted-foreground text-xs" data-testid="carrier-intel-coverage-summary">
          {missing.length === 0
            ? t("{0} provided every section of the profile.", providerName)
            : t(
                "{0} provided {1} of {2} sections. Not provided: {3}.",
                providerName,
                CARRIER_INTEL_SECTIONS.length - missing.length,
                CARRIER_INTEL_SECTIONS.length,
                formatList(missing.map((section) => labels.section[section])),
              )}
        </p>
      ) : null}
      <div className={cn("grid grid-cols-1 gap-3", columns === 2 && "xl:grid-cols-2")}>
        {visible.map(({ id, component: Card, wide }) => (
          <Card
            key={id}
            profile={profile}
            provider={provider}
            className={cn(columns === 2 && wide && "xl:col-span-2")}
          />
        ))}
      </div>
    </div>
  );
}
