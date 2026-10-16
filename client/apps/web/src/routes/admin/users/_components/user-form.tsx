import { useT } from "@trenova/shared/i18n/use-t";
import { describeApiError } from "@/lib/api-error-message";
import { RoleAutocompleteField } from "@/components/autocomplete-fields";
import { InputField, InputFieldSkeleton } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField, SwitchFieldSkeleton } from "@/components/fields/switch-field";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { Checkbox } from "@trenova/shared/components/ui/checkbox";
import { FormControl, FormGroup, FormSection } from "@trenova/shared/components/ui/form";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { LOCALE_NAMES, LOCALES } from "@trenova/shared/i18n/generated/locales";
import { statusChoices, timezoneGroupedChoices } from "@/lib/choices";
import { apiService } from "@/services/api";
import type { UserOrganization } from "@trenova/shared/types/organization";
import type {
  UserCreateFormValues,
  UserEditFormValues,
  UserOrganizationMembership,
} from "@trenova/shared/types/user";
import { Mail01Icon, SpinnerIcon } from "@trenova/shared/components/icons";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useFormContext, useWatch } from "react-hook-form";
import { toast } from "@trenova/shared/components/ui/toast";
import { UserAccessSection } from "./user-access-section";
import { UserRolesEditor } from "./user-roles-editor";

type UserFormProps =
  | { mode: "create" }
  | {
      mode: "edit";
      userId: string;
      emailAddress: string;
      isActive: boolean;
      awaitingSetup: boolean;
    };

export function UserForm(props: UserFormProps) {
  const t = useT();

  const { control } = useFormContext<UserCreateFormValues | UserEditFormValues>();
  const isEdit = props.mode === "edit";

  const timeFormatOptions = useMemo(
    () => [
      { value: "12-hour", label: t("12-hour") },
      { value: "24-hour", label: t("24-hour") },
    ],
    [t],
  );
  const localeOptions = useMemo(
    () => LOCALES.map((locale) => ({ value: locale, label: LOCALE_NAMES[locale] })),
    [],
  );

  return (
    <FormGroup cols={1}>
      <FormControl>
        <SelectField
          control={control}
          rules={{ required: true }}
          name="status"
          label={t("Status")}
          options={statusChoices}
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          rules={{ required: true }}
          name="name"
          label={t("Full name")}
          placeholder={t("Enter their full name")}
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          rules={{ required: true }}
          name="username"
          label={t("Username")}
          placeholder="johndoe"
          disabled={isEdit}
          description={isEdit ? t("Username cannot be changed") : undefined}
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          rules={{ required: true }}
          name="emailAddress"
          label={t("Email address")}
          type="email"
          placeholder={t("john@example.com")}
          description={isEdit ? undefined : t("They sign in with this address.")}
        />
      </FormControl>
      <FormControl>
        <SelectField
          control={control}
          rules={{ required: true }}
          name="timezone"
          label={t("Timezone")}
          placeholder={t("Select timezone")}
          groups={timezoneGroupedChoices}
          renderOption={(option) => (
            <span className="flex w-full items-center justify-between gap-3">
              <span>{t(option.label)}</span>
              {option.description && (
                <span className="text-muted-foreground text-xs">{t(option.description)}</span>
              )}
            </span>
          )}
        />
      </FormControl>
      <FormControl>
        <SelectField
          control={control}
          rules={{ required: true }}
          name="timeFormat"
          label={t("Time format")}
          options={timeFormatOptions}
        />
      </FormControl>
      <FormControl>
        <SelectField
          control={control}
          rules={{ required: true }}
          name="locale"
          label={t("Language")}
          description={t("Menus, labels and the emails Trenova sends them.")}
          options={localeOptions}
        />
      </FormControl>
      {props.mode === "create" ? (
        <CreateOnlyFields />
      ) : (
        <>
          <FormControl>
            <InputField control={control} name="jobTitle" label={t("Job title")} />
          </FormControl>
          <FormControl>
            <InputField control={control} name="phoneNumber" label={t("Phone number")} type="tel" />
          </FormControl>
          <FormControl>
            <SwitchField
              position="left"
              outlined
              control={control}
              name="isLocked"
              label={t("Account locked")}
              description={t("A locked account cannot sign in until it is unlocked.")}
            />
          </FormControl>
          <FormControl>
            <SwitchField
              position="left"
              outlined
              control={control}
              name="mustChangePassword"
              label={t("Require a password change at next sign-in")}
              description={t("They choose a new password before they can continue.")}
            />
          </FormControl>
          <UserRolesEditor userId={props.userId} />
          <OrganizationMembershipSection userId={props.userId} isDisabled={false} />
          <UserAccessSection
            userId={props.userId}
            emailAddress={props.emailAddress}
            isActive={props.isActive}
            awaitingSetup={props.awaitingSetup}
          />
        </>
      )}
    </FormGroup>
  );
}

function CreateOnlyFields() {
  const t = useT();
  const { control } = useFormContext<UserCreateFormValues>();
  const emailAddress = useWatch({ control, name: "emailAddress" });

  return (
    <>
      <FormControl>
        <RoleAutocompleteField
          control={control}
          name="roleIds"
          label={t("Roles")}
          description={t("You can only grant access you hold yourself.")}
          placeholder={t("Select roles")}
        />
      </FormControl>
      <FormControl>
        <Alert variant="info" size="sm">
          <Mail01Icon />
          <AlertTitle>{t("They set their own password")}</AlertTitle>
          <AlertDescription>
            {emailAddress?.trim()
              ? t(
                  "{0} will be emailed a link to choose a password and finish setting up the account. You never see or set it.",
                  emailAddress.trim(),
                )
              : t(
                  "The new user will be emailed a link to choose a password and finish setting up the account. You never see or set it.",
                )}
          </AlertDescription>
        </Alert>
      </FormControl>
    </>
  );
}

