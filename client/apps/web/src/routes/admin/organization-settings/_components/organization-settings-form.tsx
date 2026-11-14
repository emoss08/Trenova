import { useT } from "@trenova/shared/i18n/use-t";
import { UsStateAutocompleteField } from "@/components/autocomplete-fields";
import { InputField } from "@/components/fields/input-field";
import { TimezoneField } from "@/components/fields/timezone-field";
import { SwitchField } from "@/components/fields/switch-field";
import { FormSaveDock } from "@/components/form-save-dock";
import { CropResult } from "@/components/image-upload/crop-result";
import type { ImageSpec } from "@/components/image-upload/image-spec";
import { MAX_IMAGE_MEGABYTES } from "@/components/image-upload/image-file";
import { useImageUpload } from "@/components/image-upload/use-image-upload";
import { useOptimisticMutation } from "@/hooks/use-optimistic-mutation";
import { useOrganizationBranding } from "@/hooks/use-organization-branding";
import {
  organizationSettingsTabParser,
  type OrganizationSettingsTabValue,
} from "@/hooks/use-organization-setting-state";
import { updateOrganizationSettingsGraphQL } from "@/lib/graphql/organization";
import { organization } from "@/lib/queries/organization";
import { userOrganization } from "@/lib/queries/user";
import { organizationService } from "@/services/organization";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery, useQueryClient, useSuspenseQuery } from "@tanstack/react-query";
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
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import {
  organizationSettingsSchema,
  type OrganizationBrandImage,
  type OrganizationSettings,
} from "@trenova/shared/types/organization";
import {
  organizationCapabilityPresetValues,
  resolveOrganizationCapabilityPreset,
  type OrganizationCapabilityPreset,
} from "@trenova/shared/types/organization-capability";
import { Building07Icon, Shield01Icon } from "@trenova/shared/components/icons";
import { describeError } from "@trenova/shared/lib/error-presentation";
import { mailingAddressLines } from "@trenova/shared/lib/mailing-address";
import { useQueryState } from "nuqs";
import { Activity, useCallback, useMemo, useRef, useState, type ReactNode } from "react";
import { FormProvider, useForm, useFormContext, useWatch, type Resolver } from "react-hook-form";
import { BrandingPreview, type BrandImageSlot, type BrandMark } from "./branding-preview";
import { SecurityAccessWorkspace } from "./security-access-workspace";

export default function OrganizationSettingsForm() {
  const t = useT();

  const organizationId = useAuthStore((state) => state.user?.currentOrganizationId);

  if (!organizationId) {
    return (
      <div className="py-8 text-sm text-muted-foreground">
        {t("No active organization found for this session.")}
      </div>
    );
  }

  return <OrganizationSettingsTabs organizationId={organizationId} />;
}

function OrganizationSettingsTabs({ organizationId }: { organizationId: string }) {
  const t = useT();

  const queryClient = useQueryClient();

  const { data } = useSuspenseQuery(organization.detail(organizationId));

  const form = useForm<OrganizationSettings>({
    resolver: zodResolver(organizationSettingsSchema) as Resolver<OrganizationSettings>,
    defaultValues: data,
    values: data,
  });

  const [tab, setTab] = useQueryState("tab", organizationSettingsTabParser);

  const { handleSubmit, reset } = form;

  const { mutateAsync: updateOrganization } = useOptimisticMutation({
    queryKey: organization.detail(organizationId).queryKey,
    mutationFn: (values: OrganizationSettings) =>
      updateOrganizationSettingsGraphQL(organizationId, values),
    resourceName: "Organization Settings",
    resetForm: reset,
    form,
    invalidateQueries: [
      organization.detail._def,
      organization.branding._def,
      userOrganization.all._def,
    ],
  });

  const handleBrandingUpdated = useCallback(
    async (updatedOrganization: OrganizationSettings) => {
      queryClient.setQueryData(organization.detail(organizationId).queryKey, updatedOrganization);
      reset(updatedOrganization);
      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: organization.branding(organizationId).queryKey,
        }),
        queryClient.invalidateQueries({ queryKey: userOrganization.all._def }),
      ]);
    },
    [organizationId, queryClient, reset],
  );

  const onSubmit = useCallback(
    async (values: OrganizationSettings) => {
      await updateOrganization(values);
    },
    [updateOrganization],
  );

  return (
    <Tabs
      value={tab}
      onValueChange={(value) => void setTab(value as OrganizationSettingsTabValue)}
      className="gap-4"
    >
      <TabsList variant="underline">
        <TabsTab value="general">
          <Building07Icon size={16} />
          {t("General")}
        </TabsTab>
        <TabsTab value="security">
          <Shield01Icon size={16} />
          {t("Security")}
        </TabsTab>
      </TabsList>
      <TabsContent value="general" className="pb-10">
        <Activity mode={tab === "general" ? "visible" : "hidden"}>
          <FormProvider {...form}>
            <Form className="flex flex-col gap-4" onSubmit={handleSubmit(onSubmit)}>
              <BrandingForm
                organizationId={organizationId}
                onBrandingUpdated={handleBrandingUpdated}
              />
              <GeneralForm />
              <OperatingModelForm />
              <ComplianceForm />
              <AddressForm />
              <FormSaveDock saveButtonContent={t("Save changes")} />
            </Form>
          </FormProvider>
        </Activity>
      </TabsContent>
      <TabsContent value="security">
        <Activity mode={tab === "security" ? "visible" : "hidden"}>
          <SecurityAccessWorkspace organizationId={organizationId} />
        </Activity>
      </TabsContent>
    </Tabs>
  );
}

