import {
  FactoringSubmissionItemStatusBadge,
  FactoringSubmissionStatusBadge,
} from "@/components/factoring/factoring-badges";
import { KpiStrip, KpiStripItem } from "@/components/kpi/kpi-strip";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { SectionPanel } from "@/components/section-panel";
import { recordPath } from "@/config/record-links";
import { useBreadcrumbLabel } from "@/hooks/use-breadcrumb-label";
import { usePermission } from "@/hooks/use-permission";
import {
  FACTORING_PATH,
  FACTORING_TRANSPORT_METHOD_LABELS,
  factoringSubmissionAwaitsFunding,
} from "@/lib/factoring";
import type { FactoringSubmissionDetail, FactoringSubmissionItem } from "@/lib/graphql/factoring";
import { factoring as factoringQueries } from "@/lib/queries/factoring";
import { documentContentUrl } from "@/services/document";
import { useQuery } from "@tanstack/react-query";
import { ArrowLeftIcon, BankNote01Icon, Download01Icon } from "@trenova/shared/components/icons";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import {
  DescriptionEmpty,
  DescriptionItem,
  DescriptionList,
} from "@trenova/shared/components/ui/description-list";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@trenova/shared/components/ui/table";
import { useT } from "@trenova/shared/i18n/use-t";
import { describeError } from "@trenova/shared/lib/error-presentation";
import { formatUnixDateMedium, formatUnixDateTimeShort } from "@trenova/shared/lib/date";
import { formatCurrency } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useState } from "react";
import { Link, useParams } from "react-router";
import { undecidedItems } from "./_components/funding-schema";
import { RecordFundingDialog } from "./_components/record-funding-dialog";
import {
  ReserveMovementDialog,
  type ReserveMovementTarget,
} from "./_components/reserve-movement-dialog";
import {
  CancelFactoringSubmissionDialog,
  RetryFactoringSubmissionButton,
} from "./_components/submission-actions";
import { SubmissionItemActions } from "./_components/submission-item-actions";

/**
 * One schedule of accounts sent to a factor: what it carried, the funding
 * expected for it, where it went and how delivery went. A failed schedule is
 * retried or cancelled from here, which releases its invoices for another; a
 * delivered one has the factor's funding recorded here, and each funded
 * invoice's reserve released, charged back or its chargeback reversed.
 */
export function FactoringSubmissionPage() {
  const t = useT();
  const { id = "" } = useParams<{ id: string }>();
  const { data, isLoading, error, refetch } = useQuery({
    ...factoringQueries.submission(id),
    enabled: id !== "",
  });
  useBreadcrumbLabel(data?.number);

  const backButton = (
    <Button variant="outline" size="sm" render={<Link to={`${FACTORING_PATH}?view=schedules`} />}>
      <ArrowLeftIcon className="size-3.5" />
      {t("Schedules")}
    </Button>
  );

  if (isLoading) {
    return (
      <PageLayout
        pageHeaderProps={{ title: t("Factoring schedule"), description: t("Loading schedule...") }}
      >
        <Skeleton className="h-16 w-full" />
        <Skeleton className="h-40 w-full" />
        <Skeleton className="h-64 w-full" />
      </PageLayout>
    );
  }

  if (error || !data) {
    const described = error ? describeError(error) : null;
    return (
      <PageLayout
        pageHeaderProps={{
          title: t("Factoring schedule"),
          description: t("This schedule could not be loaded."),
          actions: backButton,
        }}
      >
        <Alert size="sm" variant="destructive">
          <AlertDescription className="flex items-center justify-between gap-3">
            <span>
              {described?.kind === "not-found"
                ? t("No schedule has this id, or it belongs to another organization.")
                : (described?.detail ?? t("The schedule could not be loaded."))}
            </span>
            {described?.retryable ? (
              <Button type="button" size="sm" variant="outline" onClick={() => void refetch()}>
                {t("Retry")}
              </Button>
            ) : null}
          </AlertDescription>
        </Alert>
      </PageLayout>
    );
  }

  return <SubmissionDetail submission={data} backButton={backButton} />;
}

