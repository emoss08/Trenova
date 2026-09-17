import { groupNetworkLinks } from "@/lib/carrier-intelligence";
import type { CarrierIntelNetworkKind } from "@trenova/graphql/generated/graphql";
import { formatList } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";
import {
  Fact,
  FactGrid,
  MiniTable,
  SectionNote,
  SubHeading,
  dateOrDash,
  numberOrDash,
} from "../intel-facts";
import { StatusDot } from "../status-dot";
import { useCarrierIntelLabels } from "../use-carrier-intel-labels";
import { ProfileTab, type ProfileSectionProps } from "./profile-tab";

const HIGH_RISK_KINDS: ReadonlySet<CarrierIntelNetworkKind> = new Set(["EIN", "Equipment"]);
const FREQUENT_CHANGE_THRESHOLD = 2;

function SharedCount({ value, highRisk }: { value: number | null; highRisk: boolean }) {
  return (
    <span className="inline-flex items-center gap-2">
      {highRisk && (value ?? 0) > 0 ? <StatusDot tone="critical" /> : null}
      {numberOrDash(value)}
    </span>
  );
}

export function NetworkSection({ profile, provider }: ProfileSectionProps) {
  const t = useT();
  const labels = useCarrierIntelLabels();
  const { network, changeHistory } = profile;

  return (
    <ProfileTab
      profile={profile}
      provider={provider}
      parts={[
        {
          section: "Network",
          title: t("Shared identifiers"),
          hasData: network !== null,
          render: () => {
            if (!network) {
              return null;
            }
            const linked = groupNetworkLinks(network.links ?? []);
            return (
              <div className="flex flex-col gap-5">
                <div className="flex flex-col gap-2">
                  <FactGrid className="grid-cols-2 sm:grid-cols-5">
                    <Fact label={t("EINs")}>
                      <SharedCount value={network.sharedEins} highRisk />
                    </Fact>
                    <Fact label={t("Equipment")}>
                      <SharedCount value={network.sharedEquipment} highRisk />
                    </Fact>
                    <Fact label={t("Addresses")}>
                      <SharedCount value={network.sharedAddresses} highRisk={false} />
                    </Fact>
                    <Fact label={t("Phones")}>
                      <SharedCount value={network.sharedPhones} highRisk={false} />
                    </Fact>
                    <Fact label={t("Emails")}>
                      <SharedCount value={network.sharedEmails} highRisk={false} />
                    </Fact>
                  </FactGrid>
                  <SectionNote>
                    {t(
                      "Other carriers sharing each identifier. A shared EIN or shared equipment is a common sign of a chameleon carrier.",
                    )}
                  </SectionNote>
                </div>
                {linked.length > 0 ? (
                  <div className="flex flex-col gap-1">
                    <SubHeading>{t("Linked carriers")}</SubHeading>
                    <ul className="divide-border max-h-80 divide-y overflow-y-auto">
                      {linked.map((carrier) => {
                        const highRisk = carrier.kinds.some((kind) => HIGH_RISK_KINDS.has(kind));
                        return (
                          <li
                            key={carrier.dotNumber}
                            className="flex items-center justify-between gap-3 py-2"
                            data-linked-dot={carrier.dotNumber}
                          >
                            <span className="inline-flex min-w-0 items-center gap-2 text-sm tabular-nums">
                              <StatusDot tone={highRisk ? "critical" : "medium"} />
                              {t("USDOT {0}", carrier.dotNumber)}
                            </span>
                            <span className="text-muted-foreground truncate text-xs">
                              {formatList(carrier.kinds.map((kind) => labels.networkKind[kind]))}
                            </span>
                          </li>
                        );
                      })}
                    </ul>
                  </div>
                ) : (
                  <SectionNote>
                    {t("No other carrier shares this carrier's identifiers.")}
                  </SectionNote>
                )}
              </div>
            );
          },
        },
        {
          section: "ChangeHistory",
          title: t("Change history"),
          hasData: changeHistory !== null,
          render: () =>
            changeHistory ? (
              <MiniTable
                columns={[
                  { id: "field", label: t("Field") },
                  { id: "changes", label: t("Changes"), align: "right" },
                  { id: "last", label: t("Last changed"), align: "right" },
                ]}
                rows={[
                  {
                    id: "name",
                    label: t("Name"),
                    count: changeHistory.nameChanges,
                    at: changeHistory.nameLastChangedAt,
                  },
                  {
                    id: "address",
                    label: t("Address"),
                    count: changeHistory.addressChanges,
                    at: changeHistory.addressLastChangedAt,
                  },
                  {
                    id: "phone",
                    label: t("Phone"),
                    count: changeHistory.phoneChanges,
                    at: changeHistory.phoneLastChangedAt,
                  },
                  {
                    id: "email",
                    label: t("Email"),
                    count: changeHistory.emailChanges,
                    at: changeHistory.emailLastChangedAt,
                  },
                  {
                    id: "contact",
                    label: t("Contact"),
                    count: changeHistory.contactChanges,
                    at: changeHistory.contactLastChangedAt,
                  },
                ].map((row) => ({
                  id: row.id,
                  cells: [
                    row.label,
                    <span key="changes" className="inline-flex items-center justify-end gap-2">
                      {(row.count ?? 0) >= FREQUENT_CHANGE_THRESHOLD ? (
                        <StatusDot tone="medium" />
                      ) : null}
                      {numberOrDash(row.count)}
                    </span>,
                    dateOrDash(row.at),
                  ],
                }))}
              />
            ) : null,
        },
      ]}
    />
  );
}
