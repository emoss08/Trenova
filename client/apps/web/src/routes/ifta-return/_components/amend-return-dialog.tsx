import { useT } from "@trenova/shared/i18n/use-t";
import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { amendIftaReturn, type IftaReturn } from "@/lib/graphql/ifta-return";
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
import { useEffect } from "react";
import { FormProvider, useForm, type Resolver } from "react-hook-form";
import { toast } from "sonner";
import { z } from "zod";
import { invalidateIftaReturn, invalidateOnVersionMismatch } from "./queries";

const AMEND_REASON_MIN = 10;
const AMEND_REASON_MAX = 500;

export const amendReturnSchema = z.object({
  reason: z
    .string()
    .trim()
    .min(AMEND_REASON_MIN, {
      message: "Give at least ten characters saying what the amendment corrects.",
    })
    .max(AMEND_REASON_MAX, { message: "Keep the reason under 500 characters." }),
});

export type AmendReturnValues = z.infer<typeof amendReturnSchema>;

type AmendReturnDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  ret: IftaReturnView;
  period: IftaPeriodKey;
};

export function AmendReturnDialog({ open, onOpenChange, ret, period }: AmendReturnDialogProps) {
  const t = useT();

  const queryClient = useQueryClient();
  const form = useForm<AmendReturnValues>({
    resolver: zodResolver(amendReturnSchema) as Resolver<AmendReturnValues>,
    defaultValues: { reason: "" },
  });
  const { control, handleSubmit, reset } = form;

  useEffect(() => {
    if (open) reset({ reason: "" });
  }, [open, reset]);

  const { mutateAsync, isPending } = useApiMutation<
    IftaReturn,
    AmendReturnValues,
    unknown,
    AmendReturnValues
  >({
    form,
    resourceName: "IFTA Return",
    mutationFn: (values) => amendIftaReturn({ id: ret.id, reason: values.reason.trim() }),
    onSuccess: async (created) => {
      toast.success(`Amendment ${created.amendmentNumber} opened`, {
        description:
          t("It is a fresh draft for the same quarter. The filed return is left exactly as it was."),
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
              <DialogTitle>{t("Amend the {0} return", quarterLabel(period))}</DialogTitle>
              <DialogDescription>
                {t("A filed return is never edited. This opens amendment {0} as a new draft for the same quarter, computed from the miles, fuel and rates on file now; the filed return stays exactly as it was submitted.", ret.amendmentNumber + 1)}
              </DialogDescription>
            </DialogHeader>
            <FormGroup cols={1} className="mt-4">
              <FormControl cols="full">
                <TextareaField
                  control={control}
                  name="reason"
                  label={t("Reason")}
                  placeholder={t("e.g. Two Oklahoma fuel receipts arrived after the filing")}
                  rules={{ required: true }}
                  maxLength={AMEND_REASON_MAX}
                  description={t("Between 10 and 500 characters, kept with the amendment as the record of why it exists.")}
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
                {isPending ? t("Opening...") : t("Open the amendment")}
              </Button>
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
