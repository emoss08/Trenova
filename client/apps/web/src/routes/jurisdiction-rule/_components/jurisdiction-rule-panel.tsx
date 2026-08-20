import { ComponentLoader } from "@trenova/shared/components/component-loader";
import { DataTablePanelContainer } from "@/components/data-table/data-table-panel";
import { FormCreatePanel } from "@/components/form-create-panel";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Form } from "@trenova/shared/components/ui/form";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { api } from "@trenova/shared/lib/api";
import { formatToUserTimezone } from "@trenova/shared/lib/date";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import { jurisdictionRuleSchema, type JurisdictionRule } from "@/types/jurisdiction-rule";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { BadgeCheckIcon } from "lucide-react";
import { useEffect, useState } from "react";
import { FormProvider, type Resolver, useForm } from "react-hook-form";
import { toast } from "sonner";
import { JurisdictionRuleForm } from "./jurisdiction-rule-form";
import { JurisdictionRuleVerifyDialog } from "./jurisdiction-rule-verify-dialog";

const QUERY_KEY = "jurisdiction-rule-list";

/**
 * A blank rule seeded with the federal baseline.
 *
 * These are the two limits that genuinely do not vary — 102in width and the
 * 80,000lb Interstate gross — so they are a reasonable starting point rather
 * than an empty form. Everything else is left for the editor to fill from the
 * statute, and the row lands unverified regardless.
 */
function emptyRule(): Partial<JurisdictionRule> {
  return {
    stateId: "",
    status: "Active",
    maxWidthFeet: 8.5,
    maxHeightFeet: 13.5,
    maxLengthFeet: 53,
    maxWeightPounds: 80000,
    permitLeadTimeDays: 1,
    permitValidityDays: 5,
    daylightOnly: false,
    rushHourRestricted: false,
    weekendRestricted: false,
    holidayRestricted: true,
    verificationState: "Unverified",
    sourceNote: "",
    sourceUrl: "",
  };
}

export function JurisdictionRulePanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<JurisdictionRule>) {
  const form = useForm<JurisdictionRule>({
    resolver: zodResolver(jurisdictionRuleSchema) as Resolver<JurisdictionRule>,
    defaultValues: emptyRule() as JurisdictionRule,
  });

  if (mode === "edit") {
    return (
      <JurisdictionRuleEditPanel open={open} onOpenChange={onOpenChange} row={row} form={form} />
    );
  }

  return (
    <FormCreatePanel
      open={open}
      onOpenChange={onOpenChange}
      form={form}
      url="/jurisdiction-rules/"
      queryKey={QUERY_KEY}
      title="Jurisdiction Rule"
      formComponent={<JurisdictionRuleForm />}
    />
  );
}

function JurisdictionRuleEditPanel({
  open,
  onOpenChange,
  row,
  form,
}: Pick<DataTablePanelProps<JurisdictionRule>, "open" | "onOpenChange" | "row"> & {
  form: ReturnType<typeof useForm<JurisdictionRule>>;
}) {
  const queryClient = useQueryClient();
  const [verifyOpen, setVerifyOpen] = useState(false);

  const {
    formState: { isSubmitting },
    handleSubmit,
    reset,
  } = form;

  useEffect(() => {
    if (open && row) {
      reset(row as JurisdictionRule, { keepDefaultValues: true });
    }
  }, [open, row, reset]);

  const { mutateAsync } = useApiMutation<
    JurisdictionRule,
    JurisdictionRule,
    unknown,
    JurisdictionRule
  >({
    mutationFn: async (values) =>
      api.put<JurisdictionRule>(`/jurisdiction-rules/${row?.id}/`, values),
    onSuccess: (updated) => {
      // The server clears verification whenever a limit changes, so say so
      // rather than letting the badge quietly flip in the table behind them.
      const clearedVerification =
        row?.verificationState !== "Unverified" && updated?.verificationState === "Unverified";

      toast.success("Changes have been saved", {
        description: clearedVerification
          ? "A limit changed, so this rule is unverified again and needs re-checking."
          : "Jurisdiction rule updated successfully",
      });
      void queryClient.invalidateQueries({ queryKey: [QUERY_KEY] });
    },
    form,
    resourceName: "Jurisdiction Rule",
  });

  const verificationLabel = row?.verifiedAt
    ? `${row.verificationState} on ${formatToUserTimezone(row.verifiedAt, {
        showTime: false,
        showTimeZone: false,
      })}`
    : (row?.verificationState ?? "Unverified");

  return (
    <>
      <DataTablePanelContainer
        open={open}
        onOpenChange={onOpenChange}
        title={row?.state?.name ?? "Jurisdiction Rule"}
        description={verificationLabel}
        footer={
          <>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              Cancel
            </Button>
            <Button type="submit" form="panel-edit-form" disabled={isSubmitting}>
              {isSubmitting ? "Saving..." : "Save"}
            </Button>
          </>
        }
      >
        {!row ? (
          <ComponentLoader message="Loading Jurisdiction Rule..." />
        ) : (
          <>
            <div className="mb-4 flex items-center justify-between gap-3 rounded-lg border px-4 py-3">
              <div className="space-y-0.5">
                <div className="flex items-center gap-2">
                  <span className="text-xs font-medium">Verification</span>
                  <Badge
                    variant={
                      row.verificationState === "Verified"
                        ? "active"
                        : row.verificationState === "Disputed"
                          ? "warning"
                          : "inactive"
                    }
                  >
                    {row.verificationState}
                  </Badge>
                </div>
                <p className="text-muted-foreground text-xs">
                  {row.sourceNote || "No source recorded for these limits."}
                </p>
              </div>
              <Button type="button" variant="outline" size="sm" onClick={() => setVerifyOpen(true)}>
                <BadgeCheckIcon className="size-3.5" />
                Verify
              </Button>
            </div>

            <FormProvider {...form}>
              <Form
                id="panel-edit-form"
                onSubmit={handleSubmit(async (values) => {
                  await mutateAsync(values);
                })}
              >
                <JurisdictionRuleForm />
              </Form>
            </FormProvider>
          </>
        )}
      </DataTablePanelContainer>

      <JurisdictionRuleVerifyDialog
        open={verifyOpen}
        onOpenChange={setVerifyOpen}
        rule={row ?? null}
      />
    </>
  );
}
