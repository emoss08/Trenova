import { useT } from "@trenova/shared/i18n/use-t";
import { ComponentLoader } from "@trenova/shared/components/component-loader";
import { DataTablePanelContainer } from "@/components/data-table/data-table-panel";
import { FormCreatePanel } from "@/components/form-create-panel";
import { Button } from "@trenova/shared/components/ui/button";
import { Form } from "@trenova/shared/components/ui/form";
import { SplitButton, type SplitButtonOption } from "@trenova/shared/components/ui/split-button";
import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  useEditPanelActionPreference,
  type EditPanelSaveAction,
} from "@/hooks/use-panel-action-preference";
import { api } from "@trenova/shared/lib/api";
import { formatToUserTimezone } from "@trenova/shared/lib/date";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import { factoringCompanySchema, type FactoringCompany } from "@/types/factoring-company";
import { TimeFormat } from "@trenova/shared/types/user";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { useCallback, useEffect } from "react";
import { FormProvider, type Resolver, useForm } from "react-hook-form";
import { toast } from "sonner";
import { FactoringCompanyForm } from "./factoring-company-form";
import { FactoringAgreementsSection } from "./factoring-agreements-section";
import { FactoringDeliverySection } from "./factoring-delivery-section";
import { FactoringNoticesSection } from "./factoring-notices-section";
import { useClearFactoringRecordParams } from "./use-factoring-record-redirect";
import { panelSearchParamsParser } from "@/hooks/data-table/use-data-table-state";
import { factoring as factoringQueries } from "@/lib/queries/factoring";
import { factoringCompanyRecordSearchParamsParser } from "../use-factoring-company-state";
import { useQuery } from "@tanstack/react-query";
import { useQueryStates } from "nuqs";

const SAVE_OPTIONS: SplitButtonOption<EditPanelSaveAction>[] = [
  { id: "save", label: "Save" },
  { id: "save-close", label: "Save & close" },
];

export function FactoringCompanyPanel({
  open,
  onOpenChange: onPanelOpenChange,
  mode,
  row,
}: DataTablePanelProps<FactoringCompany>) {
  const t = useT();
  const clearRecordParams = useClearFactoringRecordParams();
  const onOpenChange = (next: boolean) => {
    if (!next) clearRecordParams();
    onPanelOpenChange(next);
  };

  const form = useForm<FactoringCompany>({
    resolver: zodResolver(factoringCompanySchema) as Resolver<FactoringCompany>,
    defaultValues: {
      status: "Active" as const,
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
    },
  });

  if (mode === "edit") {
    return (
      <FactoringCompanyEditPanel open={open} onOpenChange={onOpenChange} row={row} form={form} />
    );
  }

  return (
    <FormCreatePanel
      open={open}
      onOpenChange={onOpenChange}
      form={form}
      url="/factoring-companies/"
      queryKey="factoring-company-list"
      title={t("Factoring company")}
      formComponent={<FactoringCompanyForm />}
    />
  );
}

type FactoringCompanyEditPanelProps = Pick<
  DataTablePanelProps<FactoringCompany>,
  "open" | "onOpenChange" | "row"
> & {
  form: ReturnType<typeof useForm<FactoringCompany>>;
};