/**
 * The logo is the organization's full mark, fitted whole at the top of what it sends, so it
 * keeps its own shape. It may sit inside the frame with clear ground around it.
 */
const ORGANIZATION_LOGO: ImageSpec = {
  mask: "rect",
  frame: "source",
  fit: "contain",
  maxDimension: 1024,
};

/** The icon is the square mark the sidebar's tile draws, where a logo has no room. */
const ORGANIZATION_ICON: ImageSpec = {
  mask: "tile",
  frame: "square",
  fit: "contain",
  maxDimension: 512,
};

/**
 * One of the organization's pictures as the preview needs it: where it is read from, the crop
 * flow that replaces it and the request that removes it. What goes wrong with either is told
 * to the card, which says it under the preview.
 */
function useBrandImage({
  organizationId,
  image,
  spec,
  url,
  labels,
  preview,
  onUpdated,
  onProblem,
  onLoadFailed,
}: {
  organizationId: string;
  image: OrganizationBrandImage;
  spec: ImageSpec;
  /** Where the stored picture is read from; null when there is none. */
  url: string | null;
  labels: { upload: string; change: string; cropTitle: string };
  /** The preview shown beside the crop dialog's stage. */
  preview: ReactNode;
  onUpdated: (updatedOrganization: OrganizationSettings) => Promise<void>;
  onProblem: (problem: string | null) => void;
  /** Told the address of a stored picture that did not load. */
  onLoadFailed: (url: string) => void;
}) {
  const t = useT();
  const removal = useRef(false);
  const [removing, setRemoving] = useState(false);

  const upload = useImageUpload({
    spec,
    preview,
    title: labels.cropTitle,
    confirmLabel: labels.upload,
    disabled: removing,
    onUpload: async (file) => {
      await onUpdated(await organizationService.uploadBrandImage(organizationId, image, file));
    },
    onProblem,
  });

  const remove = async () => {
    if (removal.current) return;
    removal.current = true;
    setRemoving(true);
    onProblem(null);
    try {
      await onUpdated(await organizationService.deleteBrandImage(organizationId, image));
    } catch (error) {
      onProblem(describeError(error).detail ?? t("The picture could not be removed. Try again."));
    } finally {
      removal.current = false;
      setRemoving(false);
    }
  };

  const slot: BrandImageSlot = {
    mark: storedMark(url, onLoadFailed),
    control: {
      uploadLabel: labels.upload,
      changeLabel: labels.change,
      onChoose: upload.openPicker,
      onIntent: upload.preload,
      onRemove: () => void remove(),
      removing,
      dropTarget: upload.dropTarget,
      dragging: upload.dragging,
    },
  };

  return { slot, element: upload.element };
}

/**
 * A stored picture, drawn from its signed address. The address lapses, so a picture that
 * fails to load asks for a fresh one.
 */
