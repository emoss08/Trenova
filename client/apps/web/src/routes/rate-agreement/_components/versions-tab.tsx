import { apiService } from "@/services/api";
import { useQuery } from "@tanstack/react-query";
import { Badge } from "@trenova/shared/components/ui/badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@trenova/shared/components/ui/table";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import type { RateAgreementVersion } from "@trenova/shared/types/rate";
import { ClockIcon } from "lucide-react";

type VersionsTabProps = {
  /** Absent while the agreement is being created — there is no history yet. */
  readonly rateAgreementId?: string;
};

/**
 * The header terms as they stood at each renegotiation.
 *
 * Rules carry their own effective windows, so this list is deliberately short:
 * it moves when somebody renegotiates the contract, not when they touch a lane.
 */
export function VersionsTab({ rateAgreementId }: VersionsTabProps) {
  const { data: versions } = useQuery({
    queryKey: ["rate-agreement-versions", rateAgreementId],
    queryFn: () => apiService.rateAgreementService.listVersions(rateAgreementId as string),
    enabled: Boolean(rateAgreementId),
  });

  if (!rateAgreementId) {
    return (
      <p className="text-muted-foreground text-sm">
        Save the agreement first. Versions record the terms as they stood, and there are no terms
        until there is a contract.
      </p>
    );
  }

  if (!versions || versions.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center rounded-lg border border-dashed py-12 text-center">
        <ClockIcon className="text-muted-foreground mb-3 size-8" />
        <p className="text-sm font-medium">No versions recorded yet</p>
        <p className="text-muted-foreground mt-1 max-w-sm text-xs">
          A version is written when the negotiated header terms change, so a dispute can always read
          the contract as it stood on a date.
        </p>
      </div>
    );
  }

  return (
    <div className="overflow-hidden rounded-lg border">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead className="w-16 text-xs">Version</TableHead>
            <TableHead className="text-xs">In Force</TableHead>
            <TableHead className="text-xs">What Changed</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {versions.map((version) => (
            <TableRow key={version.id ?? version.versionNumber}>
              <TableCell>
                <Badge variant="outline" className="text-2xs font-mono">
                  v{version.versionNumber}
                </Badge>
              </TableCell>
              <TableCell className="text-xs whitespace-nowrap">
                {formatUnixDateMedium(version.effectiveFrom)}
                <span className="text-muted-foreground">
                  {" — "}
                  {version.effectiveTo ? formatUnixDateMedium(version.effectiveTo) : "current"}
                </span>
              </TableCell>
              <TableCell className="text-muted-foreground text-xs">
                {describeVersion(version)}
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}

function describeVersion(version: RateAgreementVersion): string {
  if (version.changeMessage) return version.changeMessage;

  const changedFields = Object.keys(version.changeSummary ?? {});
  if (changedFields.length === 0) return "—";

  return changedFields.join(", ");
}
