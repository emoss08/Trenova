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
  Divider,
  Group,
  Modal,
  SimpleGrid,
  Tabs,
  Text,
  useMantineTheme,
} from "@mantine/core";
import { useMutation, useQueryClient } from "react-query";
import { notifications } from "@mantine/notifications";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faCheck, faXmark } from "@fortawesome/pro-solid-svg-icons";
import { useForm, UseFormReturnType, yupResolver } from "@mantine/form";
import { useMediaQuery } from "@mantine/hooks";
import { useRateStore as store } from "@/stores/DispatchStore";
import { useFormStyles } from "@/assets/styles/FormStyles";
import axios from "@/helpers/AxiosConfig";
import { APIError } from "@/types/server";
import { ValidatedTextInput } from "@/components/common/fields/TextInput";
import { ValidatedTextArea } from "@/components/common/fields/TextArea";
import { SelectInput } from "@/components/common/fields/SelectInput";
import { TChoiceProps } from "@/types";
import { rateMethodChoices, yesAndNoChoicesBoolean } from "@/helpers/constants";
import { useCustomers } from "@/hooks/useCustomers";
import { RateFormValues } from "@/types/dispatch";
import { rateSchema } from "@/helpers/schemas/DispatchSchema";
import { useCommodities } from "@/hooks/useCommodities";
import { ValidatedDateInput } from "@/components/common/fields/DateInput";
import { useLocations } from "@/hooks/useLocations";
import { ValidatedNumberInput } from "@/components/common/fields/NumberInput";
import { getNewRateNumber } from "@/services/DispatchRequestService";
import { useEquipmentTypes } from "@/hooks/useEquipmentType";
import { useOrderTypes } from "@/hooks/useOrderTypes";
import { useAccessorialCharges } from "@/hooks/useAccessorialCharges";

type Props = {
  customers: ReadonlyArray<TChoiceProps>;
  isCustomersLoading: boolean;
  isCustomersError: boolean;
  commodities: ReadonlyArray<TChoiceProps>;
  isCommoditiesLoading: boolean;
  isCommoditiesError: boolean;
  orderTypes: ReadonlyArray<TChoiceProps>;
  isOrderTypesLoading: boolean;
  isOrderTypesError: boolean;
  equipmentTypes: ReadonlyArray<TChoiceProps>;
  isEquipmentTypesLoading: boolean;
  isEquipmentTypesError: boolean;
  locations: ReadonlyArray<TChoiceProps>;
  isLocationsLoading: boolean;
  isLocationsError: boolean;
  form: UseFormReturnType<RateFormValues>;
};

