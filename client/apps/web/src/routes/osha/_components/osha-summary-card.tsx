import { useT } from "@trenova/shared/i18n/use-t";
import type { OshaLog } from "@/lib/graphql/worker-injury";
import {
  certifyBlocker,
  certifyBlockerMessage,
  ILLNESS_TYPES,
  isCertified,
  LOG_COLUMNS,
} from "@/lib/osha-log";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { FormBlock, FormFigure, FormMark } from "./osha-form-marks";

type OshaSummaryCardProps = {
  log: OshaLog;
  canUpdate: boolean;
  canCertify: boolean;
  certifying: boolean;
  reopening: boolean;
  onEditFigures: () => void;
  onCertify: () => void;
  onReopen: () => void;
};

/**
 * The 300A, laid out the way the form is: the boxed case and day counts with
 * their column letters, the numbered illness types, the establishment
 * information, and the certification block that an executive signs. Every
 * count is derived from the log; only the establishment figures are typed in.
 */
export function OshaSummaryCard({
  log,
  canUpdate,
  canCertify,
  certifying,
  reopening,
  onEditFigures,
  onCertify,
  onReopen,
}: OshaSummaryCardProps) {
  const t = useT();

  const { totals, summary } = log;
  const certified = isCertified(summary);
  const blocker = certifyBlocker(summary, totals);
  const holdReason = blocker ? certifyBlockerMessage(blocker, totals.openCases) : null;

  return (
    <section aria-label={t("Form 300A")} className="bg-card flex min-w-0 flex-col rounded-lg border">
      <header className="flex flex-wrap items-start justify-between gap-3 border-b px-4 py-3">
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-2">
            <h2 className="text-sm font-medium">
              {t("Summary of work-related injuries and illnesses, {0}", log.year)}
            </h2>
            {certified ? (
              <Badge variant="active">{t("Certified")}</Badge>
            ) : summary ? (
              <Badge variant="secondary">{t("Draft")}</Badge>
            ) : (
              <Badge variant="outline">{t("Not started")}</Badge>
            )}
          </div>
          <p className="text-muted-foreground mt-0.5 text-xs">
            {t("Form 300A. Post it from {0} to {1}, where employees can read it.", formatUnixDateMedium(log.postFrom), formatUnixDateMedium(log.postThrough))}
          </p>
        </div>
        {canUpdate || canCertify ? (
          <div className="flex shrink-0 flex-wrap items-center gap-2">
            {canUpdate && !certified ? (
              <Button size="sm" variant="outline" onClick={onEditFigures}>
                {summary ? "Edit figures" : "Start the summary"}
              </Button>
            ) : null}
            {canCertify && summary && !certified ? (
              <Button
                size="sm"
                isLoading={certifying}
                disabled={blocker !== null}
                aria-describedby={blocker ? "osha-certify-hold" : undefined}
                onClick={onCertify}
              >
                {t("Certify")}
              </Button>
            ) : null}
            {canCertify && certified ? (
              <Button size="sm" variant="ghost" isLoading={reopening} onClick={onReopen}>
                {t("Reopen")}
              </Button>
            ) : null}
          </div>
        ) : null}
      </header>

      <div className="grid gap-x-8 gap-y-5 px-4 py-4 md:grid-cols-[3fr_2fr]">
        <FormBlock title={t("Number of cases")}>
          <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
            {LOG_COLUMNS.map((column) => (
              <FormFigure
                key={column.column}
                mark={column.column}
                label={column.label}
                value={totals[column.key]}
                alarm={column.key === "deaths" && totals.deaths > 0}
                muted
              />
            ))}
          </div>
        </FormBlock>

        <FormBlock title={t("Number of days")}>
          <div className="grid grid-cols-2 gap-3">
            <FormFigure
              mark="K"
              label={t("Total days away from work")}
              value={totals.totalDaysAway}
              muted
            />
            <FormFigure
              mark="L"
              label={t("Total days of job transfer or restriction")}
              value={totals.totalDaysRestricted}
              muted
            />
          </div>
        </FormBlock>

        <FormBlock title={t("Injury and illness types")} className="md:col-span-2">
          <div className="grid grid-cols-3 gap-3 sm:grid-cols-6">
            {ILLNESS_TYPES.map((type) => (
              <FormFigure
                key={type.number}
                mark={`(${type.number})`}
                label={type.label}
                value={totals[type.key]}
                muted
              />
            ))}
          </div>
        </FormBlock>

        <FormBlock title={t("Establishment information")}>
          <dl className="divide-border/60 divide-y text-xs">
            <FormRow label={t("NAICS code")} value={summary?.naicsCode?.trim() || null} />
            <FormRow
              label={t("Annual average number of employees")}
              value={
                summary && summary.averageEmployees > 0
                  ? summary.averageEmployees.toLocaleString("en-US")
                  : null
              }
            />
            <FormRow
              label={t("Total hours worked by all employees")}
              value={
                summary && summary.totalHoursWorked > 0
                  ? summary.totalHoursWorked.toLocaleString("en-US")
                  : null
              }
            />
          </dl>
        </FormBlock>

        <FormBlock title={t("Certification")}>
          {certified ? (
            <div className="flex flex-col gap-1 text-xs">
              <p>
                <span className="font-medium">
                  {summary?.executiveName?.trim() || "An executive"}
                </span>
                {summary?.executiveTitle?.trim() ? (
                  <span className="text-muted-foreground">, {summary.executiveTitle.trim()}</span>
                ) : null}
              </p>
              <p className="text-muted-foreground">
                {t("Certified {0} {1}", formatUnixDateMedium(summary?.certifiedAt), summary?.executivePhone?.trim() ? ` · ${summary.executivePhone.trim()}` : "")}
              </p>
              <p className="text-muted-foreground">
                {summary?.submittedAt
                  ? `Submitted electronically ${formatUnixDateMedium(summary.submittedAt)}${
                      summary.submissionReference?.trim()
                        ? `, reference ${summary.submissionReference.trim()}`
                        : ""
                    }`
                  : "Not yet submitted electronically"}
              </p>
            </div>
          ) : (
            <div className="flex flex-col gap-1 text-xs">
              <p className="text-muted-foreground">
                {t("A company executive certifies that they have examined the log and believe the summary is correct and complete.")}
              </p>
              {summary?.executiveName?.trim() ? (
                <p>
                  <span className="font-medium">{summary.executiveName.trim()}</span>
                  {summary.executiveTitle?.trim() ? (
                    <span className="text-muted-foreground">, {summary.executiveTitle.trim()}</span>
                  ) : null}
                  <span className="text-muted-foreground"> {t("will sign")}</span>
                </p>
              ) : null}
              {holdReason ? (
                <p id="osha-certify-hold" className="flex items-center gap-1.5">
                  <FormMark className="text-warning-foreground border-warning/40 bg-warning/15">
                    {t("Held")}
                  </FormMark>
                  <span>{holdReason}</span>
                </p>
              ) : (
                <p className="text-foreground">{t("Ready to certify.")}</p>
              )}
            </div>
          )}
        </FormBlock>
      </div>
    </section>
  );
}

function FormRow({ label, value }: { label: string; value: string | null }) {
  const t = useT();

  return (
    <div className="flex items-baseline justify-between gap-3 py-1.5 first:pt-0 last:pb-0">
      <dt className="text-muted-foreground">{label}</dt>
      <dd className="font-medium tabular-nums">
        {value ?? <span className="text-muted-foreground/70 font-normal">{t("Not recorded")}</span>}
      </dd>
    </div>
  );
}