function storedMark(
  url: string | null,
  onLoadFailed: (url: string) => void,
): BrandMark | undefined {
  return url
    ? (className) => (
        <img
          src={url}
          alt=""
          draggable={false}
          className={className}
          onError={() => onLoadFailed(url)}
        />
      )
    : undefined;
}

const LIVE_MARK: BrandImageSlot = { mark: (className) => <CropResult className={className} /> };

function BrandingForm({
  organizationId,
  onBrandingUpdated,
}: {
  organizationId: string;
  onBrandingUpdated: (updatedOrganization: OrganizationSettings) => Promise<void>;
}) {
  const t = useT();
  const [problem, setProblem] = useState<string | null>(null);

  const { control } = useFormContext<OrganizationSettings>();
  const [name, line1, line2, city, postalCode, storedLogo, storedIcon] = useWatch({
    control,
    name: ["name", "addressLine1", "addressLine2", "city", "postalCode", "logoUrl", "iconUrl"],
  });
  const { data: detail } = useQuery(organization.detail(organizationId));

  const { branding, refresh: refreshBranding } = useOrganizationBranding(
    organizationId,
    Boolean(storedLogo || storedIcon),
  );

  // The form says which pictures exist; the branding read only says where to fetch them, and
  // may still hold the address of one that was just removed.
  const logoURL = storedLogo ? branding?.logoUrl || null : null;
  const iconURL = storedIcon ? branding?.iconUrl || null : null;

  const lines = mailingAddressLines({
    line1,
    line2,
    city,
    state: detail?.state?.abbreviation,
    postalCode,
    country: detail?.state?.countryName,
  });
  const shown = (url: string | null): BrandImageSlot => ({
    mark: storedMark(url, refreshBranding),
  });

  const logo = useBrandImage({
    organizationId,
    image: "logo",
    spec: ORGANIZATION_LOGO,
    url: logoURL,
    labels: {
      upload: t("Upload logo"),
      change: t("Change logo"),
      cropTitle: t("Position your logo"),
    },
    // An invoice carries the logo; so does the sidebar, until the organization has an icon.
    preview: (
      <BrandingPreview
        className="flex-1"
        name={name}
        lines={lines}
        logo={LIVE_MARK}
        icon={shown(iconURL)}
        surfaces={iconURL ? ["invoice"] : ["invoice", "sidebar"]}
      />
    ),
    onUpdated: onBrandingUpdated,
    onProblem: setProblem,
    onLoadFailed: refreshBranding,
  });

  const icon = useBrandImage({
    organizationId,
    image: "icon",
    spec: ORGANIZATION_ICON,
    url: iconURL,
    labels: {
      upload: t("Upload icon"),
      change: t("Change icon"),
      cropTitle: t("Position your icon"),
    },
    preview: (
      <BrandingPreview
        className="flex-1"
        name={name}
        lines={lines}
        logo={shown(logoURL)}
        icon={LIVE_MARK}
        surfaces={["sidebar"]}
      />
    ),
    onUpdated: onBrandingUpdated,
    onProblem: setProblem,
    onLoadFailed: refreshBranding,
  });

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("Organization branding")}</CardTitle>
        <CardDescription>
          {t(
            "Your icon marks this organization in the sidebar, and your logo heads the documents and emails you send. Select either in the preview to change it.",
          )}
        </CardDescription>
      </CardHeader>
      <CardContent className="flex max-w-4xl flex-col gap-2">
        <BrandingPreview
          className="h-80"
          name={name}
          lines={lines}
          logo={logo.slot}
          icon={icon.slot}
        />
        {problem ? (
          <p role="alert" className="text-xs text-destructive">
            {problem}
          </p>
        ) : (
          <p className="text-xs text-foreground-subtle">
            {iconURL
              ? t(
                  "PNG, JPG, WebP or SVG, up to {0} MB. You can also drop a file on either.",
                  MAX_IMAGE_MEGABYTES,
                )
              : t(
                  "PNG, JPG, WebP or SVG, up to {0} MB. Until you add an icon, the sidebar fits your logo in its place.",
                  MAX_IMAGE_MEGABYTES,
                )}
          </p>
        )}
        {logo.element}
        {icon.element}
      </CardContent>
    </Card>
  );
}

