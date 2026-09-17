import { carrierIntelProviderLabel, isSectionCovered } from "@/lib/carrier-intelligence";
import type { CarrierIntelProfile } from "@/lib/graphql/carrier-intelligence";
import type { CarrierIntelSection } from "@trenova/graphql/generated/graphql";
import { useT } from "@trenova/shared/i18n/use-t";
import type { ReactNode } from "react";
import { SectionNote, SubHeading } from "../intel-facts";

export type ProfileSectionProps = {
  profile: CarrierIntelProfile;
  provider: string | null | undefined;
};

export type ProfilePart = {
  section: CarrierIntelSection;
  title?: string;
  hasData: boolean;
  render: () => ReactNode;
};

export function ProfileTab({
  profile,
  provider,
  parts,
}: ProfileSectionProps & { parts: ProfilePart[] }) {
  const t = useT();
  const providerName = carrierIntelProviderLabel(provider);
  const notProvided = t("Not provided by {0}", providerName);

  if (!parts.some((part) => isSectionCovered(profile.coverage, part.section))) {
    return <SectionNote>{notProvided}</SectionNote>;
  }

  const single = parts.length === 1;

  return (
    <div className="flex flex-col gap-6">
      {parts.map((part) => {
        const body = !isSectionCovered(profile.coverage, part.section) ? (
          <SectionNote>{notProvided}</SectionNote>
        ) : !part.hasData ? (
          <SectionNote>{t("{0} returned no data for this section", providerName)}</SectionNote>
        ) : (
          part.render()
        );
        return (
          <div key={part.section} className="flex flex-col gap-2" data-section={part.section}>
            {!single && part.title ? <SubHeading>{part.title}</SubHeading> : null}
            {body}
          </div>
        );
      })}
    </div>
  );
}
