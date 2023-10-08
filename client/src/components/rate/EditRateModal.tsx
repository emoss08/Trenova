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

import {
  Badge,
  Box,
  Button,
  Divider,
  Drawer,
  Group,
  SimpleGrid,
  Tabs,
  Text,
  useMantineTheme,
} from "@mantine/core";
import React from "react";
import { useForm, UseFormReturnType, yupResolver } from "@mantine/form";
import { notifications } from "@mantine/notifications";
import { useRateStore as store } from "@/stores/DispatchStore";
import { TChoiceProps } from "@/types";
import {
  Rate,
  rateFields,
  RateFormValues as FormValues,
} from "@/types/dispatch";
import { useFormStyles } from "@/assets/styles/FormStyles";
import { SelectInput } from "@/components/common/fields/SelectInput";
import { ValidatedTextInput } from "@/components/common/fields/TextInput";
import { rateMethodChoices, yesAndNoChoicesBoolean } from "@/lib/constants";
import { ValidatedDateInput } from "@/components/common/fields/DateInput";
import { ValidatedTextArea } from "@/components/common/fields/TextArea";
import { rateSchema } from "@/lib/validations/DispatchSchema";
import { useCustomers } from "@/hooks/useCustomers";
import { useCommodities } from "@/hooks/useCommodities";
import { useLocations } from "@/hooks/useLocations";
import { useEquipmentTypes } from "@/hooks/useEquipmentType";
import { useShipmentTypes } from "@/hooks/useShipmentTypes";
import { useAccessorialCharges } from "@/hooks/useAccessorialCharges";
import { ValidatedNumberInput } from "@/components/common/fields/NumberInput";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { TableStoreProps } from "@/types/tables";

function EditRateBillingTableForm({
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
            withAsterisk
          />
          <ValidatedTextInput<FormValues>
            label="Description"
            form={form}
            name={`rateBillingTables.${index}.description`}
            placeholder="Description"
            description="Description for Rate Billing Table"
          />
          <ValidatedTextInput<FormValues>
            form={form}
            name={`rateBillingTables.${index}.unit`}
            label="Unit"
            placeholder="Unit"
            description="Unit for Rate Billing Table"
            withAsterisk
          />
          <ValidatedTextInput<FormValues>
            form={form}
            name={`rateBillingTables.${index}.chargeAmount`}
            label="Charge Amount"
            placeholder="Charge Amount"
            description="Charge Amount for Rate Billing Table"
            withAsterisk
          />
          <ValidatedTextInput<FormValues>
            form={form}
            name={`rateBillingTables.${index}.subTotal`}
            label="Sub Total"
            placeholder="Sub Total"
            description="Sub Total for Rate Billing Table"
            withAsterisk
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
            unit: Number(1),
            chargeAmount: Number(1),
            subTotal: Number(1),
          })
        }
      >
        Add Rate Billing Table
      </Button>
    </Box>
  );
}

type Props = {
  customers: ReadonlyArray<TChoiceProps>;
  isCustomersLoading: boolean;
  isCustomersError: boolean;
  commodities: ReadonlyArray<TChoiceProps>;
  isCommoditiesLoading: boolean;
  isCommoditiesError: boolean;
  shipmentTypes: ReadonlyArray<TChoiceProps>;
  isShipmentTypeLoading: boolean;
  isShipmentTypeError: boolean;
  equipmentTypes: ReadonlyArray<TChoiceProps>;
  isEquipmentTypesLoading: boolean;
  isEquipmentTypesError: boolean;
  locations: ReadonlyArray<TChoiceProps>;
  isLocationsLoading: boolean;
  isLocationsError: boolean;
  form: UseFormReturnType<FormValues>;
};

