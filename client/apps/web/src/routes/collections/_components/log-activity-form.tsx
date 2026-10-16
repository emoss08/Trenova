import { useT } from "@trenova/shared/i18n/use-t";
import { defineLabels } from "@trenova/shared/i18n/labels";
import { translate } from "@trenova/shared/i18n/runtime";
import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { InputField } from "@/components/fields/input-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { Button } from "@trenova/shared/components/ui/button";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { parseAmountToMinor } from "@trenova/shared/lib/money";
import { apiService } from "@/services/api";
import { contactActivityKinds, type ContactActivityKind } from "@/types/collection";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { useForm, useWatch } from "react-hook-form";
import { z } from "zod";

export const logActivitySchema = z
  .object({
    kind: z.enum(contactActivityKinds),
    contactName: z.string().max(200),
    summary: z.string().max(4000),
    promisedAmount: z.string(),
    promisedOn: z.number().int().nullable(),
    followUpOn: z.number().int().nullable(),
  })
  .superRefine((value, ctx) => {
    if (value.kind === "PromiseToPay") {
      const minor = parseAmountToMinor(value.promisedAmount);
      if (minor === null || minor <= 0) {
        ctx.addIssue({
          code: "custom",
          path: ["promisedAmount"],
          message: translate("Enter the promised amount, with at most two decimal places"),
        });
      }
      if (value.promisedOn === null) {
        ctx.addIssue({
          code: "custom",
          path: ["promisedOn"],
          message: translate("Enter the date the customer promised to pay by"),
        });
      }
    } else if (value.summary.trim() === "") {
      ctx.addIssue({ code: "custom", path: ["summary"], message: translate("Say what happened") });
    }
  });

export type LogActivityValues = z.infer<typeof logActivitySchema>;

const EMPTY: LogActivityValues = {
  kind: "Call",
  contactName: "",
  summary: "",
  promisedAmount: "",
  promisedOn: null,
  followUpOn: null,
};

const KIND_LABEL: Record<ContactActivityKind, string> = defineLabels({
  Call: "Call",
  Email: "Email",
  Note: "Note",
  PromiseToPay: "Promise to pay",
});

/**
 * Records a contact with the customer. A promise to pay takes the amount and
 * the date; the account then waits on it, and its follow-up moves to the day
 * after unless another date is chosen here.
 */
export function LogActivityForm({ customerId }: { customerId: string }) {
  const t = useT();
  const queryClient = useQueryClient();
  const form = useForm<LogActivityValues>({
    resolver: zodResolver(logActivitySchema),
    defaultValues: EMPTY,
  });
  const kind = useWatch({ control: form.control, name: "kind" });
  const isPromise = kind === "PromiseToPay";

  const { mutateAsync, isPending } = useApiMutation({
    mutationFn: (values: LogActivityValues) =>
      apiService.collectionService.logActivity(customerId, {
        kind: values.kind,
        summary: values.summary.trim(),
        contactName: values.contactName.trim(),
        invoiceId: null,
        promisedAmount: values.kind === "PromiseToPay" ? values.promisedAmount.trim() : null,
        promisedOn: values.kind === "PromiseToPay" ? values.promisedOn : null,
        followUpOn: values.followUpOn,
      }),
    resourceName: "CollectionActivity",
    form,
    onSuccess: (view) => {
      queryClient.setQueryData(["collection", "account", customerId], view);
      void queryClient.invalidateQueries({ queryKey: ["collection"] });
      form.reset({ ...EMPTY, kind: form.getValues("kind") });
    },
  });

  return (
    <Form
      onSubmit={form.handleSubmit((values) => mutateAsync(values))}
      className="flex flex-col gap-3"
    >
      <SegmentedControl
        aria-label={t("Kind of contact")}
        fullWidth
        value={kind}
        onValueChange={(value) => form.setValue("kind", value as ContactActivityKind)}
        items={contactActivityKinds.map((value) => ({ value, label: KIND_LABEL[value] }))}
      />
      <FormGroup cols={2}>
        <FormControl>
          <InputField
            control={form.control}
            name="contactName"
            label={t("Spoke with")}
            placeholder={t("e.g., Dana in payables")}
          />
        </FormControl>
        <FormControl>
          <AutoCompleteDateField
            control={form.control}
            name="followUpOn"
            label={t("Follow up on")}
            placeholder={isPromise ? t("Day after the promise") : t("Not scheduled")}
            clearable
          />
        </FormControl>
        {isPromise ? (
          <>
            <FormControl>
              <InputField
                control={form.control}
                name="promisedAmount"
                label={t("Promised amount")}
                placeholder="0.00"
                inputMode="decimal"
                rules={{ required: true }}
              />
            </FormControl>
            <FormControl>
              <AutoCompleteDateField
                control={form.control}
                name="promisedOn"
                label={t("Will pay by")}
                placeholder={t("Choose a date")}
                rules={{ required: true }}
              />
            </FormControl>
          </>
        ) : null}
        <FormControl cols="full">
          <TextareaField
            control={form.control}
            name="summary"
            label={isPromise ? t("Notes") : t("What happened")}
            placeholder={
              isPromise
                ? t("e.g., Paying INV-1042 and INV-1051 by ACH")
                : t("e.g., Left a voicemail; their AP runs on Thursdays")
            }
            rows={3}
          />
        </FormControl>
      </FormGroup>
      <div className="flex justify-end">
        <Button type="submit" isLoading={isPending} loadingText={t("Saving...")}>
          {isPromise ? t("Record promise") : t("Log it")}
        </Button>
      </div>
    </Form>
  );
}
