import { DocumentPanePreview } from "@/components/documents/document-pane-preview";
import {
  preloadOn,
  WorkPane,
  WorkPaneBody,
  WorkPaneCancel,
  WorkPaneFooter,
} from "@/components/work-pane";
import type { WorkerTrainingRecordRow } from "@/lib/graphql/worker-training";
import { TrainingHealthBadge } from "@trenova/shared/components/training-health-badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  DescriptionEmpty,
  DescriptionItem,
  DescriptionList,
} from "@trenova/shared/components/ui/description-list";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { describeTrainingTiming } from "@trenova/shared/lib/training";
import {
  TRAINING_CATEGORY_LABELS,
  TRAINING_DELIVERY_LABELS,
  type TrainingCategory,
  type TrainingDelivery,
} from "@trenova/shared/types/worker-training";
import type { WorkerTrainingHealth } from "@trenova/shared/types/worker-training-health";
import { useState } from "react";
import { RecordTrainingResultPane } from "../development-panes";
import { formatTrainingScore } from "./training-score";

export type CoursePaneProps = {
  /** The training record to read; the pane is open while it is set. */
  record: WorkerTrainingRecordRow | null;
  required: boolean;
  /** Null when the reader may not record results or the record is waived. */
  onRetake: ((record: WorkerTrainingRecordRow) => void) | null;
  onClose: () => void;
};

/** A course on the worker's file, read in the pane: what it is and what the record says. */
export function CoursePane({ record: requested, required, onRetake, onClose }: CoursePaneProps) {
  const t = useT();
  const [held, setHeld] = useState(requested);
  if (requested !== null && requested !== held) setHeld(requested);
  const record = requested ?? held;
  const course = record?.course;
  const score = record?.score ? Number(record.score) : null;
  const passing = course?.passingScore ? Number(course.passingScore) : null;

  return (
    <WorkPane
      open={requested !== null}
      onOpenChange={(next) => {
        if (!next) onClose();
      }}
      title={course?.name ?? t("Course")}
      contentKey={record?.id}
    >
      <WorkPaneBody>
        {record ? (
          <>
            <div className="flex items-center gap-2">
              <TrainingHealthBadge health={record.health as WorkerTrainingHealth} />
              <span className="text-xs text-muted-foreground">
                {required ? t("Required") : t("Other")}
              </span>
            </div>
            <DescriptionList layout="ruled" columns={2}>
              <DescriptionItem label={t("Category")}>
                {course ? (
                  TRAINING_CATEGORY_LABELS[course.category as TrainingCategory]
                ) : (
                  <DescriptionEmpty />
                )}
              </DescriptionItem>
              <DescriptionItem label={t("Delivery")}>
                {course ? (
                  <>
                    {TRAINING_DELIVERY_LABELS[course.delivery as TrainingDelivery]}
                    {course.durationMinutes > 0
                      ? ` · ${t("{0} min", course.durationMinutes)}`
                      : null}
                  </>
                ) : (
                  <DescriptionEmpty />
                )}
              </DescriptionItem>
              <DescriptionItem label={t("Assigned")} numeric>
                {formatUnixDateMedium(record.assignedAt)}
                {record.assignedBy?.name ? (
                  <span className="block text-xs text-muted-foreground">
                    {t("by {0}", record.assignedBy.name)}
                  </span>
                ) : null}
              </DescriptionItem>
              <DescriptionItem label={t("Completed")} numeric>
                {record.status === "Waived" ? (
                  <>
                    {t("Waived")}
                    {record.waivedReason ? (
                      <span className="block text-xs text-muted-foreground">
                        {record.waivedReason}
                      </span>
                    ) : null}
                  </>
                ) : record.completedAt ? (
                  formatUnixDateMedium(record.completedAt)
                ) : (
                  <DescriptionEmpty />
                )}
              </DescriptionItem>
              <DescriptionItem label={t("Score")} numeric>
                {score !== null ? (
                  <>
                    {t("{0}%", formatTrainingScore(score))}
                    {passing !== null ? (
                      <span className="block text-xs text-muted-foreground">
                        {t("passes at {0}%", formatTrainingScore(passing))}
                      </span>
                    ) : null}
                  </>
                ) : record.completedAt ? (
                  t("No score, passed")
                ) : (
                  <DescriptionEmpty />
                )}
              </DescriptionItem>
              <DescriptionItem label={t("Valid until")} numeric>
                {record.expiresAt ? (
                  <>
                    {formatUnixDateMedium(record.expiresAt)}
                    <span className="block text-xs text-muted-foreground">
                      {describeTrainingTiming({
                        health: record.health as WorkerTrainingHealth,
                        daysUntilExpiry: record.daysUntilExpiry,
                        daysUntilDue: record.daysUntilDue,
                      })}
                    </span>
                  </>
                ) : record.completedAt ? (
                  t("No expiry")
                ) : (
                  <DescriptionEmpty />
                )}
              </DescriptionItem>
              <DescriptionItem label={t("Certificate")} span="full">
                {record.document ? (
                  <DocumentPanePreview document={record.document} className="mt-1" />
                ) : (
                  <span className="text-foreground-subtle">{t("None attached")}</span>
                )}
              </DescriptionItem>
              {record.notes ? (
                <DescriptionItem label={t("Notes")} span="full">
                  <span className="whitespace-pre-wrap">{record.notes}</span>
                </DescriptionItem>
              ) : null}
            </DescriptionList>
          </>
        ) : null}
      </WorkPaneBody>
      <WorkPaneFooter>
        <WorkPaneCancel>{t("Close")}</WorkPaneCancel>
        {record && onRetake && record.status !== "Waived" ? (
          <Button
            type="button"
            size="sm"
            {...preloadOn(RecordTrainingResultPane)}
            onClick={() => onRetake(record)}
          >
            {t("Record a retake")}
          </Button>
        ) : null}
      </WorkPaneFooter>
    </WorkPane>
  );
}
