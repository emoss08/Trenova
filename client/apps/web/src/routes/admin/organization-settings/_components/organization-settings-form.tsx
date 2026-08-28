import { UsStateAutocompleteField } from "@/components/autocomplete-fields";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { FormSaveDock } from "@/components/form-save-dock";
import { ImageCropUploadDialog } from "@/components/image-crop-upload-dialog";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@trenova/shared/components/ui/card";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import {
  SegmentedControl,
  type SegmentedControlItem,
} from "@trenova/shared/components/ui/segmented-control";
import { Tabs, TabsContent, TabsList, TabsTab } from "@trenova/shared/components/ui/tabs";
import { useOptimisticMutation } from "@/hooks/use-optimistic-mutation";
import {
  organizationSettingsTabParser,
  type OrganizationSettingsTabValue,
} from "@/hooks/use-organization-setting-state";
import { timezoneGroupedChoices } from "@/lib/choices";
import { validateCroppableImage } from "@/lib/images/crop-image";
import { IMAGE_UPLOAD_ACCEPT, organizationLogoCropConfig } from "@/lib/images/upload-config";
import { updateOrganizationSettingsGraphQL } from "@/lib/graphql/organization";
import { queries } from "@/lib/queries";
import { isAbsoluteUrl } from "@trenova/shared/lib/utils";
import { apiService } from "@/services/api";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import {
  organizationCapabilityPresetValues,
  resolveOrganizationCapabilityPreset,
  type OrganizationCapabilityPreset,
} from "@trenova/shared/types/organization-capability";
import {
  organizationSettingsSchema,
  type OrganizationSettings,
} from "@trenova/shared/types/organization";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Building2Icon, CircleXIcon, CreditCardIcon, ShieldIcon, UploadIcon } from "lucide-react";
import { useQueryState } from "nuqs";
import type { ChangeEvent } from "react";
import { Activity, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { FormProvider, useForm, useFormContext, useWatch, type Resolver } from "react-hook-form";
import { toast } from "sonner";
import { BillingUsageTab } from "./billing-usage-tab";
import { SecurityAccessWorkspace } from "./security-access-workspace";

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
  brokerageEnabled: true,
  assetOperationsEnabled: true,
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
    resolver: zodResolver(organizationSettingsSchema) as Resolver<OrganizationSettings>,
    defaultValues: emptyOrganizationDefaults,
  });

  const [tab, setTab] = useQueryState("tab", organizationSettingsTabParser);

  const { handleSubmit, reset } = form;

  useEffect(() => {
    if (organizationQuery.data) {
      reset(organizationQuery.data);
    }
  }, [organizationQuery.data, reset]);

  const { mutateAsync: updateOrganization } = useOptimisticMutation({
    queryKey: queries.organization.detail(organizationId).queryKey,
    mutationFn: (values: OrganizationSettings) =>
      updateOrganizationSettingsGraphQL(organizationId, values),
    resourceName: "Organization Settings",
    resetForm: reset,
    form,
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
      <div className="text-muted-foreground py-8 text-sm">
        No active organization found for this session.
      </div>
    );
  }

  if (organizationQuery.isLoading) {
    return (
      <div className="text-muted-foreground py-8 text-sm">Loading organization settings...</div>
    );
  }

  return (
    <Tabs
      value={tab}
      onValueChange={(value) => void setTab(value as OrganizationSettingsTabValue)}
      className="gap-1 px-4"
    >
      <TabsList variant="underline">
        <TabsTab value="general">
          <Building2Icon size={16} />
          General
        </TabsTab>
        <TabsTab value="security">
          <ShieldIcon size={16} />
          Security
        </TabsTab>
        <TabsTab value="billing-usage">
          <CreditCardIcon size={16} />
          Billing & Usage
        </TabsTab>
      </TabsList>
      <TabsContent value="general" className="pb-10">
        <Activity mode={tab === "general" ? "visible" : "hidden"}>
          <FormProvider {...form}>
            <Form className="flex flex-col gap-4" onSubmit={handleSubmit(onSubmit)}>
              <LogoForm organizationId={organizationId} onLogoUpdated={handleLogoUpdated} />
              <GeneralForm />
              <OperatingModelForm />
              <ComplianceForm />
              <AddressForm />
              <FormSaveDock saveButtonContent="Save Changes" />
            </Form>
          </FormProvider>
        </Activity>
      </TabsContent>
      <TabsContent value="security">
        <Activity mode={tab === "security" ? "visible" : "hidden"}>
          <SecurityAccessWorkspace organizationId={organizationId} />
        </Activity>
      </TabsContent>
      <TabsContent value="billing-usage" className="pb-10">
        <Activity mode={tab === "billing-usage" ? "visible" : "hidden"}>
          <BillingUsageTab />
        </Activity>
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
  const fileInputRef = useRef<HTMLInputElement | null>(null);
  const [isRemovingLogo, setIsRemovingLogo] = useState(false);
  const [pendingFile, setPendingFile] = useState<File | null>(null);
  const [isCropOpen, setIsCropOpen] = useState(false);
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

    if (isAbsoluteUrl(rawLogoValue)) {
      return rawLogoValue;
    }

    return null;
  }, [rawLogoValue, resolvedLogoURL]);

  const handleLogoFileChange = useCallback((event: ChangeEvent<HTMLInputElement>) => {
    const selectedFile = event.target.files?.[0];
    event.target.value = "";
    if (!selectedFile) {
      return;
    }

    try {
      validateCroppableImage(selectedFile, "logos");
      setPendingFile(selectedFile);
      setIsCropOpen(true);
    } catch (error) {
      toast.error("Unsupported logo file", {
        description:
          error instanceof Error ? error.message : "Please choose a JPG, PNG, or WEBP file.",
      });
    }
  }, []);

  const handleLogoUpload = useCallback(
    async (file: File) => {
      const updatedOrganization = await apiService.organizationService.uploadLogo(
        organizationId,
        file,
      );
      await onLogoUpdated(updatedOrganization);
      toast.success("Organization logo updated");
    },
    [onLogoUpdated, organizationId],
  );

  const handleRemoveLogo = useCallback(async () => {
    if (isRemovingLogo) {
      return;
    }

    setIsRemovingLogo(true);
    try {
      const updated = await apiService.organizationService.deleteLogo(organizationId);
      await onLogoUpdated(updated);
      toast.success("Organization logo removed");
    } catch (error) {
      const message = error instanceof Error ? error.message : "Failed to remove logo";
      toast.error("Failed to remove logo", { description: message });
    }

    setIsRemovingLogo(false);
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
          <div className="bg-muted/40 relative flex h-24 w-24 items-center justify-center rounded-md border">
            {displayLogoURL ? (
              <img
                src={displayLogoURL}
                alt="Organization logo"
                className="h-full w-full rounded-md object-cover"
              />
            ) : (
              <span className="text-muted-foreground text-xs">No logo</span>
            )}
            {displayLogoURL ? (
              <button
                type="button"
                onClick={handleRemoveLogo}
                disabled={isRemovingLogo}
                className="bg-background/95 text-foreground hover:bg-muted absolute top-0 right-0 z-10 inline-flex size-6 translate-x-1/2 -translate-y-1/2 cursor-pointer items-center justify-center rounded-full shadow-xs transition-colors disabled:cursor-not-allowed disabled:opacity-50"
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
            onClick={() => fileInputRef.current?.click()}
            disabled={isRemovingLogo}
          >
            <UploadIcon className="size-4" />
            Upload Logo
          </Button>
          <input
            ref={fileInputRef}
            type="file"
            accept={IMAGE_UPLOAD_ACCEPT}
            className="hidden"
            onChange={handleLogoFileChange}
          />
        </div>
      </CardContent>

      <ImageCropUploadDialog
        open={isCropOpen}
        file={pendingFile}
        title="Crop Organization Logo"
        description="Adjust the visible bounds before uploading your organization logo."
        {...organizationLogoCropConfig}
        confirmLabel="Upload Logo"
        onClose={() => {
          setIsCropOpen(false);
          setPendingFile(null);
        }}
        onConfirm={handleLogoUpload}
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
              rules={{ required: true }}
              name="timezone"
              label="Timezone"
              placeholder="Select timezone"
              groups={timezoneGroupedChoices}
              // isReadOnly={isDisabled}
              renderOption={(option) => (
                <span className="flex w-full items-center justify-between gap-3">
                  <span>{option.label}</span>
                  {option.description && (
                    <span className="text-muted-foreground text-xs">{option.description}</span>
                  )}
                </span>
              )}
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

const OPERATING_MODEL_PRESETS: SegmentedControlItem<OrganizationCapabilityPreset>[] = [
  { value: "asset", label: "Asset", caption: "Runs its own fleet" },
  { value: "brokerage", label: "Brokerage", caption: "Buys capacity" },
  { value: "hybrid", label: "Hybrid", caption: "Both" },
];

function OperatingModelForm() {
  const { control, setValue } = useFormContext<OrganizationSettings>();
  const brokerageEnabled = useWatch({ control, name: "brokerageEnabled" });
  const assetOperationsEnabled = useWatch({ control, name: "assetOperationsEnabled" });

  const preset = useMemo(
    () => resolveOrganizationCapabilityPreset({ brokerageEnabled, assetOperationsEnabled }),
    [brokerageEnabled, assetOperationsEnabled],
  );

  const applyPreset = useCallback(
    (value: OrganizationCapabilityPreset) => {
      // "custom" describes a combination the presets do not cover; it is never
      // offered as a segment, only reflected when neither preset matches.
      if (value === "custom") {
        return;
      }

      const values = organizationCapabilityPresetValues(value);
      setValue("brokerageEnabled", values.brokerageEnabled, {
        shouldDirty: true,
        shouldTouch: true,
      });
      setValue("assetOperationsEnabled", values.assetOperationsEnabled, {
        shouldDirty: true,
        shouldTouch: true,
      });
    },
    [setValue],
  );

  return (
    <Card>
      <CardHeader>
        <CardTitle>Operating Model</CardTitle>
        <CardDescription>
          Tailors the menus to the freight this organization actually moves. This controls
          visibility only — it hides features from menus and navigation. It does not restrict
          permissions or API access, and it never changes existing records.
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <div className="flex flex-col gap-4">
          <div className="flex flex-col gap-1.5">
            <span className="text-sm font-medium">Preset</span>
            <SegmentedControl
              aria-label="Operating model preset"
              items={OPERATING_MODEL_PRESETS}
              value={preset}
              onValueChange={applyPreset}
              fullWidth
            />
            <span className="text-2xs text-muted-foreground">
              A preset simply sets the two switches below. Adjust either one on its own for a
              combination the presets do not cover.
            </span>
          </div>
          <FormGroup cols={1}>
            <FormControl>
              <SwitchField
                control={control}
                name="brokerageEnabled"
                label="Brokerage Features"
                description="Shows carriers, routing guides, tendering, and carrier settlements. Turning this off hides them from menus and navigation; it does not restrict permissions or API access."
              />
            </FormControl>
            <FormControl>
              <SwitchField
                control={control}
                name="assetOperationsEnabled"
                label="Asset Operations"
                description="Marks this organization as running its own fleet. Recorded for reporting today — it does not hide anything yet."
              />
            </FormControl>
          </FormGroup>
        </div>
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