function GeneralForm() {
  const t = useT();

  const { control } = useFormContext<OrganizationSettings>();

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("Organization details")}</CardTitle>
        <CardDescription>
          {t(
            "Core business identifiers and operational settings that define your organization profile in the system.",
          )}
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <FormGroup cols={1}>
          <FormControl>
            <InputField
              control={control}
              name="name"
              required
              label={t("Name")}
              placeholder={t("Enter organization name")}
            />
          </FormControl>
          <FormControl cols="full">
            <TimezoneField
              control={control}
              required
              name="timezone"
              label={t("Timezone")}
              placeholder={t("Select timezone")}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="loginSlug"
              label={t("Tenant login slug")}
              placeholder="acme-logistics"
              description={t("Used for tenant sign-in URLs such as /login/acme-logistics.")}
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
  const t = useT();

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
        <CardTitle>{t("Operating model")}</CardTitle>
        <CardDescription>
          {t(
            "Tailors the menus to the freight this organization actually moves. This controls visibility only — it hides features from menus and navigation. It does not restrict permissions or API access, and it never changes existing records.",
          )}
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <div className="flex flex-col gap-4">
          <div className="flex flex-col gap-1.5">
            <span className="text-sm font-medium">{t("Preset")}</span>
            <SegmentedControl
              aria-label={t("Operating model preset")}
              items={OPERATING_MODEL_PRESETS}
              value={preset}
              onValueChange={applyPreset}
              fullWidth
            />
            <span className="text-2xs text-muted-foreground">
              {t(
                "A preset simply sets the two switches below. Adjust either one on its own for a combination the presets do not cover.",
              )}
            </span>
          </div>
          <FormGroup cols={1}>
            <FormControl>
              <SwitchField
                control={control}
                name="brokerageEnabled"
                label={t("Brokerage features")}
                description={t(
                  "Shows carriers, routing guides, tendering, and carrier settlements. Turning this off hides them from menus and navigation; it does not restrict permissions or API access.",
                )}
              />
            </FormControl>
            <FormControl>
              <SwitchField
                control={control}
                name="assetOperationsEnabled"
                label={t("Asset operations")}
                description={t(
                  "Marks this organization as running its own fleet. Recorded for reporting today — it does not hide anything yet.",
                )}
              />
            </FormControl>
          </FormGroup>
        </div>
      </CardContent>
    </Card>
  );
}

function ComplianceForm() {
  const t = useT();

  const { control } = useFormContext<OrganizationSettings>();

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("Regulatory compliance")}</CardTitle>
        <CardDescription>
          {t("Regulatory identifiers required for operations and reporting.")}
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <FormGroup cols={1}>
          <FormControl>
            <InputField
              control={control}
              name="scacCode"
              required
              label={t("SCAC code")}
              placeholder={t("Enter SCAC code")}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="dotNumber"
              required
              label={t("DOT number")}
              placeholder={t("Enter DOT number")}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="taxId"
              label={t("Tax ID")}
              placeholder={t("Enter tax ID")}
            />
          </FormControl>
        </FormGroup>
      </CardContent>
    </Card>
  );
}

function AddressForm() {
  const t = useT();

  const { control } = useFormContext<OrganizationSettings>();

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("Registered address")}</CardTitle>
        <CardDescription>
          {t("Legal headquarters location used for correspondence and compliance.")}
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <FormGroup cols={2}>
          <FormControl cols="full">
            <InputField
              control={control}
              name="addressLine1"
              required
              label={t("Address Line 1")}
              placeholder={t("Enter address")}
            />
          </FormControl>
          <FormControl cols="full">
            <InputField
              control={control}
              name="addressLine2"
              label={t("Suite/unit")}
              placeholder={t("Enter suite or unit number")}
            />
          </FormControl>
          <FormControl cols={1}>
            <InputField
              control={control}
              name="city"
              required
              label={t("City")}
              placeholder={t("Enter city")}
            />
          </FormControl>
          <FormControl cols={1}>
            <UsStateAutocompleteField
              control={control}
              name="stateId"
              required
              label={t("State")}
              placeholder={t("State")}
            />
          </FormControl>
          <FormControl cols={2}>
            <InputField
              control={control}
              name="postalCode"
              required
              label={t("ZIP code")}
              placeholder={t("Enter ZIP code")}
            />
          </FormControl>
        </FormGroup>
      </CardContent>
    </Card>
  );
}
