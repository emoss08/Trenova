import { ColorOptionValue } from "@/components/fields/select-components";
import { ReviewHistoryView } from "@/components/tariff-studio/review-history-view";
import type { ReviewEntryData } from "@/components/tariff-studio/review-rounds";
import { usePermission, useReviewApproval } from "@/hooks/use-permission";
import { formulaTemplateStatusChoices, rateMatrixStatusChoices } from "@/lib/choices";
import { formulaTemplateRoutes } from "@/lib/formula-template-routes";
import type { RateMatrixRecord, RateMatrixRecordVersion } from "@/lib/graphql/rate-matrix";
import { rateMatrix } from "@/lib/queries/rate-matrix";
import { useQuery } from "@tanstack/react-query";
import { LazyComponent } from "@trenova/shared/components/error-boundary";
import {
  ArchiveIcon,
  CheckIcon,
  ClockRewindIcon,
  EyeIcon,
  MessageAlertSquareIcon,
  Send01Icon,
  XCloseIcon,
} from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useT } from "@trenova/shared/i18n/use-t";
import { Operation, Resource } from "@trenova/shared/types/permission";
import {
  formulaReviewDecisionSchema,
  type FormulaTemplateStatus,
} from "@trenova/shared/types/formula-template";
import {
  rateMatrixDimensionSchema,
  rateMatrixValueTypeSchema,
  type RateMatrixDimension,
} from "@trenova/shared/types/rate";
import { formatDistanceToNow } from "date-fns";
import { useState } from "react";
import { Link } from "react-router";
import { MatrixGridEditor } from "./matrix-grid-editor";
import { MatrixGridEditorSkeleton } from "./matrix-grid-editor-skeleton";
import {
  RateMatrixArchiveDialog,
  RateMatrixDecisionDialog,
  RateMatrixRestoreDialog,
  RateMatrixSubmitDialog,
  type RateMatrixDecision,
} from "./rate-matrix-review-dialogs";
import { RateMatrixVersionsSkeleton } from "./rate-matrix-versions-skeleton";

/** The matrix's review rounds as the shared review history reads them. */
export function matrixReviewEntries(matrix: RateMatrixRecord): ReviewEntryData[] {
  return matrix.reviews.flatMap((review) => {
    const decision = formulaReviewDecisionSchema.safeParse(review.decision);
    if (!decision.success) return [];
    return [
      {
        id: review.id,
        round: review.round,
        decision: decision.data,
        comment: review.comment,
        reviewerNote: review.reviewerNote,
        baseVersionNumber: review.baseVersionNumber,
        createdAt: review.createdAt,
        actor: review.actor,
        requestedReviewer: review.requestedReviewer,
      },
    ];
  });
}

/** What the matrix can do next from its review state. */
export function matrixActions(
  matrix: Pick<RateMatrixRecord, "status" | "hasDraftChanges" | "publishedVersionNumber">,
) {
  const inReview = matrix.status === "InReview";
  const archived = matrix.status === "Inactive";
  return {
    submit: !inReview && (archived || matrix.hasDraftChanges),
    decide: inReview,
    archive: !inReview && !archived && matrix.publishedVersionNumber > 0,
  };
}

function StatusValue({ status }: { status: FormulaTemplateStatus }) {
  const t = useT();
  const choice = rateMatrixStatusChoices.find((option) => option.value === status);
  return <ColorOptionValue color={choice?.color} value={choice ? t(choice.label) : status} />;
}

function VersionStatus({ status }: { status: FormulaTemplateStatus }) {
  const t = useT();
  const choice = formulaTemplateStatusChoices.find((option) => option.value === status);
  return (
    <ColorOptionValue
      color={choice?.color}
      value={choice ? t(choice.label) : status}
      className="text-xs"
    />
  );
}

