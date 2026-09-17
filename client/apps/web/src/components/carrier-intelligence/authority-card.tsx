import { useT } from "@trenova/shared/i18n/use-t";
import type { CarrierIntelAuthorityStatus } from "@trenova/graphql/generated/graphql";
import type { CarrierIntelProfile } from "@/lib/graphql/carrier-intelligence";
import { Badge, type BadgeVariant } from "@trenova/shared/components/ui/badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@trenova/shared/components/ui/table";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { LandmarkIcon } from "lucide-react";
import {
  IntelDate,
  IntelField,
  IntelFieldGrid,
  IntelNumber,
  IntelSectionCard,
} from "./intel-section-card";
import { useCarrierIntelLabels } from "./use-carrier-intel-labels";

type AuthorityGrant = NonNullable<NonNullable<CarrierIntelProfile["authority"]>["common"]>;

export type AuthorityCardProps = {
  profile: CarrierIntelProfile;
  provider: string | null | undefined;
  className?: string;
};

const STATUS_VARIANTS: Record<CarrierIntelAuthorityStatus, BadgeVariant> = {
  Active: "active",
  Inactive: "secondary",
  Revoked: "inactive",
  None: "outline",
  Unknown: "outline",
};

function GrantRow({ label, grant }: { label: string; grant: AuthorityGrant | null }) {
  const t = useT();
  const labels = useCarrierIntelLabels();

  return (
    <TableRow>
      <TableCell className="font-medium">{label}</TableCell>
      <TableCell>
        {grant ? (
          <div className="flex flex-wrap items-center gap-1">
            <Badge variant={STATUS_VARIANTS[grant.status]} className="max-h-5">
              {labels.authorityStatus[grant.status]}
            </Badge>
            {grant.pending ? (
              <Badge variant="info" className="max-h-5">
                {t("Pending")}
              </Badge>
            ) : null}
            {grant.underReview ? (
              <Badge variant="warning" className="max-h-5">
                {t("Under review")}
              </Badge>
            ) : null}
            {grant.revocationPending ? (
              <Badge variant="inactive" className="max-h-5">
                {t("Revocation pending")}
              </Badge>
            ) : null}
          </div>
        ) : (
          <span className="text-muted-foreground">-</span>
        )}
      </TableCell>
      <TableCell>{grant?.grantedAt ? formatUnixDateMedium(grant.grantedAt) : "-"}</TableCell>
      <TableCell className="text-right tabular-nums">
        {grant?.ageDays ?? <span className="text-muted-foreground">-</span>}
      </TableCell>
    </TableRow>
  );
}

export function AuthorityCard({ profile, provider, className }: AuthorityCardProps) {
  const t = useT();
  const authority = profile.authority;
  const revocationPending = Boolean(
    authority?.common?.revocationPending ||
    authority?.contract?.revocationPending ||
    authority?.broker?.revocationPending,
  );

  return (
    <IntelSectionCard
      title={t("Operating authority")}
      icon={LandmarkIcon}
      coverage={profile.coverage}
      provider={provider}
      emphasis={revocationPending ? "danger" : "none"}
      className={className}
      parts={[
        {
          section: "Authority",
          hasData: authority !== null,
          content: authority ? (
            <div className="flex flex-col gap-3">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>{t("Authority")}</TableHead>
                    <TableHead>{t("Status")}</TableHead>
                    <TableHead>{t("Granted")}</TableHead>
                    <TableHead className="text-right">{t("Age (days)")}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  <GrantRow label={t("Common")} grant={authority.common} />
                  <GrantRow label={t("Contract")} grant={authority.contract} />
                  <GrantRow label={t("Broker")} grant={authority.broker} />
                </TableBody>
              </Table>
              <IntelFieldGrid>
                <IntelField label={t("Total revocations")}>
                  <IntelNumber value={authority.totalRevocations} />
                </IntelField>
                <IntelField label={t("Last revocation")}>
                  <IntelDate value={authority.lastRevocationAt} />
                </IntelField>
              </IntelFieldGrid>
              {authority.history && authority.history.length > 0 ? (
                <div className="flex flex-col gap-1">
                  <h4 className="text-muted-foreground text-xs font-medium">
                    {t("Authority history")}
                  </h4>
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>{t("Type")}</TableHead>
                        <TableHead>{t("Action")}</TableHead>
                        <TableHead>{t("Served")}</TableHead>
                        <TableHead>{t("Effective")}</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {authority.history.map((entry, index) => (
                        <TableRow key={`${entry.authorityType}-${entry.action}-${index}`}>
                          <TableCell>{entry.authorityType}</TableCell>
                          <TableCell>{entry.action}</TableCell>
                          <TableCell>
                            {entry.servedAt ? formatUnixDateMedium(entry.servedAt) : "-"}
                          </TableCell>
                          <TableCell>
                            {entry.effectiveAt ? formatUnixDateMedium(entry.effectiveAt) : "-"}
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </div>
              ) : null}
            </div>
          ) : null,
        },
      ]}
    />
  );
}