function FactoringCompanyEditPanel({
  open,
  onOpenChange,
  row: listRow,
  form,
}: FactoringCompanyEditPanelProps) {
  const t = useT();
  const [{ panelEntityId }] = useQueryStates(panelSearchParamsParser);
  const [{ agreement: highlightedAgreement, noa: highlightedNoa }] = useQueryStates(
    factoringCompanyRecordSearchParamsParser,
  );
  const fetched = useQuery({
    ...factoringQueries.company(panelEntityId ?? ""),
    enabled: open && !listRow && !!panelEntityId,
  });
  const row = listRow ?? (fetched.data as FactoringCompany | null | undefined) ?? null;

  const queryClient = useQueryClient();
  const [defaultAction, setDefaultAction] = useEditPanelActionPreference();

  type EditSubmitPayload = {
    action: EditPanelSaveAction;
    values: FactoringCompany;
  };

  const {
    formState: { isSubmitting },
    handleSubmit,
    reset,
  } = form;

  const handleClose = () => {
    onOpenChange(false);
    reset();
  };

  useEffect(() => {
    if (open && row) {
      reset(row as FactoringCompany, { keepDefaultValues: true });
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, row?.id, reset]);

  const { mutateAsync } = useApiMutation<
    FactoringCompany,
    EditSubmitPayload,
    unknown,
    FactoringCompany
  >({
    mutationFn: async ({ values }) => {
      const response = await api.put<FactoringCompany>(`/factoring-companies/${row?.id}/`, values);
      return response;
    },
    onMutate: async (newValues) => {
      await queryClient.cancelQueries({
        queryKey: ["factoring-company-list"],
      });
      const previousRecord = queryClient.getQueryData(["factoring-company-list"]);
      queryClient.setQueryData(["factoring-company-list"], newValues);
      return { previousRecord, newValues };
    },
    onSuccess: (_data, variables) => {
      toast.success(t("Changes have been saved"), {
        description: t("Factoring company updated successfully"),
      });
      void queryClient.invalidateQueries({ queryKey: ["factoring-company-list"] });

      const action = variables.action;
      if (action === "save-close") {
        reset();
        onOpenChange(false);
      }
    },
    form,
    resourceName: "Factoring Company",
  });

  const onSubmit = useCallback(
    async (values: FactoringCompany, action: EditPanelSaveAction) => {
      await mutateAsync({ values, action });
    },
    [mutateAsync],
  );

  const handleOptionSelect = (action: EditPanelSaveAction) => {
    setDefaultAction(action);
    void handleSubmit((values) => onSubmit(values, action))();
  };

  const handleFormSubmit = (values: FactoringCompany) => {
    return onSubmit(values, defaultAction);
  };

  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (open && (event.ctrlKey || event.metaKey) && event.key === "Enter" && !isSubmitting) {
        event.preventDefault();
        void handleSubmit((values) => onSubmit(values, defaultAction))();
      }
    };

    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, isSubmitting, handleSubmit, defaultAction]);

  const panelDescription = row?.updatedAt
    ? t(
        "Last updated on {0}",
        formatToUserTimezone(row.updatedAt as number, {
          timeFormat: TimeFormat.enum["24-hour"],
        }),
      )
    : undefined;

  return (
    <DataTablePanelContainer
      open={open}
      onOpenChange={onOpenChange}
      title={row?.name ?? t("Factoring company")}
      description={panelDescription}
      size="lg"
      footer={
        <>
          <Button type="button" variant="outline" onClick={handleClose}>
            {t("Cancel")}
          </Button>
          <SplitButton
            options={SAVE_OPTIONS}
            selectedOption={defaultAction}
            onOptionSelect={handleOptionSelect}
            isLoading={isSubmitting}
            loadingText={t("Saving...")}
            formId="panel-edit-form"
          />
        </>
      }
    >
      {!row ? (
        <ComponentLoader message={t("Loading factoring company...")} />
      ) : (
        <div className="flex flex-col gap-6">
          <FormProvider {...form}>
            <Form id="panel-edit-form" onSubmit={handleSubmit(handleFormSubmit)}>
              <FactoringCompanyForm />
            </Form>
          </FormProvider>
          {row.id ? (
            <>
              <FactoringAgreementsSection
                factoringCompanyId={row.id}
                highlightedId={highlightedAgreement}
              />
              <FactoringNoticesSection factoringCompanyId={row.id} highlightedId={highlightedNoa} />
              <FactoringDeliverySection factoringCompanyId={row.id} />
            </>
          ) : null}
        </div>
      )}
    </DataTablePanelContainer>
  );
}