function stateSummary(matrix: RateMatrixRecord, t: ReturnType<typeof useT>): string {
  if (matrix.status === "Inactive") {
    return t(
      "Archived. No new template version can read it; those that pin a version keep reading it.",
    );
  }
  if (matrix.status === "InReview") {
    return t("The draft is waiting for a reviewer to publish it.");
  }
  if (matrix.publishedVersionNumber === 0) {
    return t("Never published. Tariffs cannot read it until a reviewer publishes the draft.");
  }
  if (matrix.hasDraftChanges) {
    return t(
      "Tariffs read version {0}. The draft has changes that are not published.",
      matrix.publishedVersionNumber,
    );
  }
  return t("Tariffs read version {0}, and the draft matches it.", matrix.publishedVersionNumber);
}

/** One published version's rates, read-only, with the axes it was published with. */
function VersionSheet({ matrixId, versionNumber }: { matrixId: string; versionNumber: number }) {
  const t = useT();
  const axes = useQuery(rateMatrix.versionAxes(matrixId, versionNumber));

  if (axes.isPending) return <MatrixGridEditorSkeleton />;
  if (axes.isError || !axes.data) {
    return (
      <p className="text-muted-foreground text-sm">{t("This version could not be loaded.")}</p>
    );
  }

  const dimensions: RateMatrixDimension[] = axes.data.dimensions.flatMap((dimension) => {
    const parsed = rateMatrixDimensionSchema.safeParse(dimension);
    return parsed.success ? [parsed.data] : [];
  });
  const valueType = rateMatrixValueTypeSchema.catch("Money").parse(axes.data.valueType);

  return (
    <LazyComponent fallback={<MatrixGridEditorSkeleton />}>
      <MatrixGridEditor
        rateMatrixId={matrixId}
        version={{ versionNumber, dimensions, valueType }}
      />
    </LazyComponent>
  );
}

function VersionRow({
  version,
  matrix,
  pins,
  viewing,
  onView,
  onRestore,
  canRestore,
}: {
  version: RateMatrixRecordVersion;
  matrix: RateMatrixRecord;
  pins: number;
  viewing: boolean;
  onView: () => void;
  onRestore: () => void;
  canRestore: boolean;
}) {
  const t = useT();
  const current = version.versionNumber === matrix.publishedVersionNumber;
  return (
    <li className="flex flex-col gap-1 px-3 py-2.5">
      <div className="flex items-center gap-3">
        <span className="font-mono text-sm">{t("v{0}", version.versionNumber)}</span>
        <div className="flex min-w-0 flex-1 flex-col">
          <span className="truncate text-sm">
            {current ? t("{0} (tariffs read this one)", version.name) : version.name}
          </span>
          <span className="text-muted-foreground text-xs">
            {t(
              "{0, plural, one {# cell} other {# cells}}, published {1} by {2}",
              version.cellCount,
              formatDistanceToNow(new Date(version.createdAt * 1000), { addSuffix: true }),
              version.publishedBy?.name ?? t("someone"),
            )}
          </span>
          {pins > 0 && (
            <span className="text-muted-foreground text-xs">
              {t(
                "{0, plural, one {# template version pins it} other {# template versions pin it}}",
                pins,
              )}
            </span>
          )}
        </div>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className="h-7 gap-1 text-xs"
          aria-pressed={viewing}
          onClick={onView}
        >
          <EyeIcon className="size-3.5" />
          {viewing ? t("Hide rates") : t("View rates")}
        </Button>
        {canRestore && (
          <Button
            type="button"
            variant="ghost"
            size="sm"
            className="h-7 gap-1 text-xs"
            onClick={onRestore}
          >
            <ClockRewindIcon className="size-3.5" />
            {t("Restore to draft")}
          </Button>
        )}
      </div>
      {version.changeMessage && (
        <p className="text-muted-foreground text-xs">{version.changeMessage}</p>
      )}
    </li>
  );
}

