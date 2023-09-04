/*
 * COPYRIGHT(c) 2023 MONTA
 *
 * This file is part of Monta.
 *
 * The Monta software is licensed under the Business Source License 1.1. You are granted the right
 * to copy, modify, and redistribute the software, but only for non-production use or with a total
 * of less than three server instances. Starting from the Change Date (November 16, 2026), the
 * software will be made available under version 2 or later of the GNU General Public License.
 * If you use the software in violation of this license, your rights under the license will be
 * terminated automatically. The software is provided "as is," and the Licensor disclaims all
 * warranties and conditions. If you use this license's text or the "Business Source License" name
 * and trademark, you must comply with the Licensor's covenants, which include specifying the
 * Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
 * Grant, and not modifying the license in any other way.
 */

import React from "react";
import {
  Box,
  Button,
  Group,
  Modal,
  Stepper,
  useMantineTheme,
} from "@mantine/core";
import { useMutation, useQueryClient } from "react-query";
import { notifications } from "@mantine/notifications";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faCheck, faX, faXmark } from "@fortawesome/pro-solid-svg-icons";
import { UseFormReturnType, useForm, yupResolver } from "@mantine/form";
import { customerTableStore as store } from "@/stores/CustomerStore";
import { CreateCustomerFormValues } from "@/types/customer";
import axios from "@/helpers/AxiosConfig";
import { APIError } from "@/types/server";
import { DeliverySlotForm } from "./CreateDeliverySlotForm";
import { CreateCustomerSchema } from "@/helpers/schemas/CustomerSchema";
import { useUsers } from "@/hooks/useUsers";
import { useLocations } from "@/hooks/useLocations";
import { customerInfoFields } from "@/utils/apps/customers/CustomerErrorContext";
import { CreateCustomerModalForm } from "@/components/customer/view/_partials/profile_forms/CreateCustomerForm";
import { CustomerRuleProfileForm } from "@/components/customer/view/_partials/profile_forms/CustomerRuleProfileForm";
import { CustomerEmailProfileForm } from "@/components/customer/view/_partials/profile_forms/CustomerEmailProfileForm";

const stepsComponent = [
  CreateCustomerModalForm, // step 0
  CustomerRuleProfileForm, // step 1
  CustomerEmailProfileForm, // step 2
  DeliverySlotForm, // step 5
];

