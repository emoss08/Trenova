import { FactoringAgreementStatusBadge } from "@/components/factoring/factoring-badges";
import { HoverGlideGroup } from "@/components/hover-glide";
import { SectionPanel, SectionPanelQuiet } from "@/components/section-panel";
import { usePermission } from "@/hooks/use-permission";
import {
  FACTORING_ACCOUNTING_TREATMENT_DESCRIPTIONS,
  FACTORING_ACCOUNTING_TREATMENT_LABELS,
  FACTORING_RECOURSE_DESCRIPTIONS,
  FACTORING_RECOURSE_LABELS,
  formatBasisPoints,
  formatMinutesOfDay,
} from "@/lib/factoring";
import type { FactoringAgreement } from "@/lib/graphql/factoring";
import { factoring as factoringQueries } from "@/lib/queries/factoring";
import { useQuery } from "@tanstack/react-query";
import { ErrorState } from "@trenova/shared/components/errors/error-state";
import {
  Copy01Icon,
  Edit05Icon,
  PlayIcon,
  PlusIcon,
  StopCircleIcon,
} from "@trenova/shared/components/icons";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  DescriptionEmpty,
  DescriptionItem,
  DescriptionList,
} from "@trenova/shared/components/ui/description-list";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { dateToUnixTimestamp, formatUnixDate } from "@trenova/shared/lib/date";
import { formatMinor } from "@trenova/shared/lib/money";
import { cn } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useState } from "react";
import { FactoringAgreementDialog } from "./factoring-agreement-dialog";
import { ActivateAgreementDialog, EndAgreementDialog } from "./factoring-agreement-status-dialogs";
import { FactoringAgreementTabSkeleton } from "./factoring-company-skeleton";
import { agreementDifferences, feeTierText, sortedTiers } from "./factoring-agreement-changes";
import {
  FactoringAgreementActivity,
  FactoringAgreementBookedTo,
  FactoringAgreementHistory,
} from "./factoring-agreement-panels";
import { FactoringFundingCalculator } from "./factoring-funding-calculator";

type Editing = { agreement: FactoringAgreement | null; template: FactoringAgreement | null };

function period(t: TranslateFn, agreement: FactoringAgreement) {
  if (agreement.status === "Draft") {
    return t("Would start {0}", formatUnixDate(agreement.effectiveFrom));
  }
  return agreement.effectiveTo
    ? t(
        "{0} to {1}",
        formatUnixDate(agreement.effectiveFrom),
        formatUnixDate(agreement.effectiveTo),
      )
    : t("{0} to now", formatUnixDate(agreement.effectiveFrom));
}

/**
 * The factor's agreements as versions: the one in force, drafts waiting for a signature with what
 * they would change, and ended ones kept for the invoices posted under them. The selected version's
 * terms can be tried on an invoice, read in full, and moved along its life.
 */
