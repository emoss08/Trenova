import { PageLayout } from "@/components/navigation/sidebar-layout";
import { UnsavedChangesGuard } from "@/components/unsaved-changes-guard";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { createFactoringCompany } from "@/lib/graphql/factoring-company";
import { factoringCompanySchema, type FactoringCompany } from "@/types/factoring-company";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { Form } from "@trenova/shared/components/ui/form";
import { useT } from "@trenova/shared/i18n/use-t";
import { useState } from "react";
import { FormProvider, useForm, type Resolver } from "react-hook-form";
import { Link, useNavigate } from "react-router";
import { FactoringCompanyEditor } from "../_components/factoring-company-editor";
import type { FactorState } from "../_components/factoring-invoice-preview";
import { FACTORING_COMPANIES_PATH, factoringCompanyPath } from "../use-factoring-company-state";

const FORM_ID = "new-factoring-company-form";

const EMPTY: FactoringCompany = {
  status: "Active",
  code: "",
  name: "",
  addressLine1: "",
  addressLine2: "",
  city: "",
  stateId: "",
  postalCode: "",
  email: "",
  phoneNumber: "",
  remittanceInstructions: "",
  noticeOfAssignment: "",
  notes: "",
  nonRecourse: false,
};

/**
 * A new factor: its name, remit-to, contact and invoice wording beside the invoice they print.
 * Once created it opens on its own page, which walks through the agreement, delivery and notices
 * it needs before it can take invoices.
 */
export function FactoringCompanyCreatePage() {
  const t = useT();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [state, setState] = useState<FactorState>({ abbreviation: null, countryName: null });
  const form = useForm<FactoringCompany>({
    resolver: zodResolver(factoringCompanySchema) as Resolver<FactoringCompany>,
    defaultValues: EMPTY,
  });
  const {
    control,
    handleSubmit,
    reset,
    formState: { isDirty },
  } = form;

  const mutation = useApiMutation<
    Awaited<ReturnType<typeof createFactoringCompany>>,
    FactoringCompany,
    unknown,
    FactoringCompany
  >({
    form,
    resourceName: "factoring company",
    mutationFn: (values) => createFactoringCompany(values),
    onSuccess: (created, values) => {
      void queryClient.invalidateQueries({ queryKey: ["factoring-company-list"] });
      reset(values);
      void navigate(factoringCompanyPath(created.id), { replace: true });
    },
  });

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("New factoring company"),
        description: t(
          "The factor's legal name and remit-to print on every invoice assigned to it. After it's created, its page walks through the agreement, delivery and notices it needs.",
        ),
      }}
    >
      <FormProvider {...form}>
        <Form id={FORM_ID} onSubmit={handleSubmit((values) => mutation.mutateAsync(values))}>
          <FactoringCompanyEditor
            control={control}
            state={state}
            onState={setState}
            footer={
              <div className="mt-6 flex max-w-160 justify-end gap-2 border-t pt-4">
                <Button
                  variant="outline"
                  nativeButton={false}
                  render={<Link to={FACTORING_COMPANIES_PATH} />}
                >
                  {t("Cancel")}
                </Button>
                <Button type="submit" form={FORM_ID} isLoading={mutation.isPending}>
                  {t("Create factoring company")}
                </Button>
              </div>
            }
          />
        </Form>
      </FormProvider>
      <UnsavedChangesGuard when={isDirty && !mutation.isPending && !mutation.isSuccess} />
    </PageLayout>
  );
}
