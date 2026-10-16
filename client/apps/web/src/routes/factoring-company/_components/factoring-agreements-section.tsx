import { FactoringAgreementStatusBadge } from "@/components/factoring/factoring-badges";
import { DateField } from "@/components/fields/date-field/date-field";
import { SectionPanel } from "@/components/section-panel";
import { handleMutationError, useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import {
  FACTORING_ACCOUNTING_TREATMENT_LABELS,
  FACTORING_RECOURSE_LABELS,
  formatBasisPoints,
  formatMinutesOfDay,
} from "@/lib/factoring";
import {
  activateFactoringAgreement,
  endFactoringAgreement,
  type FactoringAgreement,
} from "@/lib/graphql/factoring";
import { factoring as factoringQueries, invalidateFactoringQueries } from "@/lib/queries/factoring";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Edit05Icon, PlayIcon, PlusIcon, StopCircleIcon } from "@trenova/shared/components/icons";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@trenova/shared/components/ui/alert-dialog";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  DescriptionEmpty,
  DescriptionItem,
  DescriptionList,
} from "@trenova/shared/components/ui/description-list";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDate } from "@trenova/shared/lib/date";
import { cn, formatCurrency } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useEffect, useState } from "react";
import { FormProvider, useForm, type Resolver } from "react-hook-form";
import { z } from "zod";
import { FactoringAgreementDialog } from "./factoring-agreement-dialog";

/**
 * The factor's agreements, newest first: the active one is what posting and
 * readiness use, drafts wait for the contract to be signed, and ended ones
 * stay for the invoices posted under them.
 */
export function FactoringAgreementsSection({
  factoringCompanyId,
  highlightedId,
}: {
  factoringCompanyId: string;
  highlightedId?: string | null;
}) {
  const t = useT();
  const { allowed: canRead } = usePermission(Resource.FactoringAgreement, Operation.Read);
  const { allowed: canCreate } = usePermission(Resource.FactoringAgreement, Operation.Create);
  const { allowed: canUpdate } = usePermission(Resource.FactoringAgreement, Operation.Update);
  const { data, isLoading, isError, refetch } = useQuery({
    ...factoringQueries.agreements(factoringCompanyId),
    enabled: canRead,
  });
  const [editing, setEditing] = useState<{ agreement: FactoringAgreement | null } | null>(null);
  const [activating, setActivating] = useState<FactoringAgreement | null>(null);
  const [ending, setEnding] = useState<FactoringAgreement | null>(null);

  if (!canRead) {
    return null;
  }

  const agreements = data ?? [];

  return (
    <SectionPanel
      title={t("Agreements")}
      count={agreements.length}
      help={t(
        "The terms you agreed with this factor. Invoices are only ready to factor while an agreement is active.",
      )}
      action={
        canCreate ? (
          <Button type="button" size="sm" onClick={() => setEditing({ agreement: null })}>
            <PlusIcon />
            {t("New agreement")}
          </Button>
        ) : null
      }
    >
      {isLoading ? (
        <div className="space-y-2 p-3">
          <Skeleton className="h-4 w-48" />
          <Skeleton className="h-4 w-64" />
        </div>
      ) : isError ? (
        <div className="p-3">
          <Alert size="sm" variant="destructive">
            <AlertDescription className="flex items-center justify-between gap-3">
              <span>{t("The agreements could not be loaded.")}</span>
              <Button type="button" size="sm" variant="outline" onClick={() => void refetch()}>
                {t("Retry")}
              </Button>
            </AlertDescription>
          </Alert>
        </div>
      ) : agreements.length === 0 ? (
        <p className="text-muted-foreground px-3 py-3 text-sm">
          {t(
            "No agreements yet. Invoices assigned to this factor are not ready until one is active.",
          )}
        </p>
      ) : (
        <ul className="divide-y">
          {agreements.map((agreement) => (
            <AgreementRow
              key={agreement.id}
              agreement={agreement}
              highlighted={agreement.id === highlightedId}
              canUpdate={canUpdate}
              onEdit={() => setEditing({ agreement })}
              onActivate={() => setActivating(agreement)}
              onEnd={() => setEnding(agreement)}
            />
          ))}
        </ul>
      )}
      <FactoringAgreementDialog
        factoringCompanyId={factoringCompanyId}
        agreement={editing?.agreement ?? null}
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
    </SectionPanel>
  );
}