export default function FactoringAgreementTab({
  factoringCompanyId,
  highlightedId,
}: {
  factoringCompanyId: string;
  highlightedId: string | null;
}) {
  const t = useT();
  const { allowed: canCreate } = usePermission(Resource.FactoringAgreement, Operation.Create);
  const { allowed: canUpdate } = usePermission(Resource.FactoringAgreement, Operation.Update);
  const { data, isLoading, error, refetch } = useQuery(
    factoringQueries.agreements(factoringCompanyId),
  );
  const [picked, setPicked] = useState<string | null>(highlightedId);
  const [editing, setEditing] = useState<Editing | null>(null);
  const [activating, setActivating] = useState<FactoringAgreement | null>(null);
  const [ending, setEnding] = useState<FactoringAgreement | null>(null);
  const [nowUnix] = useState(() => dateToUnixTimestamp(new Date()));

  if (isLoading) return <FactoringAgreementTabSkeleton />;
  if (error || !data) {
    return (
      <ErrorState
        layout="sheet"
        error={error}
        onRetry={() => void refetch()}
        title={t("The agreements could not be loaded")}
      />
    );
  }

  const order = { Draft: 0, Active: 1, Ended: 2 } as const;
  const versions = [...data].sort(
    (a, b) => order[a.status] - order[b.status] || b.effectiveFrom - a.effectiveFrom,
  );
  const active = versions.find((agreement) => agreement.status === "Active") ?? null;
  const numbers = new Map(
    [...data]
      .sort((a, b) => a.effectiveFrom - b.effectiveFrom || a.createdAt - b.createdAt)
      .map((agreement, index) => [agreement.id, index + 1]),
  );
  const byStart = [...data].sort(
    (a, b) => a.effectiveFrom - b.effectiveFrom || a.createdAt - b.createdAt,
  );
  const selected =
    versions.find((agreement) => agreement.id === picked) ?? active ?? versions[0] ?? null;
  const previous = selected
    ? (byStart[byStart.findIndex((agreement) => agreement.id === selected.id) - 1] ?? null)
    : null;

  const dialogs = (
    <>
      <FactoringAgreementDialog
        factoringCompanyId={factoringCompanyId}
        agreement={editing?.agreement ?? null}
        template={editing?.template ?? null}
        open={editing !== null}
        onOpenChange={(open) => {
          if (!open) setEditing(null);
        }}
      />
      <ActivateAgreementDialog
        agreement={activating}
        onOpenChange={(open) => {
          if (!open) setActivating(null);
        }}
      />
      <EndAgreementDialog
        agreement={ending}
        onOpenChange={(open) => {
          if (!open) setEnding(null);
        }}
      />
    </>
  );

  if (!selected) {
    return (
      <SectionPanel title={t("Agreement")}>
        <SectionPanelQuiet>
          {t(
            "No agreement yet. Invoices assigned to this factor aren't ready to factor until one is active.",
          )}
        </SectionPanelQuiet>
        {canCreate ? (
          <div className="px-3 pb-3">
            <Button
              type="button"
              size="sm"
              onClick={() => setEditing({ agreement: null, template: null })}
            >
              <PlusIcon />
              {t("New agreement")}
            </Button>
          </div>
        ) : null}
        {dialogs}
      </SectionPanel>
    );
  }

  const tiers = sortedTiers(selected);

  return (
    <div className="grid items-start gap-4 lg:flex-1 lg:grid-cols-[15rem_minmax(0,1fr)] lg:items-stretch">
      <SectionPanel
        title={t("Versions")}
        action={
          canCreate ? (
            <Button
              type="button"
              size="xs"
              variant="ghost"
              onClick={() => setEditing({ agreement: null, template: null })}
            >
              <PlusIcon />
              {t("New agreement")}
            </Button>
          ) : null
        }
      >
        <div className="min-h-0 flex-1 overflow-y-auto bg-card">
          <HoverGlideGroup className="bg-card p-1.5">
            <ul className="flex flex-col gap-px">
              {versions.map((agreement) => {
                const changes =
                  agreement.status === "Draft" && active
                    ? agreementDifferences(t, agreement, active)
                    : [];
                const isSelected = agreement.id === selected.id;
                return (
                  <li key={agreement.id}>
                    <button
                      type="button"
                      data-glide-target
                      aria-current={isSelected ? "true" : undefined}
                      onClick={() => setPicked(agreement.id)}
                      className={cn(
                        "relative flex w-full flex-col gap-1 rounded-md ui-focus-ring px-2.5 py-2 text-left",
                        isSelected && "bg-surface-active",
                      )}
                    >
                      <span className="flex items-center justify-between gap-2 text-sm font-medium">
                        {t("v{0}", numbers.get(agreement.id) ?? 1)}
                        <FactoringAgreementStatusBadge status={agreement.status} />
                      </span>
                      <span className="text-xs text-muted-foreground">{period(t, agreement)}</span>
                      {changes.length > 0 ? (
                        <span className="flex flex-wrap gap-1">
                          {changes.map((change) => (
                            <span
                              key={change}
                              className="rounded-sm bg-info-subtle px-1.5 font-mono text-2xs text-info-subtle-foreground"
                            >
                              {change}
                            </span>
                          ))}
                        </span>
                      ) : null}
                    </button>
                  </li>
                );
              })}
            </ul>
          </HoverGlideGroup>
        </div>
      </SectionPanel>

      <div className="flex min-w-0 flex-col gap-4">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div className="flex min-w-0 flex-col gap-1">
            <div className="flex items-center gap-2">
              <h3 className="text-base font-semibold">
                {t("Agreement v{0}", numbers.get(selected.id) ?? 1)}
              </h3>
              <FactoringAgreementStatusBadge status={selected.status} />
            </div>
            <p className="text-xs text-muted-foreground">
              {selected.status === "Active"
                ? t(
                    "In force since {0}. Funding booked to this factor uses these terms; a draft takes over only when someone activates it.",
                    formatUnixDate(selected.effectiveFrom),
                  )
                : selected.status === "Draft"
                  ? t(
                      "A draft. Nothing uses these terms until it is activated, from {0}.",
                      formatUnixDate(selected.effectiveFrom),
                    )
                  : t("Ended. Invoices posted while it was in force keep these terms.")}
            </p>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            {canUpdate && selected.status !== "Ended" ? (
              <Button
                type="button"
                size="sm"
                variant="outline"
                onClick={() => setEditing({ agreement: selected, template: null })}
              >
                <Edit05Icon />
                {t("Edit")}
              </Button>
            ) : null}
            {canCreate ? (
              <Button
                type="button"
                size="sm"
                variant="outline"
                onClick={() => setEditing({ agreement: null, template: selected })}
              >
                <Copy01Icon />
                {t("New version from these terms")}
              </Button>
            ) : null}
            {canUpdate && selected.status === "Draft" ? (
              <Button type="button" size="sm" onClick={() => setActivating(selected)}>
                <PlayIcon />
                {t("Activate")}
              </Button>
            ) : null}
            {canUpdate && selected.status === "Active" ? (
              <Button type="button" size="sm" variant="outline" onClick={() => setEnding(selected)}>
                <StopCircleIcon />
                {t("End")}
              </Button>
            ) : null}
          </div>
        </div>

        <SectionPanel
          title={t("What an invoice pays")}
          tip={t(
            "Worked out by the same calculation funding uses, so these are the figures an invoice will be funded at.",
          )}
        >
          <FactoringFundingCalculator
            terms={{
              advanceRateBp: selected.advanceRateBp,
              reserveRateBp: selected.reserveRateBp,
              minimumFeeMinor: selected.minimumFeeMinor,
              feeTiers: selected.feeTiers,
            }}
            fundingDays={selected.fundingDays}
          >
            <DescriptionList layout="ruled" columns={3}>
              <DescriptionItem label={t("Advance rate")} numeric>
                {formatBasisPoints(selected.advanceRateBp)}
              </DescriptionItem>
              <DescriptionItem label={t("Reserve rate")} numeric>
                {formatBasisPoints(selected.reserveRateBp)}
                <span className="block text-xs text-muted-foreground">
                  {t("Held until released or charged back")}
                </span>
              </DescriptionItem>
              <DescriptionItem label={t("Minimum fee")} numeric>
                {formatMinor(selected.minimumFeeMinor)}
              </DescriptionItem>
              <DescriptionItem label={t("Recourse")}>
                {FACTORING_RECOURSE_LABELS[selected.recourse]}
                <span className="block text-xs text-muted-foreground">
                  {FACTORING_RECOURSE_DESCRIPTIONS[selected.recourse]}
                </span>
              </DescriptionItem>
              <DescriptionItem label={t("Chargeback window")} numeric>
                {t("{0, plural, one {# day} other {# days}}", selected.chargebackWindowDays)}
              </DescriptionItem>
              <DescriptionItem label={t("Accounting treatment")}>
                {FACTORING_ACCOUNTING_TREATMENT_LABELS[selected.accountingTreatment]}
                <span className="block text-xs text-muted-foreground">
                  {FACTORING_ACCOUNTING_TREATMENT_DESCRIPTIONS[selected.accountingTreatment]}
                </span>
              </DescriptionItem>
              <DescriptionItem label={t("Funding")} numeric>
                {t(
                  "{0, plural, one {# business day} other {# business days}}",
                  selected.fundingDays,
                )}
              </DescriptionItem>
              <DescriptionItem label={t("Daily schedule time")} numeric>
                {formatMinutesOfDay(selected.submissionTime)}
              </DescriptionItem>
              <DescriptionItem label={t("Fee tiers")} numeric>
                {tiers.length === 0 ? (
                  <DescriptionEmpty />
                ) : (
                  <ul className="flex flex-wrap gap-x-1.5 gap-y-0.5 font-mono text-xs">
                    {tiers.map((tier, index) => (
                      <li key={tier.id} className="flex gap-1.5">
                        {index > 0 ? (
                          <span aria-hidden className="text-foreground-subtle">
                            ·
                          </span>
                        ) : null}
                        {feeTierText(t, tier)}
                      </li>
                    ))}
                  </ul>
                )}
              </DescriptionItem>
              <DescriptionItem label={t("Documents the factor needs")} span="full">
                {selected.documentTypes.length > 0 ? (
                  <span className="flex flex-wrap gap-1">
                    {selected.documentTypes.map((documentType) => (
                      <Badge key={documentType.id} variant="neutral">
                        {documentType.name}
                      </Badge>
                    ))}
                  </span>
                ) : (
                  <DescriptionEmpty />
                )}
              </DescriptionItem>
            </DescriptionList>
          </FactoringFundingCalculator>
        </SectionPanel>

        <FactoringAgreementActivity agreement={selected} />
        <FactoringAgreementBookedTo agreement={selected} />

        <FactoringAgreementHistory
          agreement={selected}
          previous={previous}
          previousNumber={previous ? (numbers.get(previous.id) ?? null) : null}
          nowUnix={nowUnix}
          className="flex-1"
        />
      </div>
      {dialogs}
    </div>
  );
}
