import { FactoringRemittanceStatusBadge } from "@/components/factoring/factoring-badges";
import { KpiStrip, KpiStripItem } from "@/components/kpi/kpi-strip";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { recordPath } from "@/config/record-links";
import { useBreadcrumbLabel } from "@/hooks/use-breadcrumb-label";
import { usePermission } from "@/hooks/use-permission";
import type { RoutePrefetch } from "@/lib/route-prefetch";
import { FACTORING_PATH } from "@/lib/factoring";
import {
  canApplyRemittance,
  canDiscardRemittance,
  canEditRemittanceLines,
  FACTORING_REMITTANCE_LINE_KIND_LABELS,
  remittanceStep,
} from "@/lib/factoring-remittance";
import {
  discardFactoringRemittance,
  rematchFactoringRemittance,
  type FactoringRemittance,
} from "@/lib/graphql/factoring-remittance";
import { factoring as factoringQueries, invalidateFactoringQueries } from "@/lib/queries/factoring";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ArrowLeftIcon, BankNote01Icon, RefreshCw01Icon } from "@trenova/shared/components/icons";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { describeError } from "@trenova/shared/lib/error-presentation";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { formatCurrency } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useState } from "react";
import { Link, useParams } from "react-router";
import { toast } from "sonner";
import { ApplyRemittanceDialog } from "./_components/apply-remittance-dialog";
import { ReasonDialog } from "@/components/reason-dialog";
import { RemittanceExceptions, RemittanceMatched } from "./_components/remittance-lines";
import { RemittanceMappingStep } from "./_components/remittance-mapping";
import { RemittancePageSkeleton } from "./_components/remittance-page-skeleton";
import { RemittanceReupload } from "./_components/remittance-reupload";

export const prefetch: RoutePrefetch = ({ params }) =>
  params.id ? [factoringQueries.remittance(params.id)] : [];

/** How often a remittance whose PDF is still being read is checked again. */
const READING_POLL_MS = 5000;

/**
 * One factor remittance: the report as it was read, its lines matched to the
 * factor's schedules, the exceptions a person resolves, and the apply that
 * books the matched lines through the factoring ledger.
 */
