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

import { Box, Button, Card, SimpleGrid, Skeleton, Text } from "@mantine/core";
import React from "react";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faPencil } from "@fortawesome/pro-duotone-svg-icons";
import { Customer } from "@/types/apps/customer";
import { CustomerStats } from "@/components/customer/view/_partials/CustomerStats";
import { CustomerBillingHistoryTable } from "@/components/customer/view/_partials/CustomerBillingHistoryTable";
import { usePageStyles } from "@/styles/PageStyles";
import { USDollarFormat } from "@/lib/utils";

CustomerOverviewTab.defaultProps = {
  customer: null,
};

type CustomerOverviewTabProps = {
  customer?: Customer | null;
  isLoading: boolean;
};

type CustomerCreditBalanceProps = {
  customer: Customer;
};

function CustomerCreditBalance({ customer }: CustomerCreditBalanceProps) {
  const { classes } = usePageStyles();

  return (
    <Card className={classes.card} mt={20}>
      <Box
        style={{
          display: "flex",
          justifyContent: "space-between",
        }}
        my={20}
      >
        <Text className={classes.text} fw={600} fz={20}>
          Credit Balance
        </Text>
        <Button
          size="xs"
          leftIcon={<FontAwesomeIcon icon={faPencil} size="lg" />}
        >
          Adjust Balance
        </Button>
      </Box>
      <div
        style={{
          display: "flex",
          alignItems: "center",
        }}
      >
        <Text className={classes.text} mr="0.5%" fw={600} fz={20}>
          {USDollarFormat(customer.creditBalance)}
        </Text>

        <Text color="dimmed" fw={600} fz={15}>
          USD
        </Text>
      </div>
      <Text fz="xs" c="dimmed" mt={7}>
        Balance will increase the amount due on the customer&apos;s next
        invoice.
      </Text>
    </Card>
  );
}

export function CustomerOverviewTab({
  customer,
  isLoading,
}: CustomerOverviewTabProps) {
  const { classes } = usePageStyles();

  return isLoading ? (
    <>
      <SimpleGrid
        cols={4}
        spacing="sm"
        verticalSpacing="xl"
        breakpoints={[
          { maxWidth: "xl", cols: 2 },
          { maxWidth: "lg", cols: 1 },
          { maxWidth: "md", cols: 1 },
          { maxWidth: "xs", cols: 1 },
        ]}
      >
        <Skeleton height={150} />
        <Skeleton height={150} />
        <Skeleton height={150} />
        <Skeleton height={150} />
      </SimpleGrid>
      <Skeleton height={560} mt={20} />
    </>
  ) : (
    <>
      {customer && <CustomerStats customer={customer} />}
      <Card className={classes.card}>
        <Box
          style={{
            display: "flex",
            justifyContent: "space-between",
            alignItems: "center",
          }}
          my={20}
        >
          <Text className={classes.text} fw={600} fz={20}>
            Billing History
          </Text>
          <Button size="xs">View All</Button>
        </Box>
        {customer?.id && <CustomerBillingHistoryTable id={customer.id} />}
      </Card>
      {customer && <CustomerCreditBalance customer={customer} />}
    </>
  );
}
