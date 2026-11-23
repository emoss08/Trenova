import {
  AssignPayProfileFields,
  PayOverrideFields,
  useAssignPayProfileForm,
} from "@/components/pay/assign-pay-profile-form";
import {
  WorkPane,
  WorkPaneBody,
  WorkPaneCancel,
  WorkPaneDirtyTracker,
  WorkPaneFooter,
} from "@/components/work-pane";
import type { EffectiveWorkerPayAssignment } from "@/lib/graphql/driver-settlement";
import { Button } from "@trenova/shared/components/ui/button";
import { Form } from "@trenova/shared/components/ui/form";
import { useT } from "@trenova/shared/i18n/use-t";
import { FormProvider } from "react-hook-form";
import { overrideStart } from "../pay-model";

export type ChangeProfilePaneProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  workerId: string;
  workerName: string;
  /** The assignment in force; null when the driver has none. */
  current: EffectiveWorkerPayAssignment | null;
  onSaved: () => void;
};

/**
 * Starts a new pay assignment beside the pay profile: the profile, when it starts, the split, and
 * this driver's own rate for each component. While the current profile stays chosen its overrides
 * are offered again; another profile starts with none.
 */
export function ChangeProfilePane({
  open,
  onOpenChange,
  workerId,
  workerName,
  current,
  onSaved,
}: ChangeProfilePaneProps) {
  const t = useT();
  const initialOverrides = overrideStart(current);
  const { form, isPending, submit } = useAssignPayProfileForm({
    open,
    start: {
      workerId,
      payProfileId: current?.payProfileId ?? undefined,
      overrides: initialOverrides,
    },
    onAssigned: onSaved,
  });

  return (
    <WorkPane
      open={open}
      onOpenChange={onOpenChange}
      title={current ? t("Change pay profile") : t("Assign pay profile")}
      description={t(
        "Starts a new assignment from the effective date. Pay events before it keep the old rates.",
      )}
    >
      <WorkPaneDirtyTracker form={form} />
      <FormProvider {...form}>
        <Form onSubmit={(event) => void submit(event)}>
          <WorkPaneBody className="text-sm">
            <AssignPayProfileFields form={form} showWorker={false} showProfile />
            <PayOverrideFields
              form={form}
              open={open}
              initialProfileId={current?.payProfileId ?? undefined}
              initialOverrides={initialOverrides}
              heading={
                <div className="flex flex-col gap-0.5">
                  <h4 className="text-xs font-medium text-foreground-subtle">
                    {t("Rates for {0}", workerName)}
                  </h4>
                  <p className="text-2xs text-muted-foreground">
                    {t(
                      "Leave a rate empty to pay the profile's own. A rate you fill in is an override and shows on the pay profile.",
                    )}
                  </p>
                </div>
              }
            />
          </WorkPaneBody>
          <WorkPaneFooter>
            <WorkPaneCancel />
            <Button type="submit" size="sm" isLoading={isPending} loadingText={t("Saving...")}>
              {current ? t("Change profile") : t("Assign profile")}
            </Button>
          </WorkPaneFooter>
        </Form>
      </FormProvider>
    </WorkPane>
  );
}
