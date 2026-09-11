import { useT } from "@trenova/shared/i18n/use-t";
import type { OshaLogCase } from "@/lib/graphql/worker-injury";
import { caseLabel, illnessTypeNumber, logColumn } from "@/lib/osha-log";
import { workerRecordHref } from "@/lib/route-utils";
import { Avatar, AvatarFallback, AvatarImage } from "@trenova/shared/components/ui/avatar";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@trenova/shared/components/ui/sheet";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import {
  caseClassificationLabel,
  claimStatusLabel,
  claimStatusTone,
  classificationTone,
  illnessTypeLabel,
  injuryTreatmentLabel,
  MAX_COUNTED_DAYS,
} from "@trenova/shared/lib/injury";
import { initials } from "@trenova/shared/lib/utils";
import { LockIcon, PencilIcon, Trash2Icon } from "lucide-react";
import type { ReactNode } from "react";
import { Link } from "react-router";
import { FormMark } from "./osha-form-marks";

type OshaCaseSheetProps = {
  entry: OshaLogCase | null;
  onOpenChange: (open: boolean) => void;
  canUpdate: boolean;
  canDelete: boolean;
  onEdit: (entry: OshaLogCase) => void;
  onDelete: (entry: OshaLogCase) => void;
};

/**
 * One case in full: who, what happened, what it cost, and where the
 * workers' compensation claim stands. The log table shows the name as posted;
 * this is where the office sees who is behind a privacy case.
 */
export function OshaCaseSheet({
  entry,
  onOpenChange,
  canUpdate,
  canDelete,
  onEdit,
  onDelete,
}: OshaCaseSheetProps) {
  return (
    <Sheet open={entry !== null} onOpenChange={onOpenChange}>
      <SheetContent className="sm:max-w-md">
        {entry ? (
          <CaseDetail
            entry={entry}
            canUpdate={canUpdate}
            canDelete={canDelete}
            onEdit={onEdit}
            onDelete={onDelete}
          />
        ) : null}
      </SheetContent>
    </Sheet>
  );
}

