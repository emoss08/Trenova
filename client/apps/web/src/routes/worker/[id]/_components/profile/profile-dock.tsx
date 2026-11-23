import { FormSaveDock } from "@/components/form-save-dock";
import {
  FormFieldsProvider,
  useFormFieldRegistry,
  type FieldRegistry,
} from "@trenova/shared/lib/form-field-registry";
import { cn } from "@trenova/shared/lib/utils";
import type { Worker } from "@trenova/shared/types/worker";
import { useEffect } from "react";
import { FormProvider, useFormContext, useFormState, type UseFormReturn } from "react-hook-form";

/** What the page needs to draw the Profile form's save dock outside the Profile tab. */
export type ProfileDock = {
  form: UseFormReturn<Worker>;
  registry: FieldRegistry;
  /** The Profile `<form>`'s id, so the dock's Save submits it from any tab. */
  formId: string;
};

/** The Profile form's id on the page; one worker's page holds one Profile form. */
export function profileFormId(workerId: string): string {
  return `worker-profile-form-${workerId}`;
}

/**
 * Sits inside the Profile `<Form>` and hands the page its form and field registry. It does not
 * clear them when its effects are cleaned up, because a hidden tab (`Activity`) cleans up its
 * effects too, and the dock must stay while the person is on another tab. The page holds the dock
 * in state keyed with the worker, so another worker's page starts without one.
 */
export function ProfileDockReporter({
  formId,
  onDock,
}: {
  formId: string;
  onDock: (dock: ProfileDock) => void;
}) {
  const form = useFormContext<Worker>();
  const registry = useFormFieldRegistry();
  useEffect(() => {
    if (registry) onDock({ form, registry, formId });
  }, [form, registry, formId, onDock]);
  return null;
}

function DirtyReporter({ onDirtyChange }: { onDirtyChange: (dirty: boolean) => void }) {
  const { isDirty, isSubmitting } = useFormState<Worker>();
  const unsaved = isDirty && !isSubmitting;
  useEffect(() => {
    onDirtyChange(unsaved);
  }, [unsaved, onDirtyChange]);
  useEffect(() => () => onDirtyChange(false), [onDirtyChange]);
  return null;
}

/**
 * The Profile form's save dock, drawn by the page beside the work pane host rather than inside
 * the Profile tab: the host is a containing block, so a fixed dock inside it would sit at the end
 * of the tab instead of the viewport, and a hidden tab would hide it. Drawn here, it stays in view
 * however the page scrolls and follows the person to any tab while Profile holds changes; its
 * spacer makes room at the end of whatever tab is showing. It also tells the page whether Profile
 * holds changes, for the tab's dot and the unsaved-changes guard, since the hidden tab's own
 * effects do not run.
 */
export function WorkerProfileDock({
  dock,
  onDirtyChange,
  lifted,
}: {
  dock: ProfileDock;
  onDirtyChange: (dirty: boolean) => void;
  /** A work pane is open: rise above its footer so the pane's buttons stay reachable. */
  lifted: boolean;
}) {
  return (
    <FormProvider {...dock.form}>
      <FormFieldsProvider registry={dock.registry}>
        <DirtyReporter onDirtyChange={onDirtyChange} />
        <FormSaveDock formId={dock.formId} className={cn(lifted && "bottom-18")} />
      </FormFieldsProvider>
    </FormProvider>
  );
}
