import { KpiStrip, KpiStripItem } from "@/components/kpi/kpi-strip";
import { SectionPanel, SectionPanelQuiet } from "@/components/section-panel";
import { FACTORING_ACCOUNT_ROLE_LABELS } from "@/lib/factoring";
import type { FactoringAgreement } from "@/lib/graphql/factoring";
import { factoring as factoringQueries } from "@/lib/queries/factoring";
import { useQuery } from "@tanstack/react-query";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDate } from "@trenova/shared/lib/date";
import { formatMinor } from "@trenova/shared/lib/money";
import { cn } from "@trenova/shared/lib/utils";
import { agreementDifferences } from "./factoring-agreement-changes";

type Account = { id: string; accountCode: string; name: string } | null | undefined;

/**
 * What the schedules sent under the agreement carried: how many, how many invoices, the face value,
 * the advance and fees they were expected to bring, and what share of them has been funded.
 */
export function FactoringAgreementActivity({ agreement }: { agreement: FactoringAgreement }) {
  const t = useT();
  const { data, isLoading, error, refetch } = useQuery(
    factoringQueries.agreementActivity(agreement.id),
  );

  return (
    <SectionPanel
      title={t("Under this agreement")}
      hint={
        data?.firstSentAt && data.lastSentAt
          ? t(
              "Schedules sent {0} to {1}",
              formatUnixDate(data.firstSentAt),
              formatUnixDate(data.lastSentAt),
            )
          : undefined
      }
    >
      {isLoading ? (
        <div aria-busy aria-label={t("Loading the agreement's schedules")} className="p-3">
          <span aria-hidden className="block h-16 ui-shimmer rounded-md" />
        </div>
      ) : error || !data ? (
        <div className="p-3">
          <Alert size="sm" variant="destructive">
            <AlertDescription className="flex items-center justify-between gap-3">
              <span>{t("What this agreement carried could not be loaded.")}</span>
              <Button type="button" size="sm" variant="outline" onClick={() => void refetch()}>
                {t("Retry")}
              </Button>
            </AlertDescription>
          </Alert>
        </div>
      ) : data.schedules === 0 ? (
        <SectionPanelQuiet>
          {t("No schedule has been sent under this agreement yet.")}
        </SectionPanelQuiet>
      ) : (
        <div className="p-3">
          <KpiStrip minItemWidth="8.5rem" aria-label={t("Under this agreement")}>
            <KpiStripItem
              label={t("Schedules sent")}
              value={data.schedules.toLocaleString()}
              sub={t("{0} funded", data.fundedSchedules.toLocaleString())}
            />
            <KpiStripItem
              label={t("Invoices")}
              value={data.invoices.toLocaleString()}
              sub={t("On those schedules")}
            />
            <KpiStripItem
              label={t("Face value")}
              value={formatMinor(data.faceMinor)}
              sub={t("{0} advance expected", formatMinor(data.expectedAdvanceMinor))}
            />
            <KpiStripItem
              label={t("Fees expected")}
              tone="warning"
              value={formatMinor(data.expectedFeeMinor)}
              sub={
                data.faceMinor > 0
                  ? t(
                      "{0}% of face value",
                      ((data.expectedFeeMinor / data.faceMinor) * 100).toLocaleString(undefined, {
                        maximumFractionDigits: 2,
                      }),
                    )
                  : undefined
              }
            />
          </KpiStrip>
        </div>
      )}
    </SectionPanel>
  );
}

function AccountRow({
  role,
  account,
  note,
  muted,
}: {
  role: string;
  account: Account;
  note?: string;
  muted?: boolean;
}) {
  const t = useT();
  return (
    <div className={cn("flex min-w-0 flex-col gap-0.5", muted && "opacity-60")}>
      <dt className="text-xs font-medium text-foreground-subtle">{role}</dt>
      <dd className="m-0 flex min-w-0 flex-col gap-0.5 text-sm">
        {account ? (
          <span className="truncate">
            <span className="font-mono tabular-nums">{account.accountCode}</span>
            <span className="text-muted-foreground"> · </span>
            {account.name}
          </span>
        ) : (
          <span className="text-muted-foreground">{t("Organization default")}</span>
        )}
        {note ? <span className="text-xs text-muted-foreground">{note}</span> : null}
      </dd>
    </div>
  );
}

