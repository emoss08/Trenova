// src/pages/organization-form.tsx
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { FormSaveDock } from "@/components/form";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Form, FormControl, FormGroup } from "@/components/ui/form";
import { useFormWithSave } from "@/hooks/use-form-with-save";
import { broadcastQueryInvalidation } from "@/hooks/use-invalidate-query";
import { queries } from "@/lib/queries";
import {
  OrganizationSchema,
  organizationSchema,
} from "@/lib/schemas/organization-schema";
import { TIMEZONES } from "@/lib/timezone/timezone";
import { updateOrganization } from "@/services/organization";
import { useUser } from "@/stores/user-store";
import { OrganizationType } from "@/types/organization";
import { yupResolver } from "@hookform/resolvers/yup";
import { useQuery } from "@tanstack/react-query";
import { useEffect } from "react";

export default function OrganizationForm() {
  const user = useUser();

  // Get the organization data
  const userOrg = useQuery({
    ...queries.organization.getOrgById(user?.currentOrganizationId ?? ""),
  });

  // Get state options for the form
  const usStates = useQuery({
    ...queries.usState.options(),
  });
  const usStateOptions = usStates.data?.results ?? [];

  // Set up the form with save functionality
  const {
    control,
    reset,
    handleSubmit,
    formState: { isDirty, isSubmitting },
    onSubmit,
  } = useFormWithSave({
    resourceName: "Organization",
    formOptions: {
      resolver: yupResolver(organizationSchema),
      defaultValues: {},
      mode: "onChange",
    },
    mutationFn: async (values: OrganizationSchema) => {
      const response = await updateOrganization(
        user?.currentOrganizationId ?? "",
        values,
      );
      return response.data;
    },
    onSuccess: () => {
      broadcastQueryInvalidation({
        queryKey: ["organization", "getUserOrganizations", "getOrgById"],
        options: {
          correlationId: `update-organization-${Date.now()}`,
        },
        config: {
          predicate: true,
          refetchType: "all",
        },
      });
    },
    successMessage: "Changes have been saved",
    successDescription: "Organization updated successfully",
  });

  // Load organization data into the form when available
  useEffect(() => {
    if (userOrg.data && !userOrg.isLoading) {
      reset(userOrg.data);
    }
  }, [userOrg.data, userOrg.isLoading, reset]);

  return (
    <Form onSubmit={handleSubmit(onSubmit)}>
      <div className="flex flex-col gap-4 pb-10">
        <Card>
          <CardHeader>
            <CardTitle>Organization Settings</CardTitle>
            <CardDescription>
              Essential for operational efficiency and compliance, this section
              captures your organization&apos;s core details.
            </CardDescription>
          </CardHeader>
          <CardContent className="max-w-prose">
            <FormGroup cols={1}>
              <FormControl>
                <InputField
                  control={control}
                  name="name"
                  label="Name"
                  placeholder="Enter organization name"
                  description="The legal business name of your transportation company as registered with relevant authorities."
                />
              </FormControl>
              <FormControl>
                <SelectField
                  control={control}
                  name="orgType"
                  label="Organization Type"
                  options={Object.values(OrganizationType).map((type) => ({
                    label: type,
                    value: type,
                  }))}
                  placeholder="Select operation type"
                  description="Defines your operational model: Asset-based (own fleet), Brokerage (freight matching), or Both (combined operations)."
                />
              </FormControl>
              <FormControl>
                <InputField
                  control={control}
                  name="scacCode"
                  label="SCAC Code"
                  rules={{ required: true }}
                  placeholder="Enter SCAC code"
                  description="Your Standard Carrier Alpha Code, a unique identifier required for EDI transactions and shipping documents."
                />
              </FormControl>
              <FormControl>
                <InputField
                  control={control}
                  name="dotNumber"
                  label="DOT Number"
                  rules={{ required: true }}
                  placeholder="Enter DOT number"
                  description="Your USDOT number for interstate operations. Required for carriers and brokers operating across state lines."
                />
              </FormControl>
              <FormControl>
                <InputField
                  control={control}
                  name="taxId"
                  label="Tax ID"
                  placeholder="Enter Tax ID"
                  description="Your Tax Identification Number (TIN) for tax purposes."
                />
              </FormControl>
              <FormControl cols="full">
                <SelectField
                  control={control}
                  name="timezone"
                  options={TIMEZONES.map((timezone) => ({
                    label: timezone.label,
                    value: timezone.value,
                  }))}
                  rules={{ required: true }}
                  label="Operating Timezone"
                  placeholder="Select timezone"
                  description="Primary timezone for dispatch operations, load scheduling, and ETA calculations. Affects all time-based operations in the system."
                />
              </FormControl>
            </FormGroup>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Address Information</CardTitle>
            <CardDescription>
              The address information of where your organization is legally
              registered.
            </CardDescription>
          </CardHeader>
          <CardContent className="max-w-prose">
            <FormGroup cols={2}>
              <FormControl cols="full">
                <InputField
                  control={control}
                  rules={{ required: true }}
                  name="addressLine1"
                  label="Street Address"
                  placeholder="Enter street address"
                  description="Primary business location address for official correspondence and regulatory compliance."
                />
              </FormControl>
              <FormControl cols="full">
                <InputField
                  control={control}
                  name="addressLine2"
                  label="Suite/Unit"
                  placeholder="Enter suite or unit number"
                  description="Additional address details such as suite number, floor, or building identifier."
                />
              </FormControl>
              <FormControl cols={1}>
                <InputField
                  control={control}
                  rules={{ required: true }}
                  name="city"
                  label="City"
                  placeholder="Enter city"
                  description="City of your primary business operations."
                />
              </FormControl>
              <FormControl cols={1}>
                <SelectField
                  control={control}
                  rules={{ required: true }}
                  name="stateId"
                  label="State"
                  placeholder="State"
                  menuPlacement="top"
                  description="The U.S. state where the organization is situated."
                  options={usStateOptions}
                  isLoading={usStates.isLoading}
                  isFetchError={usStates.isError}
                />
              </FormControl>
              <FormControl cols={2}>
                <InputField
                  control={control}
                  rules={{ required: true }}
                  name="postalCode"
                  label="ZIP Code"
                  placeholder="Enter ZIP code"
                  description="Postal code for your business location."
                />
              </FormControl>
            </FormGroup>
          </CardContent>
        </Card>
        <FormSaveDock isDirty={isDirty} isSubmitting={isSubmitting} />
      </div>
    </Form>
  );
}
