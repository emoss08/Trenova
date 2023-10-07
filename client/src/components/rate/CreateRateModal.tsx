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

import React, { Suspense } from "react";
import {
  Badge,
  Box,
  Button,
  Divider,
  Group,
  Modal,
  Tabs,
  useMantineTheme,
} from "@mantine/core";
import { notifications } from "@mantine/notifications";
import { useForm, UseFormReturnType, yupResolver } from "@mantine/form";
import { useMediaQuery } from "@mantine/hooks";
import { useRateStore as store } from "@/stores/DispatchStore";
import { ValidatedTextInput } from "@/components/common/fields/TextInput";
import { SelectInput } from "@/components/common/fields/SelectInput";
import { TChoiceProps } from "@/types";
import { useCustomers } from "@/hooks/useCustomers";
import {
  Rate,
  rateFields,
  RateFormValues as FormValues,
} from "@/types/dispatch";
import { rateSchema } from "@/lib/schemas/DispatchSchema";
import { useCommodities } from "@/hooks/useCommodities";
import { useLocations } from "@/hooks/useLocations";
import { ValidatedNumberInput } from "@/components/common/fields/NumberInput";
import { useEquipmentTypes } from "@/hooks/useEquipmentType";
import { useShipmentTypes } from "@/hooks/useShipmentTypes";
import { useAccessorialCharges } from "@/hooks/useAccessorialCharges";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { TableStoreProps } from "@/types/tables";

function CreateRateBillingTableForm({
  accessorialCharges,
  isAccessorialChargesLoading,
  isAccessorialChargesError,
  form,
}: {
  accessorialCharges: ReadonlyArray<TChoiceProps>;
  isAccessorialChargesLoading: boolean;
  isAccessorialChargesError: boolean;
  form: UseFormReturnType<FormValues>;
}) {
  const theme = useMantineTheme();

  const fields = form.values.rateBillingTables?.map((item, index) => {
    const { accessorialCharge } = item;

    return (
      <>
        <Group mt="xs" key={accessorialCharge}>
          <SelectInput<FormValues>
            form={form}
            name={`rateBillingTables.${index}.accessorialCharge`}
            data={accessorialCharges}
            isLoading={isAccessorialChargesLoading}
            isError={isAccessorialChargesError}
            label="Accessorial Charge"
            placeholder="Select Accessorial Charge"
            description="Accessorial Charge associated with this Rate"
          />
          <ValidatedTextInput<FormValues>
            label="Description"
            form={form}
            name={`rateBillingTables.${index}.description`}
            placeholder="Description"
            description="Description for Rate Billing Table"
          />
          <ValidatedNumberInput<FormValues>
            form={form}
            name={`rateBillingTables.${index}.unit`}
            label="Unit"
            placeholder="Unit"
            description="Unit for Rate Billing Table"
          />
          <ValidatedNumberInput<FormValues>
            form={form}
            name={`rateBillingTables.${index}.chargeAmount`}
            label="Charge Amount"
            placeholder="Charge Amount"
            description="Charge Amount for Rate Billing Table"
          />
          <ValidatedNumberInput<FormValues>
            form={form}
            name={`rateBillingTables.${index}.subTotal`}
            label="Sub Total"
            placeholder="Sub Total"
            description="Sub Total for Rate Billing Table"
          />
          <Button
            mt={40}
            variant="subtle"
            style={{
              color:
                theme.colorScheme === "dark"
                  ? theme.colors.gray[0]
                  : theme.colors.dark[9],
              backgroundColor: "transparent",
            }}
            size="sm"
            compact
            onClick={() => form.removeListItem("rateBillingTables", index)}
          >
            Remove Rate Billing Table
          </Button>
        </Group>
        <Divider variant="dashed" mt={20} />
      </>
    );
  });

  return (
    <Box>
      {fields}
      <Button
        variant="subtle"
        style={{
          color:
            theme.colorScheme === "dark"
              ? theme.colors.gray[0]
              : theme.colors.dark[9],
          backgroundColor: "transparent",
        }}
        size="sm"
        compact
        mt={20}
        onClick={() =>
          form.insertListItem("rateBillingTables", {
            accessorialCharge: "",
            description: "",
            unit: 1,
            chargeAmount: 1,
            subTotal: 1,
          })
        }
      >
        Add Rate Billing Table
      </Button>
    </Box>
  );
}

