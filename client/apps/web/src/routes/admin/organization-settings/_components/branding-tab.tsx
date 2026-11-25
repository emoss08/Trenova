import { CropResult } from "@trenova/shared/components/image-upload/crop-result";
import { MAX_IMAGE_MEGABYTES } from "@trenova/shared/components/image-upload/image-file";
import type { ImageSpec } from "@trenova/shared/components/image-upload/image-spec";
import { useImageUpload } from "@trenova/shared/components/image-upload/use-image-upload";
import {
  SettingsReadout,
  SettingsSection,
  SettingsSections,
} from "@/components/settings/settings-section";
import { useOrganizationBranding } from "@/hooks/use-organization-branding";
import { organization } from "@/lib/queries/organization";
import { userOrganization } from "@/lib/queries/user";
import { organizationService } from "@/services/organization";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import { describeError } from "@trenova/shared/lib/error-presentation";
import { mailingAddressLines } from "@trenova/shared/lib/mailing-address";
import type {
  OrganizationBrandImage,
  OrganizationSettings,
} from "@trenova/shared/types/organization";
import { useRef, useState, type ReactNode } from "react";
import { useFormContext, useWatch } from "react-hook-form";
import { BrandingPreview, type BrandImageSlot, type BrandMark } from "./branding-preview";

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

type BrandingUpdated = (updatedOrganization: OrganizationSettings) => Promise<void>;

/**
 * Takes in an organization a picture upload or removal returned. A picture is saved the
 * moment it is chosen, not with the form, and the save moves the organization's version on:
 * the form takes the new record under whatever the person has changed and not saved yet, so
 * those edits stay and the next save is not refused as out of date.
 */
function useBrandingUpdated(organizationId: string): BrandingUpdated {
  const queryClient = useQueryClient();
  const { reset } = useFormContext<OrganizationSettings>();

  return async (updatedOrganization) => {
    queryClient.setQueryData(organization.detail(organizationId).queryKey, updatedOrganization);
    reset(updatedOrganization, { keepDirtyValues: true, keepDirty: true });
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: organization.branding(organizationId).queryKey }),
      queryClient.invalidateQueries({ queryKey: userOrganization.all._def }),
    ]);
  };
}

/**
 * One of the organization's pictures as the preview needs it: where it is read from, the crop
 * flow that replaces it and the request that removes it. What goes wrong with either is told
 * to the tab, which says it under the preview.
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
  onUpdated: BrandingUpdated;
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

/**
 * The organization's logo and icon, changed where they are seen: the logo heads the sample
 * invoice and the icon marks the sidebar, and pressing either replaces or removes it. A
 * picture is saved as soon as it is cropped, apart from the form's save.
 */
export function BrandingTab({ organizationId }: { organizationId: string }) {
  const t = useT();
  const [problem, setProblem] = useState<string | null>(null);
  const onUpdated = useBrandingUpdated(organizationId);

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
    onUpdated,
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
    onUpdated,
    onProblem: setProblem,
    onLoadFailed: refreshBranding,
  });

  return (
    <SettingsSections>
      <SettingsSection
        title={t("Logo and icon")}
        description={t(
          "Your icon marks this organization in the sidebar, and your logo heads the documents and emails you send. Select either in the preview to change it; it is saved as soon as you place it.",
        )}
      >
        <SettingsReadout>
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
        </SettingsReadout>
      </SettingsSection>
      {logo.element}
      {icon.element}
    </SettingsSections>
  );
}
