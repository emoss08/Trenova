import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { SectionPanel, SectionPanelQuiet } from "@/components/section-panel";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import {
  EXTRACTION_ROLLOUT_KEY,
  EXTRACTION_ROLLOUT_REPORT_KEY,
  fetchExtractionRollout,
  fetchExtractionRolloutReport,
  updateExtractionRollout,
  type ExtractionRollout,
  type ExtractionRolloutReport,
} from "@/lib/graphql/extraction-rollout";
import { queries } from "@/lib/queries";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import {
  DescriptionEmpty,
  DescriptionItem,
  DescriptionList,
} from "@trenova/shared/components/ui/description-list";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { AlertCircleIcon, Sliders01Icon, XOctagonIcon } from "@trenova/shared/components/icons";
import { useState } from "react";
import { toast } from "sonner";
import { formatShare } from "../quality-model";
import { RolloutStateBadge } from "./extraction-badges";
import { EXTRACTION_STALE_MS } from "./extraction-model";
import { RolloutFigures } from "./rollout-figures";
import { rolloutDraftOf, rolloutInput, rolloutState } from "./rollout-model";
import { RolloutSettingsDialog } from "./rollout-settings-dialog";
import { ShadowFieldsTable } from "./shadow-fields-table";

function RolloutSummary({ rollout }: { rollout: ExtractionRollout }) {
  const t = useT();
  const providers = useQuery({ ...queries.aiProvider.list() });
  const provider = providers.data?.find((item) => item.id === rollout.providerId);

  return (
    <DescriptionList columns={4} className="p-3">
      <DescriptionItem label={t("Status")}>
        <RolloutStateBadge value={rolloutState(rollout)} t={t} />
      </DescriptionItem>
      <DescriptionItem label={t("Candidate provider")}>
        {provider ? t("{0} · {1}", provider.name, provider.model) : <DescriptionEmpty />}
      </DescriptionItem>
      <DescriptionItem label={t("Share of documents")} numeric>
        {t("{0}%", rollout.percent)}
      </DescriptionItem>
      <DescriptionItem label={t("Comparing since")}>
        {rollout.startedAt ? (
          <HoverCardTimestamp timestamp={rollout.startedAt} />
        ) : (
          <DescriptionEmpty />
        )}
      </DescriptionItem>
    </DescriptionList>
  );
}

function HaltAlert({ rollout }: { rollout: ExtractionRollout }) {
  const t = useT();
  if (!rollout.haltedAt || !rollout.haltReason) return null;

  const candidate = formatShare(rollout.haltCandidateRate);
  const baseline = formatShare(rollout.haltBaselineRate);

  return (
    <Alert variant="destructive" size="sm">
      <XOctagonIcon />
      <AlertTitle>{t("A guard stopped the rollout")}</AlertTitle>
      <AlertDescription>
        <p>
          {rollout.haltReason === "AccuracyDrop"
            ? t(
                "The candidate read {0} of confirmed fields correctly against {1} for production, more than {2} points below.",
                candidate,
                baseline,
                rollout.maxAccuracyDropPoints,
              )
            : t(
                "The candidate's answers were unusable on {0} of its documents against {1} for production, more than {2} points above.",
                candidate,
                baseline,
                rollout.maxRejectionIncreasePoints,
              )}
        </p>
        <p>
          {t(
            "Every document is back on production. Save the settings with the rollout on to start a new comparison.",
          )}
        </p>
        <p className="text-muted-foreground flex flex-wrap items-center gap-1 text-xs">
          {t("Stopped")}
          <HoverCardTimestamp timestamp={rollout.haltedAt} />
        </p>
      </AlertDescription>
    </Alert>
  );
}

function GuardProgress({ report }: { report: ExtractionRolloutReport }) {
  const t = useT();
  const { rollout } = report;
  const scored = Math.min(report.candidateAccuracy.scored, report.productionAccuracy.scored);
  const candidateSettled =
    report.candidate.accepted + report.candidate.rejected + report.candidate.failed;
  const controlSettled = report.control.accepted + report.control.rejected + report.control.failed;
  const settled = Math.min(candidateSettled, controlSettled);

  return (
    <SectionPanel
      title={t("Guards")}
      help={t(
        "A guard acts only once both sides have enough evidence, so a handful of hard documents cannot stop the rollout on their own.",
      )}
    >
      <DescriptionList columns={2} className="p-3">
        <DescriptionItem label={t("Accuracy guard")}>
          <div className="flex flex-col gap-0.5">
            <span>
              {t(
                "Stops when the candidate is more than {0} points below production",
                rollout.maxAccuracyDropPoints,
              )}
            </span>
            <span className="text-muted-foreground text-xs">
              {scored >= report.minGuardScoredFields
                ? t("Watching")
                : t(
                    "{0} of {1} confirmed fields on each side before it can act",
                    scored,
                    report.minGuardScoredFields,
                  )}
            </span>
          </div>
        </DescriptionItem>
        <DescriptionItem label={t("Unusable answer guard")}>
          <div className="flex flex-col gap-0.5">
            <span>
              {t(
                "Stops when the candidate's answers are unusable more than {0} points more often than production's",
                rollout.maxRejectionIncreasePoints,
              )}
            </span>
            <span className="text-muted-foreground text-xs">
              {settled >= report.minGuardExtractions
                ? t("Watching")
                : t(
                    "{0} of {1} finished extractions on each side before it can act",
                    settled,
                    report.minGuardExtractions,
                  )}
            </span>
          </div>
        </DescriptionItem>
      </DescriptionList>
    </SectionPanel>
  );
}

