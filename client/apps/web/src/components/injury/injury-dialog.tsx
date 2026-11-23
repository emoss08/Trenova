import type { WorkerInjuryRow } from "@/lib/graphql/worker-injury";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Form } from "@trenova/shared/components/ui/form";
import { useT } from "@trenova/shared/i18n/use-t";
import { FormProvider } from "react-hook-form";
import { InjuryFormFields, useInjuryForm } from "./injury-form";

export type InjuryDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  workerId: string;
  injury: WorkerInjuryRow | null;
  /** Set when the case is opened from a safety event. */
  safetyEventId?: string | null;
};

/**
 * The injury case form as a dialog, for the OSHA log, which has no work pane. The form lives
 * inside the dialog's content, which is mounted only while it is open, so each opening starts
 * from the case it was opened on.
 */
export function InjuryDialog({ open, onOpenChange, ...props }: InjuryDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="lg">
        <InjuryDialogForm {...props} onClose={() => onOpenChange(false)} />
      </DialogContent>
    </Dialog>
  );
}

function InjuryDialogForm({
  workerId,
  injury,
  safetyEventId = null,
  onClose,
}: Omit<InjuryDialogProps, "open" | "onOpenChange"> & { onClose: () => void }) {
  const t = useT();
  const { form, isEdit, isPending, submit } = useInjuryForm({
    workerId,
    injury,
    safetyEventId,
    onSaved: onClose,
  });

  return (
    <>
      <DialogHeader>
        <DialogTitle>{isEdit ? t("Edit case") : t("Record an injury or illness")}</DialogTitle>
        <DialogDescription>
          {t(
            "Every case is kept, recordable or not. The decision not to record one is itself worth a record.",
          )}
        </DialogDescription>
      </DialogHeader>
      <FormProvider {...form}>
        <Form className="flex flex-col gap-4" onSubmit={submit}>
          <InjuryFormFields form={form} isEdit={isEdit} />
          <DialogFooter>
            <Button type="button" size="sm" variant="outline" onClick={onClose}>
              {t("Cancel")}
            </Button>
            <Button type="submit" size="sm" isLoading={isPending}>
              {isEdit ? t("Save changes") : t("Record case")}
            </Button>
          </DialogFooter>
        </Form>
      </FormProvider>
    </>
  );
}