function CreateRateBillingTableForm({
  accessorialCharges,
  isAccessorialChargesLoading,
  isAccessorialChargesError,
  form,
}: {
  accessorialCharges: ReadonlyArray<TChoiceProps>;
  isAccessorialChargesLoading: boolean;
  isAccessorialChargesError: boolean;
  form: UseFormReturnType<RateFormValues>;
}) {
  const theme = useMantineTheme();

  const fields = form.values.rateBillingTables?.map((item, index) => {
    const { accessorialCharge } = item;

    return (
      <>
        <Group mt="xs" key={accessorialCharge}>
          <SelectInput<RateFormValues>
            form={form}
            name={`rateBillingTables.${index}.accessorialCharge`}
            data={accessorialCharges}
            isLoading={isAccessorialChargesLoading}
            isError={isAccessorialChargesError}
            label="Accessorial Charge"
            placeholder="Select Accessorial Charge"
            description="Accessorial Charge associated with this Rate"
          />
          <ValidatedTextInput<RateFormValues>
            label="Description"
            form={form}
            name={`rateBillingTables.${index}.description`}
            placeholder="Description"
            description="Description for Rate Billing Table"
          />
          <ValidatedNumberInput<RateFormValues>
            form={form}
            name={`rateBillingTables.${index}.unit`}
            label="Unit"
            placeholder="Unit"
            description="Unit for Rate Billing Table"
          />
          <ValidatedNumberInput<RateFormValues>
            form={form}
            name={`rateBillingTables.${index}.chargeAmount`}
            label="Charge Amount"
            placeholder="Charge Amount"
            description="Charge Amount for Rate Billing Table"
          />
          <ValidatedNumberInput<RateFormValues>
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

export function CreateRateModalForm({
  customers,
  isCustomersLoading,
  isCustomersError,
  commodities,
  isCommoditiesLoading,
  isCommoditiesError,
  orderTypes,
  isOrderTypesLoading,
  isOrderTypesError,
  equipmentTypes,
  isEquipmentTypesLoading,
  isEquipmentTypesError,
  locations,
  isLocationsLoading,
  isLocationsError,
  form,
}: Props) {
  const { classes } = useFormStyles();

  const fetchRate = React.useCallback(async () => {
    try {
      const rateNumber = await getNewRateNumber();
      form.setFieldValue("rateNumber", rateNumber);
    } catch (err) {
      console.error("Error fetching rate number:", err);
    }
  }, []);

  // fetch rate number and assign it to rateNumber field once when component mounts
  React.useEffect(() => {
    fetchRate();
  }, [fetchRate]);

  return (
    <Box className={classes.div}>
      <SimpleGrid cols={4} breakpoints={[{ maxWidth: "lg", cols: 1 }]}>
        <SelectInput<RateFormValues>
          form={form}
          name="isActive"
          label="Is Active"
          description="Is this Rate active?"
          placeholder="Is Active"
          withAsterisk
          data={yesAndNoChoicesBoolean}
        />
        <ValidatedTextInput<RateFormValues>
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
        <SelectInput<RateFormValues>
          form={form}
          name="customer"
          label="Customer"
          placeholder="Customer"
          description="Customer associated with this Rate"
          data={customers} // TODO(WOLFRED): add context menu's or add creatable to select fields
          isLoading={isCustomersLoading}
          isError={isCustomersError}
        />
        <SelectInput<RateFormValues>
          form={form}
          name="commodity"
          label="Commodity"
          placeholder="Commodity"
          description="Commodity associated with this Rate"
          data={commodities}
          isLoading={isCommoditiesLoading}
          isError={isCommoditiesError}
        />
        <ValidatedDateInput<RateFormValues>
          form={form}
          name="effectiveDate"
          label="Effective Date"
          placeholder="Effective Date"
          description="Effective Date for Rate"
          withAsterisk
        />
        <ValidatedDateInput<RateFormValues>
          form={form}
          name="expirationDate"
          label="Expiration Date"
          placeholder="Expiration Date"
          description="Expiration Date for Rate"
          withAsterisk
        />
        <SelectInput<RateFormValues>
          form={form}
          name="originLocation"
          label="Origin Location"
          placeholder="Origin Location"
          description="Origin Location associated with this Rate"
          data={locations}
          isLoading={isLocationsLoading}
          isError={isLocationsError}
        />
        <SelectInput<RateFormValues>
          form={form}
          name="destinationLocation"
          label="Destination Location"
          placeholder="Destination Location"
          description="Destination Location associated with this Rate"
          data={locations}
          isLoading={isLocationsLoading}
          isError={isLocationsError}
        />
        <SelectInput<RateFormValues>
          form={form}
          name="equipmentType"
          label="Equipment Type"
          placeholder="Equipment Type"
          description="Equipment Type associated with this Rate"
          data={equipmentTypes}
          isLoading={isEquipmentTypesLoading}
          isError={isEquipmentTypesError}
        />
        <SelectInput<RateFormValues>
          form={form}
          name="orderType"
          label="Order Type"
          placeholder="Order Type"
          description="Order Type associated with this Rate"
          data={orderTypes}
          isLoading={isOrderTypesLoading}
          isError={isOrderTypesError}
        />
        <SelectInput<RateFormValues>
          form={form}
          name="rateMethod"
          label="Rate Method"
          placeholder="Rate Method"
          description="Rate Method associated with this Rate"
          data={rateMethodChoices}
        />
        <ValidatedNumberInput<RateFormValues>
          form={form}
          name="rateAmount"
          label="Rate Amount"
          placeholder="Rate Amount"
          description="Rate Amount associated with this Rate"
        />
      </SimpleGrid>
      <ValidatedTextArea<RateFormValues>
        form={form}
        name="comments"
        label="Comments"
        description="Additional Comments for Rate"
        placeholder="Comments"
      />
    </Box>
  );
}

export function CreateRateModal() {
  const [showCreateModal, setShowCreateModal] = store.use("createModalOpen");
  const [activeTab, setActiveTab] = React.useState<string | null>("overview");
  const [loading, setLoading] = React.useState<boolean>(false);
  const isMobile = useMediaQuery("(max-width: 50em)");

  const queryClient = useQueryClient();

  const mutation = useMutation(
    (values: RateFormValues) => axios.post("/rates/", values),
    {
      onSuccess: () => {
        queryClient
          .invalidateQueries({
            queryKey: ["rate-table-data"],
          })
          .then(() => {
            notifications.show({
              title: "Success",
              message: "Rate created successfully",
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
            } else if (
              e.attr === "__all__" &&
              e.detail ===
                "Rate with this rate number and Organization already exists."
            ) {
              form.setFieldError("rate_number", e.detail);
            } else if (e.code === "invalid") {
              form.setFieldError(e.attr, e.detail);
            }
          });
        }
      },
      onSettled: () => {
        setLoading(false);
      },
    },
  );

  const form = useForm<RateFormValues>({
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

  const submitForm = (values: RateFormValues) => {
    setLoading(true);
    mutation.mutate(values);
  };

  console.log("Form Values", form.values);

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
    selectOrderType,
    isLoading: isOrderTypesLoading,
    isError: isOrderTypesError,
  } = useOrderTypes(showCreateModal);

  const {
    selectAccessorialChargeData,
    isLoading: isAccessorialChargesLoading,
    isError: isAccessorialChargesError,
  } = useAccessorialCharges(showCreateModal);

  if (!showCreateModal) return null;

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
                <Tabs.Tab value="overview">Overview</Tabs.Tab>
                <Tabs.Tab value="rate-billing-table">Billing Tables</Tabs.Tab>
              </Tabs.List>
              <Tabs.Panel value="overview" pt="xs">
                <CreateRateModalForm
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
                  orderTypes={selectOrderType}
                  isOrderTypesError={isOrderTypesError}
                  isOrderTypesLoading={isOrderTypesLoading}
                  form={form}
                />
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
