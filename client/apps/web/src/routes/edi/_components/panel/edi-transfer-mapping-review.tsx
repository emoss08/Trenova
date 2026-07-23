import { Badge } from "@/components/ui/badge";
import type { EDIMappingProfileItem, EDIMappingResolution } from "@/types/edi";
import { mappingKey } from "../edi-display-utils";
import { TargetLookup } from "../edi-target-lookup";
import { EDIEmptyState } from "./edi-panel-primitives";

type MappingReviewProps = {
  canResolve: boolean;
  inlineMappings: Record<string, EDIMappingProfileItem>;
  mappingRows: EDIMappingResolution[];
  setInlineMappings: React.Dispatch<React.SetStateAction<Record<string, EDIMappingProfileItem>>>;
  unresolved: EDIMappingResolution[];
};

export function MappingReview({
  canResolve,
  inlineMappings,
  mappingRows,
  setInlineMappings,
  unresolved,
}: MappingReviewProps) {
  if (!canResolve) {
    return <MappingSummary mappingRows={mappingRows} />;
  }

  return (
    <div className="flex flex-col gap-3 rounded-md border p-3">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div>
          <div className="font-medium">Mapping Preview</div>
          <div className="text-xs text-muted-foreground">
            Resolve required mappings before accepting and creating the receiving shipment.
          </div>
        </div>
        <Badge variant={unresolved.length === 0 ? "active" : "outline"}>
          {unresolved.length === 0 ? "Ready" : `${unresolved.length} unresolved`}
        </Badge>
      </div>
      {unresolved.length === 0 ? (
        <MappingSummary mappingRows={mappingRows} />
      ) : (
        unresolved.map((row) => (
          <div
            key={mappingKey(row.entityType, row.sourceId)}
            className="grid gap-2 md:grid-cols-[1fr_1fr]"
          >
            <div className="rounded-md border bg-muted/20 p-3 text-sm">
              <div className="text-xs font-medium text-muted-foreground">Source value</div>
              <div className="mt-1 font-medium">{row.sourceLabel || "Unlabeled source value"}</div>
              <div className="mt-1 text-xs text-muted-foreground">{row.entityType}</div>
            </div>
            <TargetLookup
              label="Local record"
              entityType={row.entityType}
              value={inlineMappings[mappingKey(row.entityType, row.sourceId)]?.targetId ?? ""}
              onChange={(target) => {
                const key = mappingKey(row.entityType, row.sourceId);
                setInlineMappings((current) => ({
                  ...current,
                  [key]: {
                    entityType: row.entityType,
                    sourceId: row.sourceId,
                    sourceLabel: row.sourceLabel ?? "",
                    targetId: target.targetId,
                    targetLabel: target.targetLabel,
                  },
                }));
              }}
            />
          </div>
        ))
      )}
    </div>
  );
}

export function MappingSummary({ mappingRows }: { mappingRows: EDIMappingResolution[] }) {
  if (mappingRows.length === 0) {
    return <EDIEmptyState message="No mapping requirements were returned for this transfer." />;
  }

  return (
    <div className="grid gap-2">
      {mappingRows.map((row) => (
        <div
          key={mappingKey(row.entityType, row.sourceId)}
          className="rounded-md border bg-muted/20 p-3"
        >
          <div className="flex items-center justify-between gap-2">
            <span className="text-sm font-medium">{row.entityType}</span>
            <Badge variant={row.resolved ? "active" : "outline"}>
              {row.resolved ? "Resolved" : "Unresolved"}
            </Badge>
          </div>
          <div className="mt-3 grid gap-2 md:grid-cols-2">
            <div>
              <div className="text-xs font-medium text-muted-foreground">Source value</div>
              <div className="mt-1 truncate text-sm">
                {row.sourceLabel || "Unlabeled source value"}
              </div>
            </div>
            <div>
              <div className="text-xs font-medium text-muted-foreground">Local record</div>
              <div className="mt-1 truncate text-sm">
                {row.targetLabel || (row.resolved ? "Mapped local record" : "No mapping saved")}
              </div>
            </div>
          </div>
        </div>
      ))}
    </div>
  );
}
