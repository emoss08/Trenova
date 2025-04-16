import { InputField } from "@/components/fields/input-field";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { FormControl, FormGroup } from "@/components/ui/form";
import { Icon } from "@/components/ui/icons";
import { LazyImage } from "@/components/ui/image";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { integrationKeys } from "@/config/query-keys";
import { generateDateTimeStringFromUnixTimestamp } from "@/lib/date";
import { http } from "@/lib/http-client";
import { cn } from "@/lib/utils";
import type { Integration } from "@/types/integrations/integration";
import {
  faCheck,
  faCheckCircle,
  faCog,
  faInfoCircle,
  faTriangleExclamation,
  faWarning,
} from "@fortawesome/pro-regular-svg-icons";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { FormProvider, useForm, useFormContext } from "react-hook-form";
import { toast } from "sonner";
import { GoogleMapsForm } from "../_forms/google-maps";
import { integrationImages } from "../_utils/integration";

interface PCMilerConfigData {
  username: string;
  password: string;
  licenseKey: string;
}

// API functions
async function configureIntegration(
  integrationId: string,
  data: Record<string, any>,
) {
  return http.put(`/integrations/${integrationId}`, {
    configuration: data,
  });
}

async function testIntegrationConnection(integrationId: string) {
  return http.post<{ isValid: boolean; message: string; status: string }>(
    `/integrations/${integrationId}/test-connection`,
  );
}

interface IntegrationConfigDialogProps {
  integration: Integration;
  open: boolean;
  onClose: () => void;
}

export function IntegrationConfigDialog({
  integration,
  open,
  onClose,
}: IntegrationConfigDialogProps) {
  const queryClient = useQueryClient();
  const [activeTab, setActiveTab] = useState("info");
  const [testResult, setTestResult] = useState<{
    isValid: boolean;
    message: string;
  } | null>(null);

  // Set up form methods based on integration type
  const formMethods = useForm({
    defaultValues:
      integration.configuration || getDefaultValues(integration.type),
  });

  // Reset form if integration changes
  useEffect(() => {
    formMethods.reset(
      integration.configuration || getDefaultValues(integration.type),
    );
  }, [integration, formMethods]);

  // Handle form submission
  const handleSubmit = (data: Record<string, any>) => {
    configureMutation.mutate(data);
  };

  // Check if we have existing configuration data
  const hasExistingConfig =
    integration.configuration &&
    Object.keys(integration.configuration).length > 0;

  // Mutations
  const configureMutation = useMutation({
    mutationFn: (data: Record<string, any>) =>
      configureIntegration(integration.id, data),
    onSuccess: () => {
      toast.success("Integration configured successfully");
      queryClient.invalidateQueries({ queryKey: integrationKeys.list() });
      setTestResult({
        isValid: true,
        message: "Configuration saved successfully",
      });
      // Don't switch tabs automatically - stay on the current tab
    },
    onError: (error: any) => {
      toast.error("Configuration failed", {
        description: error.message || "An unknown error occurred",
      });
    },
  });

  const testConnectionMutation = useMutation({
    mutationFn: () => testIntegrationConnection(integration.id),
    onSuccess: (response) => {
      const data = response.data;
      setTestResult({
        isValid: data.isValid,
        message: data.message,
      });

      if (data.isValid) {
        toast.success("Connection test successful", {
          description: data.message,
        });
      } else {
        toast.warning("Connection test failed", {
          description: data.message,
        });
      }

      queryClient.invalidateQueries({ queryKey: integrationKeys.list() });
    },
    onError: (error: any) => {
      toast.error("Connection test failed", {
        description: error.message || "An unknown error occurred",
      });
      setTestResult({
        isValid: false,
        message: error.message || "An unknown error occurred",
      });
    },
  });

  return (
    <Dialog open={open} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <LazyImage
              src={integrationImages[integration.type]}
              layout="fixed"
              width={20}
              height={20}
            />
            {integration.name}
          </DialogTitle>
          <DialogDescription>
            Configure and manage your {integration.name} integration
          </DialogDescription>
        </DialogHeader>
        <DialogBody>
          <Tabs value={activeTab} onValueChange={setActiveTab}>
            <TabsList className="grid w-full grid-cols-3">
              <TabsTrigger value="info">Information</TabsTrigger>
              <TabsTrigger value="configure">Configure</TabsTrigger>
              <TabsTrigger value="test">Test</TabsTrigger>
            </TabsList>

            <TabsContent value="info" className="space-y-4 py-4">
              <InfoTab
                integration={integration}
                testResult={testResult}
                onConfigureClick={() => setActiveTab("configure")}
                onTestClick={() => testConnectionMutation.mutate()}
                isTestLoading={testConnectionMutation.isPending}
                hasExistingConfig={hasExistingConfig ?? false}
              />
            </TabsContent>

            <TabsContent value="configure" className="py-4">
              <FormProvider {...formMethods}>
                <form onSubmit={formMethods.handleSubmit(handleSubmit)}>
                  <ConfigurationForm
                    integrationType={integration.type}
                    hasExistingConfig={hasExistingConfig ?? false}
                  />
                  <DialogFooter className="mt-6">
                    <Button
                      type="button"
                      variant="outline"
                      onClick={onClose}
                      disabled={configureMutation.isPending}
                    >
                      Cancel
                    </Button>
                    <Button
                      type="submit"
                      disabled={configureMutation.isPending}
                    >
                      {configureMutation.isPending
                        ? "Saving..."
                        : hasExistingConfig
                          ? "Update Configuration"
                          : "Save Configuration"}
                    </Button>
                  </DialogFooter>
                </form>
              </FormProvider>
            </TabsContent>

            <TabsContent value="test" className="space-y-4 py-4">
              <TestTab
                integration={integration}
                testResult={testResult}
                onTestClick={() => testConnectionMutation.mutate()}
                isTestLoading={testConnectionMutation.isPending}
                onClose={onClose}
              />
            </TabsContent>
          </Tabs>
        </DialogBody>
      </DialogContent>
    </Dialog>
  );
}

