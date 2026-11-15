import { UsStateAutocompleteField } from "@/components/autocomplete-fields";
import { InputField } from "@/components/fields/input-field";
import { TimezoneField } from "@/components/fields/timezone-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { useSignOut } from "@/hooks/use-sign-out";
import { updateOrganizationSettingsGraphQL } from "@/lib/graphql/organization";
import { organization } from "@/lib/queries/organization";
import { userOrganization } from "@/lib/queries/user";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { toast } from "@trenova/shared/components/ui/toast";
import { translate } from "@trenova/shared/i18n/runtime";
import { useT } from "@trenova/shared/i18n/use-t";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import type { OrganizationSettings } from "@trenova/shared/types/organization";
import type { OrganizationSetup } from "@trenova/shared/types/user";
import { useState } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { z } from "zod";
import { AuthErrorText, AuthSubmit } from "./auth-field";
import { AuthHeading, AuthQuietButton, StepCrumbs } from "./auth-primitives";

export const organizationSetupSchema = z.object({
  name: z
    .string()
    .trim()
    .min(1, { error: () => translate("Enter your company name") })
    .max(100, { error: () => translate("Company name must be 100 characters or fewer") }),
  timezone: z.string().min(1, { error: () => translate("Select a timezone") }),
  addressLine1: z
    .string()
    .trim()
    .min(1, { error: () => translate("Enter a street address") })
    .max(150, { error: () => translate("Address must be 150 characters or fewer") }),
  addressLine2: z
    .string()
    .trim()
    .max(150, {
      error: () => translate("Address must be 150 characters or fewer"),
    }),
  city: z
    .string()
    .trim()
    .min(1, { error: () => translate("Enter a city") })
    .max(100, { error: () => translate("City must be 100 characters or fewer") }),
  stateId: z.string().min(1, { error: () => translate("Select a state") }),
  postalCode: z
    .string()
    .trim()
    .min(1, { error: () => translate("Enter a ZIP code") }),
  scacCode: z
    .string()
    .trim()
    .regex(/^[A-Za-z]{4}$/, { error: () => translate("A SCAC code is four letters") }),
  dotNumber: z
    .string()
    .trim()
    .regex(/^\d{1,8}$/, { error: () => translate("A DOT number is 1 to 8 digits") }),
});

export type OrganizationSetupValues = z.infer<typeof organizationSetupSchema>;

const SETUP_FIELDS = [
  "name",
  "timezone",
  "addressLine1",
  "city",
  "stateId",
  "postalCode",
  "scacCode",
  "dotNumber",
] as const satisfies readonly (keyof OrganizationSetupValues)[];

/**
 * What the form opens with: the organization as stored, with every field the server named
 * as missing left empty, so a stand-in the install wrote (a placeholder city or SCAC) is
 * never offered back as though somebody had chosen it.
 */
export function organizationSetupDefaults(
  stored: OrganizationSettings,
  missing: readonly string[],
): OrganizationSetupValues {
  const values: OrganizationSetupValues = {
    name: stored.name ?? "",
    timezone: stored.timezone ?? "",
    addressLine1: stored.addressLine1 ?? "",
    addressLine2: stored.addressLine2 ?? "",
    city: stored.city ?? "",
    stateId: stored.stateId ?? "",
    postalCode: stored.postalCode ?? "",
    scacCode: stored.scacCode ?? "",
    dotNumber: stored.dotNumber ?? "",
  };
  for (const field of SETUP_FIELDS) {
    if (missing.includes(field)) {
      values[field] = "";
    }
  }
  return values;
}

/** The organization as the existing update takes it: what is stored, with the step's answers. */
export function organizationWithSetup(
  stored: OrganizationSettings,
  values: OrganizationSetupValues,
): OrganizationSettings {
  return {
    ...stored,
    name: values.name,
    timezone: values.timezone,
    addressLine1: values.addressLine1,
    addressLine2: values.addressLine2,
    city: values.city,
    stateId: values.stateId,
    postalCode: values.postalCode,
    scacCode: values.scacCode.toUpperCase(),
    dotNumber: values.dotNumber,
  };
}

/**
 * The step a session meets while its organization lacks the details the product needs. An
 * administrator fills them in here; anybody else is told who can.
 */
export function OrganizationSetupStep({ setup }: { setup: OrganizationSetup }) {
  if (!setup.canComplete) {
    return <OrganizationSetupBlocked />;
  }
  return <OrganizationSetupLoader missing={setup.missing} />;
}

function OrganizationSetupBlocked() {
  const t = useT();
  const signOut = useSignOut();

  return (
    <>
      <StepCrumbs left={t("Organization setup")} right={t("Waiting on an administrator")} />
      <AuthHeading title={t("Your organization is not set up yet")}>
        {t(
          "An administrator needs to finish setting up your organization before you can use Trenova. Ask them to sign in and complete it, then sign in again.",
        )}
      </AuthHeading>
      <AuthSubmit onClick={() => void signOut()} trailingIcon={null}>
        {t("Sign out")}
      </AuthSubmit>
    </>
  );
}