/** The ledger accounts the agreement's factoring is booked to; an empty one uses the organization's. */
export function FactoringAgreementBookedTo({ agreement }: { agreement: FactoringAgreement }) {
  const t = useT();
  const sale = agreement.accountingTreatment === "Sale";

  return (
    <SectionPanel
      title={t("Booked to")}
      tip={t("An empty account uses your organization's factoring default.")}
    >
      <dl className="m-0 grid gap-x-6 gap-y-4 p-3 sm:grid-cols-2 xl:grid-cols-4">
        <AccountRow
          role={FACTORING_ACCOUNT_ROLE_LABELS.DueFromFactor}
          account={agreement.dueFromFactorAccount}
          note={sale ? t("The reserve the factor holds") : t("Not used for secured borrowing")}
          muted={!sale}
        />
        <AccountRow
          role={FACTORING_ACCOUNT_ROLE_LABELS.Liability}
          account={agreement.liabilityAccount}
          note={sale ? t("Not used for a sale") : t("What is owed the factor")}
          muted={sale}
        />
        <AccountRow
          role={FACTORING_ACCOUNT_ROLE_LABELS.FeeExpense}
          account={agreement.feeExpenseAccount}
          note={t("Where the factor's fee is expensed")}
        />
        <AccountRow
          role={FACTORING_ACCOUNT_ROLE_LABELS.Cash}
          account={agreement.cashAccount}
          note={t("Where the factor's payments land")}
        />
      </dl>
    </SectionPanel>
  );
}

type Moment = { key: string; label: string; at: number | null | undefined; future?: boolean };

/**
 * The agreement's life so far, in order, and what it changed from the version before it, so a
 * reviewer sees when terms moved and by how much.
 */
export function FactoringAgreementHistory({
  agreement,
  previous,
  previousNumber,
  nowUnix,
  className,
}: {
  agreement: FactoringAgreement;
  previous: FactoringAgreement | null;
  previousNumber: number | null;
  nowUnix: number;
  className?: string;
}) {
  const t = useT();
  const moments: Moment[] = [
    { key: "drafted", label: t("Drafted"), at: agreement.createdAt },
    { key: "activated", label: t("Activated"), at: agreement.activatedAt },
    {
      key: "starts",
      label: agreement.effectiveFrom > nowUnix ? t("Takes effect") : t("In force from"),
      at: agreement.effectiveFrom,
      future: agreement.effectiveFrom > nowUnix,
    },
    agreement.endedAt
      ? { key: "ended", label: t("Ended"), at: agreement.endedAt }
      : {
          key: "ends",
          label: t("Ends"),
          at: agreement.effectiveTo,
          future: true,
        },
  ];
  const changes = previous ? agreementDifferences(t, agreement, previous) : [];

  return (
    <SectionPanel title={t("History")} className={className}>
      <div className="grid flex-1 gap-4 p-3 md:grid-cols-2">
        <ol className="m-0 flex list-none flex-col p-0">
          {moments.map((moment, index) => (
            <li key={moment.key} className="grid grid-cols-[0.75rem_minmax(0,1fr)] gap-3">
              <span aria-hidden className="flex flex-col items-center">
                <span
                  className={cn(
                    "mt-1.5 size-2 shrink-0 rounded-full border-[1.5px]",
                    moment.at && !moment.future
                      ? "border-foreground bg-foreground"
                      : "border-border-strong bg-card",
                  )}
                />
                {index < moments.length - 1 ? <span className="w-px flex-1 bg-border" /> : null}
              </span>
              <span className="flex flex-col pb-4">
                <span className="text-sm font-medium">{moment.label}</span>
                <span className="text-xs text-muted-foreground">
                  {moment.at
                    ? formatUnixDate(moment.at)
                    : moment.key === "ends"
                      ? t("No end date; runs until someone ends it")
                      : t("Not yet")}
                </span>
              </span>
            </li>
          ))}
        </ol>
        <div className="flex min-w-0 flex-col gap-2">
          <span className="text-xs font-medium text-foreground-subtle">
            {previous && previousNumber !== null
              ? t("Changes from v{0}", previousNumber)
              : t("Changes")}
          </span>
          {!previous ? (
            <p className="m-0 text-sm text-muted-foreground">
              {t("The first agreement with this factor; nothing to compare it with.")}
            </p>
          ) : changes.length === 0 ? (
            <p className="m-0 text-sm text-muted-foreground">
              {t("The same terms as the version before it.")}
            </p>
          ) : (
            <ul className="m-0 flex list-none flex-wrap gap-1.5 p-0">
              {changes.map((change) => (
                <li
                  key={change}
                  className="rounded-sm bg-info-subtle px-1.5 py-0.5 font-mono text-xs text-info-subtle-foreground"
                >
                  {change}
                </li>
              ))}
            </ul>
          )}
          {agreement.notes ? (
            <div className="mt-2 flex flex-col gap-1">
              <span className="text-xs font-medium text-foreground-subtle">{t("Notes")}</span>
              <p className="m-0 text-sm whitespace-pre-line">{agreement.notes}</p>
            </div>
          ) : null}
        </div>
      </div>
    </SectionPanel>
  );
}
