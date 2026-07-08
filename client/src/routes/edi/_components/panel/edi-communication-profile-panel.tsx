import {
  EDIConnectionAutocompleteField,
  EDIPartnerAutocompleteField,
  OrganizationAutocompleteField,
} from "@/components/autocomplete-fields";
import { DataTablePanelContainer } from "@/components/data-table/data-table-panel";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { Button } from "@/components/ui/button";
import { Form, FormControl, FormGroup, FormSection } from "@/components/ui/form";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { statusChoices } from "@/lib/choices";
import { apiService } from "@/services/api";
import type { DataTablePanelProps } from "@/types/data-table";
import type { EDICommunicationProfile, EDIConnectionTestResult } from "@/types/edi";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { KeyRoundIcon, RadioTowerIcon, ServerIcon, ShieldCheckIcon } from "lucide-react";
import { useEffect } from "react";
import { useForm, useWatch } from "react-hook-form";
import { toast } from "sonner";
import {
  communicationProfileFormSchema,
  communicationProfileMethodOptions,
  type CommunicationProfileFormValues,
} from "../edi-schemas";
import {
  SecretProfileFields,
  TransportProfileFields,
  X12EnvelopeFields,
} from "./edi-communication-profile-fields";
import {
  getProfileFormDefaults,
  toCommunicationProfileRequest,
} from "./edi-communication-profile-form";
import { invalidateEDICommunicationProfiles } from "./edi-panel-invalidation";
import { EDIEmptyState } from "./edi-panel-primitives";

function notifyConnectionTestResult(result: EDIConnectionTestResult) {
  const describe = (statuses: string[]) =>
    result.checks
      .filter((check) => statuses.includes(check.status))
      .map((check) => `${check.name}: ${check.message ?? check.status}`)
      .join("\n");
  if (result.success) {
    const warnings = describe(["warning"]);
    if (warnings) {
      toast.warning("Connection test passed with warnings", { description: warnings });
      return;
    }
    toast.success("Connection test passed", { description: describe(["passed"]) });
    return;
  }
  toast.error("Connection test failed", { description: describe(["failed", "warning"]) });
}