function EditRateModalForm({
  customers,
  isCustomersLoading,
  isCustomersError,
  commodities,
  isCommoditiesLoading,
  isCommoditiesError,
  shipmentTypes,
  isShipmentTypeLoading,
  isShipmentTypeError,
  equipmentTypes,
  isEquipmentTypesLoading,
  isEquipmentTypesError,
  locations,
  isLocationsLoading,
  isLocationsError,
  form,
}: Props): React.ReactElement {
  const { classes } = useFormStyles();

  return (
    <Box className={classes.div}>
      <SimpleGrid cols={4} breakpoints={[{ maxWidth: "lg", cols: 1 }]}>
        <SelectInput<FormValues>
          form={form}
          name="isActive"
          label="Is Active"
          description="Is this Rate active?"
          placeholder="Is Active"
          withAsterisk
          data={yesAndNoChoicesBoolean}
        />
        <ValidatedTextInput<FormValues>
          form={form}
          name="rateNumber"
          label="Rate Number"
          placeholder="Rate Number"
          description="Unique Number for Rate"
          disabled
          withAsterisk
        />
      </SimpleGrid>
      <Box my={10}>
        <div
          style={{
            textAlign: "center",
            display: "flex",
            justifyContent: "center",
            alignItems: "center",
            flexDirection: "column",
          }}
        >
          <Text fw={400} className={classes.text}>
            Rate Details
          </Text>
        </div>
        <Divider variant="dashed" />
      </Box>
      <SimpleGrid cols={4} breakpoints={[{ maxWidth: "sm", cols: 1 }]}>
        <SelectInput<FormValues>
          form={form}
          name="customer"
          label="Customer"
          placeholder="Customer"
          description="Customer associated with this Rate"
          data={customers} // TODO(WOLFRED): add context menu's or add creatable to select fields
          isLoading={isCustomersLoading}
          isError={isCustomersError}
        />
        <SelectInput<FormValues>
          form={form}
          name="commodity"
          label="Commodity"
          placeholder="Commodity"
          description="Commodity associated with this Rate"
          data={commodities}
          isLoading={isCommoditiesLoading}
          isError={isCommoditiesError}
        />
        <ValidatedDateInput<FormValues>
          form={form}
          name="effectiveDate"
          label="Effective Date"
          placeholder="Effective Date"
          description="Effective Date for Rate"
          withAsterisk
        />
        <ValidatedDateInput<FormValues>
          form={form}
          name="expirationDate"
          label="Expiration Date"
          placeholder="Expiration Date"
          description="Expiration Date for Rate"
          withAsterisk
        />
        <SelectInput<FormValues>
          form={form}
          name="originLocation"
          label="Origin Location"
          placeholder="Origin Location"
          description="Origin Location associated with this Rate"
          data={locations}
          isLoading={isLocationsLoading}
          isError={isLocationsError}
        />
        <SelectInput<FormValues>
          form={form}
          name="destinationLocation"
          label="Destination Location"
          placeholder="Destination Location"
          description="Dest. Location associated with this Rate"
          data={locations}
          isLoading={isLocationsLoading}
          isError={isLocationsError}
        />
        <SelectInput<FormValues>
          form={form}
          name="equipmentType"
          label="Equipment Type"
          placeholder="Equipment Type"
          description="Equipment Type associated with this Rate"
          data={equipmentTypes}
          isLoading={isEquipmentTypesLoading}
          isError={isEquipmentTypesError}
        />
        <SelectInput<FormValues>
          form={form}
          name="orderType"
          label="Order Type"
          placeholder="Order Type"
          description="Order Type associated with this Rate"
          data={shipmentTypes}
          isLoading={isShipmentTypeLoading}
          isError={isShipmentTypeError}
        />
        <SelectInput<FormValues>
          form={form}
          name="rateMethod"
          label="Rate Method"
          placeholder="Rate Method"
          description="Rate Method associated with this Rate"
          data={rateMethodChoices}
          withAsterisk
        />
        <ValidatedNumberInput<FormValues>
          form={form}
          name="rateAmount"
          label="Rate Amount"
          thousandsSeparator=","
          decimalSeparator="."
          precision={2}
          hideControls
          placeholder="Rate Amount"
          description="Rate Amount associated with this Rate"
          withAsterisk
        />
        <ValidatedNumberInput<FormValues>
          form={form}
          name="distanceOverride"
          label="Distance Override"
          thousandsSeparator=","
          decimalSeparator="."
          precision={2}
          hideControls
          placeholder="Distance Override"
          description="Dist. Override associated with this Rate"
        />
      </SimpleGrid>
      <ValidatedTextArea<FormValues>
        form={form}
        name="comments"
        label="Comments"
        description="Additional Comments for Rate"
        placeholder="Comments"
      />
    </Box>
  );
}