function AgreementRow({
  agreement,
  highlighted,
  canUpdate,
  onEdit,
  onActivate,
  onEnd,
}: {
  agreement: FactoringAgreement;
  highlighted: boolean;
  canUpdate: boolean;
  onEdit: () => void;
  onActivate: () => void;
  onEnd: () => void;
}) {
  const t = useT();
  const tiers = [...agreement.feeTiers].sort((a, b) => a.daysFrom - b.daysFrom);

  return (
    <li
      className={cn("flex flex-col gap-3 px-3 py-3", highlighted && "bg-brand-subtle")}
      data-testid="factoring-agreement-row"
      aria-current={highlighted ? "true" : undefined}
    >
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="flex min-w-0 items-center gap-2">
          <FactoringAgreementStatusBadge status={agreement.status} />
          <span className="text-sm">
            {agreement.effectiveTo
              ? t(
                  "{0} to {1}",
                  formatUnixDate(agreement.effectiveFrom),
                  formatUnixDate(agreement.effectiveTo),
                )
              : t("From {0}", formatUnixDate(agreement.effectiveFrom))}
          </span>
        </div>
        {canUpdate ? (
          <div className="flex items-center gap-2">
            {agreement.status !== "Ended" ? (
              <Button type="button" size="sm" variant="outline" onClick={onEdit}>
                <Edit05Icon />
                {t("Edit")}
              </Button>
            ) : null}
            {agreement.status === "Draft" ? (
              <Button type="button" size="sm" variant="outline" onClick={onActivate}>
                <PlayIcon />
                {t("Activate")}
              </Button>
            ) : null}
            {agreement.status === "Active" ? (
              <Button type="button" size="sm" variant="outline" onClick={onEnd}>
                <StopCircleIcon />
                {t("End")}
              </Button>
            ) : null}
          </div>
        ) : null}
      </div>
      <DescriptionList columns={3}>
        <DescriptionItem label={t("Advance rate")} numeric>
          {formatBasisPoints(agreement.advanceRateBp)}
        </DescriptionItem>
        <DescriptionItem label={t("Reserve rate")} numeric>
          {formatBasisPoints(agreement.reserveRateBp)}
        </DescriptionItem>
        <DescriptionItem label={t("Minimum fee")} numeric>
          {formatCurrency(agreement.minimumFeeMinor / 100)}
        </DescriptionItem>
        <DescriptionItem label={t("Recourse")}>
          {FACTORING_RECOURSE_LABELS[agreement.recourse]}
        </DescriptionItem>
        <DescriptionItem label={t("Accounting treatment")}>
          {FACTORING_ACCOUNTING_TREATMENT_LABELS[agreement.accountingTreatment]}
        </DescriptionItem>
        <DescriptionItem label={t("Chargeback window")} numeric>
          {t("{0, plural, one {# day} other {# days}}", agreement.chargebackWindowDays)}
        </DescriptionItem>
        <DescriptionItem label={t("Funding")} numeric>
          {t("{0, plural, one {# business day} other {# business days}}", agreement.fundingDays)}
        </DescriptionItem>
        <DescriptionItem label={t("Daily schedule time")} numeric>
          {formatMinutesOfDay(agreement.submissionTime)}
        </DescriptionItem>
        <DescriptionItem label={t("Version")} numeric>
          {agreement.version}
        </DescriptionItem>
        <DescriptionItem label={t("Fees")} span="full">
          <ul className="space-y-0.5">
            {tiers.map((tier) => (
              <li key={tier.id} className="tabular-nums">
                {tier.daysTo == null
                  ? t(
                      "Day {0} on: {1} plus {2}",
                      tier.daysFrom,
                      formatBasisPoints(tier.rateBp),
                      formatCurrency(tier.flatMinor / 100),
                    )
                  : t(
                      "Days {0} to {1}: {2} plus {3}",
                      tier.daysFrom,
                      tier.daysTo,
                      formatBasisPoints(tier.rateBp),
                      formatCurrency(tier.flatMinor / 100),
                    )}
              </li>
            ))}
          </ul>
        </DescriptionItem>
        <DescriptionItem label={t("Documents the factor needs")} span="full">
          {agreement.documentTypes.length > 0 ? (
            <span className="flex flex-wrap gap-1">
              {agreement.documentTypes.map((documentType) => (
                <Badge key={documentType.id} variant="neutral">
                  {documentType.name}
                </Badge>
              ))}
            </span>
          ) : (
            <DescriptionEmpty />
          )}
        </DescriptionItem>
        {agreement.notes ? (
          <DescriptionItem label={t("Notes")} span="full" valueClassName="whitespace-pre-line">
            {agreement.notes}
          </DescriptionItem>
        ) : null}
      </DescriptionList>
    </li>
  );
}

