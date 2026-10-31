import { permitChecklist } from "@/lib/permit-checklist";
import { shipment as shipmentQueries } from "@/lib/queries/shipment";
import { useQuery } from "@tanstack/react-query";
import { RulerIcon, ShieldQuestionIcon } from "@trenova/shared/components/icons";
import {
  Alert,
  AlertAction,
  AlertDescription,
  AlertTitle,
} from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import {
  dimensionRows,
  escortSummary,
  pickupIsTooSoon,
  restrictionSummary,
  unverifiedJurisdictions,
} from "@trenova/shared/lib/permit";
import type { Permit, PermitRequirement } from "@trenova/shared/types/permit";
import type { Shipment } from "@trenova/shared/types/shipment";
import { useState, type ReactNode } from "react";
import { PermitRecordDialog, PermitWaiveDialog } from "../permit-dialogs";
import { PermitSection, PermitSummary, SizeMeters, TripNeeds } from "./permit-overview";
import { PermitStateList } from "./permit-state-list";
import { PermitsTabSkeleton } from "./permits-tab-skeleton";

function Notice({ children }: { children: ReactNode }) {
  return (
    <div className="flex items-start gap-2.5 rounded-lg border border-dashed px-3 py-2.5">
      <RulerIcon aria-hidden className="text-foreground-muted mt-0.5 size-3.5 shrink-0" />
      <p className="text-foreground-muted text-xs">{children}</p>
    </div>
  );
}

/**
 * The shipment's oversize permits as a checklist: how many are still to file and when the load
 * can leave at the earliest, the load's size against the tightest limit on the route, a line per
 * state (what it needs, the permit that covers it, or why it was waived), what the trip needs
 * beyond the paper, and which limits are still unconfirmed. It reads the saved shipment: the
 * assessment is derived from what was last saved, so it follows each saved version.
 */
export default function ShipmentPermitsTab({ shipment }: { shipment: Shipment | undefined }) {
  const t = useT();
  const shipmentId = shipment?.id ?? "";

  const assessmentQuery = useQuery({
    ...shipmentQueries.permitAssessment(shipmentId, shipment?.version),
    enabled: !!shipmentId,
  });
  // The assessment's requirements are freshly derived and carry no id, so actions read the
  // persisted set; the derived list stands in for a shipment saved before the engine ran.
  const { data: persisted } = useQuery({
    ...shipmentQueries.permitRequirements(shipmentId),
    enabled: !!shipmentId,
  });
  const { data: permits } = useQuery({
    ...shipmentQueries.permits(shipmentId),
    enabled: !!shipmentId,
  });

  const [recording, setRecording] = useState<PermitRequirement | null>(null);
  const [waiving, setWaiving] = useState<PermitRequirement | null>(null);
  const [editing, setEditing] = useState<Permit | null>(null);

  if (!shipmentId) return null;
  if (assessmentQuery.isPending) return <PermitsTabSkeleton />;

  const assessment = assessmentQuery.data;
  if (assessmentQuery.isError || !assessment) {
    return (
      <Alert variant="destructive" size="sm">
        <ShieldQuestionIcon />
        <AlertTitle>{t("The permit assessment could not be loaded")}</AlertTitle>
        <AlertDescription>
          {t("Permits, limits and escorts for this shipment are not shown until it loads.")}
        </AlertDescription>
        <AlertAction>
          <Button
            type="button"
            variant="ghost"
            size="xs"
            onClick={() => void assessmentQuery.refetch()}
          >
            {t("Retry")}
          </Button>
        </AlertAction>
      </Alert>
    );
  }

  if (assessment.measurements.widthFeet === 0 && assessment.measurements.lengthFeet === 0) {
    return (
      <Notice>
        {t(
          "Add length, width and height to the commodity lines. Deck fit, permits, escorts and lead time are all worked out from those numbers.",
        )}
      </Notice>
    );
  }

  const requirements = persisted?.length ? persisted : (assessment.requirements ?? []);
  const checklist = permitChecklist(requirements, permits ?? []);
  const unverified = unverifiedJurisdictions(assessment);
  const scheduledPickupAt = shipment?.moves?.[0]?.stops?.[0]?.scheduledWindowStart;

  return (
    <div className="flex flex-col gap-6">
      <PermitSummary
        assessment={assessment}
        open={checklist.open}
        resolved={checklist.resolved}
        pickupTooSoon={pickupIsTooSoon(assessment, scheduledPickupAt)}
      />

      {!assessment.routeResolved && (
        <Notice>
          {t(
            "No state rules matched the stops on this shipment, so no limits were checked. That is not the same as a legal load: add the stop locations, or make sure the states on this route have rules set up.",
          )}
        </Notice>
      )}

      <SizeMeters rows={dimensionRows(assessment)} />

      {checklist.entries.length > 0 && (
        <PermitSection
          title={t("By state")}
          meta={
            checklist.open > 0 ? (
              <span className="text-warning-foreground text-2xs font-medium tabular-nums">
                {t("{0, plural, one {# to file} other {# to file}}", checklist.open)}
              </span>
            ) : (
              <span className="text-foreground-muted text-2xs">{t("All resolved")}</span>
            )
          }
        >
          <PermitStateList
            entries={checklist.entries}
            onRecord={setRecording}
            onWaive={setWaiving}
            onEdit={setEditing}
          />
        </PermitSection>
      )}

      <TripNeeds
        escorts={escortSummary(requirements)}
        restrictions={restrictionSummary(assessment.jurisdictions)}
      />

      {unverified.length > 0 && (
        <Alert variant="warning" size="sm">
          <ShieldQuestionIcon />
          <AlertTitle>
            {t(
              "Unconfirmed limits for {0}",
              unverified.map((jurisdiction) => jurisdiction.stateCode).join(", "),
            )}
          </AlertTitle>
          <AlertDescription>
            {t(
              "These limits come from Trenova's researched baseline and your organization has not confirmed them with the issuing authority. Check them in jurisdiction rules before relying on them for a permit filing.",
            )}
          </AlertDescription>
        </Alert>
      )}

      <PermitRecordDialog
        open={recording !== null}
        onOpenChange={(next) => !next && setRecording(null)}
        shipmentId={shipmentId}
        requirement={recording}
      />
      <PermitWaiveDialog
        open={waiving !== null}
        onOpenChange={(next) => !next && setWaiving(null)}
        shipmentId={shipmentId}
        requirement={waiving}
      />
      <PermitRecordDialog
        open={editing !== null}
        onOpenChange={(next) => !next && setEditing(null)}
        shipmentId={shipmentId}
        requirement={null}
        permit={editing}
      />
    </div>
  );
}
