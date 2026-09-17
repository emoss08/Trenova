import { useT } from "@trenova/shared/i18n/use-t";
import type { CarrierIntelProfile } from "@/lib/graphql/carrier-intelligence";
import { Badge } from "@trenova/shared/components/ui/badge";
import { BuildingIcon } from "lucide-react";
import {
  IntelDate,
  IntelField,
  IntelFieldGrid,
  IntelNumber,
  IntelSectionCard,
  IntelValue,
} from "./intel-section-card";

type IntelAddress = NonNullable<NonNullable<CarrierIntelProfile["identity"]>["physicalAddress"]>;

export type IdentityCardProps = {
  profile: CarrierIntelProfile;
  provider: string | null | undefined;
  className?: string;
};

function AddressValue({ address }: { address: IntelAddress | null | undefined }) {
  const t = useT();

  if (!address) {
    return <span className="text-muted-foreground">-</span>;
  }

  const locality = [address.city, address.state].filter(Boolean).join(", ");
  const lines = [address.line1, [locality, address.postalCode].filter(Boolean).join(" ")].filter(
    (line): line is string => Boolean(line),
  );

  return (
    <span className="flex flex-col whitespace-normal">
      {lines.length > 0 ? lines.map((line) => <span key={line}>{line}</span>) : "-"}
      {address.undelivered ? (
        <Badge variant="warning" className="mt-1 max-h-5">
          {t("Mail returned undeliverable")}
        </Badge>
      ) : null}
    </span>
  );
}

export function IdentityCard({ profile, provider, className }: IdentityCardProps) {
  const t = useT();
  const identity = profile.identity;
  const docket = identity?.docketNumber
    ? `${identity.docketPrefix ?? ""}${identity.docketNumber}`
    : null;

  return (
    <IntelSectionCard
      title={t("Identity")}
      icon={BuildingIcon}
      coverage={profile.coverage}
      provider={provider}
      className={className}
      parts={[
        {
          section: "Identity",
          hasData: identity !== null,
          content: identity ? (
            <IntelFieldGrid>
              <IntelField label={t("Legal name")}>
                <IntelValue value={identity.legalName} />
              </IntelField>
              <IntelField label={t("DBA name")}>
                <IntelValue value={identity.dbaName} />
              </IntelField>
              <IntelField label={t("USDOT number")}>
                <span className="font-mono">{identity.dotNumber}</span>
              </IntelField>
              <IntelField label={t("Docket")}>
                <IntelValue value={docket} />
              </IntelField>
              <IntelField label={t("USDOT status")}>
                <IntelValue value={identity.usdotStatus} />
              </IntelField>
              <IntelField label={t("EIN")}>
                <IntelValue value={identity.ein} />
              </IntelField>
              <IntelField label={t("Entity type")}>
                <IntelValue value={identity.entityType} />
              </IntelField>
              <IntelField label={t("Carrier operation")}>
                <IntelValue value={identity.carrierOperation} />
              </IntelField>
              <IntelField label={t("USDOT added")}>
                <IntelDate value={identity.dotAddedAt} />
              </IntelField>
              <IntelField label={t("USDOT age (days)")}>
                <IntelNumber value={identity.dotAgeDays} />
              </IntelField>
              <IntelField label={t("Physical address")}>
                <AddressValue address={identity.physicalAddress} />
              </IntelField>
              <IntelField label={t("Mailing address")}>
                <AddressValue address={identity.mailingAddress} />
              </IntelField>
            </IntelFieldGrid>
          ) : null,
        },
      ]}
    />
  );
}
