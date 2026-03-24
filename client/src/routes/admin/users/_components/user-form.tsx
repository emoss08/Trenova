import { RoleAutocompleteField } from "@/components/autocomplete-fields";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { FormControl, FormGroup } from "@/components/ui/form";
import { ScrollArea } from "@/components/ui/scroll-area";
import { statusChoices, timezoneChoices } from "@/lib/choices";
import { apiService } from "@/services/api";
import type { UserOrganization } from "@/types/organization";
import type { User, UserOrganizationMembership } from "@/types/user";
import { Loader2Icon } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useFormContext } from "react-hook-form";
import { toast } from "sonner";

export function UserForm({
  isEdit,
  isDisabled,
  editUserId,
}: {
  isEdit?: boolean;
  isDisabled?: boolean;
  editUserId?: string | null;
}) {
  const { control } = useFormContext<User>();

  return (
    <FormGroup cols={1}>
      <FormControl>
        <SelectField
          control={control}
          rules={{ required: true }}
          name="status"
          label="Status"
          options={statusChoices}
          isReadOnly={isDisabled}
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          rules={{ required: true }}
          name="name"
          label="Full Name"
          placeholder="Enter your full name"
          disabled={isDisabled}
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          rules={{ required: true }}
          name="username"
          label="Username"
          placeholder="johndoe"
          disabled={isDisabled || isEdit}
          description={isEdit ? "Username cannot be changed" : undefined}
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          rules={{ required: true }}
          name="emailAddress"
          label="Email Address"
          type="email"
          placeholder="john@example.com"
          disabled={isDisabled}
        />
      </FormControl>
      <FormControl>
        <SelectField
          control={control}
          rules={{ required: true }}
          name="timezone"
          label="Timezone"
          placeholder="Select timezone"
          options={timezoneChoices}
          isReadOnly={isDisabled}
        />
      </FormControl>
      <FormControl>
        <SwitchField
          position="left"
          outlined
          control={control}
          rules={{ required: true }}
          name="mustChangePassword"
          label="Require password change on first login"
          description="User will be prompted to set a new password after signing in"
        />
      </FormControl>
      <FormControl>
        <RoleAutocompleteField
          control={control}
          name="assignments"
          label="Roles"
          description="System access permissions and privileges"
          placeholder="Select roles"
        />
      </FormControl>
      {isEdit && editUserId && (
        <OrganizationMembershipSection
          userId={editUserId}
          isDisabled={Boolean(isDisabled)}
        />
      )}
    </FormGroup>
  );
}

function OrganizationMembershipSection({
  userId,
  isDisabled,
}: {
  userId: string;
  isDisabled: boolean;
}) {
  const [availableOrganizations, setAvailableOrganizations] = useState<
    UserOrganization[]
  >([]);
  const [selectedOrgIDs, setSelectedOrgIDs] = useState<string[]>([]);
  const [defaultOrganizationID, setDefaultOrganizationID] = useState<
    string | null
  >(null);
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
    setSelectedOrgIDs(
      memberships.map((membership) => membership.organizationId),
    );
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
        setError("Unable to save organization access.");
        console.error("Failed to save organization memberships", err);
        setIsSaving(false);
        return null;
      });

    if (!updatedMemberships) {
      return;
    }

    setSelectedOrgIDs(
      updatedMemberships.map((membership) => membership.organizationId),
    );
    setDefaultOrganizationID(getDefaultOrganizationID(updatedMemberships));

    toast.success("Organization access updated");
    setIsSaving(false);
  }, [selectedOrgIDs, userId]);

  return (
    <FormControl>
      <div className="space-y-3 rounded-lg border border-border p-4">
        <div className="space-y-1">
          <h4 className="text-sm font-medium">Organization Access</h4>
          <p className="text-xs text-muted-foreground">
            Choose which organizations this user can access in the current
            business unit.
          </p>
        </div>

        {isLoading ? (
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <Loader2Icon className="size-4 animate-spin" />
            Loading organizations...
          </div>
        ) : availableOrganizations.length === 0 ? (
          <p className="text-xs text-muted-foreground">
            No organizations are available for assignment.
          </p>
        ) : (
          <ScrollArea className="flex max-h-42 flex-col gap-2 rounded-md border border-border/60 bg-muted p-1">
            {availableOrganizations.map((organization) => (
              <label
                key={organization.id}
                className="flex cursor-pointer items-start gap-2 rounded-md p-2 text-sm hover:bg-accent/40"
              >
                <Checkbox
                  checked={selectedOrgIDs.includes(organization.id)}
                  onCheckedChange={() => toggleOrganization(organization.id)}
                  disabled={isDisabled || isSaving}
                />
                <span className="inline-flex flex-wrap items-center gap-2">
                  <span>{organization.name}</span>
                  {organization.id === defaultOrganizationID && (
                    <span className="rounded bg-muted px-1.5 py-0.5 text-2xs text-muted-foreground uppercase">
                      default
                    </span>
                  )}
                  {(organization.city || organization.state) && (
                    <span className="text-xs text-muted-foreground">
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
          <p className="text-xs text-muted-foreground">
            Default organization:{" "}
            {orgByID.get(defaultOrganizationID)?.name ?? defaultOrganizationID}
          </p>
        )}

        {error && <p className="text-xs text-destructive">{error}</p>}

        <div className="flex justify-end">
          <Button
            type="button"
            variant="outline"
            onClick={saveMemberships}
            disabled={isDisabled || isLoading || isSaving}
            isLoading={isSaving}
            loadingText="Saving..."
          >
            Save Organization Access
          </Button>
        </div>
      </div>
    </FormControl>
  );
}

function getDefaultOrganizationID(
  memberships: UserOrganizationMembership[],
): string | null {
  const defaultMembership = memberships.find(
    (membership) => membership.isDefault,
  );
  return defaultMembership?.organizationId ?? null;
}
