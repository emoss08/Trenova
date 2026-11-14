import { FormSaveDock } from "@/components/form-save-dock";
import { useSettingsTab, tabHoldingError } from "@/components/settings/settings-tab";
import { SettingsTabs } from "@/components/settings/settings-tabs";
import { useOptimisticMutation } from "@/hooks/use-optimistic-mutation";
import { dispatchControl } from "@/lib/queries/dispatch-control";
import { dispatchControlService } from "@/services/dispatch-control";
import { dispatchControlSchema, type DispatchControl } from "@/types/dispatch-control";
import { zodResolver } from "@hookform/resolvers/zod";
import { useSuspenseQuery } from "@tanstack/react-query";
import { Form } from "@trenova/shared/components/ui/form";
import { useOrgCapabilities } from "@trenova/shared/hooks/use-org-capabilities";
import { useT } from "@trenova/shared/i18n/use-t";
import {
  hasOrganizationCapability,
  OrganizationCapability,
} from "@trenova/shared/types/organization-capability";
import type { ReactNode } from "react";
import { FormProvider, useForm, type Resolver } from "react-hook-form";
import { dispatchControlTabs, type DispatchControlTab } from "../dispatch-control-tabs";
import { AssignmentTab } from "./assignment-tab";
import { ComplianceTab } from "./compliance-tab";
import { CostsTab } from "./costs-tab";
import { ExceptionsTab } from "./exceptions-tab";
import { LearningTab } from "./learning-tab";
import { MonitoringTab } from "./monitoring-tab";
import { TendersTab } from "./tenders-tab";

/**
 * Dispatch control: one form, saved once, split into a tab for each job a person comes here
 * to do. Which tabs there are follows what the organization does: planning, its costs, what
 * it has learned and DOT compliance for one that runs a fleet, carrier tenders for a brokerage.
 */
export default function DispatchControlForm() {
  const t = useT();

  const { data } = useSuspenseQuery({
    ...dispatchControl.get(),
  });

  const form = useForm<DispatchControl>({
    resolver: zodResolver(dispatchControlSchema) as Resolver<DispatchControl>,
    defaultValues: data,
  });

  const { handleSubmit, reset, getFieldState } = form;

  const capabilities = useOrgCapabilities();
  const specs = dispatchControlTabs({
    runsAssets: hasOrganizationCapability(capabilities, OrganizationCapability.AssetOperations),
    brokers: hasOrganizationCapability(capabilities, OrganizationCapability.Brokerage),
  });
  const [openTab, selectTab] = useSettingsTab(specs);

  const { mutateAsync } = useOptimisticMutation({
    queryKey: dispatchControl.get._def,
    mutationFn: async (values: DispatchControl) => dispatchControlService.update(values),
    resourceName: "Dispatch Control",
    resetForm: reset,
    form,
    invalidateQueries: [dispatchControl.get._def],
  });

  // A save refused for a field on another tab would otherwise fail out of sight.
  const openRefusedTab = () => {
    const tab = tabHoldingError(specs, openTab, (field) => getFieldState(field).invalid);
    if (tab) selectTab(tab);
  };

  const onSubmit = async (values: DispatchControl) => {
    try {
      await mutateAsync(values);
    } catch {
      openRefusedTab();
    }
  };

  const content = (tab: DispatchControlTab): ReactNode => {
    switch (tab) {
      case "assignment":
        return <AssignmentTab />;
      case "costs":
        return <CostsTab />;
      case "learning":
        return (
          <LearningTab
            shown={openTab === "learning"}
            onOpenAssignment={() => selectTab("assignment")}
          />
        );
      case "monitoring":
        return <MonitoringTab />;
      case "exceptions":
        return <ExceptionsTab />;
      case "tenders":
        return <TendersTab />;
      case "compliance":
        return <ComplianceTab />;
    }
  };

  return (
    <FormProvider {...form}>
      <Form onSubmit={handleSubmit(onSubmit, openRefusedTab)}>
        <div className="flex flex-col gap-4 pb-14">
          <SettingsTabs
            tabs={specs.map((spec) => ({ ...spec, content: content(spec.value) }))}
            value={openTab}
            onValueChange={selectTab}
          />
          <FormSaveDock saveButtonContent={t("Save changes")} />
        </div>
      </Form>
    </FormProvider>
  );
}
