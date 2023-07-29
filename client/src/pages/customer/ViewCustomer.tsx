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

import { Suspense } from "react";
import { useParams } from "react-router-dom";
import { useQuery, useQueryClient } from "react-query";
import { getCustomerDetails } from "@/requests/CustomerRequestFactory";
import { ViewCustomerNavbar } from "./ViewCustomerNavbar";
import { Box, Button, Card, Grid, Skeleton, Tabs, Text } from "@mantine/core";
import { usePageStyles } from "@/styles/PageStyles";
import { ViewCustomerTable } from "@/components/customer/ViewCustomerTable";
import { CustomerStats } from "@/components/customer/CustomerStats";

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
      return getCustomerDetails(id);
    },
    initialData: () => {
      return queryClient.getQueryData(["customer", id]);
    },
    staleTime: Infinity,
  });

  if (isCustomerDataLoading) {
    return <Skeleton height={500} />;
  }

  if (!customerData) {
    return <div>Customer not found</div>;
  }

  return (
    <Grid gutter="md">
      <Grid.Col span={12} sm={6} md={4} lg={3} xl={3}>
        <Suspense fallback={<Skeleton height={500} />}>
          <ViewCustomerNavbar customer={customerData} />
        </Suspense>
      </Grid.Col>
      <Grid.Col span={12} sm={6} md={8} lg={9} xl={9}>
        <Tabs defaultValue="overview">
          <Tabs.List grow mb={20}>
            <Tabs.Tab value="overview">Overview</Tabs.Tab>
            <Tabs.Tab value="second">Second tab</Tabs.Tab>
            <Tabs.Tab value="third">Third tab</Tabs.Tab>
          </Tabs.List>

          {/** Overview Tab */}
          <Tabs.Panel value="overview" pt="xs">
            {id && <CustomerStats id={id} />}
            <Card className={classes.card} withBorder>
              <Box
                style={{
                  display: "flex",
                  justifyContent: "space-between",
                  alignItems: "center",
                }}
                my={20}
              >
                <Text fw={600} fz={20}>
                  Billing History
                </Text>
                <Button size="xs">View All</Button>
              </Box>

              <Suspense fallback={<Skeleton height={500} />}></Suspense>
              {id ? <ViewCustomerTable id={id} /> : <Skeleton height={500} />}
            </Card>
          </Tabs.Panel>
        </Tabs>
      </Grid.Col>
    </Grid>
  );
}