function CaseDetail({
  entry,
  canUpdate,
  canDelete,
  onEdit,
  onDelete,
}: Omit<OshaCaseSheetProps, "entry" | "onOpenChange"> & { entry: OshaLogCase }) {
  const t = useT();

  const column = logColumn(entry.classification);
  const typeNumber = illnessTypeNumber(entry.illnessType);
  const workerName = entry.worker
    ? `${entry.worker.firstName} ${entry.worker.lastName}`
    : entry.logName;
  const capped = entry.daysAway >= MAX_COUNTED_DAYS || entry.daysRestricted >= MAX_COUNTED_DAYS;

  return (
    <>
      <SheetHeader className="pr-10">
        <div className="flex flex-wrap items-center gap-2">
          <SheetTitle>{t("Case {0}", caseLabel(entry))}</SheetTitle>
          <Badge variant={classificationTone(entry.classification)}>
            {caseClassificationLabel(entry.classification)}
          </Badge>
          {entry.status === "Open" ? <Badge variant="warning">{t("Open")}</Badge> : null}
          {entry.recordable ? null : <Badge variant="secondary">{t("Off the log")}</Badge>}
        </div>
        <SheetDescription>
          {t("Occurred {0}{1}", formatUnixDateMedium(entry.occurredAt), entry.reportedAt ? `, reported ${formatUnixDateMedium(entry.reportedAt)}` : "")}
        </SheetDescription>
      </SheetHeader>

      <div className="flex min-h-0 flex-1 flex-col gap-5 overflow-y-auto px-4 pb-2">
        <Link
          to={workerRecordHref(entry.workerId, "safety")}
          className="hover:bg-accent -mx-2 flex items-center gap-2.5 rounded-md px-2 py-1.5 transition-colors"
        >
          <Avatar size="sm">
            {entry.worker?.profilePicUrl ? (
              <AvatarImage src={entry.worker.profilePicUrl} alt="" />
            ) : null}
            <AvatarFallback className="text-2xs font-medium">
              {entry.worker ? initials(entry.worker.firstName, entry.worker.lastName) : "?"}
            </AvatarFallback>
          </Avatar>
          <span className="flex min-w-0 flex-col leading-tight">
            <span className="truncate text-xs font-medium">{workerName}</span>
            <span className="text-muted-foreground text-2xs">
              {t("Open the worker's safety tab")}
            </span>
          </span>
        </Link>

        {entry.privacyCase ? (
          <p className="bg-muted/60 text-muted-foreground flex items-start gap-2 rounded-md px-3 py-2 text-xs">
            <LockIcon className="mt-0.5 size-3.5 shrink-0" />
            <span>
              {t("Privacy case. The posted log reads “{0}”; the name stays on the confidential list.", entry.logName)}
            </span>
          </p>
        ) : null}

        <Section title={t("What happened")}>
          <p className="text-xs leading-relaxed">{entry.description}</p>
          <dl className="mt-2 text-xs">
            <Row label={t("Where")}>{entry.location?.trim() || null}</Row>
            <Row label={t("Body part")}>{entry.bodyPart?.trim() || null}</Row>
            <Row label={t("Object or substance")}>{entry.harmfulAgent?.trim() || null}</Row>
          </dl>
        </Section>

        <Section title={t("Outcome")}>
          <dl className="text-xs">
            <Row label={t("Log column")}>
              {column ? (
                <span className="inline-flex items-center gap-1.5">
                  <FormMark className="text-foreground">{column}</FormMark>
                  {caseClassificationLabel(entry.classification)}
                </span>
              ) : (
                caseClassificationLabel(entry.classification)
              )}
            </Row>
            <Row label={t("Type")}>
              <span className="inline-flex items-center gap-1.5">
                {typeNumber !== null ? <FormMark>({typeNumber})</FormMark> : null}
                {illnessTypeLabel(entry.illnessType)}
              </span>
            </Row>
            <Row label={t("Treatment")}>{injuryTreatmentLabel(entry.treatment)}</Row>
            <Row label={t("Days away from work")}>
              <span className="inline-flex items-center gap-1.5">
                <FormMark>K</FormMark>
                {entry.daysAway}
              </span>
            </Row>
            <Row label={t("Days restricted or transferred")}>
              <span className="inline-flex items-center gap-1.5">
                <FormMark>L</FormMark>
                {entry.daysRestricted}
              </span>
            </Row>
            <Row label={t("Returned to work")}>
              {entry.returnedToWorkAt ? formatUnixDateMedium(entry.returnedToWorkAt) : null}
            </Row>
            <Row label={t("Status")}>
              {entry.status === "Open" ? "Open, days may still accrue" : "Closed"}
            </Row>
          </dl>
          {capped ? (
            <p className="text-muted-foreground mt-2 text-2xs">
              {t("Counting stopped at {0} days, as the log requires.", MAX_COUNTED_DAYS)}
            </p>
          ) : null}
        </Section>

        <Section title={t("Workers' compensation claim")}>
          {entry.claimStatus === "NotFiled" ? (
            <p className="text-muted-foreground text-xs">{t("No claim has been filed.")}</p>
          ) : (
            <dl className="text-xs">
              <Row label={t("Status")}>
                <Badge variant={claimStatusTone(entry.claimStatus)}>
                  {claimStatusLabel(entry.claimStatus)}
                </Badge>
              </Row>
              <Row label={t("Claim number")}>{entry.claimNumber?.trim() || null}</Row>
              <Row label={t("Carrier")}>{entry.claimCarrier?.trim() || null}</Row>
              <Row label={t("Filed")}>
                {entry.claimFiledAt ? formatUnixDateMedium(entry.claimFiledAt) : null}
              </Row>
              <Row label={t("Closed")}>
                {entry.claimClosedAt ? formatUnixDateMedium(entry.claimClosedAt) : null}
              </Row>
            </dl>
          )}
        </Section>

        {entry.notes?.trim() ? (
          <Section title={t("Notes")}>
            <p className="text-muted-foreground text-xs leading-relaxed whitespace-pre-line">
              {entry.notes.trim()}
            </p>
          </Section>
        ) : null}
      </div>

      {canUpdate || canDelete ? (
        <SheetFooter className="flex-row justify-end border-t">
          {canDelete ? (
            <Button
              variant="ghost"
              size="sm"
              className="text-destructive hover:text-destructive mr-auto"
              onClick={() => onDelete(entry)}
            >
              <Trash2Icon className="size-3.5" />
              {t("Delete case")}
            </Button>
          ) : null}
          {canUpdate ? (
            <Button variant="outline" size="sm" onClick={() => onEdit(entry)}>
              <PencilIcon className="size-3.5" />
              {t("Edit case")}
            </Button>
          ) : null}
        </SheetFooter>
      ) : null}
    </>
  );
}

function Section({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section aria-label={title} className="flex flex-col gap-1.5">
      <h3 className="text-muted-foreground text-xs font-medium">{title}</h3>
      {children}
    </section>
  );
}

function Row({ label, children }: { label: string; children: ReactNode }) {
  const t = useT();

  return (
    <div className="border-border/60 flex items-center justify-between gap-3 border-b py-1.5 last:border-0">
      <dt className="text-muted-foreground shrink-0">{label}</dt>
      <dd className="min-w-0 truncate text-right font-medium tabular-nums">
        {children ?? <span className="text-muted-foreground/70 font-normal">{t("Not given")}</span>}
      </dd>
    </div>
  );
}
