import { useT } from "@trenova/shared/i18n/use-t";
import { UserAutocompleteField } from "@/components/autocomplete-fields";
import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { APPROVAL_DELEGATIONS_KEY, delegateApproval } from "@/lib/graphql/org-structure";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
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
import { getTodayDate } from "@trenova/shared/lib/date";
import { approvalScopeHint, approvalScopeLabel } from "@trenova/shared/lib/org-structure";
import {
  approvalScopeSchema,
  delegationFormSchema,
  type DelegationFormValues,
} from "@trenova/shared/types/org-structure";
import { useEffect } from "react";
import { FormProvider, useForm, useWatch, type Resolver } from "react-hook-form";
import { toast } from "sonner";

const SCOPE_OPTIONS = approvalScopeSchema.options.map((value) => ({
  value,
  label: approvalScopeLabel(value),
  description: approvalScopeHint(value),
}));

export type DelegationDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
};

function emptyDelegation(): DelegationFormValues {
  return {
    delegateId: "",
    scope: "All",
    startsAt: getTodayDate(),
    endsAt: null,
    reason: null,
  };
}

export function DelegationDialog({ open, onOpenChange }: DelegationDialogProps) {
  const t = useT();

  const queryClient = useQueryClient();
  const form = useForm<DelegationFormValues>({
    resolver: zodResolver(delegationFormSchema) as Resolver<DelegationFormValues>,
    defaultValues: emptyDelegation(),
  });
  const { control, handleSubmit, reset } = form;

  useEffect(() => {
    if (!open) return;
    reset(emptyDelegation());
  }, [open, reset]);

  const endsAt = useWatch({ control, name: "endsAt" });

  const { mutateAsync, isPending } = useApiMutation<
    { id: string },
    DelegationFormValues,
    unknown,
    DelegationFormValues
  >({
    form,
    resourceName: "Delegation",
    mutationFn: (values) =>
      delegateApproval({
        delegateId: values.delegateId,
        scope: values.scope,
        startsAt: values.startsAt,
        endsAt: values.endsAt ?? undefined,
        reason: values.reason ?? undefined,
      }),
    onSuccess: () => {
      toast.success(t("Cover arranged"), {
        description:
          t("They can now approve what you can approve, for the people you manage — and nothing beyond that."),
      });
      void queryClient.invalidateQueries({ queryKey: [APPROVAL_DELEGATIONS_KEY] });
      onOpenChange(false);
    },
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{t("Arrange cover")}</DialogTitle>
          <DialogDescription>
            {t("Hand your approvals to somebody else while you are away. Cover widens what they can act on; it never widens what you could approve yourself.")}
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
            <FormGroup className="pb-2" cols={2}>
              <FormControl cols="full">
                <UserAutocompleteField<DelegationFormValues>
                  control={control}
                  name="delegateId"
                  label={t("Who is covering")}
                  placeholder={t("Select a colleague")}
                  description={t("The colleague who approves in your place while the cover runs.")}
                  rules={{ required: true }}
                />
              </FormControl>
              <FormControl cols="full">
                <SelectField<DelegationFormValues>
                  control={control}
                  name="scope"
                  label={t("What they can approve")}
                  options={SCOPE_OPTIONS}
                  placeholder={t("Pick what is covered")}
                  description={t("Limits the cover to one kind of approval, or hands over all of them.")}
                  rules={{ required: true }}
                />
              </FormControl>
              <FormControl>
                <AutoCompleteDateField<DelegationFormValues>
                  control={control}
                  name="startsAt"
                  label={t("From")}
                  placeholder={t("e.g. Today")}
                  description={t("The first day they can approve on your behalf.")}
                  rules={{ required: true }}
                />
              </FormControl>
              <FormControl>
                <AutoCompleteDateField<DelegationFormValues>
                  control={control}
                  name="endsAt"
                  label={t("Until")}
                  placeholder={t("e.g. Next Friday")}
                  description={t("The last day of cover; approvals come back to you after it.")}
                />
              </FormControl>
              {endsAt ? null : (
                <FormControl cols="full">
                  <Alert variant="warning">
                    <AlertDescription>
                      {t("With no end date this runs until you call it back. Set one if you are covering a specific absence.")}
                    </AlertDescription>
                  </Alert>
                </FormControl>
              )}
              <FormControl cols="full">
                <InputField<DelegationFormValues>
                  control={control}
                  name="reason"
                  label={t("Why")}
                  placeholder={t("e.g. Annual leave")}
                  description={t("Kept with the delegation so an approval made under it can be explained.")}
                />
              </FormControl>
            </FormGroup>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                {t("Cancel")}
              </Button>
              <Button type="submit" isLoading={isPending}>
                {t("Arrange")}
              </Button>
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
