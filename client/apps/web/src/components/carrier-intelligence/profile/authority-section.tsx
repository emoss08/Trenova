import type { CarrierIntelProfile } from "@/lib/graphql/carrier-intelligence";
import type { CarrierIntelAuthorityStatus } from "@trenova/graphql/generated/graphql";
import { useT } from "@trenova/shared/i18n/use-t";
import {
  EmptyValue,
  Fact,
  FactGrid,
  MiniTable,
  Muted,
  SubHeading,
  dateOrDash,
  numberOrDash,
  textOrDash,
} from "../intel-facts";
import { StatusDot, type StatusTone } from "../status-dot";
import { useCarrierIntelLabels } from "../use-carrier-intel-labels";
import { useIntelAgeFormatter } from "../use-intel-age";
import { ProfileTab, type ProfileSectionProps } from "./profile-tab";

type AuthorityGrant = NonNullable<NonNullable<CarrierIntelProfile["authority"]>["common"]>;

const AUTHORITY_TONE: Record<CarrierIntelAuthorityStatus, StatusTone> = {
  Active: "success",
  Inactive: "neutral",
  Revoked: "critical",
  None: "neutral",
  Unknown: "neutral",
};

function GrantStatus({ grant }: { grant: AuthorityGrant | null }) {
  const t = useT();
  const labels = useCarrierIntelLabels();
  if (!grant) {
    return <EmptyValue />;
  }
  const flags = [
    grant.pending ? t("pending") : null,
    grant.underReview ? t("under review") : null,
    grant.revocationPending ? t("revocation pending") : null,
  ].filter(Boolean);
  return (
    <span className="inline-flex items-center gap-2">
      <StatusDot tone={grant.revocationPending ? "critical" : AUTHORITY_TONE[grant.status]} />
      <span>{labels.authorityStatus[grant.status]}</span>
      {flags.length > 0 ? <Muted>{flags.join(", ")}</Muted> : null}
    </span>
  );
}

export function AuthoritySection({ profile, provider }: ProfileSectionProps) {
  const t = useT();
  const formatAge = useIntelAgeFormatter();
  const authority = profile.authority;

  return (
    <ProfileTab
      profile={profile}
      provider={provider}
      parts={[
        {
          section: "Authority",
          hasData: authority !== null,
          render: () => {
            if (!authority) {
              return null;
            }
            const grants = [
              { id: "common", label: t("Common"), grant: authority.common },
              { id: "contract", label: t("Contract"), grant: authority.contract },
              { id: "broker", label: t("Broker"), grant: authority.broker },
            ];
            const history = authority.history ?? [];
            return (
              <div className="flex flex-col gap-5">
                <MiniTable
                  columns={[
                    { id: "type", label: t("Authority") },
                    { id: "status", label: t("Status") },
                    { id: "granted", label: t("Granted") },
                    { id: "age", label: t("Age"), align: "right" },
                  ]}
                  rows={grants.map(({ id, label, grant }) => ({
                    id,
                    cells: [
                      label,
                      <GrantStatus key="status" grant={grant} />,
                      dateOrDash(grant?.grantedAt),
                      formatAge(grant?.ageDays) ?? <EmptyValue key="age" />,
                    ],
                  }))}
                />
                <FactGrid>
                  <Fact label={t("Revocations")}>{numberOrDash(authority.totalRevocations)}</Fact>
                  <Fact label={t("Last revocation")}>{dateOrDash(authority.lastRevocationAt)}</Fact>
                  <Fact label={t("USDOT added")}>{dateOrDash(profile.identity?.dotAddedAt)}</Fact>
                  <Fact label={t("USDOT age")}>
                    {formatAge(profile.identity?.dotAgeDays) ?? <EmptyValue />}
                  </Fact>
                </FactGrid>
                {history.length > 0 ? (
                  <div className="flex flex-col gap-1">
                    <SubHeading>{t("History")}</SubHeading>
                    <MiniTable
                      columns={[
                        { id: "served", label: t("Served") },
                        { id: "type", label: t("Authority") },
                        { id: "action", label: t("Action") },
                      ]}
                      rows={history.map((entry, index) => ({
                        id: `${entry.authorityType}-${entry.action}-${entry.servedAt ?? "none"}-${index}`,
                        cells: [
                          dateOrDash(entry.servedAt),
                          textOrDash(entry.authorityType),
                          textOrDash(entry.action),
                        ],
                      }))}
                    />
                  </div>
                ) : null}
              </div>
            );
          },
        },
      ]}
    />
  );
}
