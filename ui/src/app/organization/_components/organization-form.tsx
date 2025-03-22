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
import { useSuspenseQuery } from "@tanstack/react-query";
import { useEffect } from "react";
import { FormProvider, useFormContext } from "react-hook-form";

export default function OrganizationForm() {
  const user = useUser();

  // Get the organization data
  const userOrg = useSuspenseQuery({
    ...queries.organization.getOrgById(user?.currentOrganizationId ?? ""),
  });

  // Set up the form with save functionality
  const form = useFormWithSave({
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

  const {
    reset,
    handleSubmit,
    formState: { isDirty, isSubmitting },
    onSubmit,
  } = form;

  // Load organization data into the form when available
  useEffect(() => {
    if (userOrg.data && !userOrg.isLoading) {
      reset(userOrg.data);
    }
  }, [userOrg.data, userOrg.isLoading, reset]);

  return (
    <FormProvider {...form}>
      <Form onSubmit={handleSubmit(onSubmit)}>
        <div className="flex flex-col gap-4 pb-10">
          <GeneralForm />
          <ComplianceForm />
          <AddressForm />
          <FormSaveDock isDirty={isDirty} isSubmitting={isSubmitting} />
        </div>
      </Form>
    </FormProvider>
  );
}

function GeneralForm() {
  const { control } = useFormContext();

  return (
    <Card>
      <CardHeader>
        <CardTitle>Organization Details</CardTitle>
        <CardDescription>
          Core business identifiers and operational settings that define your
          transportation company&apos;s profile in the system
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
              description="Legal business name as registered with regulatory authorities"
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
              description="Primary timezone for operations, scheduling, and reporting activities"
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="id"
              label="Organization ID"
              readOnly
              placeholder="Enter organization ID"
              description="System-generated unique identifier for this organization"
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="businessUnitId"
              label="Parent Business Unit ID"
              readOnly
              placeholder="Enter parent business unit ID"
              description="Identifier for associated parent business unit in the corporate hierarchy"
            />
          </FormControl>
        </FormGroup>
      </CardContent>
    </Card>
  );
}

function ComplianceForm() {
  const { control } = useFormContext();

  return (
    <Card>
      <CardHeader>
        <CardTitle>Regulatory Compliance</CardTitle>
        <CardDescription>
          Essential regulatory identifiers required for interstate commerce, EDI
          transactions, and legal compliance reporting
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <FormGroup cols={1}>
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
              description="Business model classification: Asset-based, Brokerage, or Hybrid operations"
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="scacCode"
              label="SCAC Code"
              rules={{ required: true }}
              placeholder="Enter SCAC code"
              description="Standard Carrier Alpha Code for EDI transactions and carrier identification"
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="dotNumber"
              label="DOT Number"
              rules={{ required: true }}
              placeholder="Enter DOT number"
              description="USDOT number required for interstate commerce authorization"
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="taxId"
              label="Tax ID"
              placeholder="Enter Tax ID"
              description="Federal Employer Identification Number (FEIN) or equivalent tax identifier"
            />
          </FormControl>
        </FormGroup>
      </CardContent>
    </Card>
  );
}

function AddressForm() {
  const { control } = useFormContext();

  // Get state options for the form
  const usStates = useSuspenseQuery({
    ...queries.usState.options(),
  });
  const usStateOptions = usStates.data?.results ?? [];

  return (
    <Card>
      <CardHeader>
        <CardTitle>Registered Address</CardTitle>
        <CardDescription>
          Legal headquarters location used for official correspondence,
          regulatory filings, and service area determination
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
              description="Primary business location for official correspondence"
            />
          </FormControl>
          <FormControl cols="full">
            <InputField
              control={control}
              name="addressLine2"
              label="Suite/Unit"
              placeholder="Enter suite or unit number"
              description="Additional location details (suite, floor, building identifier)"
            />
          </FormControl>
          <FormControl cols={1}>
            <InputField
              control={control}
              rules={{ required: true }}
              name="city"
              label="City"
              placeholder="Enter city"
              description="City of registered business operations"
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
              description="State jurisdiction for business operations and taxation"
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
              description="Postal code for location-based services and correspondence"
            />
          </FormControl>
        </FormGroup>
      </CardContent>
    </Card>
  );
}
