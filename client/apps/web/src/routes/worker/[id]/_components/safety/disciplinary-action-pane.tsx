import { WorkPane, WorkPaneBody, WorkPaneFooter } from "@/components/work-pane";
import type { WorkerDisciplinaryActionRow } from "@/lib/graphql/worker-safety";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { DescriptionItem, DescriptionList } from "@trenova/shared/components/ui/description-list";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import {
  DISCIPLINARY_LEVEL_LABELS,
  DISCIPLINARY_STATUS_LABELS,
  type DisciplinaryLevel,
  type DisciplinaryStatus,
} from "@trenova/shared/types/worker-safety";

export type DisciplinaryActionPaneProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** Null when the link names an action no longer on the record. */
  action: WorkerDisciplinaryActionRow | null;
  canRescind: boolean;
  onOpenEvent: (safetyEventId: string) => void;
  onRescind: (action: WorkerDisciplinaryActionRow) => void;
};

/**
 * One disciplinary action read beside the stair: when and by whom it was issued, until when it
 * counts, the reason the driver was given, the office's details, and the driver's answer.
 */
export function DisciplinaryActionPane({
  open,
  onOpenChange,
  action,
  canRescind,
  onOpenEvent,
  onRescind,
}: DisciplinaryActionPaneProps) {
  const t = useT();
  if (!action) {
    return (
      <WorkPane open={open} onOpenChange={onOpenChange} title={t("Disciplinary action")}>
        <WorkPaneBody className="text-sm">
          <p className="text-sm text-muted-foreground">
            {t("This action is no longer on the record.")}
          </p>
        </WorkPaneBody>
      </WorkPane>
    );
  }
  const level = action.level as DisciplinaryLevel;
  const status = action.status as DisciplinaryStatus;
  const eventId = action.safetyEventId;

  return (
    <WorkPane
      open={open}
      onOpenChange={onOpenChange}
      title={t(DISCIPLINARY_LEVEL_LABELS[level] ?? level)}
      contentKey={action.id}
    >
      <WorkPaneBody className="text-sm">
        <Badge variant="neutral" dot={status === "Active"} className="self-start">
          {t(DISCIPLINARY_STATUS_LABELS[status] ?? status)}
        </Badge>
        <DescriptionList layout="ruled" columns={2}>
          <DescriptionItem label={t("Issued")} numeric>
            {formatUnixDateMedium(action.issuedAt)}
          </DescriptionItem>
          <DescriptionItem label={t("Issued by")}>
            {action.issuedBy?.name ?? t("Not recorded")}
          </DescriptionItem>
          <DescriptionItem label={t("Rolls off")} numeric>
            {action.expiresAt ? formatUnixDateMedium(action.expiresAt) : t("Never")}
          </DescriptionItem>
          <DescriptionItem label={t("Acknowledged")} numeric>
            {action.acknowledgedAt ? formatUnixDateMedium(action.acknowledgedAt) : t("Not yet")}
          </DescriptionItem>
          {action.suspensionDays ? (
            <DescriptionItem label={t("Suspension")} numeric span={2}>
              {t("{0, plural, one {# day} other {# days}}", action.suspensionDays)}
            </DescriptionItem>
          ) : null}
        </DescriptionList>
        <section className="flex flex-col gap-1.5">
          <h4 className="text-xs font-medium text-foreground-subtle">{t("Reason")}</h4>
          <p className="text-pretty">{action.reason}</p>
        </section>
        {action.details ? (
          <section className="flex flex-col gap-1.5">
            <h4 className="text-xs font-medium text-foreground-subtle">{t("Details")}</h4>
            <p className="text-pretty">{action.details}</p>
          </section>
        ) : null}
        {action.workerComment ? (
          <section className="flex flex-col gap-1.5">
            <h4 className="text-xs font-medium text-foreground-subtle">
              {t("The driver's answer")}
            </h4>
            <blockquote className="border-l-2 border-border-strong pl-2.5 text-pretty">
              {action.workerComment}
            </blockquote>
          </section>
        ) : null}
        {status === "Rescinded" ? (
          <section className="flex flex-col gap-1.5">
            <h4 className="text-xs font-medium text-foreground-subtle">{t("Rescinded")}</h4>
            <p className="text-pretty">
              {action.rescindedAt
                ? t(
                    "{0}: {1}",
                    formatUnixDateMedium(action.rescindedAt),
                    action.rescindReason ?? "",
                  )
                : (action.rescindReason ?? "")}
            </p>
          </section>
        ) : null}
      </WorkPaneBody>
      {eventId || (canRescind && status === "Active") ? (
        <WorkPaneFooter
          leading={
            canRescind && status === "Active" ? (
              <Button
                size="sm"
                variant="outline"
                className="text-danger-foreground"
                onClick={() => onRescind(action)}
              >
                {t("Rescind")}
              </Button>
            ) : null
          }
        >
          {eventId ? (
            <Button size="sm" variant="outline" onClick={() => onOpenEvent(eventId)}>
              {t("Open the event")}
            </Button>
          ) : null}
        </WorkPaneFooter>
      ) : null}
    </WorkPane>
  );
}
