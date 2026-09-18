import { CARRIER_INTEL_SECTIONS, carrierIntelProviderLabel } from "@/lib/carrier-intelligence";
import type { CarrierIntelProfile } from "@/lib/graphql/carrier-intelligence";
import { Tabs, TabsContent, TabsList, TabsTab } from "@trenova/shared/components/ui/tabs";
import { formatList } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useState, type ComponentType } from "react";
import { SectionNote } from "./intel-facts";
import { AuthoritySection } from "./profile/authority-section";
import { CompanySection } from "./profile/company-section";
import { FleetSection } from "./profile/fleet-section";
import { InsuranceSection } from "./profile/insurance-section";
import { LanesSection } from "./profile/lanes-section";
import { NetworkSection } from "./profile/network-section";
import type { ProfileSectionProps } from "./profile/profile-tab";
import { SafetySection } from "./profile/safety-section";
import { useCarrierIntelLabels } from "./use-carrier-intel-labels";

export const CARRIER_INTEL_PROFILE_SECTIONS = [
  "company",
  "authority",
  "insurance",
  "safety",
  "fleet",
  "network",
  "lanes",
] as const;

export type CarrierIntelProfileSectionId = (typeof CARRIER_INTEL_PROFILE_SECTIONS)[number];

const SECTION_COMPONENTS: Record<
  CarrierIntelProfileSectionId,
  ComponentType<ProfileSectionProps>
> = {
  company: CompanySection,
  authority: AuthoritySection,
  insurance: InsuranceSection,
  safety: SafetySection,
  fleet: FleetSection,
  network: NetworkSection,
  lanes: LanesSection,
};

export type CarrierIntelProfileViewProps = {
  profile: CarrierIntelProfile;
  provider: string | null | undefined;
  sections?: readonly CarrierIntelProfileSectionId[];
  showCoverageSummary?: boolean;
  className?: string;
};

export function CarrierIntelProfileView({
  profile,
  provider,
  sections = CARRIER_INTEL_PROFILE_SECTIONS,
  showCoverageSummary = false,
  className,
}: CarrierIntelProfileViewProps) {
  const t = useT();
  const labels = useCarrierIntelLabels();
  const [active, setActive] = useState<CarrierIntelProfileSectionId | null>(null);
  const current = active && sections.includes(active) ? active : (sections[0] ?? null);
  const providerName = carrierIntelProviderLabel(provider);
  const missing = CARRIER_INTEL_SECTIONS.filter((section) => !profile.coverage.includes(section));

  const tabLabels: Record<CarrierIntelProfileSectionId, string> = {
    company: t("Company"),
    authority: t("Authority"),
    insurance: t("Insurance"),
    safety: t("Safety"),
    fleet: t("Fleet"),
    network: t("Network signals"),
    lanes: t("Lanes"),
  };

  return (
    <div className={cn("flex flex-col gap-4", className)}>
      {current ? (
        <Tabs
          value={current}
          onValueChange={(value) => {
            const next = sections.find((section) => section === value);
            if (next) {
              setActive(next);
            }
          }}
          className="gap-4"
        >
          <TabsList variant="underline" className="w-full justify-start border-b">
            {sections.map((id) => (
              <TabsTab key={id} value={id} className="grow-0 px-2 text-sm">
                {tabLabels[id]}
              </TabsTab>
            ))}
          </TabsList>
          {sections.map((id) => {
            const Section = SECTION_COMPONENTS[id];
            return (
              <TabsContent key={id} value={id}>
                <Section profile={profile} provider={provider} />
              </TabsContent>
            );
          })}
        </Tabs>
      ) : null}
      {showCoverageSummary ? (
        <SectionNote>
          <span data-testid="carrier-intel-coverage-summary">
            {missing.length === 0
              ? t("{0} provided every section of the profile.", providerName)
              : t(
                  "{0} provided {1} of {2} sections. Not provided: {3}.",
                  providerName,
                  CARRIER_INTEL_SECTIONS.length - missing.length,
                  CARRIER_INTEL_SECTIONS.length,
                  formatList(missing.map((section) => labels.section[section])),
                )}
          </span>
        </SectionNote>
      ) : null}
    </div>
  );
}