/**
 * A candidate provider serving a share of real extractions: its answers fill
 * shipment drafts for the documents it is given, the rest stay on production,
 * and the two are compared on what people confirm. Guards stop it on their own
 * when it does worse, and one switch stops it at any time.
 */
export function RolloutView() {
  const t = useT();
  const queryClient = useQueryClient();
  const [editing, setEditing] = useState(false);
  const { allowed: canUpdate } = usePermission(Resource.AIProvider, Operation.Update);

  const rollout = useQuery({
    queryKey: [EXTRACTION_ROLLOUT_KEY],
    queryFn: ({ signal }) => fetchExtractionRollout({ signal }),
    staleTime: EXTRACTION_STALE_MS,
  });
  const report = useQuery({
    queryKey: [EXTRACTION_ROLLOUT_REPORT_KEY],
    queryFn: ({ signal }) => fetchExtractionRolloutReport({ signal }),
    staleTime: EXTRACTION_STALE_MS,
  });

  const stop = useApiMutation({
    mutationFn: (current: ExtractionRollout) => {
      const input = rolloutInput({ ...rolloutDraftOf(current), enabled: false }, current.version);
      if (!input) {
        throw new Error("The rollout settings have problems");
      }
      return updateExtractionRollout(input);
    },
    onSuccess: async (saved) => {
      toast.success(t("The rollout is off; every document is read by production"));
      queryClient.setQueryData([EXTRACTION_ROLLOUT_KEY], saved);
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: [EXTRACTION_ROLLOUT_KEY] }),
        queryClient.invalidateQueries({ queryKey: [EXTRACTION_ROLLOUT_REPORT_KEY] }),
      ]);
    },
    resourceName: t("Rollout"),
  });

  const current = rollout.data;

  return (
    <div className="flex min-w-0 flex-col gap-4">
      <SectionPanel
        title={t("Gradual rollout")}
        help={t(
          "The candidate's answers are used for real on its share of documents and billed as production usage. Accuracy counts only documents a person confirmed.",
        )}
        action={
          canUpdate && current ? (
            <div className="flex items-center gap-2">
              {current.serving ? (
                <Button
                  type="button"
                  size="sm"
                  variant="destructive"
                  isLoading={stop.isPending}
                  onClick={() => stop.mutate(current)}
                >
                  <XOctagonIcon />
                  {t("Stop rollout")}
                </Button>
              ) : null}
              <Button type="button" size="sm" variant="outline" onClick={() => setEditing(true)}>
                <Sliders01Icon />
                {t("Edit settings")}
              </Button>
            </div>
          ) : undefined
        }
      >
        {rollout.isError ? (
          <SectionPanelQuiet>{t("The rollout could not be loaded.")}</SectionPanelQuiet>
        ) : current ? (
          <RolloutSummary rollout={current} />
        ) : (
          <Skeleton className="m-3 h-10" aria-busy />
        )}
      </SectionPanel>

      {current ? <HaltAlert rollout={current} /> : null}

      {report.isError ? (
        <Alert variant="destructive" size="sm">
          <AlertCircleIcon />
          <AlertDescription>
            {t("The rollout comparison could not be loaded. Try again shortly.")}
          </AlertDescription>
        </Alert>
      ) : report.data ? (
        report.data.rollout.startedAt ? (
          <>
            <RolloutFigures report={report.data} />
            <GuardProgress report={report.data} />
            {report.data.truncated ? (
              <Alert variant="info" size="sm">
                <AlertDescription>
                  {t("Only the most recent corrections are counted in the field comparison.")}
                </AlertDescription>
              </Alert>
            ) : null}
            <SectionPanel
              title={t("Accuracy by field")}
              hint={t("biggest shortfall first")}
              help={t(
                "Correct fields divided by fields a person confirmed: the candidate on the documents it read, production on the rest.",
              )}
            >
              {report.data.fields.length > 0 ? (
                <ShadowFieldsTable fields={report.data.fields} />
              ) : (
                <SectionPanelQuiet>
                  {t(
                    "No document read during the rollout has been confirmed yet. Scores appear once someone creates a shipment from one.",
                  )}
                </SectionPanelQuiet>
              )}
            </SectionPanel>
          </>
        ) : (
          <SectionPanel title={t("Comparison")}>
            <SectionPanelQuiet>
              {t(
                "Choose a candidate and turn the rollout on to send it a share of real documents.",
              )}
            </SectionPanelQuiet>
          </SectionPanel>
        )
      ) : (
        <Skeleton className="h-16" aria-busy />
      )}

      {current ? (
        <RolloutSettingsDialog open={editing} onOpenChange={setEditing} rollout={current} />
      ) : null}
    </div>
  );
}
