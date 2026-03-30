import { UsStateAutocompleteField } from "@/components/autocomplete-fields";
import { formatFileSize, type RejectedFile } from "@/components/documents/document-upload-zone";
import { UploadPanel } from "@/components/documents/upload-panel";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { FormSaveDock } from "@/components/form-save-dock";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Form, FormControl, FormGroup } from "@/components/ui/form";
import { Tabs, TabsContent, TabsList, TabsTab } from "@/components/ui/tabs";
import { useOptimisticMutation } from "@/hooks/use-optimistic-mutation";
import { useUploadWithProgress } from "@/hooks/use-upload-with-progress";
import { timezoneChoices } from "@/lib/choices";
import { convertOrganizationLogoToWebP } from "@/lib/images/organization-logo";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import { useAuthStore } from "@/stores/auth-store";
import { organizationSettingsSchema, type OrganizationSettings } from "@/types/organization";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Building2Icon, CircleXIcon, ShieldIcon, UploadIcon } from "lucide-react";
import { parseAsStringLiteral, useQueryState } from "nuqs";
import { useCallback, useEffect, useMemo, useState } from "react";
import { FormProvider, useForm, useFormContext, useWatch } from "react-hook-form";
import { toast } from "sonner";
import { MicrosoftSSOCard } from "./microsoft-sso-card";

const tabValues = ["general", "security"] as const;
const LOGO_ACCEPT = ".jpg,.jpeg,.png,.webp";
const LOGO_MAX_SIZE = 5 * 1024 * 1024;

const emptyOrganizationDefaults: OrganizationSettings = {
  id: "",
  version: 0,
  createdAt: 0,
  updatedAt: 0,
  bucketName: "",
  businessUnitId: "",
  loginSlug: "",
  name: "",
  scacCode: "",
  dotNumber: "",
  logoUrl: "",
  addressLine1: "",
  addressLine2: "",
  city: "",
  stateId: "",
  postalCode: "",
  timezone: "",
  taxId: "",
  state: null,
};

export default function OrganizationSettingsForm() {
  const queryClient = useQueryClient();
  const organizationId = useAuthStore((state) => state.user?.currentOrganizationId) ?? "";

  const organizationQuery = useQuery({
    ...queries.organization.detail(organizationId),
    enabled: Boolean(organizationId),
  });

  const form = useForm<OrganizationSettings>({
    resolver: zodResolver(organizationSettingsSchema),
    defaultValues: emptyOrganizationDefaults,
  });

  const [tab, setTab] = useQueryState(
    "tab",
    parseAsStringLiteral(tabValues)
      .withOptions({ history: "push", shallow: true })
      .withDefault("general"),
  );

  const { handleSubmit, setError, reset } = form;

  useEffect(() => {
    if (organizationQuery.data) {
      reset(organizationQuery.data);
    }
  }, [organizationQuery.data, reset]);

  const { mutateAsync: updateOrganization } = useOptimisticMutation({
    queryKey: queries.organization.detail(organizationId).queryKey,
    mutationFn: (values: OrganizationSettings) =>
      apiService.organizationService.update(organizationId, values),
    resourceName: "Organization Settings",
    resetForm: reset,
    setFormError: setError,
    invalidateQueries: [
      queries.organization.detail._def,
      queries.organization.logo._def,
      queries.userOrganization.all._def,
    ],
  });

  const handleLogoUpdated = useCallback(
    async (updatedOrganization: OrganizationSettings) => {
      queryClient.setQueryData(
        queries.organization.detail(organizationId).queryKey,
        updatedOrganization,
      );
      reset(updatedOrganization);
      const logoQueryKey = queries.organization.logo(organizationId).queryKey;

      if (updatedOrganization.logoUrl) {
        await queryClient.invalidateQueries({
          queryKey: logoQueryKey,
        });
      } else {
        queryClient.removeQueries({
          queryKey: logoQueryKey,
          exact: true,
        });
      }

      await queryClient.invalidateQueries({
        queryKey: queries.userOrganization.all._def,
      });
    },
    [organizationId, queryClient, reset],
  );

  const onSubmit = useCallback(
    async (values: OrganizationSettings) => {
      if (!organizationId) {
        return;
      }

      await updateOrganization(values);
    },
    [organizationId, updateOrganization],
  );

  if (!organizationId) {
    return (
      <div className="py-8 text-sm text-muted-foreground">
        No active organization found for this session.
      </div>
    );
  }

  if (organizationQuery.isLoading) {
    return (
      <div className="py-8 text-sm text-muted-foreground">Loading organization settings...</div>
    );
  }

  return (
    <Tabs value={tab} onValueChange={(value) => setTab(value)} className="gap-1">
      <TabsList variant="underline">
        <TabsTab value="general">
          <Building2Icon size={16} />
          General
        </TabsTab>
        <TabsTab value="security">
          <ShieldIcon size={16} />
          Security
        </TabsTab>
      </TabsList>
      <TabsContent value="general" className="pb-10">
        <FormProvider {...form}>
          <Form className="flex flex-col gap-4" onSubmit={handleSubmit(onSubmit)}>
            <LogoForm organizationId={organizationId} onLogoUpdated={handleLogoUpdated} />
            <GeneralForm />
            <ComplianceForm />
            <AddressForm />
            <FormSaveDock saveButtonContent="Save Changes" />
          </Form>
        </FormProvider>
      </TabsContent>
      <TabsContent value="security">
        <MicrosoftSSOCard organizationId={organizationId} />
      </TabsContent>
    </Tabs>
  );
}