/**
 * A matrix's review state and its published versions. Editors change the draft; a reviewer
 * publishes it as the next numbered version, which formula template versions pin when they are
 * approved, so publishing never reprices a contract on its own.
 */
export function RateMatrixVersions({ rateMatrixId }: { rateMatrixId?: string }) {
  const t = useT();
  const record = useQuery({
    ...rateMatrix.record(rateMatrixId ?? ""),
    enabled: Boolean(rateMatrixId),
  });
  const canSubmit = usePermission(Resource.RateMatrix, Operation.Submit).allowed;
  const approval = useReviewApproval(Resource.RateMatrix, record.data?.submittedById);
  const canReject = usePermission(Resource.RateMatrix, Operation.Reject).allowed;
  const canArchive = usePermission(Resource.RateMatrix, Operation.Archive).allowed;
  const canUpdate = usePermission(Resource.RateMatrix, Operation.Update).allowed;

  const [submitting, setSubmitting] = useState(false);
  const [decision, setDecision] = useState<RateMatrixDecision | null>(null);
  const [archiving, setArchiving] = useState(false);
  const [restoring, setRestoring] = useState<number | null>(null);
  const [viewing, setViewing] = useState<number | null>(null);

  if (!rateMatrixId) {
    return (
      <p className="text-muted-foreground text-sm">
        {t("Save the matrix first. Its draft is published from here once a reviewer approves it.")}
      </p>
    );
  }
  if (record.isPending) return <RateMatrixVersionsSkeleton />;
  if (record.isError) {
    return (
      <div className="flex flex-col items-start gap-2">
        <p className="text-muted-foreground text-sm">
          {t("This matrix's versions could not be loaded.")}
        </p>
        <Button type="button" variant="outline" size="xs" onClick={() => void record.refetch()}>
          {t("Retry")}
        </Button>
      </div>
    );
  }

  const matrix = record.data;
  const actions = matrixActions(matrix);
  const pinsByVersion = new Map<number, number>();
  for (const usage of matrix.usages) {
    pinsByVersion.set(
      usage.matrixVersionNumber,
      (pinsByVersion.get(usage.matrixVersionNumber) ?? 0) + 1,
    );
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="bg-card flex flex-wrap items-center justify-between gap-3 rounded-md border p-3">
        <div className="flex min-w-0 flex-col gap-1">
          <StatusValue status={matrix.status} />
          <p className="text-muted-foreground text-xs">{stateSummary(matrix, t)}</p>
          {matrix.reviewComment && matrix.status === "Draft" && (
            <p className="text-xs">{t("Reviewer: {0}", matrix.reviewComment)}</p>
          )}
        </div>
        <div className="flex flex-wrap gap-2">
          {actions.submit && canSubmit && (
            <Button type="button" size="sm" className="gap-1.5" onClick={() => setSubmitting(true)}>
              <Send01Icon className="size-3.5" />
              {t("Submit for review")}
            </Button>
          )}
          {actions.decide && canReject && (
            <>
              <Button
                type="button"
                variant="outline"
                size="sm"
                className="gap-1.5"
                onClick={() => setDecision("requestChanges")}
              >
                <MessageAlertSquareIcon className="size-3.5" />
                {t("Request changes")}
              </Button>
              <Button
                type="button"
                variant="outline"
                size="sm"
                className="gap-1.5"
                onClick={() => setDecision("reject")}
              >
                <XCloseIcon className="size-3.5" />
                {t("Reject")}
              </Button>
            </>
          )}
          {actions.decide && approval.allowed && (
            <Button
              type="button"
              size="sm"
              className="gap-1.5"
              onClick={() => setDecision("approve")}
            >
              <CheckIcon className="size-3.5" />
              {t("Publish")}
            </Button>
          )}
          {actions.decide && approval.needsSecondReviewer && (
            <Tooltip>
              <TooltipTrigger
                render={
                  <span>
                    <Button type="button" size="sm" className="gap-1.5" disabled>
                      <CheckIcon className="size-3.5" />
                      {t("Publish")}
                    </Button>
                  </span>
                }
              />
              <TooltipContent>
                {t("You submitted this draft, so another reviewer has to publish it")}
              </TooltipContent>
            </Tooltip>
          )}
          {actions.archive && canArchive && (
            <Button
              type="button"
              variant="outline"
              size="sm"
              className="gap-1.5"
              onClick={() => setArchiving(true)}
            >
              <ArchiveIcon className="size-3.5" />
              {t("Archive")}
            </Button>
          )}
        </div>
      </div>

      <section className="flex flex-col gap-2" aria-label={t("Published versions")}>
        <h3 className="text-sm font-semibold">{t("Published versions")}</h3>
        {matrix.versions.length === 0 ? (
          <p className="text-muted-foreground text-xs">{t("No version has been published yet.")}</p>
        ) : (
          <ul className="divide-y rounded-md border">
            {matrix.versions.map((version) => (
              <VersionRow
                key={version.id}
                version={version}
                matrix={matrix}
                pins={pinsByVersion.get(version.versionNumber) ?? 0}
                viewing={viewing === version.versionNumber}
                onView={() =>
                  setViewing((current) =>
                    current === version.versionNumber ? null : version.versionNumber,
                  )
                }
                onRestore={() => setRestoring(version.versionNumber)}
                canRestore={canUpdate && matrix.status !== "InReview"}
              />
            ))}
          </ul>
        )}
        {viewing !== null && <VersionSheet matrixId={matrix.id} versionNumber={viewing} />}
      </section>

      <section className="flex flex-col gap-2" aria-label={t("Pinned by")}>
        <h3 className="text-sm font-semibold">{t("Pinned by")}</h3>
        {matrix.usages.length === 0 ? (
          <p className="text-muted-foreground text-xs">
            {t(
              "No formula template version pins this matrix yet. Approving a template version that reads tables.{0} pins its newest published version.",
              matrix.code,
            )}
          </p>
        ) : (
          <ul className="divide-y rounded-md border">
            {matrix.usages.map((usage) => (
              <li
                key={`${usage.templateId}:${usage.templateVersion}`}
                className="flex items-center gap-3 px-3 py-2"
              >
                <div className="flex min-w-0 flex-1 flex-col gap-0.5">
                  <Link
                    to={formulaTemplateRoutes.edit(usage.templateId)}
                    className="ui-focus-ring truncate rounded-sm text-sm font-medium hover:underline"
                  >
                    {usage.templateName}
                  </Link>
                  <span className="text-muted-foreground flex items-center gap-2 text-xs">
                    {t("v{0}", usage.templateVersion)}
                    <VersionStatus status={usage.versionStatus} />
                  </span>
                </div>
                <span className="text-muted-foreground shrink-0 font-mono text-xs">
                  {t("pins v{0}", usage.matrixVersionNumber)}
                </span>
              </li>
            ))}
          </ul>
        )}
      </section>

      <section className="flex flex-col gap-2" aria-label={t("Review history")}>
        <h3 className="text-sm font-semibold">{t("Review history")}</h3>
        <ReviewHistoryView
          reviews={matrixReviewEntries(matrix)}
          isLoading={false}
          isError={false}
          emptyText={t("This matrix has never been submitted for review.")}
        />
      </section>

      <RateMatrixSubmitDialog open={submitting} onOpenChange={setSubmitting} matrix={matrix} />
      <RateMatrixDecisionDialog
        decision={decision}
        onClose={() => setDecision(null)}
        matrix={matrix}
      />
      <RateMatrixArchiveDialog open={archiving} onOpenChange={setArchiving} matrix={matrix} />
      <RateMatrixRestoreDialog
        versionNumber={restoring}
        onClose={() => setRestoring(null)}
        matrix={matrix}
      />
    </div>
  );
}
