import { InjuryFormFields, useInjuryForm } from "@/components/injury/injury-form";
import {
  WorkPane,
  WorkPaneBody,
  WorkPaneCancel,
  WorkPaneDirtyTracker,
  WorkPaneFooter,
} from "@/components/work-pane";
import type { WorkerInjuryRow } from "@/lib/graphql/worker-injury";
import { Button } from "@trenova/shared/components/ui/button";
import { Form } from "@trenova/shared/components/ui/form";
import { useT } from "@trenova/shared/i18n/use-t";
import { FormProvider } from "react-hook-form";

export type InjuryPaneProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  workerId: string;
  /** The case being edited; null records a new one. */
  injury: WorkerInjuryRow | null;
  onSaved: () => void;
};

/**
 * The injury case form in the work pane, beside the worker's safety record. The page mounts it
 * fresh at each opening (`lazyWorkPane`), so the form starts from the case it was opened on.
 */
export function InjuryPane({ open, onOpenChange, workerId, injury, onSaved }: InjuryPaneProps) {
  const t = useT();
  const { form, isEdit, isPending, submit } = useInjuryForm({
    workerId,
    injury,
    onSaved,
  });

  return (
    <WorkPane
      open={open}
      onOpenChange={onOpenChange}
      title={isEdit ? t("Edit case") : t("Record an injury or illness")}
      description={t(
        "Every case is kept, recordable or not. The decision not to record one is itself worth a record.",
      )}
      contentKey={injury?.id ?? "new"}
    >
      <WorkPaneDirtyTracker form={form} />
      <FormProvider {...form}>
        <Form onSubmit={submit}>
          <WorkPaneBody className="text-sm">
            <InjuryFormFields form={form} isEdit={isEdit} />
          </WorkPaneBody>
          <WorkPaneFooter>
            <WorkPaneCancel />
            <Button type="submit" size="sm" isLoading={isPending}>
              {isEdit ? t("Save changes") : t("Record case")}
            </Button>
          </WorkPaneFooter>
        </Form>
      </FormProvider>
    </WorkPane>
  );
}
