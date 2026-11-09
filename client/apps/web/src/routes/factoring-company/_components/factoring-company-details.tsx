import type { ChangeFields } from "@/components/form-changes";
import { FormSaveDock } from "@/components/form-save-dock";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import type { FactoringCompanyRecord } from "@/lib/graphql/factoring";
import { updateFactoringCompany } from "@/lib/graphql/factoring-company";
import { factoring as factoringQueries } from "@/lib/queries/factoring";
import { factoringCompanySchema, type FactoringCompany } from "@/types/factoring-company";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Form } from "@trenova/shared/components/ui/form";
import { useT } from "@trenova/shared/i18n/use-t";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useEffect, useRef, useState } from "react";
import { FormProvider, useForm, useWatch, type Resolver } from "react-hook-form";
import type { FactoringCompanyTab } from "../use-factoring-company-state";
import { FactoringCompanyEditor } from "./factoring-company-editor";
import type { FactorState } from "./factoring-invoice-preview";

/** The record as the form holds it: optional text as empty strings, so a cleared field compares equal. */
export function factoringCompanyFormValues(record: FactoringCompanyRecord): FactoringCompany {
  return {
    id: record.id,
    organizationId: record.organizationId,
    businessUnitId: record.businessUnitId,
    version: record.version,
    createdAt: record.createdAt,
    updatedAt: record.updatedAt,
    status: record.status,
    code: record.code,
    name: record.name,
    addressLine1: record.addressLine1,
    addressLine2: record.addressLine2 ?? "",
    city: record.city,
    stateId: record.stateId,
    postalCode: record.postalCode,
    email: record.email ?? "",
    phoneNumber: record.phoneNumber ?? "",
    remittanceInstructions: record.remittanceInstructions ?? "",
    noticeOfAssignment: record.noticeOfAssignment ?? "",
    notes: record.notes ?? "",
    nonRecourse: record.nonRecourse,
  };
}

function storedState(record: FactoringCompanyRecord): FactorState {
  return {
    abbreviation: record.state?.abbreviation ?? null,
    countryName: record.state?.countryName ?? null,
  };
}

/**
 * The factor's own details beside the invoice they print: its name and code, remit-to, contact,
 * and the wording every assigned invoice carries. Saving goes through the dock, which names what
 * changed; leaving with unsaved edits asks first.
 */
export default function FactoringCompanyDetails({
  company,
  recourseFromAgreement,
  onTab,
  onDirtyChange,
}: {
  company: FactoringCompanyRecord;
  recourseFromAgreement: boolean;
  onTab: (tab: FactoringCompanyTab) => void;
  /**
   * Hears whether edits are unsaved. The page keeps this tab mounted when another is open, so it
   * guards leaving and marks the tab rather than this tab, whose effects stop while hidden.
   */
  onDirtyChange: (dirty: boolean) => void;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const { allowed: canUpdate } = usePermission(Resource.FactoringCompany, Operation.Update);
  const [state, setState] = useState<FactorState>(() => storedState(company));
  const overview = useQuery(factoringQueries.overview(company.id));
  const notices = overview.data?.notices;
  const form = useForm<FactoringCompany>({
    resolver: zodResolver(factoringCompanySchema) as Resolver<FactoringCompany>,
    defaultValues: factoringCompanyFormValues(company),
  });
  const {
    control,
    handleSubmit,
    reset,
    formState: { isDirty },
  } = form;

  // The form follows the stored record only when it moves to a new version, so neither a refetch
  // of the same version nor the tab being shown again (Activity re-runs effects) wipes edits in
  // progress. The ref survives the tab being hidden; an effect alone would reset on every return.
  const loadedVersion = useRef(`${company.id}:${company.version}`);
  useEffect(() => {
    const version = `${company.id}:${company.version}`;
    if (loadedVersion.current === version) return;
    loadedVersion.current = version;
    reset(factoringCompanyFormValues(company));
    setState(storedState(company));
  }, [company, reset]);

  // A reset (or picking the stored state again) puts the stored state's abbreviation and country
  // back on the invoice preview; adjusted during render, not in an effect.
  const stateId = useWatch({ control, name: "stateId" });
  const [shownStateId, setShownStateId] = useState(stateId);
  if (stateId !== shownStateId) {
    setShownStateId(stateId);
    if (stateId === company.stateId) setState(storedState(company));
  }

  const mutation = useApiMutation<
    Awaited<ReturnType<typeof updateFactoringCompany>>,
    FactoringCompany,
    unknown,
    FactoringCompany
  >({
    form,
    resourceName: "factoring company",
    mutationFn: (values) => updateFactoringCompany(company.id, values),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: factoringQueries.company(company.id).queryKey,
      });
      void queryClient.invalidateQueries({
        queryKey: factoringQueries.overview(company.id).queryKey,
      });
      void queryClient.invalidateQueries({ queryKey: ["factoring-company-list"] });
    },
  });

  const unsaved = isDirty && !mutation.isPending;
  useEffect(() => {
    onDirtyChange(unsaved);
  }, [unsaved, onDirtyChange]);

  const changeFields: ChangeFields = {
    status: { label: t("Status") },
    code: { label: t("Code") },
    name: { label: t("Name") },
    addressLine1: { label: t("Address line 1") },
    addressLine2: { label: t("Address line 2") },
    city: { label: t("City") },
    stateId: { label: t("State"), format: () => state.abbreviation ?? "" },
    postalCode: { label: t("Postal code") },
    email: { label: t("Email") },
    phoneNumber: { label: t("Phone") },
    nonRecourse: { label: t("Non-recourse") },
    remittanceInstructions: { label: t("Remittance instructions") },
    noticeOfAssignment: { label: t("Notice of assignment") },
    notes: { label: t("Notes") },
  };

  return (
    <FormProvider {...form}>
      <Form
        onSubmit={handleSubmit((values) => mutation.mutateAsync(values))}
        className="flex min-w-0 flex-1 flex-col"
      >
        <FactoringCompanyEditor
          control={control}
          state={state}
          onState={setState}
          readOnly={!canUpdate}
          recourseFromAgreement={recourseFromAgreement}
          onOpenAgreement={() => onTab("agreement")}
          remitNotices={notices ? notices.awaiting + notices.acknowledged : 0}
          onOpenCustomers={() => onTab("customers")}
        />
        {canUpdate ? <FormSaveDock changeFields={changeFields} /> : null}
      </Form>
    </FormProvider>
  );
}
