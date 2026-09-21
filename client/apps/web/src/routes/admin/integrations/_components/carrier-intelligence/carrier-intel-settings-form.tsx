import { handleMutationError } from "@/hooks/use-api-mutation";
import { carrierIntelProviderLabel } from "@/lib/carrier-intelligence";
import type { CarrierIntelControl, CarrierIntelSettings } from "@/lib/graphql/carrier-intelligence";
import {
  updateCarrierIntelControl,
  type CarrierIntelCostEstimate,
} from "@/lib/graphql/carrier-intel-settings";
import { queries } from "@/lib/queries";
import { zodResolver } from "@hookform/resolvers/zod";
import type { CarrierIntelControlPatchInput } from "@trenova/graphql/generated/graphql";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { DialogFooter } from "@trenova/shared/components/ui/dialog";
import { Form } from "@trenova/shared/components/ui/form";
import { TabsPanel } from "@trenova/shared/components/ui/tabs";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useCallback, useMemo, useState } from "react";
import { FormProvider, useForm, type FieldErrors, type Resolver } from "react-hook-form";
import { toast } from "sonner";
import {
  buildControlPatch,
  buildSettingsSchema,
  isCostConfirmationRequiredError,
  remapRuleFieldErrors,
  requiresCostConfirmation,
  settingsTabForErrors,
  toSettingsFormValues,
  type CarrierIntelSettingsFormValues,
  type CarrierIntelSettingsTab,
} from "./carrier-intel-settings-schema";
import { CostConfirmationDialog } from "./cost-confirmation-dialog";
import { CarrierIntelMonitoringTab } from "./monitoring-tab";
import { CarrierIntelRulesTab } from "./rules-tab";
import { CarrierIntelSpendTab } from "./spend-tab";

type SaveVariables = {
  input: CarrierIntelControlPatchInput;
  values: CarrierIntelSettingsFormValues;
};

type PendingConfirmation = SaveVariables & { estimate: CarrierIntelCostEstimate };