// Helper components to improve readability
function InfoTab({
  integration,
  testResult,
  onConfigureClick,
  onTestClick,
  isTestLoading,
  hasExistingConfig,
}: {
  integration: Integration;
  testResult: { isValid: boolean; message: string } | null;
  onConfigureClick: () => void;
  onTestClick: () => void;
  isTestLoading: boolean;
  hasExistingConfig: boolean;
}) {
  return (
    <div className="space-y-4">
      <div>
        <h3 className="text-sm font-medium">Status</h3>
        <div
          className={cn(
            "mt-1 inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-semibold",
            integration.enabled
              ? "bg-green-100 text-green-800"
              : "bg-gray-100 text-gray-800",
          )}
        >
          <Icon
            icon={integration.enabled ? faCheck : faCog}
            className="mr-1 h-3 w-3"
          />
          {integration.enabled ? "Active" : "Inactive"}
        </div>
      </div>

      <div>
        <h3 className="text-sm font-medium">Description</h3>
        <p className="mt-1 text-sm text-muted-foreground">
          {integration.description}
        </p>
      </div>

      {hasExistingConfig && (
        <div className="rounded-md bg-blue-50 p-3">
          <h3 className="flex items-center text-sm font-medium text-blue-800">
            <Icon icon={faInfoCircle} className="mr-2 h-4 w-4 text-blue-500" />
            Configuration Status
          </h3>
          <p className="mt-1 text-sm text-blue-700">
            This integration is{" "}
            {integration.enabled
              ? "active and configured"
              : "configured but not active"}
            . You can {integration.enabled ? "manage" : "update"}{" "}
          </p>
          <p className="mt-1 text-xs text-blue-600">
            Last updated:{" "}
            {generateDateTimeStringFromUnixTimestamp(integration.updatedAt)}
          </p>
        </div>
      )}

      {integration.lastError && (
        <div className="rounded-md bg-red-50 p-3">
          <h3 className="flex items-center text-sm font-medium text-red-800">
            <Icon icon={faTriangleExclamation} className="mr-2 h-4 w-4" />
            Last Error
          </h3>
          <p className="mt-1 text-sm text-red-700">{integration.lastError}</p>
        </div>
      )}

      {testResult && (
        <div
          className={cn(
            "rounded-md p-3",
            testResult.isValid ? "bg-green-50" : "bg-amber-50",
          )}
        >
          <h3
            className={cn(
              "flex items-center text-sm font-medium",
              testResult.isValid ? "text-green-800" : "text-amber-800",
            )}
          >
            <Icon
              icon={testResult.isValid ? faCheckCircle : faWarning}
              className={cn(
                "mr-2 h-4 w-4",
                testResult.isValid ? "text-green-500" : "text-amber-500",
              )}
            />
            {testResult.isValid ? "Test Successful" : "Test Result"}
          </h3>
          <p
            className={cn(
              "mt-1 text-sm",
              testResult.isValid ? "text-green-700" : "text-amber-700",
            )}
          >
            {testResult.message}
          </p>
        </div>
      )}

      <div className="flex justify-end space-x-2 pt-4">
        <Button variant="outline" onClick={onConfigureClick}>
          <Icon icon={faCog} className="mr-2 h-4 w-4" />
          {hasExistingConfig ? "Edit Configuration" : "Configure"}
        </Button>
        <Button onClick={onTestClick} disabled={isTestLoading}>
          {isTestLoading ? "Testing..." : "Test Connection"}
        </Button>
      </div>
    </div>
  );
}