export function UserFormSkeleton() {
  return (
    <div className="flex flex-col gap-4" aria-busy="true">
      {Array.from({ length: 7 }, (_, index) => (
        <InputFieldSkeleton key={index} />
      ))}
      <SwitchFieldSkeleton />
      <SwitchFieldSkeleton />
    </div>
  );
}

function OrganizationMembershipSection({
  userId,
  isDisabled,
}: {
  userId: string;
  isDisabled: boolean;
}) {
  const t = useT();

  const [availableOrganizations, setAvailableOrganizations] = useState<UserOrganization[]>([]);
  const [selectedOrgIDs, setSelectedOrgIDs] = useState<string[]>([]);
  const [defaultOrganizationID, setDefaultOrganizationID] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const loadMembershipData = useCallback(async () => {
    setIsLoading(true);
    setError(null);

    const [organizations, memberships] = await Promise.all([
      apiService.userService.getUserOrganizations(),
      apiService.userService.listOrganizationMemberships(userId),
    ]).catch((err) => {
      setError("Unable to load organization access.");
      console.error("Failed to load organization memberships", err);
      setIsLoading(false);
      return [null, null] as const;
    });

    if (!organizations || !memberships) {
      return;
    }

    setAvailableOrganizations(organizations);
    setSelectedOrgIDs(memberships.map((membership) => membership.organizationId));
    setDefaultOrganizationID(getDefaultOrganizationID(memberships));
    setIsLoading(false);
  }, [userId]);

  useEffect(() => {
    const timer = window.setTimeout(() => {
      void loadMembershipData();
    }, 0);

    return () => window.clearTimeout(timer);
  }, [loadMembershipData]);

  const orgByID = useMemo(() => {
    return new Map(availableOrganizations.map((org) => [org.id, org]));
  }, [availableOrganizations]);

  const toggleOrganization = useCallback((organizationID: string) => {
    setSelectedOrgIDs((current) =>
      current.includes(organizationID)
        ? current.filter((id) => id !== organizationID)
        : [...current, organizationID],
    );
  }, []);

  const saveMemberships = useCallback(async () => {
    setIsSaving(true);
    setError(null);

    const updatedMemberships = await apiService.userService
      .replaceOrganizationMemberships(userId, {
        organizationIds: selectedOrgIDs,
      })
      .catch((err) => {
        setError(describeApiError(err, t("Unable to save organization access.")));
        console.error("Failed to save organization memberships", err);
        setIsSaving(false);
        return null;
      });

    if (!updatedMemberships) {
      return;
    }

    setSelectedOrgIDs(updatedMemberships.map((membership) => membership.organizationId));
    setDefaultOrganizationID(getDefaultOrganizationID(updatedMemberships));

    toast.success(t("Organization access updated"));
    setIsSaving(false);
  }, [selectedOrgIDs, userId, t]);

  return (
    <FormControl>
      <FormSection
        className="border-border gap-3 rounded-lg border p-4"
        title={t("Organization access")}
        description={t(
          "Choose which organizations this user can access in the current business unit.",
        )}
      >
        {isLoading ? (
          <div className="text-muted-foreground flex items-center gap-2 text-sm">
            <SpinnerIcon className="size-4 animate-spin" />
            {t("Loading organizations...")}
          </div>
        ) : availableOrganizations.length === 0 ? (
          <p className="text-muted-foreground text-xs">
            {t("No organizations are available for assignment.")}
          </p>
        ) : (
          <ScrollArea className="border-border/60 bg-muted flex max-h-42 flex-col gap-2 rounded-md border p-1">
            {availableOrganizations.map((organization) => (
              <label
                key={organization.id}
                className="hover:bg-accent/40 flex cursor-pointer items-start gap-2 rounded-md p-2 text-sm"
              >
                <Checkbox
                  checked={selectedOrgIDs.includes(organization.id)}
                  onCheckedChange={() => toggleOrganization(organization.id)}
                  disabled={isDisabled || isSaving}
                />
                <span className="inline-flex flex-wrap items-center gap-2">
                  <span>{organization.name}</span>
                  {organization.id === defaultOrganizationID && (
                    <span className="bg-muted text-xs text-muted-foreground rounded-md px-1.5 py-0.5">
                      default
                    </span>
                  )}
                  {(organization.city || organization.state) && (
                    <span className="text-muted-foreground text-xs">
                      {organization.city}
                      {organization.city && organization.state ? ", " : ""}
                      {organization.state}
                    </span>
                  )}
                </span>
              </label>
            ))}
          </ScrollArea>
        )}

        {selectedOrgIDs.length > 0 && defaultOrganizationID && (
          <p className="text-muted-foreground text-xs">
            {t(
              "Default organization: {0}",
              orgByID.get(defaultOrganizationID)?.name ?? defaultOrganizationID,
            )}
          </p>
        )}

        {error && <p className="text-destructive text-xs">{error}</p>}

        <div className="flex justify-end">
          <Button
            type="button"
            variant="outline"
            onClick={saveMemberships}
            disabled={isDisabled || isLoading || isSaving || selectedOrgIDs.length === 0}
            isLoading={isSaving}
            loadingText={t("Saving...")}
          >
            {t("Save organization access")}
          </Button>
        </div>
      </FormSection>
    </FormControl>
  );
}

function getDefaultOrganizationID(memberships: UserOrganizationMembership[]): string | null {
  const defaultMembership = memberships.find((membership) => membership.isDefault);
  return defaultMembership?.organizationId ?? null;
}