export function CarrierIntelSettingsForm({
  settings,
  open,
  activeTab,
  canManage,
  onTabChange,
  onClose,
}: {
  settings: CarrierIntelSettings;
  open: boolean;
  activeTab: string;
  canManage: boolean;
  onTabChange: (tab: CarrierIntelSettingsTab) => void;
  onClose: () => void;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const {
    carrierIntelControl: savedControl,
    carrierIntelRuleCatalog: catalog,
    carrierIntelProvider: provider,
  } = settings;

  const schema = useMemo(() => buildSettingsSchema(catalog), [catalog]);
  const defaultValues = useMemo(
    () => toSettingsFormValues(savedControl, catalog),
    [savedControl, catalog],
  );
  const storedCodes = useMemo(
    () => new Set(savedControl.rules.map((rule) => rule.code)),
    [savedControl.rules],
  );

  const form = useForm<CarrierIntelSettingsFormValues>({
    resolver: zodResolver(schema) as Resolver<CarrierIntelSettingsFormValues>,
    defaultValues,
    values: defaultValues,
    resetOptions: { keepDirtyValues: true },
  });
  const { handleSubmit, reset, formState } = form;

  const [pending, setPending] = useState<PendingConfirmation | null>(null);
  const [isEstimating, setIsEstimating] = useState(false);

  const revealErrors = useCallback(
    (errors: FieldErrors<CarrierIntelSettingsFormValues>) => {
      const tab = settingsTabForErrors(errors);
      if (tab && tab !== activeTab) {
        onTabChange(tab);
      }
    },
    [activeTab, onTabChange],
  );

  const requestConfirmation = useCallback(
    async (variables: SaveVariables) => {
      const { values } = variables;
      const recentlyUsed = values.enrollmentPolicy === "RecentlyUsed";
      setIsEstimating(true);
      try {
        const estimate = await queryClient.fetchQuery(
          queries.carrierIntelSettings.costEstimate({
            policy: values.enrollmentPolicy,
            recentUsageDays: recentlyUsed ? values.recentUsageDays : undefined,
            includeOpenTenders: recentlyUsed ? values.includeOpenTenders : undefined,
          }),
        );
        setPending({ ...variables, estimate });
      } catch (error) {
        handleMutationError({ error, resourceName: "Carrier monitoring cost estimate" });
      } finally {
        setIsEstimating(false);
      }
    },
    [queryClient],
  );

  const saveMutation = useMutation({
    mutationFn: ({ input }: SaveVariables) => updateCarrierIntelControl(input),
    onSuccess: async (control: CarrierIntelControl) => {
      setPending(null);
      queryClient.setQueryData<CarrierIntelSettings>(
        queries.carrierIntelSettings.settings().queryKey,
        (previous) => (previous ? { ...previous, carrierIntelControl: control } : previous),
      );
      reset(toSettingsFormValues(control, catalog));
      toast.success(t("Carrier intelligence settings saved"));
      await queryClient.invalidateQueries({ queryKey: queries.carrierIntelSettings._def });
    },
    onError: (error, variables) => {
      if (isCostConfirmationRequiredError(error) && !variables.input.confirmEstimatedCost) {
        void requestConfirmation(variables);
        return;
      }
      setPending(null);
      handleMutationError<CarrierIntelSettingsFormValues>({
        error: remapRuleFieldErrors(error, variables.values.rules),
        form,
        resourceName: "Carrier intelligence settings",
      });
      revealErrors(form.formState.errors);
    },
  });

  const onValid = async (values: CarrierIntelSettingsFormValues) => {
    const input = buildControlPatch({ values, original: defaultValues, catalog, storedCodes });
    if (!input) {
      toast.info(t("There are no changes to save"));
      return;
    }
    if (requiresCostConfirmation(values, defaultValues)) {
      await requestConfirmation({ input, values });
      return;
    }
    await saveMutation.mutateAsync({ input, values }).catch(() => undefined);
  };

  const confirmPending = () => {
    if (!pending) {
      return;
    }
    saveMutation.mutate({
      input: { ...pending.input, confirmEstimatedCost: true },
      values: pending.values,
    });
  };

  const readOnly = !canManage;
  const providerName = carrierIntelProviderLabel(provider.provider);
  const isSettingsTab = activeTab !== "connection";

  return (
    <FormProvider {...form}>
      <Form
        onSubmit={handleSubmit(onValid, revealErrors)}
        className={cn("flex min-h-0 flex-1 flex-col", !isSettingsTab && "hidden")}
      >
        <div className="min-h-0 flex-1 overflow-y-auto px-1 py-2">
          {!provider.configured ? (
            <div className="border-border bg-muted/30 text-muted-foreground mb-4 rounded-md border p-3 text-sm">
              {t(
                "No carrier intelligence provider is connected yet. Settings can be prepared now and take effect once a provider is enabled.",
              )}
            </div>
          ) : null}
          <TabsPanel value="rules">
            <CarrierIntelRulesTab
              catalog={catalog}
              sections={provider.sections}
              providerName={providerName}
              providerConfigured={provider.configured}
              readOnly={readOnly}
            />
          </TabsPanel>
          <TabsPanel value="monitoring">
            <CarrierIntelMonitoringTab
              open={open && activeTab === "monitoring"}
              readOnly={readOnly}
              canManage={canManage}
              providerConfigured={provider.configured}
            />
          </TabsPanel>
          <TabsPanel value="spend">
            <CarrierIntelSpendTab
              open={open && activeTab === "spend"}
              readOnly={readOnly}
              canManage={canManage}
            />
          </TabsPanel>
        </div>
        {isSettingsTab ? (
          <DialogFooter className="sm:items-center sm:justify-between">
            <span className="text-muted-foreground text-xs">
              {readOnly
                ? t("You can view these settings but not change them.")
                : formState.isDirty
                  ? t("You have unsaved changes.")
                  : t("Policy version {0}", savedControl.policyVersion)}
            </span>
            <div className="flex flex-col-reverse gap-2 sm:flex-row">
              <Button type="button" variant="outline" onClick={onClose}>
                {t("Close")}
              </Button>
              {canManage ? (
                <>
                  <Button
                    type="button"
                    variant="outline"
                    disabled={!formState.isDirty || saveMutation.isPending}
                    onClick={() => reset(defaultValues)}
                  >
                    {t("Discard")}
                  </Button>
                  <Button
                    type="submit"
                    disabled={!formState.isDirty}
                    isLoading={saveMutation.isPending || isEstimating}
                    loadingText={isEstimating ? t("Estimating...") : t("Saving...")}
                  >
                    {t("Save changes")}
                  </Button>
                </>
              ) : null}
            </div>
          </DialogFooter>
        ) : null}
      </Form>
      <CostConfirmationDialog
        open={pending !== null}
        estimate={pending?.estimate ?? null}
        isSaving={saveMutation.isPending}
        onConfirm={confirmPending}
        onCancel={() => setPending(null)}
      />
    </FormProvider>
  );
}