function ConfigurationForm({
  integrationType,
  hasExistingConfig,
}: {
  integrationType: string;
  hasExistingConfig: boolean;
}) {
  return (
    <FormGroup className="gap-y-3">
      {integrationType === "GoogleMaps" && <GoogleMapsForm />}
      {integrationType === "PCMiler" && <PCMilerForm />}
      {!["GoogleMaps", "PCMiler"].includes(integrationType) && (
        <div className="flex flex-col items-center justify-center py-8 text-center">
          <Icon icon={faInfoCircle} className="mb-4 h-12 w-12 text-blue-500" />
          <h3 className="mb-2 text-lg font-medium">
            Configuration Not Available
          </h3>
          <p className="mb-6 text-sm text-muted-foreground">
            Configuration options for this integration are not available yet.
          </p>
        </div>
      )}

      {hasExistingConfig && (
        <div className="mt-2 rounded-md bg-blue-50 p-3">
          <p className="text-xs text-blue-700">
            <Icon icon={faInfoCircle} className="mr-1 h-3 w-3" />
            You are editing an existing configuration.
          </p>
        </div>
      )}
    </FormGroup>
  );
}

function PCMilerForm() {
  const { control, formState } = useFormContext<PCMilerConfigData>();
  const { dirtyFields } = formState;

  return (
    <>
      <FormControl>
        <InputField
          control={control}
          name="username"
          label="Username"
          rules={{ required: true }}
          placeholder="Enter your PCMiler username"
          description="Your PCMiler account username."
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          name="password"
          label="Password"
          rules={{ required: true }}
          placeholder="Enter your PCMiler password"
          type="password"
          description="Your PCMiler account password."
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          name="licenseKey"
          label="License Key"
          rules={{ required: true }}
          placeholder="Enter your PCMiler license key"
          description="The license key for your PCMiler subscription."
        />
      </FormControl>
      {Object.keys(dirtyFields).length > 0 && (
        <div className="mt-2 rounded-md bg-blue-50 p-2">
          <p className="text-xs text-blue-700">
            <Icon icon={faInfoCircle} className="mr-1 h-3 w-3" />
            Changes will be saved when you click Update Configuration.
          </p>
        </div>
      )}
    </>
  );
}

function TestTab({
  integration,
  testResult,
  onTestClick,
  isTestLoading,
  onClose,
}: {
  integration: Integration;
  testResult: { isValid: boolean; message: string } | null;
  onTestClick: () => void;
  isTestLoading: boolean;
  onClose: () => void;
}) {
  const hasConfiguration =
    integration.configuration &&
    Object.keys(integration.configuration).length > 0;

  return (
    <div className="space-y-4">
      <p className="text-sm">
        Test your {integration.name} integration to ensure it&apos;s properly
        configured and working.
      </p>

      {!hasConfiguration && (
        <div className="rounded-md bg-amber-50 p-4">
          <h3 className="flex items-center text-sm font-medium text-amber-800">
            <Icon icon={faWarning} className="mr-2 h-4 w-4 text-amber-500" />
            Configuration Needed
          </h3>
          <p className="mt-1 text-sm text-amber-700">
            This integration hasn&apos;t been configured yet. Please configure
            it first before testing.
          </p>
        </div>
      )}

      {testResult && (
        <div
          className={cn(
            "rounded-md p-4",
            testResult.isValid ? "bg-green-50" : "bg-amber-50",
          )}
        >
          <h3
            className={cn(
              "flex items-center text-sm font-medium",
              testResult.isValid ? "text-green-800" : "text-amber-800",
            )}
          >
            <Icon
              icon={testResult.isValid ? faCheckCircle : faWarning}
              className={cn(
                "mr-2 h-4 w-4",
                testResult.isValid ? "text-green-500" : "text-amber-500",
              )}
            />
            {testResult.isValid ? "Test Successful" : "Test Result"}
          </h3>
          <p
            className={cn(
              "mt-1 text-sm",
              testResult.isValid ? "text-green-700" : "text-amber-700",
            )}
          >
            {testResult.message}
          </p>
        </div>
      )}

      <div className="flex justify-end space-x-2 pt-4">
        <Button variant="outline" onClick={onClose}>
          Close
        </Button>
        <Button
          onClick={onTestClick}
          disabled={isTestLoading || !hasConfiguration}
        >
          {isTestLoading ? "Testing..." : "Test Connection"}
        </Button>
      </div>
    </div>
  );
}

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
