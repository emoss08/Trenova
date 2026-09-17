import { useT } from "@trenova/shared/i18n/use-t";
import type { CarrierIntelProfile } from "@/lib/graphql/carrier-intelligence";
import { Badge } from "@trenova/shared/components/ui/badge";
import { ContactIcon } from "lucide-react";
import {
  IntelBoolean,
  IntelDate,
  IntelField,
  IntelFieldGrid,
  IntelNumber,
  IntelSectionCard,
  IntelValue,
} from "./intel-section-card";

export type ContactsOperationsCardProps = {
  profile: CarrierIntelProfile;
  provider: string | null | undefined;
  className?: string;
};

function TagList({ values }: { values: readonly string[] | null }) {
  if (!values || values.length === 0) {
    return <span className="text-muted-foreground">-</span>;
  }
  return (
    <span className="flex flex-wrap gap-1 whitespace-normal">
      {values.map((value) => (
        <Badge key={value} variant="secondary" className="max-h-5">
          {value}
        </Badge>
      ))}
    </span>
  );
}

export function ContactsOperationsCard({
  profile,
  provider,
  className,
}: ContactsOperationsCardProps) {
  const t = useT();
  const contacts = profile.contacts;
  const operations = profile.operations;

  return (
    <IntelSectionCard
      title={t("Contacts & operations")}
      icon={ContactIcon}
      coverage={profile.coverage}
      provider={provider}
      className={className}
      parts={[
        {
          section: "Contacts",
          heading: t("Contacts"),
          hasData: contacts !== null,
          content: contacts ? (
            <IntelFieldGrid>
              <IntelField label={t("Primary contact")}>
                <IntelValue value={contacts.primaryContact} />
              </IntelField>
              <IntelField label={t("Secondary contact")}>
                <IntelValue value={contacts.secondaryContact} />
              </IntelField>
              <IntelField label={t("Phone")}>
                <IntelValue value={contacts.phone} />
              </IntelField>
              <IntelField label={t("Cell phone")}>
                <IntelValue value={contacts.cellphone} />
              </IntelField>
              <IntelField label={t("Fax")}>
                <IntelValue value={contacts.fax} />
              </IntelField>
              <IntelField label={t("Email")}>
                <IntelValue value={contacts.email} />
              </IntelField>
            </IntelFieldGrid>
          ) : null,
        },
        {
          section: "Operations",
          heading: t("Operations"),
          hasData: operations !== null,
          content: operations ? (
            <IntelFieldGrid>
              <IntelField label={t("Classification")} className="sm:col-span-2">
                <TagList values={operations.classification} />
              </IntelField>
              <IntelField label={t("Cargo carried")} className="sm:col-span-2">
                <TagList values={operations.cargoCarried} />
              </IntelField>
              <IntelField label={t("Hazmat carrier")}>
                <IntelBoolean value={operations.hazmatCarrier} />
              </IntelField>
              <IntelField label={t("PHMSA registered")}>
                <IntelBoolean value={operations.phmsa} trueTone="good" />
              </IntelField>
              <IntelField label={t("MCS-150 filed")}>
                <IntelDate value={operations.mcs150At} />
              </IntelField>
              <IntelField label={t("MCS-150 mileage")}>
                <IntelNumber value={operations.mcs150Mileage} />
              </IntelField>
              <IntelField label={t("BOC-3 on file")}>
                <IntelBoolean value={operations.boc3OnFile} trueTone="good" />
              </IntelField>
              <IntelField label={t("BOC-3 agent")}>
                <IntelValue value={operations.boc3Agent} />
              </IntelField>
              <IntelField label={t("SmartWay partner")}>
                <IntelBoolean value={operations.smartWay} trueTone="good" />
              </IntelField>
              <IntelField label={t("CARB compliant")}>
                <IntelBoolean value={operations.carbCompliant} trueTone="good" />
              </IntelField>
            </IntelFieldGrid>
          ) : null,
        },
      ]}
    />
  );
}