export function EditRateModalBody({
  rate,
  drawerOpen,
  onCancel,
}: {
  rate: Rate;
  drawerOpen: boolean;
  onCancel: () => void;
}) {
  const [activeTab, setActiveTab] = React.useState<string | null>("overview");
  const [loading, setLoading] = React.useState(false);
  const { classes } = useFormStyles();

  const form = useForm<FormValues>({
    validate: yupResolver(rateSchema),
    initialValues: {
      isActive: rate.isActive,
      rateNumber: rate.rateNumber,
      customer: rate.customer,
      effectiveDate: new Date(rate.effectiveDate as Date),
      expirationDate: new Date(rate.expirationDate as Date),
      commodity: rate.commodity,
      orderType: rate.orderType,
      equipmentType: rate.equipmentType,
      originLocation: rate.originLocation,
      destinationLocation: rate.destinationLocation,
      rateMethod: rate.rateMethod,
      rateAmount: Number(rate.rateAmount as number) || 0,
      distanceOverride: Number(rate.distanceOverride as number) || 0,
      comments: rate.comments,
      rateBillingTables: rate?.rateBillingTables,
    },
  });

  const mutation = useCustomMutation<FormValues, TableStoreProps<Rate>>(
    form,
    notifications,
    {
      method: "PUT",
      path: `/rates/${rate.id}`,
      successMessage: "Rate updated successfully.",
      queryKeysToInvalidate: ["rate-table-data"],
      additionalInvalidateQueries: ["rates"],
      closeModal: true,
      errorMessage: "Failed to update rate.",
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

  const {
    selectCustomersData,
    isLoading: isCustomersLoading,
    isError: isCustomersError,
  } = useCustomers(drawerOpen);

  const {
    selectCommodityData,
    isLoading: isCommoditiesLoading,
    isError: isCommoditiesError,
  } = useCommodities(drawerOpen);

  const {
    selectLocationData,
    isLoading: isLocationsLoading,
    isError: isLocationsError,
  } = useLocations(drawerOpen);

  const {
    selectEquipmentType,
    isLoading: isEquipmentTypesLoading,
    isError: isEquipmentTypesError,
  } = useEquipmentTypes(drawerOpen);

  const {
    selectShipmentType,
    isLoading: isShipmentTypeLoading,
    isError: isShipmentTypeError,
  } = useShipmentTypes(drawerOpen);

  const {
    selectAccessorialChargeData,
    isLoading: isAccessorialChargesLoading,
    isError: isAccessorialChargesError,
  } = useAccessorialCharges(drawerOpen);

  return (
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
          >
            Overview
          </Tabs.Tab>
          <Tabs.Tab
            color={getErrorCount("rate-billing-table") > 0 ? "red" : "blue"}
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
          <EditRateModalForm
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
            shipmentTypes={selectShipmentType}
            isShipmentTypeError={isShipmentTypeError}
            isShipmentTypeLoading={isShipmentTypeLoading}
            form={form}
          />
        </Tabs.Panel>
        <Tabs.Panel value="rate-billing-table" pt="xs">
          <EditRateBillingTableForm
            accessorialCharges={selectAccessorialChargeData}
            isAccessorialChargesLoading={isAccessorialChargesLoading}
            isAccessorialChargesError={isAccessorialChargesError}
            form={form}
          />
        </Tabs.Panel>
      </Tabs>
      <Group position="right" mt="md">
        <Button
          variant="subtle"
          onClick={onCancel}
          color="gray"
          type="button"
          className={classes.control}
        >
          Cancel
        </Button>
        <Button
          color="white"
          type="submit"
          className={classes.control}
          loading={loading}
        >
          Submit
        </Button>
      </Group>
    </form>
  );
}

export function RateDrawer() {
  const [drawerOpen, setDrawerOpen] = store.use("drawerOpen");
  const [rate] = store.use("selectedRecord");
  const onCancel = () => setDrawerOpen(false);

  return (
    <Drawer.Root
      position="right"
      opened={drawerOpen}
      onClose={() => setDrawerOpen(false)}
      size="60%"
    >
      <Drawer.Overlay />
      <Drawer.Content>
        <Drawer.Header>
          <Drawer.Title>Edit Rate: {rate && rate.rateNumber}</Drawer.Title>
          <Drawer.CloseButton />
        </Drawer.Header>
        <Drawer.Body>
          {rate ? (
            <EditRateModalBody
              rate={rate}
              drawerOpen={drawerOpen}
              onCancel={onCancel}
            />
          ) : (
            <Text>Rate not found</Text>
          )}
        </Drawer.Body>
      </Drawer.Content>
    </Drawer.Root>
  );
}