export function CreateCustomerModal(): React.ReactElement {
  const [showCreateModal, setShowCreateModal] = store.use("createModalOpen");
  const [loading, setLoading] = React.useState<boolean>(false);
  const [activeStep, setActiveStep] = React.useState<number>(0);
  const queryClient = useQueryClient();
  const theme = useMantineTheme();
  const {
    selectUsersData,
    isLoading: isUsersLoading,
    isError: isUsersError,
  } = useUsers(showCreateModal);
  const {
    selectLocationData,
    isLoading: isLocationsLoading,
    isError: isLocationsError,
  } = useLocations(showCreateModal);
  const CurrentStepComponent = stepsComponent[activeStep];
  const [attemptedNext, setAttemptedNext] = React.useState(false);

  const nextStep = () => {
    form.validate();
    setAttemptedNext(true);
  };

  const mutation = useMutation(
    (values: CreateCustomerFormValues) => axios.post("/customers/", values),
    {
      onSuccess: () => {
        queryClient
          .invalidateQueries({
            queryKey: ["customers-table-data"],
          })
          .then(() => {
            notifications.show({
              title: "Success",
              message: "Customer updated successfully",
              color: "green",
              withCloseButton: true,
              icon: <FontAwesomeIcon icon={faCheck} />,
            });
            store.set("createModalOpen", false);
          });
      },
      onError: (error: any) => {
        const { data } = error.response;
        if (data.type === "validation_error") {
          data.errors.forEach((e: APIError) => {
            form.setFieldError(e.attr, e.detail);
            if (e.attr === "non_field_errors") {
              notifications.show({
                title: "Error",
                message: e.detail,
                color: "red",
                withCloseButton: true,
                icon: <FontAwesomeIcon icon={faXmark} />,
                autoClose: 10_000, // 10 seconds
              });
            }
          });
        }
      },
      onSettled: () => {
        setLoading(false);
      },
    },
  );

  const form: UseFormReturnType<CreateCustomerFormValues> =
    useForm<CreateCustomerFormValues>({
      validate: yupResolver(CreateCustomerSchema),
      initialValues: {
        status: "A",
        name: "",
        addressLine1: "",
        addressLine2: "",
        city: "",
        state: "",
        zipCode: "",
        hasCustomerPortal: "N",
        autoMarkReadyToBill: "N",
        advocate: "",
        deliverySlots: [],
        emailProfile: {
          subject: "",
          comment: "",
          fromAddress: "",
          blindCopy: "",
          readReceipt: false,
          readReceiptTo: "",
          attachmentName: "",
        },
      },
    });

  const validateCurrentStep: () => boolean = React.useCallback(() => {
    switch (activeStep) {
      case 0: // Assuming step 0 is "customerInfo"
        return customerInfoFields.every((field: string) => !form.errors[field]);
      case 2: // Email Profile
        return Object.keys(form.errors).every(
          (field: string) => !field.startsWith("emailProfile"),
        );
      case 5: // Delivery Slots
        return Object.keys(form.errors).every(
          (field: string) => !field.startsWith("deliverySlots"),
        );
      default:
        return true;
    }
  }, [activeStep, form.errors]);

  React.useEffect(() => {
    if (attemptedNext) {
      if (validateCurrentStep()) {
        setActiveStep((current) => (current < 3 ? current + 1 : current));
      }
      setAttemptedNext(false); // Reset the flag
    }
  }, [form.errors, attemptedNext, validateCurrentStep]);

  const submitForm = (values: CreateCustomerFormValues) => {
    setLoading(true);
    mutation.mutate(values);
  };

  // const nextStep = () =>
  //   setActiveStep((current) => (current < 3 ? current + 1 : current));
  const prevStep = () =>
    setActiveStep((current) => (current > 0 ? current - 1 : current));

  // Modify the validateCurrentStep function to return error count for each step
  const getErrorCountForStep = (step: number) => {
    switch (step) {
      case 0: // Assuming step 0 is "customerInfo"
        return customerInfoFields.filter((field) => form.errors[field]).length;
      case 6: // Assuming step 1 is "deliverySlots"
        return Object.keys(form.errors).filter((field) =>
          field.startsWith("deliverySlots"),
        ).length;
      // ... add cases for other steps as needed
      default:
        return 0; // Default to 0 for steps that don't have validation
    }
  };

  return (
    <Modal.Root
      opened={showCreateModal}
      onClose={() => setShowCreateModal(false)}
      size="90%"
    >
      <Modal.Overlay />
      <Modal.Content>
        <Modal.Header>
          <Modal.Title>Create Customer</Modal.Title>
          <Modal.CloseButton />
        </Modal.Header>
        <Modal.Body>
          <Stepper active={activeStep} onStepClick={setActiveStep}>
            <Stepper.Step
              icon={
                getErrorCountForStep(0) > 0 ? (
                  <FontAwesomeIcon icon={faX} color="red" />
                ) : undefined
              }
              color={getErrorCountForStep(0) > 0 ? "red" : undefined}
              label="Customer Info"
              description="Enter Customer Info"
            />
            <Stepper.Step label="Rule Profile" description="Add Rule Profile" />
            <Stepper.Step
              label="Email Profile"
              description="Add Rule Profile"
            />
            <Stepper.Step label="Fuel Profile" description="Add Rule Profile" />
            <Stepper.Step label="Contacts" description="Add Contacts" />
            <Stepper.Step
              label="Delivery Slots"
              description="Add Delivery Slots"
            />
            <Stepper.Completed>
              Completed, click back button to get to previous step
            </Stepper.Completed>
          </Stepper>
          <form onSubmit={form.onSubmit((values) => submitForm(values))}>
            <Box mx={250} my={20}>
              <CurrentStepComponent
                form={form}
                isLocationsError={isLocationsError}
                isLocationsLoading={isLocationsLoading}
                isUsersError={isUsersError}
                isUsersLoading={isUsersLoading}
                users={selectUsersData}
                locations={selectLocationData}
              />
            </Box>
          </form>
          <Group position="center" mt="xl">
            {/* on first step add cancel button */}
            {activeStep === 0 ? (
              <Button
                variant="subtle"
                color={theme.colorScheme === "dark" ? "gray" : "dark"}
                onClick={() => {
                  setShowCreateModal(false);
                  form.reset();
                }}
              >
                Cancel
              </Button>
            ) : (
              <Button variant="default" onClick={prevStep}>
                Back
              </Button>
            )}
            <Button onClick={nextStep} disabled={!validateCurrentStep()}>
              Next step
            </Button>
          </Group>
        </Modal.Body>
      </Modal.Content>
    </Modal.Root>
  );
}
