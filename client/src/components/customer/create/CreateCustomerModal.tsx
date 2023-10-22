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
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { notifications } from "@mantine/notifications";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faCheck, faX, faXmark } from "@fortawesome/pro-solid-svg-icons";
import { useForm, UseFormReturnType, yupResolver } from "@mantine/form";
import { customerTableStore as store } from "@/stores/CustomerStore";
import { CreateCustomerFormValues } from "@/types/customer";
import axios from "@/lib/axiosConfig";
import { APIError } from "@/types/server";
import { DeliverySlotForm } from "@/components/customer/create/steps/DeliverySlotForm";
import { CreateCustomerSchema } from "@/lib/validations/CustomerSchema";
import { useUsers } from "@/hooks/useUsers";
import { useLocations } from "@/hooks/useLocations";
import { CreateCustomerModalForm } from "@/components/customer/create/steps/CustomerForm";
import { CustomerRuleProfileForm } from "@/components/customer/create/steps/CustomerRuleProfileForm";
import { CustomerEmailProfileForm } from "@/components/customer/create/steps/CustomerEmailProfileForm";
import { CustomerContactForm } from "@/components/customer/create/steps/CustomerContactForm";
import { useDocumentClass } from "@/hooks/useDocumentClass";

type ErrorCountType = (step: number) => number;

const customerInfoFields: ReadonlyArray<string> = [
  "name",
  "addressLine1",
  "addressLine2",
  "city",
  "state",
  "zipCode",
];

const stepsComponent = [
  CreateCustomerModalForm, // step 0
  CustomerRuleProfileForm, // step 1
  CustomerEmailProfileForm, // step 2
  CustomerContactForm, // step 3
  DeliverySlotForm, // step 4
];

export function CreateCustomerModal(): React.ReactElement {
  const [showCreateModal, setShowCreateModal] = store.use("createModalOpen");
  const [loading, setLoading] = store.use("loading");
  const [activeStep, setActiveStep] = store.use("activeStep");
  const [attemptedNext, setAttemptedNext] = store.use("attemptedNext");
  const nextButtonRef = React.useRef<HTMLButtonElement>(null);
  const submitButtonRef = React.useRef<HTMLButtonElement>(null);

  const queryClient = useQueryClient();
  const theme = useMantineTheme();

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
        ruleProfile: {
          name: "",
          documentClass: [],
        },
        customerContacts: [],
      },
    });

  const CurrentStepComponent = stepsComponent[activeStep];
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
  const {
    selectDocumentClassData,
    isLoading: isDocumentClassesLoading,
    isError: isDocumentClassesError,
  } = useDocumentClass(showCreateModal);

  const getErrorCountForStep: ErrorCountType = (step) => {
    switch (step) {
      case 0:
        return customerInfoFields.filter((field) => form.errors[field]).length;
      case 1:
        return Object.keys(form.errors).filter((field) =>
          field.startsWith("ruleProfile"),
        ).length;
      case 2:
        return Object.keys(form.errors).filter((field) =>
          field.startsWith("emailProfile"),
        ).length;
      case 3:
        return Object.keys(form.errors).filter((field) =>
          field.startsWith("customerContacts"),
        ).length;
      case 4:
        return Object.keys(form.errors).filter((field) =>
          field.startsWith("deliverySlots"),
        ).length;
      default:
        return 0;
    }
  };

  const validateCurrentStep: () => boolean = React.useCallback(() => {
    switch (activeStep) {
      case 0: // Assuming step 0 is "customerInfo"
        return customerInfoFields.every((field: string) => !form.errors[field]);
      case 1: // Rule Profile
        return Object.keys(form.errors).every(
          (field: string) => !field.startsWith("ruleProfile"),
        );
      case 2: // Email Profile
        return Object.keys(form.errors).every(
          (field: string) => !field.startsWith("emailProfile"),
        );
      case 3: // Contacts
        return Object.keys(form.errors).every(
          (field: string) => !field.startsWith("customerContacts"),
        );
      case 4: // Delivery Slots
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
        setActiveStep((current) => current + 1);
      }
      setAttemptedNext(false); // Reset the flag
    }
  }, [
    form.errors,
    attemptedNext,
    validateCurrentStep,
    setAttemptedNext,
    setActiveStep,
  ]);

  const nextStep = () => {
    form.validate();
    setAttemptedNext(true);
  };

  const prevStep = () =>
    setActiveStep((current) => (current > 0 ? current - 1 : current));

  const handleModalClose = () => {
    setShowCreateModal(false);
    form.reset();
  };

  const submitForm = (values: CreateCustomerFormValues) => {
    setLoading(true);
    mutation.mutate(values);
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
            if (e.attr === "nonFieldErrors") {
              notifications.show({
                title: "Error",
                message: e.detail,
                color: "red",
                withCloseButton: true,
                icon: <FontAwesomeIcon icon={faXmark} />,
                autoClose: 10_000, // 10 seconds
              });
            } else if (
              e.attr === "All" &&
              e.detail ===
                "Customer Rule Profile with this Name and Organization already exists."
            ) {
              form.setFieldError("ruleProfile.name", e.detail);
            }
          });
        }
      },
      onSettled: () => {
        setLoading(false);
      },
    },
  );

  return (
    <Modal.Root opened={showCreateModal} onClose={handleModalClose} size="90%">
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
            <Stepper.Step
              icon={
                getErrorCountForStep(0) > 0 ? (
                  <FontAwesomeIcon icon={faX} color="red" />
                ) : undefined
              }
              color={getErrorCountForStep(0) > 0 ? "red" : undefined}
              label="Rule Profile"
              description="Add Rule Profile"
            />
            <Stepper.Step
              icon={
                getErrorCountForStep(0) > 0 ? (
                  <FontAwesomeIcon icon={faX} color="red" />
                ) : undefined
              }
              color={getErrorCountForStep(0) > 0 ? "red" : undefined}
              label="Email Profile"
              description="Add Email Profile"
            />
            <Stepper.Step
              icon={
                getErrorCountForStep(0) > 0 ? (
                  <FontAwesomeIcon icon={faX} color="red" />
                ) : undefined
              }
              color={getErrorCountForStep(0) > 0 ? "red" : undefined}
              label="Contacts"
              description="Add Contacts"
            />
            <Stepper.Step
              icon={
                getErrorCountForStep(0) > 0 ? (
                  <FontAwesomeIcon icon={faX} color="red" />
                ) : undefined
              }
              color={getErrorCountForStep(0) > 0 ? "red" : undefined}
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
                documentClasses={selectDocumentClassData}
                isDocumentClassesError={isDocumentClassesError}
                isDocumentClassesLoading={isDocumentClassesLoading}
                key={activeStep}
                users={selectUsersData}
                locations={selectLocationData}
              />
            </Box>
            <Group position="center" mt="xl">
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

              {activeStep === stepsComponent.length - 1 ? (
                <Button
                  type="submit"
                  ref={submitButtonRef}
                  key="submit-button"
                  loading={loading}
                  disabled={!validateCurrentStep()}
                >
                  Submit
                </Button>
              ) : (
                <Button
                  onClick={async () => nextStep()}
                  disabled={!validateCurrentStep()}
                  ref={nextButtonRef}
                >
                  Next step
                </Button>
              )}
            </Group>
          </form>
        </Modal.Body>
      </Modal.Content>
    </Modal.Root>
  );
}
