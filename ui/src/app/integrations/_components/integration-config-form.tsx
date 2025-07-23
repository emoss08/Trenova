/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import { Button, FormSaveButton } from "@/components/ui/button";
import { DialogBody, DialogFooter } from "@/components/ui/dialog";
import { Form } from "@/components/ui/form";
import { useFormWithSave } from "@/hooks/use-form-with-save";
import { broadcastQueryInvalidation } from "@/hooks/use-invalidate-query";
import { queries } from "@/lib/queries";
import type { IntegrationSchema } from "@/lib/schemas/integration-schema";
import { upperFirst } from "@/lib/utils";
import { api } from "@/services/api";
import { useUser } from "@/stores/user-store";
import { IntegrationType } from "@/types/integration";
import { useEffect } from "react";
import { FormProvider } from "react-hook-form";
import { GoogleMapsForm } from "../_forms/google-maps";
import { PCMilerForm } from "../_forms/pc-miler";

// Helper function to get default values for each integration type
function getDefaultValues(integrationType: string): Record<string, any> {
  switch (integrationType) {
    case "google_maps":
      return { apiKey: "" };
    case "pcmiler":
      return { username: "", password: "", licenseKey: "" };
    case "stripe":
      return { secretKey: "", publishableKey: "" };
    case "auth0":
      return { domain: "", clientId: "", clientSecret: "" };
    case "tracking":
      return { apiKey: "", endpoint: "" };
    default:
      return {};
  }
}

type IntegrationConfigFormProps = {
  integration: IntegrationSchema;
  onOpenChange: (open: boolean) => void;
};

export function IntegrationConfigForm({
  integration,
  onOpenChange,
}: IntegrationConfigFormProps) {
  const user = useUser();

  const form = useFormWithSave({
    resourceName: "Integration",
    formOptions: {
      defaultValues:
        integration.configuration || getDefaultValues(integration.type),
      mode: "onChange",
    },
    mutationFn: async (data: Record<string, any>) => {
      const response = await api.integrations.update(
        integration.id!,
        data,
        user?.id || "",
      );
      return response.data;
    },
    onSuccess: () => {
      onOpenChange(false);
      broadcastQueryInvalidation({
        queryKey: [...queries.integration.getIntegrations._def],
        options: {
          correlationId: `update-integration-${Date.now()}`,
        },
        config: {
          predicate: true,
          refetchType: "all",
        },
      });
    },
  });

  const {
    handleSubmit,
    reset,
    onSubmit,
    formState: { isSubmitting, isSubmitSuccessful },
  } = form;

  // Reset the form when the mutation is successful
  // This is recommended by react-hook-form - https://react-hook-form.com/docs/useform/reset
  useEffect(() => {
    reset();
  }, [isSubmitSuccessful, integration, reset]);

  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (
        (event.ctrlKey || event.metaKey) &&
        event.key === "Enter" &&
        !isSubmitting
      ) {
        event.preventDefault();
        handleSubmit(onSubmit)();
      }
    };

    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [isSubmitting, handleSubmit, onSubmit]);

  return (
    <>
      <FormProvider {...form}>
        <Form onSubmit={handleSubmit(onSubmit)}>
          <DialogBody>
            <ConfigurationForm integrationType={integration.type} />
          </DialogBody>
          <DialogFooter className="mt-6">
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(!open)}
            >
              Cancel
            </Button>
            <FormSaveButton
              isSubmitting={isSubmitting}
              title={`${upperFirst(integration.name)} Integration`}
            />
          </DialogFooter>
        </Form>
      </FormProvider>
    </>
  );
}

function ConfigurationForm({
  integrationType,
}: {
  integrationType: IntegrationType;
}) {
  switch (integrationType) {
    case IntegrationType.GoogleMaps:
      return <GoogleMapsForm />;
    case IntegrationType.PCMiler:
      return <PCMilerForm />;
    default:
      return null;
  }
}