function SubmissionDetail({
  submission,
  backButton,
}: {
  submission: FactoringSubmissionDetail;
  backButton: React.ReactNode;
}) {
  const t = useT();
  const { allowed: canRetry } = usePermission(Resource.FactoringSubmission, Operation.Update);
  const { allowed: canCancel } = usePermission(Resource.FactoringSubmission, Operation.Cancel);
  const { allowed: canBook } = usePermission(Resource.FactoringReserve, Operation.Create);
  const [cancelling, setCancelling] = useState(false);
  const [recordingFunding, setRecordingFunding] = useState(false);
  const [movement, setMovement] = useState<ReserveMovementTarget | null>(null);
  const { data: agreements } = useQuery({
    ...factoringQueries.agreements(submission.factoringCompanyId),
    enabled: canBook,
  });
  const recourse =
    agreements?.find((agreement) => agreement.id === submission.agreementId)?.recourse ===
    "Recourse";
  const money = (minor: number) => formatCurrency(minor / 100, submission.currencyCode);
  const factorName = submission.factoringCompany?.name ?? submission.factoringCompanyId;
  const failed = submission.status === "Failed";
  const canRecordFunding =
    canBook &&
    factoringSubmissionAwaitsFunding(submission.status) &&
    undecidedItems(submission).length > 0;

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Schedule {0}", submission.number),
        description: t(
          "{0, plural, one {# invoice} other {# invoices}} to {1}",
          submission.invoiceCount,
          factorName,
        ),
        actions: (
          <div className="flex items-center gap-2">
            {backButton}
            {failed && canCancel ? (
              <Button type="button" variant="outline" size="sm" onClick={() => setCancelling(true)}>
                {t("Cancel schedule")}
              </Button>
            ) : null}
            {failed && canRetry ? <RetryFactoringSubmissionButton submission={submission} /> : null}
            {canRecordFunding ? (
              <Button type="button" size="sm" onClick={() => setRecordingFunding(true)}>
                <BankNote01Icon className="size-3.5" />
                {t("Record funding")}
              </Button>
            ) : null}
          </div>
        ),
      }}
    >
      <StatusCallout submission={submission} />
      <KpiStrip aria-label={t("Expected funding")}>
        <KpiStripItem
          label={t("Face value")}
          value={money(submission.faceMinor)}
          sub={t("{0, plural, one {# invoice} other {# invoices}}", submission.invoiceCount)}
        />
        <KpiStripItem
          label={t("Expected advance")}
          value={money(submission.expectedAdvanceMinor)}
        />
        <KpiStripItem label={t("Expected fee")} value={money(submission.expectedFeeMinor)} />
        <KpiStripItem
          label={t("Reserve")}
          value={money(submission.expectedReserveMinor)}
          sub={t("Released when customers pay")}
        />
      </KpiStrip>
      <div className="grid gap-4 lg:grid-cols-2">
        <SectionPanel title={t("Delivery")}>
          <div className="p-3">
            <DescriptionList columns={2}>
              <DescriptionItem label={t("Status")}>
                <FactoringSubmissionStatusBadge status={submission.status} />
              </DescriptionItem>
              <DescriptionItem label={t("Factor")}>
                <Link
                  to={recordPath("factoring_company", submission.factoringCompanyId)}
                  className="text-brand hover:underline"
                >
                  {factorName}
                </Link>
              </DescriptionItem>
              <DescriptionItem label={t("Method")}>
                {FACTORING_TRANSPORT_METHOD_LABELS[submission.transportMethod]}
              </DescriptionItem>
              <DescriptionItem label={t("Sent to")}>
                <span className="break-all">{submission.transportTarget}</span>
              </DescriptionItem>
              <DescriptionItem label={t("Reference")}>
                {submission.transportReference ? (
                  <span className="font-mono text-xs break-all">
                    {submission.transportReference}
                  </span>
                ) : (
                  <DescriptionEmpty />
                )}
              </DescriptionItem>
              <DescriptionItem label={t("Attempts")} numeric>
                {submission.attempts}
              </DescriptionItem>
              <DescriptionItem label={t("Submitted")}>
                {formatUnixDateTimeShort(submission.createdAt)}
              </DescriptionItem>
              <DescriptionItem label={t("Sent")}>
                {submission.sentAt ? (
                  formatUnixDateTimeShort(submission.sentAt)
                ) : (
                  <DescriptionEmpty />
                )}
              </DescriptionItem>
            </DescriptionList>
          </div>
        </SectionPanel>
        <SectionPanel
          title={t("Documents")}
          help={t(
            "The schedule of accounts the factor reads, the same rows as a CSV for its import, and one packet per invoice with its supporting documents.",
          )}
        >
          <div className="flex flex-col gap-2 p-3 text-sm">
            <DocumentLink
              documentId={submission.schedulePdfDocumentId}
              label={t("Schedule of accounts (PDF)")}
            />
            <DocumentLink
              documentId={submission.scheduleCsvDocumentId}
              label={t("Schedule of accounts (CSV)")}
            />
          </div>
        </SectionPanel>
      </div>
      <SectionPanel title={t("Invoices")} count={submission.items.length}>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t("Invoice #")}</TableHead>
              <TableHead>{t("Customer")}</TableHead>
              <TableHead className="text-right">{t("Days")}</TableHead>
              <TableHead className="text-right">{t("Face value")}</TableHead>
              <TableHead className="text-right">{t("Advance")}</TableHead>
              <TableHead className="text-right">{t("Fee")}</TableHead>
              <TableHead className="text-right">{t("Reserve")}</TableHead>
              <TableHead>{t("Packet")}</TableHead>
              <TableHead>{t("Status")}</TableHead>
              {canBook ? (
                <TableHead>
                  <span className="sr-only">{t("Actions")}</span>
                </TableHead>
              ) : null}
            </TableRow>
          </TableHeader>
          <TableBody>
            {submission.items.map((item) => (
              <TableRow key={item.id}>
                <TableCell>
                  <Link
                    to={recordPath("invoice", item.invoiceId)}
                    className="font-mono font-medium hover:underline"
                  >
                    {item.invoiceNumber}
                  </Link>
                </TableCell>
                <TableCell>{item.customer?.name ?? ""}</TableCell>
                <TableCell className="text-right tabular-nums">{item.expectedDays}</TableCell>
                <TableCell className="text-right tabular-nums">{money(item.faceMinor)}</TableCell>
                <TableCell className="text-right tabular-nums">
                  <ItemAmount
                    actual={item.advanceMinor}
                    expected={item.expectedAdvanceMinor}
                    money={money}
                  />
                </TableCell>
                <TableCell className="text-right tabular-nums">
                  <ItemAmount
                    actual={item.feeMinor}
                    expected={item.expectedFeeMinor}
                    money={money}
                  />
                </TableCell>
                <TableCell className="text-right tabular-nums">
                  <ItemAmount
                    actual={item.reserveMinor}
                    expected={item.expectedReserveMinor}
                    money={money}
                  />
                </TableCell>
                <TableCell>
                  {item.packetDocumentId ? (
                    <a
                      href={documentContentUrl(item.packetDocumentId, "download")}
                      className="text-brand hover:underline"
                    >
                      {t("Download")}
                    </a>
                  ) : (
                    <span className="text-muted-foreground">{t("Not built yet")}</span>
                  )}
                </TableCell>
                <TableCell>
                  <div className="flex flex-col items-start gap-0.5">
                    <FactoringSubmissionItemStatusBadge status={item.status} />
                    <ItemStatusNote item={item} />
                  </div>
                </TableCell>
                {canBook ? (
                  <TableCell className="text-right">
                    <SubmissionItemActions
                      item={item}
                      recourse={recourse}
                      onSelect={(action) => setMovement({ item, action })}
                    />
                  </TableCell>
                ) : null}
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </SectionPanel>
      <CancelFactoringSubmissionDialog
        submission={cancelling ? submission : null}
        onOpenChange={(open) => {
          if (!open) setCancelling(false);
        }}
      />
      <RecordFundingDialog
        submission={submission}
        open={recordingFunding}
        onOpenChange={setRecordingFunding}
      />
      <ReserveMovementDialog
        submission={submission}
        target={movement}
        onOpenChange={(open) => {
          if (!open) setMovement(null);
        }}
      />
    </PageLayout>
  );
}

function StatusCallout({ submission }: { submission: FactoringSubmissionDetail }) {
  const t = useT();

  switch (submission.status) {
    case "Sending":
      return (
        <Alert size="sm" variant="info">
          <AlertDescription>
            {t(
              "The schedule is being built and delivered. A failed attempt is retried on its own before the schedule is marked as failed.",
            )}
          </AlertDescription>
        </Alert>
      );
    case "Failed":
      return (
        <Alert size="sm" variant="destructive">
          <AlertTitle>
            {submission.failedAt
              ? t("Delivery failed {0}", formatUnixDateTimeShort(submission.failedAt))
              : t("Delivery failed")}
          </AlertTitle>
          <AlertDescription>
            {submission.lastError ?? t("The factor could not be reached.")}{" "}
            {t(
              "Fix the factor's delivery settings and retry, or cancel the schedule to put its invoices back in the queue.",
            )}
          </AlertDescription>
        </Alert>
      );
    case "Cancelled":
      return (
        <Alert size="sm">
          <AlertDescription>
            {submission.cancelledAt
              ? t(
                  "Cancelled {0}: {1}",
                  formatUnixDateTimeShort(submission.cancelledAt),
                  submission.cancelReason ?? "",
                )
              : t("Cancelled: {0}", submission.cancelReason ?? "")}
          </AlertDescription>
        </Alert>
      );
    case "Sent":
      return (
        <Alert size="sm" variant="info">
          <AlertDescription>
            {submission.sentAt
              ? t(
                  "Delivered {0}. Record the factor's funding once its report or remittance arrives.",
                  formatUnixDateTimeShort(submission.sentAt),
                )
              : t("Delivered. Record the factor's funding once its report or remittance arrives.")}
          </AlertDescription>
        </Alert>
      );
    case "PartiallyFunded":
      return (
        <Alert size="sm" variant="info">
          <AlertDescription>
            {t(
              "The factor has decided some invoices on this schedule. Record the rest when its next report arrives.",
            )}
          </AlertDescription>
        </Alert>
      );
    case "Funded":
      return (
        <Alert size="sm" variant="success">
          <AlertDescription>
            {submission.fundedAt
              ? t(
                  "Every invoice is decided; funding was first recorded {0}. Release each reserve as the factor remits it.",
                  formatUnixDateTimeShort(submission.fundedAt),
                )
              : t("Every invoice is decided. Release each reserve as the factor remits it.")}
          </AlertDescription>
        </Alert>
      );
    case "Rejected":
      return (
        <Alert size="sm" variant="warning">
          <AlertDescription>
            {t(
              "The factor rejected every invoice on this schedule. They are back in the queue to send again or keep.",
            )}
          </AlertDescription>
        </Alert>
      );
  }
}

/** The amount the factor actually moved, or the expected one in a muted tone until it has. */
function ItemAmount({
  actual,
  expected,
  money,
}: {
  actual?: number | null;
  expected: number;
  money: (minor: number) => string;
}) {
  if (actual != null) {
    return <span>{money(actual)}</span>;
  }
  return <span className="text-muted-foreground">{money(expected)}</span>;
}

/** Why an invoice stands where it does on the schedule: its rejection, settlement or chargeback. */
function ItemStatusNote({ item }: { item: FactoringSubmissionItem }) {
  const t = useT();

  switch (item.status) {
    case "Rejected":
      return item.rejectionReason ? (
        <span className="text-muted-foreground text-xs">{item.rejectionReason}</span>
      ) : null;
    case "Funded":
      return item.fundedOn ? (
        <span className="text-muted-foreground text-xs">
          {t("Funded {0}", formatUnixDateMedium(item.fundedOn))}
        </span>
      ) : null;
    case "Settled":
      return item.settledOn ? (
        <span className="text-muted-foreground text-xs">
          {t("Reserve released {0}", formatUnixDateMedium(item.settledOn))}
        </span>
      ) : null;
    case "ChargedBack":
      return (
        <span className="text-muted-foreground text-xs">
          {item.chargedBackOn
            ? t("Charged back {0}", formatUnixDateMedium(item.chargedBackOn))
            : null}
          {item.chargebackReason ? ` · ${item.chargebackReason}` : null}
        </span>
      );
    case "Pending":
    case "Released":
      return null;
  }
}

function DocumentLink({ documentId, label }: { documentId?: string | null; label: string }) {
  const t = useT();

  if (!documentId) {
    return (
      <div className="flex items-center justify-between gap-3">
        <span>{label}</span>
        <span className="text-muted-foreground">{t("Not built yet")}</span>
      </div>
    );
  }

  return (
    <div className="flex items-center justify-between gap-3">
      <a
        href={documentContentUrl(documentId, "view")}
        target="_blank"
        rel="noreferrer"
        className="text-brand hover:underline"
      >
        {label}
      </a>
      <Button
        variant="ghost"
        size="icon-sm"
        aria-label={t("Download {0}", label)}
        render={<a href={documentContentUrl(documentId, "download")} />}
      >
        <Download01Icon className="size-3.5" />
      </Button>
    </div>
  );
}
