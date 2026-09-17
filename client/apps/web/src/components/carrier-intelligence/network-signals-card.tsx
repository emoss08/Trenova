import { useT } from "@trenova/shared/i18n/use-t";
import type { CarrierIntelNetworkKind } from "@trenova/graphql/generated/graphql";
import type { CarrierIntelProfile } from "@/lib/graphql/carrier-intelligence";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { Badge } from "@trenova/shared/components/ui/badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@trenova/shared/components/ui/table";
import { cn } from "@trenova/shared/lib/utils";
import { NetworkIcon, SirenIcon } from "lucide-react";
import { IntelSectionCard } from "./intel-section-card";
import { useCarrierIntelLabels } from "./use-carrier-intel-labels";

export type NetworkSignalsCardProps = {
  profile: CarrierIntelProfile;
  provider: string | null | undefined;
  className?: string;
};

const HIGH_RISK_KINDS: ReadonlySet<CarrierIntelNetworkKind> = new Set(["EIN", "Equipment"]);

export function NetworkSignalsCard({ profile, provider, className }: NetworkSignalsCardProps) {
  const t = useT();
  const labels = useCarrierIntelLabels();
  const network = profile.network;

  const counters: { kind: CarrierIntelNetworkKind; count: number | null }[] = network
    ? [
        { kind: "Address", count: network.sharedAddresses },
        { kind: "Phone", count: network.sharedPhones },
        { kind: "Email", count: network.sharedEmails },
        { kind: "EIN", count: network.sharedEins },
        { kind: "Equipment", count: network.sharedEquipment },
      ]
    : [];
  const totalShared = counters.reduce((sum, counter) => sum + (counter.count ?? 0), 0);
  const highRisk = counters.some(
    (counter) => HIGH_RISK_KINDS.has(counter.kind) && (counter.count ?? 0) > 0,
  );
  const links = network?.links ?? [];

  return (
    <IntelSectionCard
      title={t("Network signals")}
      icon={NetworkIcon}
      coverage={profile.coverage}
      provider={provider}
      emphasis={highRisk ? "danger" : totalShared > 0 ? "warning" : "none"}
      className={className}
      parts={[
        {
          section: "Network",
          hasData: network !== null,
          content: network ? (
            <div className="flex flex-col gap-3">
              {totalShared > 0 ? (
                <Alert variant={highRisk ? "destructive" : "warning"}>
                  <SirenIcon />
                  <AlertTitle>
                    {highRisk
                      ? t("Identity overlaps with other carriers")
                      : t("Contact details shared with other carriers")}
                  </AlertTitle>
                  <AlertDescription>
                    {highRisk
                      ? t(
                          "A shared EIN or shared equipment is a common sign of chameleon carriers and double brokering. Confirm who you are dealing with before tendering.",
                        )
                      : t(
                          "Shared addresses, phones or emails can be innocent (a factoring company, a shared office) but are also how fraudulent carriers reappear under new names.",
                        )}
                  </AlertDescription>
                </Alert>
              ) : (
                <p className="text-muted-foreground text-sm">
                  {t("No other carriers share this carrier's identifying details.")}
                </p>
              )}
              <ul className="grid grid-cols-2 gap-2 sm:grid-cols-5">
                {counters.map((counter) => {
                  const count = counter.count ?? 0;
                  const danger = HIGH_RISK_KINDS.has(counter.kind) && count > 0;
                  return (
                    <li
                      key={counter.kind}
                      className={cn(
                        "flex flex-col gap-0.5 rounded-md border px-2.5 py-2",
                        danger && "border-red-600/50 bg-red-600/10",
                        !danger && count > 0 && "border-yellow-600/40 bg-yellow-600/10",
                      )}
                    >
                      <span className="text-muted-foreground text-xs">
                        {labels.networkKind[counter.kind]}
                      </span>
                      <span
                        className={cn(
                          "text-lg font-semibold tabular-nums",
                          danger && "text-red-700 dark:text-red-400",
                        )}
                      >
                        {counter.count ?? "-"}
                      </span>
                    </li>
                  );
                })}
              </ul>
              {links.length > 0 ? (
                <div className="max-h-72 overflow-auto rounded-md border">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>{t("Link")}</TableHead>
                        <TableHead>{t("Shared value")}</TableHead>
                        <TableHead>{t("Other carrier")}</TableHead>
                        <TableHead>{t("Status")}</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {links.map((link, index) => (
                        <TableRow
                          key={`${link.kind}-${link.dotNumber ?? ""}-${link.value ?? ""}-${index}`}
                        >
                          <TableCell>
                            <Badge
                              variant={HIGH_RISK_KINDS.has(link.kind) ? "inactive" : "warning"}
                              className="max-h-5"
                            >
                              {labels.networkKind[link.kind]}
                            </Badge>
                          </TableCell>
                          <TableCell className="max-w-48 truncate" title={link.value ?? undefined}>
                            {link.value ?? "-"}
                          </TableCell>
                          <TableCell>
                            <span className="flex flex-col">
                              <span>{link.legalName ?? "-"}</span>
                              {link.dotNumber ? (
                                <span className="text-muted-foreground font-mono text-2xs">
                                  {t("USDOT {0}", link.dotNumber)}
                                </span>
                              ) : null}
                            </span>
                          </TableCell>
                          <TableCell>{link.status ?? "-"}</TableCell>
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
