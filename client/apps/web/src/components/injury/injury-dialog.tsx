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

/** The injury case form as a dialog, for the OSHA log, which has no work pane. */
export function InjuryDialog({
  open,
  onOpenChange,
  workerId,
  injury,
  safetyEventId = null,
}: InjuryDialogProps) {
  const t = useT();
  const { form, isEdit, isPending, submit } = useInjuryForm({
    open,
    workerId,
    injury,
    safetyEventId,
    onSaved: () => onOpenChange(false),
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="lg">
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
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                {t("Cancel")}
              </Button>
              <Button type="submit" isLoading={isPending}>
                {isEdit ? t("Save changes") : t("Record case")}
              </Button>
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
