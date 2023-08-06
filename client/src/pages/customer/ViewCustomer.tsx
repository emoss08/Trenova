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

import { useParams } from "react-router-dom";
import { useQuery, useQueryClient } from "react-query";
import { Box, Button, Card, Grid, Tabs, Text } from "@mantine/core";
import { getCustomerDetailsWithMetrics } from "@/requests/CustomerRequestFactory";
import { usePageStyles } from "@/styles/PageStyles";
import { CustomerStats } from "@/components/customer/CustomerStats";
import { CustomerBillingHistoryTable } from "@/components/customer/CustomerBillingHistoryTable";
import { ViewCustomerNavbar } from "@/components/customer/ViewCustomerNavbar";
import { MetricsSkeleton } from "@/components/customer/_partials/MetricsSkeleton";
import { CustomerCreditBalance } from "@/components/customer/CustomerCreditBalance";

export default function ViewCustomer() {
  const { classes } = usePageStyles();
  const { id } = useParams<{ id: string }>();
  const queryClient = useQueryClient();

  const { data: customerData, isLoading: isCustomerDataLoading } = useQuery({
    queryKey: ["customer", id],
    queryFn: () => {
      if (!id) {
        return Promise.resolve(null);
      }
      return getCustomerDetailsWithMetrics(id);
    },
    initialData: () => queryClient.getQueryData(["customerWithMetrics", id]),
    staleTime: Infinity,
  });

  return isCustomerDataLoading ? (
    <MetricsSkeleton />
  ) : (
    <Grid gutter="md">
      <Grid.Col span={12} sm={6} md={4} lg={3} xl={3}>
        {customerData && <ViewCustomerNavbar customer={customerData} />}
      </Grid.Col>
      <Grid.Col span={12} sm={6} md={8} lg={9} xl={9}>
        <Tabs defaultValue="overview">
          <Tabs.List grow mb={20}>
            <Tabs.Tab value="overview">Overview</Tabs.Tab>
            <Tabs.Tab value="second">Events & Logs</Tabs.Tab>
            <Tabs.Tab value="third">Statements</Tabs.Tab>
          </Tabs.List>

          {/** Overview Tab */}
          <Tabs.Panel value="overview" pt="xs">
            {customerData && <CustomerStats customer={customerData} />}
            <Card className={classes.card} withBorder>
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
              {id && <CustomerBillingHistoryTable id={id} />}
            </Card>
            {customerData && (
              <CustomerCreditBalance customer={customerData} />
            )}
          </Tabs.Panel>
        </Tabs>
      </Grid.Col>
    </Grid>
  );
}