function LogoForm({
  organizationId,
  onLogoUpdated,
}: {
  organizationId: string;
  onLogoUpdated: (updatedOrganization: OrganizationSettings) => Promise<void>;
}) {
  const [isUploadOpen, setIsUploadOpen] = useState(false);
  const [isRemovingLogo, setIsRemovingLogo] = useState(false);
  const { control } = useFormContext<OrganizationSettings>();
  const rawLogoValue = useWatch({ control, name: "logoUrl" });

  const { data: resolvedLogoURL } = useQuery({
    ...queries.organization.logo(organizationId),
    enabled: Boolean(rawLogoValue),
    retry: false,
  });

  const displayLogoURL = useMemo(() => {
    if (resolvedLogoURL) {
      return resolvedLogoURL;
    }

    if (
      rawLogoValue &&
      (rawLogoValue.startsWith("http://") || rawLogoValue.startsWith("https://"))
    ) {
      return rawLogoValue;
    }

    return null;
  }, [rawLogoValue, resolvedLogoURL]);

  const {
    uploads,
    uploadFiles,
    cancelUpload,
    retryUpload,
    removeUpload,
    clearCompleted,
    isUploading,
  } = useUploadWithProgress({
    resourceId: organizationId,
    resourceType: "organization-logo",
    maxConcurrent: 1,
    uploadEndpoint: `/organizations/${organizationId}/logo`,
    parseResponse: (response) => organizationSettingsSchema.parse(response as OrganizationSettings),
    invalidateQueryKey: queries.organization.detail(organizationId).queryKey,
    transformFile: (file) => convertOrganizationLogoToWebP(file),
    onSuccess: async (result) => {
      await onLogoUpdated(result as OrganizationSettings);
      toast.success("Organization logo updated");
    },
    onError: (error, file) => {
      toast.error(`Failed to upload ${file.name}`, {
        description: error.message,
      });
    },
  });

  const handleFilesRejected = useCallback((rejectedFiles: RejectedFile[]) => {
    rejectedFiles.forEach(({ file, reason }) => {
      if (reason === "size") {
        toast.error(`File too large: ${file.name}`, {
          description: `Maximum file size is 5MB. This file is ${formatFileSize(file.size)}.`,
        });
        return;
      }

      if (reason === "type") {
        toast.error(`Unsupported file type: ${file.name}`, {
          description: "Only JPG, PNG, and WEBP files are supported.",
        });
      }
    });
  }, []);

  const handleRemoveLogo = useCallback(async () => {
    if (isRemovingLogo) {
      return;
    }

    try {
      setIsRemovingLogo(true);
      const updated = await apiService.organizationService.deleteLogo(organizationId);
      await onLogoUpdated(updated);
      toast.success("Organization logo removed");
    } catch (error) {
      const message = error instanceof Error ? error.message : "Failed to remove logo";
      toast.error("Failed to remove logo", { description: message });
    } finally {
      setIsRemovingLogo(false);
    }
  }, [isRemovingLogo, onLogoUpdated, organizationId]);

  return (
    <Card>
      <CardHeader>
        <CardTitle>Organization Branding</CardTitle>
        <CardDescription>
          Upload and manage your organization logo used across the application.
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <div className="space-y-4">
          <div className="relative flex h-24 w-24 items-center justify-center rounded-md border bg-muted/40">
            {displayLogoURL ? (
              <img
                src={displayLogoURL}
                alt="Organization logo"
                className="h-full w-full rounded-md object-cover"
              />
            ) : (
              <span className="text-xs text-muted-foreground">No logo</span>
            )}
            {displayLogoURL ? (
              <button
                type="button"
                onClick={handleRemoveLogo}
                disabled={isRemovingLogo || isUploading}
                className="absolute top-0 right-0 z-10 inline-flex size-6 translate-x-1/2 -translate-y-1/2 cursor-pointer items-center justify-center rounded-full bg-background/95 text-foreground shadow-xs transition-colors hover:bg-muted disabled:cursor-not-allowed disabled:opacity-50"
                aria-label="Remove logo"
                title="Remove logo"
              >
                <CircleXIcon className="size-4" />
              </button>
            ) : null}
          </div>

          <Button
            type="button"
            variant="secondary"
            onClick={() => setIsUploadOpen(true)}
            disabled={isUploading || isRemovingLogo}
          >
            <UploadIcon className="size-4" />
            Upload Logo
          </Button>
        </div>
      </CardContent>

      <UploadPanel
        isOpen={isUploadOpen}
        onClose={() => setIsUploadOpen(false)}
        uploads={uploads}
        onFilesSelected={uploadFiles}
        onFilesRejected={handleFilesRejected}
        onCancel={cancelUpload}
        onRetry={retryUpload}
        onRemove={removeUpload}
        onClearCompleted={clearCompleted}
        disabled={isUploading || isRemovingLogo}
        title="Upload Organization Logo"
        accept={LOGO_ACCEPT}
        maxFileSize={LOGO_MAX_SIZE}
        multiple={false}
        supportedFormatsLabel="JPG, PNG, WEBP"
        maxFileSizeLabel="5 MB"
      />
    </Card>
  );
}