export function CreateRateModal() {
  const [showCreateModal, setShowCreateModal] = store.use("createModalOpen");
  const [activeTab, setActiveTab] = React.useState<string | null>("overview");
  const [loading, setLoading] = React.useState<boolean>(false);
  const isMobile = useMediaQuery("(max-width: 50em)");

  const form = useForm<FormValues>({
    validate: yupResolver(rateSchema),
    initialValues: {
      isActive: true,
      rateNumber: "",
      customer: "",
      effectiveDate: new Date(),
      expirationDate: new Date(),
      commodity: "",
      orderType: "",
      equipmentType: "",
      originLocation: "",
      destinationLocation: "",
      rateMethod: "F",
      rateAmount: Number(0),
      distanceOverride: Number(0),
      comments: "",
      rateBillingTables: [],
    },
  });

  const mutation = useCustomMutation<FormValues, TableStoreProps<Rate>>(
    form,
    notifications,
    {
      method: "POST",
      path: "/rates/",
      successMessage: "Rate created successfully.",
      queryKeysToInvalidate: ["rate-table-data"],
      additionalInvalidateQueries: ["rates"],
      closeModal: true,
      errorMessage: "Failed to create rate.",
    },
    () => setLoading(false),
    store,
  );

  type ErrorCountType = (tab: string | null) => number;

  const getErrorCount: ErrorCountType = (tab) => {
    switch (tab) {
      case "overview":
        return rateFields.filter((field: string) => form.errors[field]).length;
      case "rate-billing-table":
        return Object.keys(form.errors).filter((field) =>
          field.startsWith("rateBillingTables"),
        ).length;
      default:
        return 0;
    }
  };

  const submitForm = (values: FormValues) => {
    setLoading(true);
    mutation.mutate(values);
  };

  // Requests
  const {
    selectCustomersData,
    isLoading: isCustomersLoading,
    isError: isCustomersError,
  } = useCustomers(showCreateModal);

  const {
    selectCommodityData,
    isLoading: isCommoditiesLoading,
    isError: isCommoditiesError,
  } = useCommodities(showCreateModal);

  const {
    selectLocationData,
    isLoading: isLocationsLoading,
    isError: isLocationsError,
  } = useLocations(showCreateModal);

  const {
    selectEquipmentType,
    isLoading: isEquipmentTypesLoading,
    isError: isEquipmentTypesError,
  } = useEquipmentTypes(showCreateModal);

  const {
    selectShipmentType,
    isLoading: isOrderTypesLoading,
    isError: isOrderTypesError,
  } = useShipmentTypes(showCreateModal);

  const {
    selectAccessorialChargeData,
    isLoading: isAccessorialChargesLoading,
    isError: isAccessorialChargesError,
  } = useAccessorialCharges(showCreateModal);

  const RateFormContent = React.lazy(() => import("./RateForm"));

  return (
    <Modal.Root
      opened={showCreateModal}
      onClose={() => setShowCreateModal(false)}
      fullScreen={isMobile}
      transitionProps={{ transition: "fade", duration: 200 }}
      styles={{
        content: {
          minWidth: "60%",
        },
      }}
    >
      <Modal.Overlay />
      <Modal.Content>
        <Modal.Header>
          <Modal.Title>Create Rate</Modal.Title>
          <Modal.CloseButton />
        </Modal.Header>
        <Modal.Body>
          <form onSubmit={form.onSubmit((values) => submitForm(values))}>
            <Tabs
              defaultValue="overview"
              value={activeTab}
              onTabChange={setActiveTab}
            >
              <Tabs.List>
                <Tabs.Tab
                  color={getErrorCount("overview") > 0 ? "red" : "blue"}
                  value="overview"
                  rightSection={
                    getErrorCount("overview") > 0 ? (
                      <Badge
                        w={16}
                        h={16}
                        sx={{ pointerEvents: "none" }}
                        variant="filled"
                        size="xs"
                        p={0}
                        color="red"
                      >
                        {getErrorCount("overview")}
                      </Badge>
                    ) : undefined
                  }
                >
                  Overview
                </Tabs.Tab>
                <Tabs.Tab
                  color={
                    getErrorCount("rate-billing-table") > 0 ? "red" : "blue"
                  }
                  rightSection={
                    getErrorCount("rate-billing-table") > 0 ? (
                      <Badge
                        w={16}
                        h={16}
                        sx={{ pointerEvents: "none" }}
                        variant="filled"
                        size="xs"
                        p={0}
                        color="red"
                      >
                        {getErrorCount("rate-billing-table")}
                      </Badge>
                    ) : undefined
                  }
                  value="rate-billing-table"
                >
                  Rate Billing Table
                </Tabs.Tab>
              </Tabs.List>
              <Tabs.Panel value="overview" pt="xs">
                <Suspense fallback={<div>Loading...</div>}>
                  <RateFormContent
                    customers={selectCustomersData}
                    isCustomersLoading={isCustomersLoading}
                    isCustomersError={isCustomersError}
                    commodities={selectCommodityData}
                    isCommoditiesLoading={isCommoditiesLoading}
                    isCommoditiesError={isCommoditiesError}
                    locations={selectLocationData}
                    isLocationsLoading={isLocationsLoading}
                    isLocationsError={isLocationsError}
                    equipmentTypes={selectEquipmentType}
                    isEquipmentTypesLoading={isEquipmentTypesLoading}
                    isEquipmentTypesError={isEquipmentTypesError}
                    orderTypes={selectShipmentType}
                    isOrderTypesError={isOrderTypesError}
                    isOrderTypesLoading={isOrderTypesLoading}
                    form={form}
                  />
                </Suspense>
              </Tabs.Panel>
              <Tabs.Panel value="rate-billing-table" pt="xs">
                <CreateRateBillingTableForm
                  accessorialCharges={selectAccessorialChargeData}
                  isAccessorialChargesLoading={isAccessorialChargesLoading}
                  isAccessorialChargesError={isAccessorialChargesError}
                  form={form}
                />
              </Tabs.Panel>
            </Tabs>
            <Group position="right" mt="md">
              <Button type="submit" loading={loading}>
                Submit
              </Button>
            </Group>
          </form>
        </Modal.Body>
      </Modal.Content>
    </Modal.Root>
  );
}
