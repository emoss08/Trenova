import { useT } from "@trenova/shared/i18n/use-t";
import { Badge } from "@trenova/shared/components/ui/badge";
import type { EDIMappingProfileItem, EDIMappingResolution } from "@trenova/shared/types/edi";
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
  const t = useT();

  if (!canResolve) {
    return <MappingSummary mappingRows={mappingRows} />;
  }

  return (
    <div className="flex flex-col gap-3 rounded-md border p-3">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div>
          <div className="font-medium">{t("Mapping Preview")}</div>
          <div className="text-muted-foreground text-xs">
            {t("Resolve required mappings before accepting and creating the receiving shipment.")}
          </div>
        </div>
        <Badge variant={unresolved.length === 0 ? "active" : "outline"}>
          {unresolved.length === 0 ? t("Ready") : t("{0} unresolved", unresolved.length)}
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
            <div className="bg-muted/20 rounded-md border p-3 text-sm">
              <div className="text-muted-foreground text-xs font-medium">{t("Source value")}</div>
              <div className="mt-1 font-medium">{row.sourceLabel || t("Unlabeled source value")}</div>
              <div className="text-muted-foreground mt-1 text-xs">{row.entityType}</div>
            </div>
            <TargetLookup
              label={t("Local record")}
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
  const t = useT();

  if (mappingRows.length === 0) {
    return <EDIEmptyState message={t("No mapping requirements were returned for this transfer.")} />;
  }

  return (
    <div className="grid gap-2">
      {mappingRows.map((row) => (
        <div
          key={mappingKey(row.entityType, row.sourceId)}
          className="bg-muted/20 rounded-md border p-3"
        >
          <div className="flex items-center justify-between gap-2">
            <span className="text-sm font-medium">{row.entityType}</span>
            <Badge variant={row.resolved ? "active" : "outline"}>
              {row.resolved ? t("Resolved") : t("Unresolved")}
            </Badge>
          </div>
          <div className="mt-3 grid gap-2 md:grid-cols-2">
            <div>
              <div className="text-muted-foreground text-xs font-medium">{t("Source value")}</div>
              <div className="mt-1 truncate text-sm">
                {row.sourceLabel || t("Unlabeled source value")}
              </div>
            </div>
            <div>
              <div className="text-muted-foreground text-xs font-medium">{t("Local record")}</div>
              <div className="mt-1 truncate text-sm">
                {row.targetLabel || (row.resolved ? t("Mapped local record") : t("No mapping saved"))}
              </div>
            </div>
          </div>
        </div>
      ))}
    </div>
  );
}
