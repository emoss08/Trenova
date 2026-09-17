import { useT } from "@trenova/shared/i18n/use-t";
import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  CARRIER_INTEL_NOTE_MAX_LENGTH,
  reviewNoteFormSchema,
  type ReviewNoteFormValues,
} from "@/lib/carrier-intelligence";
import { markCarrierIntelReviewed } from "@/lib/graphql/carrier-intelligence";
import { zodResolver } from "@hookform/resolvers/zod";
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

export type MarkReviewedDialogProps = {
  carrierId: string;
  blockingCount: number;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onReviewed: () => void;
};

const DEFAULT_VALUES: ReviewNoteFormValues = { note: "" };

export function MarkReviewedDialog({
  carrierId,
  blockingCount,
  open,
  onOpenChange,
  onReviewed,
}: MarkReviewedDialogProps) {
  const t = useT();

  const form = useForm<ReviewNoteFormValues>({
    resolver: zodResolver(reviewNoteFormSchema) as Resolver<ReviewNoteFormValues>,
    defaultValues: DEFAULT_VALUES,
  });
  const { control, handleSubmit, reset } = form;

  useEffect(() => {
    if (open) {
      reset(DEFAULT_VALUES);
    }
  }, [open, reset]);

  const { mutateAsync, isPending } = useApiMutation<
    boolean,
    ReviewNoteFormValues,
    unknown,
    ReviewNoteFormValues
  >({
    form,
    resourceName: "Carrier intelligence review",
    mutationFn: (values) => markCarrierIntelReviewed(carrierId, values.note),
    onSuccess: () => {
      toast.success(t("Carrier marked reviewed"));
      onReviewed();
      onOpenChange(false);
    },
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{t("Mark reviewed")}</DialogTitle>
          <DialogDescription>
            {blockingCount > 0
              ? t(
                  "Records that someone looked at the latest vetting. It does not clear the {0, plural, one {# blocker} other {# blockers}}; those still need an override or a fix.",
                  blockingCount,
                )
              : t("Records that someone looked at the latest vetting and signed off on it.")}
          </DialogDescription>
        </DialogHeader>
        <FormProvider {...form}>
          <Form
            onSubmit={(submitEvent) => {
              submitEvent.preventDefault();
              submitEvent.stopPropagation();
              void handleSubmit((values) => mutateAsync(values))(submitEvent);
            }}
          >
            <FormGroup cols={1} className="pb-2">
              <FormControl>
                <TextareaField<ReviewNoteFormValues>
                  control={control}
                  name="note"
                  label={t("Review note")}
                  placeholder={t("What was checked and what was concluded")}
                  rules={{ required: true }}
                  maxLength={CARRIER_INTEL_NOTE_MAX_LENGTH}
                />
              </FormControl>
            </FormGroup>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                {t("Cancel")}
              </Button>
              <Button type="submit" isLoading={isPending}>
                {t("Mark reviewed")}
              </Button>
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