function OrganizationSetupLoader({ missing }: { missing: readonly string[] }) {
  const t = useT();
  const organizationId = useAuthStore((state) => state.user?.currentOrganizationId) ?? "";
  const query = useQuery({ ...organization.detail(organizationId), enabled: !!organizationId });

  if (query.isError) {
    return (
      <>
        <StepCrumbs left={t("Organization setup")} right={t("Company details")} />
        <AuthHeading title={t("Your organization could not be loaded")}>
          {t("Trenova could not read your organization's details. Try again in a moment.")}
        </AuthHeading>
        <AuthSubmit onClick={() => void query.refetch()} busy={query.isFetching}>
          {t("Try again")}
        </AuthSubmit>
      </>
    );
  }

  if (!query.data) {
    return <OrganizationSetupSkeleton />;
  }

  return <OrganizationSetupForm stored={query.data} missing={missing} />;
}

export function OrganizationSetupSkeleton() {
  return (
    <div className="flex flex-col gap-4" aria-busy="true">
      <Skeleton className="h-3 w-40" />
      <Skeleton className="h-7 w-64" />
      <Skeleton className="h-10 w-full" />
      {SETUP_FIELDS.map((field) => (
        <div key={field} className="flex flex-col gap-1.5">
          <Skeleton className="h-3 w-24" />
          <Skeleton className="h-8 w-full" />
        </div>
      ))}
      <Skeleton className="h-11 w-full" />
    </div>
  );
}

function OrganizationSetupForm({
  stored,
  missing,
}: {
  stored: OrganizationSettings;
  missing: readonly string[];
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const signOut = useSignOut();
  const [confirmFailed, setConfirmFailed] = useState(false);
  const [confirming, setConfirming] = useState(false);

  const form = useForm<OrganizationSetupValues>({
    resolver: zodResolver(organizationSetupSchema),
    defaultValues: organizationSetupDefaults(stored, missing),
  });
  const { control, handleSubmit, formState } = form;
  const rootError = formState.errors.root?.message;

  const { mutateAsync, isPending } = useApiMutation({
    mutationFn: (values: OrganizationSetupValues) =>
      updateOrganizationSettingsGraphQL(stored.id, organizationWithSetup(stored, values)),
    form,
    resourceName: "Organization",
    onSuccess: async (saved) => {
      queryClient.setQueryData(organization.detail(stored.id).queryKey, saved);
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: organization.branding(stored.id).queryKey }),
        queryClient.invalidateQueries({ queryKey: userOrganization.all._def }),
      ]);
      await confirmSetUp();
    },
  });

  const confirmSetUp = async () => {
    setConfirming(true);
    try {
      const session = await useAuthStore.getState().checkSession();
      if (session.status !== "signed_in") {
        setConfirmFailed(true);
        return;
      }
      setConfirmFailed(false);
      toast.success(t("Your organization is set up"));
    } finally {
      setConfirming(false);
    }
  };

  return (
    <>
      <StepCrumbs left={t("Organization setup")} right={t("Company details")} />
      <AuthHeading title={t("Set up your organization")}>
        {t(
          "Trenova prints these on your invoices, rate confirmations and other documents. You can change them later in organization settings.",
        )}
      </AuthHeading>

      <FormProvider {...form}>
        <form
          className="flex flex-col gap-4"
          noValidate
          onSubmit={handleSubmit((values) => void mutateAsync(values))}
        >
          <InputField
            control={control}
            name="name"
            required
            label={t("Company name")}
            placeholder={t("Enter organization name")}
            disabled={isPending}
          />
          <TimezoneField
            control={control}
            name="timezone"
            required
            label={t("Timezone")}
            placeholder={t("Select timezone")}
          />
          <InputField
            control={control}
            name="addressLine1"
            required
            label={t("Address Line 1")}
            placeholder={t("Enter address")}
            disabled={isPending}
          />
          <InputField
            control={control}
            name="addressLine2"
            label={t("Suite/unit")}
            placeholder={t("Enter suite or unit number")}
            disabled={isPending}
          />
          <div className="grid grid-cols-2 gap-3">
            <InputField
              control={control}
              name="city"
              required
              label={t("City")}
              placeholder={t("Enter city")}
              disabled={isPending}
            />
            <UsStateAutocompleteField
              control={control}
              name="stateId"
              required
              label={t("State")}
              placeholder={t("State")}
            />
          </div>
          <InputField
            control={control}
            name="postalCode"
            required
            label={t("ZIP code")}
            placeholder={t("Enter ZIP code")}
            disabled={isPending}
          />
          <div className="grid grid-cols-2 gap-3">
            <InputField
              control={control}
              name="scacCode"
              required
              label={t("SCAC code")}
              placeholder="ACME"
              maxLength={4}
              disabled={isPending}
            />
            <InputField
              control={control}
              name="dotNumber"
              required
              label={t("DOT number")}
              placeholder="1234567"
              inputMode="numeric"
              maxLength={8}
              disabled={isPending}
            />
          </div>
          {rootError ? <AuthErrorText>{rootError}</AuthErrorText> : null}
          {confirmFailed ? (
            <>
              <AuthErrorText>
                {t(
                  "Your details are saved, but Trenova could not reach the server to continue. Check your connection and try again.",
                )}
              </AuthErrorText>
              <AuthSubmit onClick={() => void confirmSetUp()} busy={confirming}>
                {t("Continue")}
              </AuthSubmit>
            </>
          ) : (
            <AuthSubmit type="submit" busy={isPending || confirming} busyLabel={t("Saving")}>
              {t("Save and continue")}
            </AuthSubmit>
          )}
        </form>
      </FormProvider>
      <div className="mt-4">
        <AuthQuietButton onClick={() => void signOut()}>{t("Sign out")}</AuthQuietButton>
      </div>
    </>
  );
}
