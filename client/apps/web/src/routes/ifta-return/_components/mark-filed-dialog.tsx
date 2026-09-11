import { useT } from "@trenova/shared/i18n/use-t";
import { DateField } from "@/components/fields/date-field/date-field";
import { InputField } from "@/components/fields/input-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { markIftaReturnFiled, type IftaReturn } from "@/lib/graphql/ifta-return";
import { quarterLabel, type IftaPeriodKey, type IftaReturnView } from "@/lib/ifta-return";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { getEndOfDay, getTodayDate } from "@trenova/shared/lib/date";
import { useEffect, useMemo } from "react";
import { FormProvider, useForm, type Resolver } from "react-hook-form";
import { toast } from "sonner";
import { z } from "zod";
import { invalidateIftaReturn, invalidateOnVersionMismatch } from "./queries";

const FILING_REFERENCE_MAX = 100;

/**
 * The filing cannot predate the moment the worksheet was locked, and a return
 * cannot be filed in the future, so both bounds come from the return itself
 * rather than from anything the form can be talked into.
 */
export function markFiledSchema(finalizedAt: number | null, latestFiledAt: number) {
  return z.object({
    filedAt: z
      .number({ message: "Choose the date the return was filed." })
      .min(finalizedAt ?? 0, {
        message: "A return cannot be filed before it was finalized.",
      })
      .max(latestFiledAt, { message: "A return cannot be filed in the future." }),
    filingReference: z
      .string()
      .nullable()
      .transform((value) => {
        const trimmed = value?.trim() ?? "";
        return trimmed === "" ? null : trimmed;
      })
      .pipe(
        z
          .string()
          .max(FILING_REFERENCE_MAX, {
            message: `Keep the reference under ${FILING_REFERENCE_MAX} characters.`,
          })
          .nullable(),
      ),
  });
}

export type MarkFiledValues = { filedAt: number; filingReference: string | null };

type MarkFiledDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  ret: IftaReturnView;
  period: IftaPeriodKey;
};

export function MarkFiledDialog({ open, onOpenChange, ret, period }: MarkFiledDialogProps) {
  const t = useT();

  const queryClient = useQueryClient();
  const schema = useMemo(
    () => markFiledSchema(ret.finalizedAt ?? null, getEndOfDay()),
    [ret.finalizedAt],
  );
  const form = useForm<MarkFiledValues>({
    resolver: zodResolver(schema) as Resolver<MarkFiledValues>,
    defaultValues: { filedAt: getTodayDate(), filingReference: null },
  });
  const { control, handleSubmit, reset } = form;

  useEffect(() => {
    if (open) reset({ filedAt: getTodayDate(), filingReference: null });
  }, [open, reset]);

  const { mutateAsync, isPending } = useApiMutation<
    IftaReturn,
    MarkFiledValues,
    unknown,
    MarkFiledValues
  >({
    form,
    resourceName: "IFTA Return",
    mutationFn: (values) =>
      markIftaReturnFiled({
        id: ret.id,
        version: ret.version,
        filedAt: values.filedAt,
        filingReference: values.filingReference,
      }),
    onSuccess: async () => {
      toast.success(t("Return marked filed"), {
        description:
          t("It is immutable now. A correction opens a new draft through an amendment, leaving this one as filed."),
      });
      await invalidateIftaReturn(queryClient, period);
      onOpenChange(false);
    },
    onError: (error) => invalidateOnVersionMismatch(error, queryClient, period),
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <FormProvider {...form}>
          <Form
            onSubmit={(event) => {
              event.preventDefault();
              event.stopPropagation();
              void handleSubmit((values) => mutateAsync(values))(event);
            }}
          >
            <DialogHeader>
              <DialogTitle>{t("Mark the {0} return filed", quarterLabel(period))}</DialogTitle>
              <DialogDescription>
                {t("This records that the finalized worksheet went to the base jurisdiction. The return becomes immutable: a correction opens a new draft as an amendment and leaves this one as it was filed.")}
              </DialogDescription>
            </DialogHeader>
            <FormGroup cols={2} className="mt-4">
              <FormControl>
                <DateField
                  control={control}
                  name="filedAt"
                  label={t("Filed on")}
                  placeholder={t("Pick the filing date")}
                  description={t("Between the day the return was finalized and today.")}
                />
              </FormControl>
              <FormControl>
                <InputField
                  control={control}
                  name="filingReference"
                  label={t("Filing reference")}
                  placeholder={t("e.g. TX-2026Q2-88213")}
                  maxLength={FILING_REFERENCE_MAX}
                  description={t("Optional. The confirmation number the jurisdiction gave you.")}
                />
              </FormControl>
            </FormGroup>
            <DialogFooter className="mt-4">
              <Button
                type="button"
                variant="outline"
                onClick={() => onOpenChange(false)}
                disabled={isPending}
              >
                {t("Cancel")}
              </Button>
              <Button type="submit" disabled={isPending}>
                {isPending ? "Recording..." : "Mark filed"}
              </Button>
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