export function CommunicationProfilePanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<EDICommunicationProfile>) {
  const queryClient = useQueryClient();
  const profile = mode === "edit" ? row : null;
  const isEdit = !!profile;
  const form = useForm<CommunicationProfileFormValues>({
    resolver: zodResolver(communicationProfileFormSchema),
    defaultValues: getProfileFormDefaults(profile),
    mode: "onChange",
  });
  const { control, handleSubmit, reset, setError } = form;
  const method = useWatch({ control, name: "method" });
  const authMode = useWatch({ control, name: "config.authMode" });

  const mutation = useApiMutation({
    mutationFn: (values: CommunicationProfileFormValues) => {
      const request = toCommunicationProfileRequest(values, profile);
      return isEdit
        ? apiService.ediService.updateCommunicationProfile(profile.id, request)
        : apiService.ediService.createCommunicationProfile(request);
    },
    setFormError: setError,
    resourceName: "Communication Profile",
    onSuccess: async () => {
      toast.success(isEdit ? "Communication profile updated" : "Communication profile created");
      onOpenChange(false);
      await invalidateEDICommunicationProfiles(queryClient);
    },
  });

  useEffect(() => {
    if (!open) return;
    reset(getProfileFormDefaults(profile));
  }, [open, profile, reset]);

  const testConnectionMutation = useMutation({
    mutationFn: () => {
      if (!profile) throw new Error("Profile is required");
      return apiService.ediService.testProfileConnection(profile.id);
    },
    onSuccess: (result) => notifyConnectionTestResult(result),
    onError: () => toast.error("The connection test could not be run"),
  });
  const isDirty = form.formState.isDirty;
  const canTestConnection = isEdit && method !== "Internal";

  return (
    <DataTablePanelContainer
      open={open}
      onOpenChange={onOpenChange}
      title={isEdit ? "Edit Communication Profile" : "New Communication Profile"}
      description="Configure the transport profile and envelope values used for this organization."
      size="xl"
      footer={
        <div className="flex w-full items-center justify-end gap-2">
          {canTestConnection && (
            <Button
              type="button"
              variant="outline"
              className="mr-auto"
              onClick={() => testConnectionMutation.mutate()}
              isLoading={testConnectionMutation.isPending}
              disabled={isDirty}
              title={
                isDirty
                  ? "Save your changes before testing the connection"
                  : "Verify certificates, credentials, and endpoint reachability"
              }
            >
              Test Connection
            </Button>
          )}
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button type="submit" form="edi-profile-form" isLoading={mutation.isPending}>
            Save Profile
          </Button>
        </div>
      }
    >
      <Form
        id="edi-profile-form"
        className="min-h-0"
        onSubmit={(event) => {
          event.stopPropagation();
          void handleSubmit((values) => mutation.mutate(values))(event);
        }}
      >
        <Tabs defaultValue="overview" className="gap-3">
          <TabsList variant="underline" className="grid w-full grid-cols-4 border-b border-border">
            <TabsTrigger value="overview">
              <ServerIcon data-icon="inline-start" />
              Overview
            </TabsTrigger>
            <TabsTrigger value="transport">
              <RadioTowerIcon data-icon="inline-start" />
              Transport
            </TabsTrigger>
            <TabsTrigger value="envelope">
              <ShieldCheckIcon data-icon="inline-start" />
              Envelope
            </TabsTrigger>
            <TabsTrigger value="secrets">
              <KeyRoundIcon data-icon="inline-start" />
              Secrets
            </TabsTrigger>
          </TabsList>
          <TabsContent value="overview" className="space-y-3">
            <FormSection title="Profile Identity" className="rounded-md border bg-muted/20 p-3">
              <FormGroup cols={2}>
                <FormControl>
                  <InputField
                    control={control}
                    name="name"
                    label="Name"
                    placeholder="Profile name"
                    rules={{ required: true }}
                  />
                </FormControl>
                <FormControl>
                  <SelectField
                    control={control}
                    name="method"
                    label="Method"
                    options={communicationProfileMethodOptions}
                    rules={{ required: true }}
                  />
                </FormControl>
                <FormControl>
                  <SelectField
                    control={control}
                    name="status"
                    label="Status"
                    options={statusChoices}
                    rules={{ required: true }}
                  />
                </FormControl>
                <FormControl>
                  <EDIPartnerAutocompleteField
                    control={control}
                    name="ediPartnerId"
                    label="Partner"
                    placeholder="Select partner"
                    description="Trading partner this transport profile delivers documents for."
                    clearable
                  />
                </FormControl>
                <FormControl cols="full">
                  <TextareaField
                    control={control}
                    name="description"
                    label="Description"
                    placeholder="Operational notes for this profile"
                  />
                </FormControl>
              </FormGroup>
            </FormSection>
            {method === "Internal" && (
              <FormSection title="Internal Routing" className="rounded-md border bg-muted/20 p-3">
                <FormGroup cols={2}>
                  <FormControl>
                    <EDIConnectionAutocompleteField
                      control={control}
                      name="ediConnectionId"
                      label="Connection"
                      placeholder="Select connection"
                      description="Accepted organization connection this profile routes through."
                      clearable
                    />
                  </FormControl>
                  <FormControl>
                    <OrganizationAutocompleteField
                      control={control}
                      name="config.connectedOrganizationId"
                      label="Connected Organization"
                      placeholder="Select organization"
                      description="Organization that receives documents delivered over this profile."
                      clearable
                    />
                  </FormControl>
                </FormGroup>
              </FormSection>
            )}
          </TabsContent>
          <TabsContent value="transport" className="space-y-3">
            <TransportProfileFields control={control} method={method} authMode={authMode} />
          </TabsContent>
          <TabsContent value="envelope" className="space-y-3">
            {method === "Internal" ? (
              <EDIEmptyState message="Internal profiles use organization routing and do not require X12 interchange identifiers." />
            ) : (
              <X12EnvelopeFields control={control} />
            )}
          </TabsContent>
          <TabsContent value="secrets" className="space-y-3">
            <SecretProfileFields
              control={control}
              method={method}
              profile={profile}
              authMode={authMode}
            />
          </TabsContent>
        </Tabs>
      </Form>
    </DataTablePanelContainer>
  );
}