function GeneralForm() {
  const { control } = useFormContext<OrganizationSettings>();

  return (
    <Card>
      <CardHeader>
        <CardTitle>Organization Details</CardTitle>
        <CardDescription>
          Core business identifiers and operational settings that define your organization profile
          in the system.
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <FormGroup cols={1}>
          <FormControl>
            <InputField
              control={control}
              name="name"
              rules={{ required: true }}
              label="Name"
              placeholder="Enter organization name"
            />
          </FormControl>
          <FormControl cols="full">
            <SelectField
              control={control}
              name="timezone"
              rules={{ required: true }}
              label="Operating Timezone"
              placeholder="Select timezone"
              options={timezoneChoices}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="loginSlug"
              label="Tenant Login Slug"
              placeholder="acme-logistics"
              description="Used for tenant sign-in URLs such as /login/acme-logistics."
            />
          </FormControl>
        </FormGroup>
      </CardContent>
    </Card>
  );
}

function ComplianceForm() {
  const { control } = useFormContext<OrganizationSettings>();

  return (
    <Card>
      <CardHeader>
        <CardTitle>Regulatory Compliance</CardTitle>
        <CardDescription>
          Regulatory identifiers required for operations and reporting.
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <FormGroup cols={1}>
          <FormControl>
            <InputField
              control={control}
              name="scacCode"
              rules={{ required: true }}
              label="SCAC Code"
              placeholder="Enter SCAC code"
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="dotNumber"
              rules={{ required: true }}
              label="DOT Number"
              placeholder="Enter DOT number"
            />
          </FormControl>
          <FormControl>
            <InputField control={control} name="taxId" label="Tax ID" placeholder="Enter Tax ID" />
          </FormControl>
        </FormGroup>
      </CardContent>
    </Card>
  );
}

function AddressForm() {
  const { control } = useFormContext<OrganizationSettings>();

  return (
    <Card>
      <CardHeader>
        <CardTitle>Registered Address</CardTitle>
        <CardDescription>
          Legal headquarters location used for correspondence and compliance.
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <FormGroup cols={2}>
          <FormControl cols="full">
            <InputField
              control={control}
              name="addressLine1"
              rules={{ required: true }}
              label="Address Line 1"
              placeholder="Enter address"
            />
          </FormControl>
          <FormControl cols="full">
            <InputField
              control={control}
              name="addressLine2"
              label="Suite/Unit"
              placeholder="Enter suite or unit number"
            />
          </FormControl>
          <FormControl cols={1}>
            <InputField
              control={control}
              name="city"
              rules={{ required: true }}
              label="City"
              placeholder="Enter city"
            />
          </FormControl>
          <FormControl cols={1}>
            <UsStateAutocompleteField
              control={control}
              name="stateId"
              rules={{ required: true }}
              label="State"
              placeholder="State"
            />
          </FormControl>
          <FormControl cols={2}>
            <InputField
              control={control}
              name="postalCode"
              rules={{ required: true }}
              label="ZIP Code"
              placeholder="Enter ZIP code"
            />
          </FormControl>
        </FormGroup>
      </CardContent>
    </Card>
  );
}
