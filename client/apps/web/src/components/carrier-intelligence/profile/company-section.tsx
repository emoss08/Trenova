import type { CarrierIntelProfile } from "@/lib/graphql/carrier-intelligence";
import { formatNumber } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";
import type { ReactNode } from "react";
import {
  BooleanValue,
  EmptyValue,
  Fact,
  FactGrid,
  Muted,
  dateOrDash,
  listOrDash,
  textOrDash,
} from "../intel-facts";
import { StatusDot } from "../status-dot";
import { useIntelAgeFormatter } from "../use-intel-age";
import { ProfileTab, type ProfileSectionProps } from "./profile-tab";

type IntelAddress = NonNullable<NonNullable<CarrierIntelProfile["identity"]>["physicalAddress"]>;

function AddressValue({ address }: { address: IntelAddress | null | undefined }) {
  const t = useT();
  if (!address) {
    return <EmptyValue />;
  }
  const locality = [address.city, address.state].filter(Boolean).join(", ");
  const lines = [address.line1, [locality, address.postalCode].filter(Boolean).join(" ")].filter(
    (line): line is string => Boolean(line),
  );
  return (
    <span className="flex flex-col">
      {lines.length > 0 ? lines.map((line) => <span key={line}>{line}</span>) : <EmptyValue />}
      {address.undelivered ? (
        <span className="text-muted-foreground inline-flex items-center gap-2 text-xs">
          <StatusDot tone="medium" />
          {t("Mail returned undeliverable")}
        </span>
      ) : null}
    </span>
  );
}

export function CompanySection({ profile, provider }: ProfileSectionProps) {
  const t = useT();
  const formatAge = useIntelAgeFormatter();
  const { identity, contacts, operations } = profile;

  return (
    <ProfileTab
      profile={profile}
      provider={provider}
      parts={[
        {
          section: "Identity",
          title: t("Identity"),
          hasData: identity !== null,
          render: (): ReactNode =>
            identity ? (
              <FactGrid>
                <Fact label={t("Legal name")}>{textOrDash(identity.legalName)}</Fact>
                <Fact label={t("DBA name")}>{textOrDash(identity.dbaName)}</Fact>
                <Fact label={t("USDOT number")}>{identity.dotNumber}</Fact>
                <Fact label={t("Docket")}>
                  {identity.docketNumber ? (
                    `${identity.docketPrefix ?? ""}${identity.docketNumber}`
                  ) : (
                    <EmptyValue />
                  )}
                </Fact>
                <Fact label={t("USDOT status")}>{textOrDash(identity.usdotStatus)}</Fact>
                <Fact label={t("EIN")}>{textOrDash(identity.ein)}</Fact>
                <Fact label={t("Entity type")}>{textOrDash(identity.entityType)}</Fact>
                <Fact label={t("Carrier operation")}>{textOrDash(identity.carrierOperation)}</Fact>
                <Fact label={t("USDOT added")}>{dateOrDash(identity.dotAddedAt)}</Fact>
                <Fact label={t("USDOT age")}>
                  {formatAge(identity.dotAgeDays) ?? <EmptyValue />}
                </Fact>
                <Fact label={t("Physical address")} wrap>
                  <AddressValue address={identity.physicalAddress} />
                </Fact>
                <Fact label={t("Mailing address")} wrap>
                  <AddressValue address={identity.mailingAddress} />
                </Fact>
              </FactGrid>
            ) : null,
        },
        {
          section: "Contacts",
          title: t("Contacts"),
          hasData: contacts !== null,
          render: () =>
            contacts ? (
              <FactGrid>
                <Fact label={t("Primary contact")}>{textOrDash(contacts.primaryContact)}</Fact>
                <Fact label={t("Secondary contact")}>{textOrDash(contacts.secondaryContact)}</Fact>
                <Fact label={t("Phone")}>{textOrDash(contacts.phone)}</Fact>
                <Fact label={t("Cell phone")}>{textOrDash(contacts.cellphone)}</Fact>
                <Fact label={t("Fax")}>{textOrDash(contacts.fax)}</Fact>
                <Fact label={t("Email")}>{textOrDash(contacts.email)}</Fact>
              </FactGrid>
            ) : null,
        },
        {
          section: "Operations",
          title: t("Operations"),
          hasData: operations !== null,
          render: () =>
            operations ? (
              <FactGrid>
                <Fact label={t("Classification")} wrap className="sm:col-span-2">
                  {listOrDash(operations.classification)}
                </Fact>
                <Fact label={t("Cargo carried")} wrap className="sm:col-span-2">
                  {listOrDash(operations.cargoCarried)}
                </Fact>
                <Fact label={t("Hazmat carrier")}>
                  <BooleanValue value={operations.hazmatCarrier} />
                </Fact>
                <Fact label={t("PHMSA registered")}>
                  <BooleanValue value={operations.phmsa} />
                </Fact>
                <Fact label={t("MCS-150 filed")}>
                  {operations.mcs150At ? (
                    <span>
                      {dateOrDash(operations.mcs150At)}
                      {operations.mcs150Mileage !== null ? (
                        <Muted>
                          {" · "}
                          {t("{0} mi", formatNumber(operations.mcs150Mileage))}
                        </Muted>
                      ) : null}
                    </span>
                  ) : (
                    <EmptyValue />
                  )}
                </Fact>
                <Fact label={t("BOC-3 on file")}>
                  {operations.boc3OnFile && operations.boc3Agent ? (
                    <span>
                      {t("Yes")}
                      <Muted> · {operations.boc3Agent}</Muted>
                    </span>
                  ) : (
                    <BooleanValue value={operations.boc3OnFile} />
                  )}
                </Fact>
                <Fact label={t("SmartWay partner")}>
                  <BooleanValue value={operations.smartWay} />
                </Fact>
                <Fact label={t("CARB compliant")}>
                  <BooleanValue value={operations.carbCompliant} />
                </Fact>
              </FactGrid>
            ) : null,
        },
      ]}
    />
  );
}
