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
  Box,
  Button,
  Divider,
  Group,
  Modal,
  Select,
  SimpleGrid,
  Tabs,
  Text,
  Textarea,
  TextInput,
} from "@mantine/core";
import React from "react";
import { useMediaQuery } from "@mantine/hooks";
import { DateInput } from "@mantine/dates";
import { Rate, RateBillingTable } from "@/types/dispatch";
import { TChoiceProps } from "@/types";
import { useFormStyles } from "@/assets/styles/FormStyles";
import { useRateStore as store } from "@/stores/DispatchStore";
import { useAccessorialCharges } from "@/hooks/useAccessorialCharges";
import { rateMethodChoices, yesAndNoChoicesBoolean } from "@/lib/constants";
import { useCustomers } from "@/hooks/useCustomers";
import { useCommodities } from "@/hooks/useCommodities";
import { useLocations } from "@/hooks/useLocations";
import { useEquipmentTypes } from "@/hooks/useEquipmentType";
import { useOrderTypes } from "@/hooks/useOrderTypes";
import { ViewSelectInput } from "@/components/common/fields/SelectInput";

function ViewRateBillingTableModal({
  accessorialCharges,
  rateBillingTables,
}: {
  accessorialCharges: ReadonlyArray<TChoiceProps>;
  rateBillingTables: Array<RateBillingTable>;
}) {
  const { classes } = useFormStyles();
  const fields = rateBillingTables?.map((item, index) => {
    const { accessorialCharge } = item;

    return (
      <>
        <Group mt="xs" key={accessorialCharge}>
          <Select
            data={accessorialCharges}
            value={rateBillingTables[index].accessorialCharge}
            label="Accessorial Charge"
            placeholder="Select Accessorial Charge"
            description="Accessorial Charge associated with this Rate"
            className={classes.fields}
            variant="filled"
            readOnly
          />
          <TextInput
            label="Description"
            value={rateBillingTables[index].description || ""}
            placeholder="Description"
            description="Description for Rate Billing Table"
            className={classes.fields}
            variant="filled"
            readOnly
          />
          <TextInput
            label="Unit"
            value={rateBillingTables[index].unit}
            placeholder="Unit"
            description="Unit for Rate Billing Table"
            className={classes.fields}
            variant="filled"
            readOnly
          />
          <TextInput
            label="Charge Amount"
            placeholder="Charge Amount"
            value={parseFloat(
              String(rateBillingTables[index].chargeAmount),
            ).toFixed(2)}
            description="Charge Amount for Rate Billing Table"
            className={classes.fields}
            variant="filled"
            readOnly
          />
          <TextInput
            label="Sub Total"
            placeholder="Sub Total"
            value={parseFloat(
              String(rateBillingTables[index].subTotal),
            ).toFixed(2)}
            description="Sub Total for Rate Billing Table"
            className={classes.fields}
            variant="filled"
            readOnly
          />
        </Group>
      </>
    );
  });

  if (rateBillingTables.length === 0) {
    return (
      <Box>
        <Text fw={400} className={classes.text}>
          No Rate Billing Table found
        </Text>
      </Box>
    );
  }

  return <Box>{fields}</Box>;
}

type ViewRateModalFormProps = {
  customers: ReadonlyArray<TChoiceProps>;
  commodities: ReadonlyArray<TChoiceProps>;
  orderTypes: ReadonlyArray<TChoiceProps>;
  equipmentTypes: ReadonlyArray<TChoiceProps>;
  locations: ReadonlyArray<TChoiceProps>;
  rate: Rate;
};