export function FactoringRemittancePage() {
  const t = useT();
  const { id = "" } = useParams<{ id: string }>();
  const { data, isLoading, error, refetch } = useQuery({
    ...factoringQueries.remittance(id),
    enabled: id !== "",
    refetchInterval: (query) =>
      query.state.data && remittanceStep(query.state.data) === "reading" ? READING_POLL_MS : false,
  });
  useBreadcrumbLabel(data?.fileName ?? undefined);

  const backButton = (
    <Button variant="outline" size="sm" render={<Link to={`${FACTORING_PATH}?view=remittances`} />}>
      <ArrowLeftIcon className="size-3.5" />
      {t("Remittances")}
    </Button>
  );

  if (isLoading) {
    return (
      <PageLayout
        pageHeaderProps={{ title: t("Factor remittance"), description: t("Loading remittance...") }}
      >
        <RemittancePageSkeleton />
      </PageLayout>
    );
  }

  if (error || !data) {
    const described = error ? describeError(error) : null;
    return (
      <PageLayout
        pageHeaderProps={{
          title: t("Factor remittance"),
          description: t("This remittance could not be loaded."),
          actions: backButton,
        }}
      >
        <Alert size="sm" variant="destructive">
          <AlertDescription className="flex items-center justify-between gap-3">
            <span>
              {described?.kind === "not-found"
                ? t("No remittance has this id, or it belongs to another organization.")
                : (described?.detail ?? t("The remittance could not be loaded."))}
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

  return <RemittanceDetail remittance={data} backButton={backButton} />;
}

function RemittanceDetail({
  remittance,
  backButton,
}: {
  remittance: FactoringRemittance;
  backButton: React.ReactNode;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const { allowed: canUpdate } = usePermission(Resource.FactoringRemittance, Operation.Update);
  const { allowed: canApply } = usePermission(Resource.FactoringRemittance, Operation.Import);
  const { allowed: canCancel } = usePermission(Resource.FactoringRemittance, Operation.Cancel);
  const [applying, setApplying] = useState(false);
  const [discarding, setDiscarding] = useState(false);
  const step = remittanceStep(remittance);
  const showsLines = step === "review" || step === "done";
  const lines = useQuery({
    ...factoringQueries.remittanceLines(remittance.id, null),
    enabled: showsLines,
  });
  const money = (minor: number) => formatCurrency(minor / 100, remittance.currencyCode);
  const factorName = remittance.factoringCompany?.name ?? remittance.factoringCompanyId;

  const rematch = useMutation({
    mutationFn: () =>
      rematchFactoringRemittance({ id: remittance.id, version: remittance.version }),
    onSuccess: (matched) => {
      invalidateFactoringQueries(queryClient);
      toast.success(t("Remittance matched again"), {
        description: t(
          "{0, plural, one {# line matched} other {# lines matched}}, {1, plural, one {# to resolve} other {# to resolve}}.",
          matched.matchedCount,
          matched.exceptionCount,
        ),
      });
    },
    onError: (error) => {
      const described = describeError(error);
      toast.error(t("The remittance was not matched again"), {
        description: described.detail ?? described.message,
      });
    },
  });

  const discard = useMutation({
    mutationFn: (reason: string) =>
      discardFactoringRemittance({ id: remittance.id, version: remittance.version, reason }),
    onSuccess: () => {
      invalidateFactoringQueries(queryClient);
      setDiscarding(false);
      toast.success(t("Remittance discarded"));
    },
    onError: (error) => {
      const described = describeError(error);
      toast.error(t("The remittance was not discarded"), {
        description: described.detail ?? described.message,
      });
    },
  });

  return (
    <PageLayout
      pageHeaderProps={{
        title: remittance.fileName || t("Factor remittance"),
        description: t(
          "{0} report from {1}",
          FACTORING_REMITTANCE_LINE_KIND_LABELS[remittance.defaultKind],
          factorName,
        ),
        actions: (
          <div className="flex items-center gap-2">
            {backButton}
            {canCancel && canDiscardRemittance(remittance.status) ? (
              <Button type="button" variant="outline" size="sm" onClick={() => setDiscarding(true)}>
                {t("Discard")}
              </Button>
            ) : null}
            {canUpdate && canEditRemittanceLines(remittance.status) ? (
              <Button
                type="button"
                variant="outline"
                size="sm"
                isLoading={rematch.isPending}
                onClick={() => rematch.mutate()}
              >
                <RefreshCw01Icon className="size-3.5" />
                {t("Match again")}
              </Button>
            ) : null}
            {canApply && canApplyRemittance(remittance) ? (
              <Button type="button" size="sm" onClick={() => setApplying(true)}>
                <BankNote01Icon className="size-3.5" />
                {t("Apply")}
              </Button>
            ) : null}
          </div>
        ),
      }}
    >
      <StatusCallout remittance={remittance} />
      <KpiStrip aria-label={t("Remittance totals")}>
        <KpiStripItem
          label={t("Lines")}
          value={String(remittance.lineCount)}
          sub={<FactoringRemittanceStatusBadge status={remittance.status} />}
        />
        <KpiStripItem label={t("Matched")} value={String(remittance.matchedCount)} />
        <KpiStripItem label={t("To resolve")} value={String(remittance.exceptionCount)} />
        <KpiStripItem label={t("Booked")} value={String(remittance.appliedCount)} />
        <KpiStripItem
          label={t("Cash")}
          value={money(remittance.totalCashMinor)}
          sub={
            remittance.remittedOn
              ? t("Remitted {0}", formatUnixDateMedium(remittance.remittedOn))
              : undefined
          }
        />
      </KpiStrip>
      {step === "mapping" && canUpdate ? <RemittanceMappingStep remittance={remittance} /> : null}
      {(step === "failed" || step === "upload") && canUpdate ? (
        <RemittanceReupload remittance={remittance} />
      ) : null}
      {showsLines ? (
        <div className="grid gap-4 xl:grid-cols-2">
          <RemittanceExceptions remittance={remittance} lines={lines.data ?? []} />
          <RemittanceMatched remittance={remittance} lines={lines.data ?? []} />
        </div>
      ) : null}
      <ApplyRemittanceDialog remittance={remittance} open={applying} onOpenChange={setApplying} />
      <ReasonDialog
        open={discarding}
        title={t("Discard remittance")}
        description={t(
          "A discarded remittance is set aside for good; nothing on it can be booked afterwards.",
        )}
        placeholder={t("Such as: a duplicate of yesterday's report")}
        confirmLabel={t("Discard")}
        destructive
        pending={discard.isPending}
        onConfirm={(reason) => discard.mutate(reason)}
        onOpenChange={setDiscarding}
      />
      {remittance.inboundMessageId ? (
        <p className="text-muted-foreground text-xs">
          {t("Arrived by email.")}{" "}
          <Link
            to={recordPath("inbound_message", remittance.inboundMessageId)}
            className="text-brand"
          >
            {t("Open the message")}
          </Link>
        </p>
      ) : null}
    </PageLayout>
  );
}

function StatusCallout({ remittance }: { remittance: FactoringRemittance }) {
  const t = useT();
  switch (remittanceStep(remittance)) {
    case "reading":
      return (
        <Alert size="sm" variant="info">
          <AlertTitle>{t("Reading the PDF")}</AlertTitle>
          <AlertDescription>
            {t("Its text is being read; the lines appear here once it is matched.")}
          </AlertDescription>
        </Alert>
      );
    case "upload":
      return (
        <Alert size="sm" variant="info">
          <AlertDescription>{t("Upload the factor's file to read it.")}</AlertDescription>
        </Alert>
      );
    case "mapping":
      return (
        <Alert size="sm" variant="warning">
          <AlertTitle>{t("Map the columns")}</AlertTitle>
          <AlertDescription>
            {t(
              "The file's headers did not say which columns hold the invoice and the amounts. Map them once and this factor's next file is read on its own.",
            )}
          </AlertDescription>
        </Alert>
      );
    case "failed":
      return (
        <Alert size="sm" variant="destructive">
          <AlertTitle>{t("The remittance could not be read")}</AlertTitle>
          <AlertDescription>{remittance.error}</AlertDescription>
        </Alert>
      );
    case "done":
      return remittance.status === "Discarded" ? (
        <Alert size="sm" variant="info">
          <AlertTitle>{t("Discarded")}</AlertTitle>
          <AlertDescription>{remittance.discardReason}</AlertDescription>
        </Alert>
      ) : null;
    case "review":
      return remittance.status === "PartiallyApplied" ? (
        <Alert size="sm" variant="warning">
          <AlertDescription>
            {t(
              "Part of this remittance is booked. Resolve or ignore the lines that are left, then apply again.",
            )}
          </AlertDescription>
        </Alert>
      ) : null;
  }
}
