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
import { broadcastQueryInvalidation } from "@/hooks/use-invalidate-query";
import { queries } from "@/lib/queries";
import {
  organizationSchema,
  type OrganizationSchema,
} from "@/lib/schemas/organization-schema";
import { TIMEZONES } from "@/lib/timezone/timezone";
import { updateOrganization } from "@/services/organization";
import { useUser } from "@/stores/user-store";
import type { APIError } from "@/types/errors";
import { OrganizationType } from "@/types/organization";
import { zodResolver } from "@hookform/resolvers/zod";
import {
  useMutation,
  useQueryClient,
  useSuspenseQuery,
} from "@tanstack/react-query";
import { useCallback } from "react";
import {
  FormProvider,
  useForm,
  useFormContext,
  type Path,
} from "react-hook-form";
import { toast } from "sonner";

export default function OrganizationForm() {
  const user = useUser();
  const queryClient = useQueryClient();
  const userOrg = useSuspenseQuery({
    ...queries.organization.getOrgById(user?.currentOrganizationId ?? ""),
  });

  const form = useForm({
    resolver: zodResolver(organizationSchema),
    defaultValues: userOrg.data,
  });

  const {
    handleSubmit,
    formState: { isDirty, isSubmitting },
    setError,
  } = form;

  const { mutateAsync } = useMutation({
    mutationFn: async (values: OrganizationSchema) => {
      const response = await updateOrganization(
        user?.currentOrganizationId ?? "",
        values,
      );

      return response.data;
    },
    onMutate: async (newValues) => {
      // * Cancel any outgoing refetches so they don't overwrite our optimistic update
      await queryClient.cancelQueries({
        queryKey: queries.organization.getShipmentControl._def,
      });

      // * Snapshot the previous value
      const previousShipmentControl = queryClient.getQueryData([
        queries.organization.getShipmentControl._def,
      ]);

      // * Optimistically update to the new value
      queryClient.setQueryData(
        [queries.organization.getShipmentControl._def],
        newValues,
      );

      return { previousShipmentControl, newValues };
    },
    onSuccess: () => {
      toast.success("Organization updated successfully");

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
    onError: (error: APIError, _, context) => {
      // * Rollback the optimistic update
      queryClient.setQueryData(
        [queries.organization.getShipmentControl._def],
        context?.previousShipmentControl,
      );

      if (error.isValidationError()) {
        error.getFieldErrors().forEach((fieldError) => {
          setError(fieldError.name as Path<OrganizationSchema>, {
            message: fieldError.reason,
          });
        });
      }

      if (error.isRateLimitError()) {
        toast.error("Rate limit exceeded", {
          description:
            "You have exceeded the rate limit. Please try again later.",
        });
      }
    },
    onSettled: () => {
      // * Invalidate the query to refresh the data
      queryClient.invalidateQueries({
        queryKey: queries.organization.getOrgById._def,
      });
    },
  });

  const onSubmit = useCallback(
    async (values: OrganizationSchema) => {
      await mutateAsync(values);
    },
    [mutateAsync],
  );

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