function ViewRateModalForm({
  customers,
  commodities,
  orderTypes,
  equipmentTypes,
  locations,
  rate,
}: ViewRateModalFormProps): React.ReactElement {
  const { classes } = useFormStyles();

  return (
    <Box className={classes.div}>
      <SimpleGrid cols={4} breakpoints={[{ maxWidth: "lg", cols: 1 }]}>
        <ViewSelectInput
          label="Is Active"
          description="Is this Rate active?"
          placeholder="Is Active"
          data={yesAndNoChoicesBoolean}
          className={classes.fields}
          value={rate.isActive}
          variant="filled"
          readOnly
        />
        <TextInput
          label="Rate Number"
          placeholder="Rate Number"
          description="Unique Number for Rate"
          className={classes.fields}
          value={rate.rateNumber}
          variant="filled"
          readOnly
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
        <Select
          label="Customer"
          placeholder="Customer"
          description="Customer associated with this Rate"
          data={customers}
          className={classes.fields}
          value={rate.customer}
          variant="filled"
          readOnly
        />
        <Select
          label="Commodity"
          placeholder="Commodity"
          description="Commodity associated with this Rate"
          data={commodities}
          className={classes.fields}
          value={rate.commodity}
          variant="filled"
          readOnly
        />
        <DateInput
          label="Effective Date"
          placeholder="Effective Date"
          description="Effective Date for Rate"
          className={classes.fields}
          value={new Date(rate.effectiveDate)}
          variant="filled"
          readOnly
        />
        <DateInput
          label="Expiration Date"
          placeholder="Expiration Date"
          description="Expiration Date for Rate"
          className={classes.fields}
          value={new Date(rate.expirationDate)}
          variant="filled"
          readOnly
        />
        <Select
          label="Origin Location"
          placeholder="Origin Location"
          description="Origin Location associated with this Rate"
          data={locations}
          className={classes.fields}
          value={rate.originLocation}
          variant="filled"
          readOnly
        />
        <Select
          label="Destination Location"
          placeholder="Destination Location"
          description="Dest. Location associated with this Rate"
          data={locations}
          className={classes.fields}
          value={rate.destinationLocation}
          variant="filled"
          readOnly
        />
        <Select
          label="Equipment Type"
          placeholder="Equipment Type"
          description="Equipment Type associated with this Rate"
          data={equipmentTypes}
          className={classes.fields}
          value={rate.equipmentType}
          variant="filled"
          readOnly
        />
        <Select
          label="Order Type"
          placeholder="Order Type"
          description="Order Type associated with this Rate"
          data={orderTypes}
          className={classes.fields}
          value={rate.orderType}
          variant="filled"
          readOnly
        />
        <Select
          label="Rate Method"
          placeholder="Rate Method"
          description="Rate Method associated with this Rate"
          data={rateMethodChoices}
          className={classes.fields}
          value={rate.rateMethod}
          variant="filled"
          readOnly
        />
        <TextInput
          label="Rate Amount"
          placeholder="Rate Amount"
          description="Rate Amount associated with this Rate"
          className={classes.fields}
          value={parseFloat(String(rate.rateAmount)).toFixed(2)}
          variant="filled"
          readOnly
        />
        <TextInput
          label="Distance Override"
          placeholder="Distance Override"
          description="Dist. Override associated with this Rate"
          className={classes.fields}
          value={rate.distanceOverride || ""}
          variant="filled"
          readOnly
        />
      </SimpleGrid>
      <Textarea
        label="Comments"
        description="Additional Comments for Rate"
        placeholder="Comments"
        className={classes.fields}
        value={rate.comments || ""}
        variant="filled"
        readOnly
      />
    </Box>
  );
}

export function ViewRateModal() {
  const [showViewModal, setShowViewModal] = store.use("viewModalOpen");
  const [rate] = store.use("selectedRecord");
  const { classes } = useFormStyles();
  const [activeTab, setActiveTab] = React.useState<string | null>("overview");
  const isMobile = useMediaQuery("(max-width: 50em)");

  const { selectCustomersData } = useCustomers(showViewModal);

  const { selectCommodityData } = useCommodities(showViewModal);

  const { selectLocationData } = useLocations(showViewModal);

  const { selectEquipmentType } = useEquipmentTypes(showViewModal);

  const { selectOrderType } = useOrderTypes(showViewModal);

  const { selectAccessorialChargeData } = useAccessorialCharges(showViewModal);

  return (
    <Modal.Root
      opened={showViewModal}
      onClose={() => setShowViewModal(false)}
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
          <Modal.Title>View Rate</Modal.Title>
          <Modal.CloseButton />
        </Modal.Header>
        <Modal.Body>
          <Tabs
            defaultValue="overview"
            value={activeTab}
            onTabChange={setActiveTab}
          >
            <Tabs.List>
              <Tabs.Tab value="overview">Overview</Tabs.Tab>
              <Tabs.Tab value="rate-billing-table">Rate Billing Table</Tabs.Tab>
            </Tabs.List>
            <Tabs.Panel value="overview" pt="xs">
              {rate && (
                <ViewRateModalForm
                  orderTypes={selectOrderType}
                  customers={selectCustomersData}
                  commodities={selectCommodityData}
                  equipmentTypes={selectEquipmentType}
                  locations={selectLocationData}
                  rate={rate}
                />
              )}
            </Tabs.Panel>
            <Tabs.Panel value="rate-billing-table" pt="xs">
              {rate && (
                <ViewRateBillingTableModal
                  rateBillingTables={rate.rateBillingTables || []}
                  accessorialCharges={selectAccessorialChargeData}
                />
              )}
            </Tabs.Panel>
          </Tabs>
          <Group position="right" mt="md">
            <Button
              color="white"
              type="submit"
              className={classes.control}
              onClick={() => {
                store.set("viewModalOpen", false);
                store.set("editModalOpen", true);
              }}
            >
              Edit Rate
            </Button>
          </Group>
        </Modal.Body>
      </Modal.Content>
    </Modal.Root>
  );
}
