import { useT } from "@trenova/shared/i18n/use-t";
import { InputField } from "@/components/fields/input-field";
import { SwitchField } from "@/components/fields/switch-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { carrierPanelPath } from "@/lib/carrier-links";
import { withExistingCarrier } from "@/lib/carrier-sourcing";
import {
  CARRIER_INTEL_LOOKUP_KEY,
  CARRIER_SOURCING_SEARCH_KEY,
  importSourcedCarrier,
  type CarrierIntelProspectLookup,
  type CarrierSourcingPage,
  type ImportedSourcedCarrier,
} from "@/lib/graphql/carrier-sourcing";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
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
import { useEffect } from "react";
import { FormProvider, useForm, type Resolver } from "react-hook-form";
import { useNavigate } from "react-router";
import { toast } from "sonner";
import {
  CARRIER_CODE_MAX_LENGTH,
  importSourcedCarrierSchema,
  type ImportSourcedCarrierFormValues,
} from "./sourcing-schema";

export type ImportCandidate = {
  dotNumber: string;
  legalName: string | null;
};

export type ImportCarrierDialogProps = {
  candidate: ImportCandidate | null;
  canEnrollMonitoring: boolean;
  onOpenChange: (open: boolean) => void;
};

export function ImportCarrierDialog({
  candidate,
  canEnrollMonitoring,
  onOpenChange,
}: ImportCarrierDialogProps) {
  const t = useT();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const open = candidate !== null;

  const defaults: ImportSourcedCarrierFormValues = {
    code: "",
    enrollMonitoring: canEnrollMonitoring,
  };

  const form = useForm<ImportSourcedCarrierFormValues>({
    resolver: zodResolver(importSourcedCarrierSchema) as Resolver<ImportSourcedCarrierFormValues>,
    defaultValues: defaults,
  });
  const { control, handleSubmit, reset } = form;

  useEffect(() => {
    if (open) {
      reset({ code: "", enrollMonitoring: canEnrollMonitoring });
    }
  }, [canEnrollMonitoring, open, reset]);

  const { mutateAsync, isPending } = useApiMutation<
    ImportedSourcedCarrier,
    ImportSourcedCarrierFormValues,
    unknown,
    ImportSourcedCarrierFormValues
  >({
    form,
    resourceName: "Carrier",
    mutationFn: (values) => {
      if (!candidate) {
        throw new Error("No carrier selected");
      }
      return importSourcedCarrier({
        dotNumber: candidate.dotNumber,
        code: values.code === "" ? null : values.code.toUpperCase(),
        enrollMonitoring: canEnrollMonitoring && values.enrollMonitoring,
      });
    },
    onSuccess: async (carrier) => {
      const dotNumber = candidate?.dotNumber ?? carrier.dotNumber ?? "";
      queryClient.setQueriesData<CarrierSourcingPage>(
        { queryKey: [CARRIER_SOURCING_SEARCH_KEY] },
        (page) =>
          page ? { ...page, items: withExistingCarrier(page.items, dotNumber, carrier.id) } : page,
      );
      queryClient.setQueriesData<CarrierIntelProspectLookup>(
        { queryKey: [CARRIER_INTEL_LOOKUP_KEY] },
        (lookup) =>
          lookup && lookup.snapshot.dotNumber === dotNumber
            ? { ...lookup, existingCarrierId: carrier.id }
            : lookup,
      );
      await queryClient.invalidateQueries({ queryKey: ["carrier-list"] });
      toast.success(t("{0} imported as {1}", carrier.name, carrier.code), {
        description: t("The carrier starts in Pending compliance until it is reviewed."),
      });
      onOpenChange(false);
      void navigate(carrierPanelPath(carrier.id, "intelligence"));
    },
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{t("Import carrier")}</DialogTitle>
          <DialogDescription>
            {candidate
              ? t(
                  "Creates {0} (USDOT {1}) from the provider's FMCSA record, attaches its intelligence and opens it so you can finish onboarding.",
                  candidate.legalName || t("this carrier"),
                  candidate.dotNumber,
                )
              : t("Create a carrier from the provider's record.")}
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
            <FormGroup cols={1} className="pb-2">
              <FormControl>
                <InputField<ImportSourcedCarrierFormValues>
                  control={control}
                  name="code"
                  label={t("Carrier code")}
                  placeholder={t("Generated from the name")}
                  description={t(
                    "Leave blank to generate one from the legal name and USDOT number.",
                  )}
                  maxLength={CARRIER_CODE_MAX_LENGTH}
                  autoComplete="off"
                />
              </FormControl>
              {canEnrollMonitoring ? (
                <FormControl className="min-h-0">
                  <SwitchField<ImportSourcedCarrierFormValues>
                    control={control}
                    name="enrollMonitoring"
                    label={t("Monitor this carrier")}
                    description={t(
                      "Watch for authority, insurance and safety changes as soon as it is created.",
                    )}
                    position="left"
                  />
                </FormControl>
              ) : null}
            </FormGroup>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                {t("Cancel")}
              </Button>
              <Button type="submit" isLoading={isPending} disabled={!candidate}>
                {t("Import carrier")}
              </Button>
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
