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
import {
  AssignPayProfileFields,
  PayOverrideFields,
  useAssignPayProfileForm,
} from "./assign-pay-profile-form";

/** The pay assignment form as a dialog, for the pay profile page. */
export function AssignPayProfileDialog({
  open,
  onOpenChange,
  workerId,
  payProfileId,
  onAssigned,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  workerId?: string;
  payProfileId?: string;
  onAssigned?: () => void;
}) {
  const t = useT();
  const { form, isPending, submit } = useAssignPayProfileForm({
    open,
    start: { workerId, payProfileId },
    onAssigned: () => {
      onOpenChange(false);
      onAssigned?.();
    },
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="lg" className="max-h-[85vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{t("Assign pay profile")}</DialogTitle>
          <DialogDescription>
            {t(
              "The driver's pay is computed from this profile for every shipment delivered on or after the effective date. Any currently-open assignment ends automatically — no cleanup needed.",
            )}
          </DialogDescription>
        </DialogHeader>
        <FormProvider {...form}>
          <Form id="assign-pay-profile-form" onSubmit={(event) => void submit(event)}>
            <AssignPayProfileFields
              form={form}
              showWorker={!workerId}
              showProfile={!payProfileId}
            />
          </Form>
        </FormProvider>
        <PayOverrideFields
          form={form}
          open={open}
          initialProfileId={payProfileId}
          className="rounded-lg border p-3"
          heading={
            <div>
              <p className="text-xs font-medium">{t("Driver-specific rate overrides")}</p>
              <p className="text-xs text-muted-foreground">
                {t(
                  "Leave blank to use the profile rate. An override replaces the component's base rate and any mileage bands for this driver only.",
                )}
              </p>
            </div>
          }
        />
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {t("Cancel")}
          </Button>
          <Button form="assign-pay-profile-form" type="submit" isLoading={isPending}>
            {t("Assign profile")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