function ActivateAgreementDialog({
  agreement,
  onOpenChange,
}: {
  agreement: FactoringAgreement | null;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const { mutate, isPending } = useMutation({
    mutationFn: async () => {
      if (!agreement) throw new Error("No agreement selected");
      return activateFactoringAgreement({ id: agreement.id, version: agreement.version });
    },
    onSuccess: () => {
      invalidateFactoringQueries(queryClient);
      onOpenChange(false);
    },
    onError: (error) => handleMutationError({ error, resourceName: "factoring agreement" }),
  });

  return (
    <AlertDialog open={agreement !== null} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{t("Activate this agreement?")}</AlertDialogTitle>
          <AlertDialogDescription>
            {t(
              "Invoices posted to this factor from its start date use these terms, and every open invoice assigned to the factor is checked again.",
            )}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>{t("Cancel")}</AlertDialogCancel>
          <AlertDialogAction isLoading={isPending} onClick={() => mutate()}>
            {t("Activate")}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}

const END_FORM_ID = "end-factoring-agreement-form";

const endSchema = z.object({ endAt: z.number().nullable() });
type EndValues = z.infer<typeof endSchema>;

function EndAgreementDialog({
  agreement,
  onOpenChange,
}: {
  agreement: FactoringAgreement | null;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const form = useForm<EndValues>({
    resolver: zodResolver(endSchema) as Resolver<EndValues>,
    defaultValues: { endAt: null },
  });
  const { reset } = form;

  useEffect(() => {
    if (agreement) reset({ endAt: null });
  }, [agreement, reset]);

  const mutation = useApiMutation<FactoringAgreement, EndValues, unknown, EndValues>({
    form,
    resourceName: "factoring agreement",
    mutationFn: (values) => {
      if (!agreement) throw new Error("No agreement selected");
      return endFactoringAgreement({
        id: agreement.id,
        version: agreement.version,
        endAt: values.endAt,
      });
    },
    onSuccess: () => {
      invalidateFactoringQueries(queryClient);
      onOpenChange(false);
    },
  });

  return (
    <Dialog open={agreement !== null} onOpenChange={onOpenChange}>
      <DialogContent size="sm">
        <DialogHeader>
          <DialogTitle>{t("End this agreement?")}</DialogTitle>
          <DialogDescription>
            {t(
              "Invoices posted after it ends are not ready to factor until another agreement is active. Invoices already posted keep the terms they were posted under.",
            )}
          </DialogDescription>
        </DialogHeader>
        <FormProvider {...form}>
          <Form id={END_FORM_ID} onSubmit={form.handleSubmit((values) => mutation.mutate(values))}>
            <FormGroup cols={1}>
              <FormControl>
                <DateField
                  control={form.control}
                  name="endAt"
                  label={t("Ends")}
                  placeholder={t("Now")}
                  description={t("Leave blank to end it now.")}
                  clearable
                />
              </FormControl>
            </FormGroup>
          </Form>
        </FormProvider>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            {t("Cancel")}
          </Button>
          <Button
            type="submit"
            form={END_FORM_ID}
            variant="destructive"
            isLoading={mutation.isPending}
          >
            {t("End agreement")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
