import { useT } from "@trenova/shared/i18n/use-t";
import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { reopenIftaReturn, type IftaReturn } from "@/lib/graphql/ifta-return";
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

const REOPEN_REASON_MIN = 10;
const REOPEN_REASON_MAX = 500;

export const reopenReturnSchema = z.object({
  reason: z
    .string()
    .trim()
    .min(REOPEN_REASON_MIN, {
      message: "Give at least ten characters saying why the return is being reopened.",
    })
    .max(REOPEN_REASON_MAX, { message: "Keep the reason under 500 characters." }),
});

export type ReopenReturnValues = z.infer<typeof reopenReturnSchema>;

type ReopenReturnDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  ret: IftaReturnView;
  period: IftaPeriodKey;
};

export function ReopenReturnDialog({ open, onOpenChange, ret, period }: ReopenReturnDialogProps) {
  const t = useT();

  const queryClient = useQueryClient();
  const form = useForm<ReopenReturnValues>({
    resolver: zodResolver(reopenReturnSchema) as Resolver<ReopenReturnValues>,
    defaultValues: { reason: "" },
  });
  const { control, handleSubmit, reset } = form;

  useEffect(() => {
    if (open) reset({ reason: "" });
  }, [open, reset]);

  const { mutateAsync, isPending } = useApiMutation<
    IftaReturn,
    ReopenReturnValues,
    unknown,
    ReopenReturnValues
  >({
    form,
    resourceName: "IFTA Return",
    mutationFn: (values) =>
      reopenIftaReturn({ id: ret.id, version: ret.version, reason: values.reason.trim() }),
    onSuccess: async () => {
      toast.success(t("Return reopened"), {
        description: t(
          "It is a draft again and recomputes with the data on file. The reason is kept with the return.",
        ),
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
              <DialogTitle>{t("Reopen the {0} return?", quarterLabel(period))}</DialogTitle>
              <DialogDescription>
                {t(
                  "The worksheet unlocks and its figures move with the data again, so anything already reported to the base jurisdiction can drift from what is on file. The reason is kept with the return and shown in its audit trail.",
                )}
              </DialogDescription>
            </DialogHeader>
            <FormGroup cols={1} className="mt-4">
              <FormControl cols="full">
                <TextareaField
                  control={control}
                  name="reason"
                  label={t("Reason")}
                  placeholder={t("e.g. Oklahoma published its Q2 rate after we finalized")}
                  rules={{ required: true }}
                  maxLength={REOPEN_REASON_MAX}
                  description={t(
                    "Between 10 and 500 characters, kept with the return as the record of why it was unlocked.",
                  )}
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
                {t("Keep it finalized")}
              </Button>
              <Button type="submit" disabled={isPending}>
                {isPending ? t("Reopening...") : t("Reopen return")}
              </Button>
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
