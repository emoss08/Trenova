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
import { useParams } from "react-router-dom";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Grid, Tabs } from "@mantine/core";
import { getCustomerDetailsWithMetrics } from "@/services/CustomerRequestService";
import { ViewCustomerNavbar } from "@/components/customer/view/_partials/ViewCustomerNavbar";
import { CustomerOverviewTab } from "@/components/customer/view/_partials/CustomerOverviewTab";
import { CustomerProfileTab } from "@/components/customer/view/_partials/CustomerProfileTab";
import { customerStore as store } from "@/stores/CustomerStore";

export default function ViewCustomer() {
  const { id } = useParams<{ id: string }>();
  const queryClient = useQueryClient();
  const [activeTab, setActiveTab] = store.use("activeTab");

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

  return (
    <Grid gutter="md">
      <Grid.Col span={12} sm={6} md={4} lg={3} xl={3}>
        <ViewCustomerNavbar
          customer={customerData}
          isLoading={isCustomerDataLoading}
        />
      </Grid.Col>
      <Grid.Col span={12} sm={6} md={8} lg={9} xl={9}>
        <Tabs
          value={activeTab}
          onTabChange={setActiveTab}
          defaultValue="overview"
        >
          <Tabs.List grow mb={20}>
            <Tabs.Tab value="overview">Overview</Tabs.Tab>
            <Tabs.Tab value="profile">Profiles & Settings</Tabs.Tab>
            <Tabs.Tab value="third">Events & Logs</Tabs.Tab>
          </Tabs.List>

          {/** Overview Tab */}
          <Tabs.Panel value="overview" pt="xs">
            <Grid.Col span={12} sm={12} md={12} lg={12} xl={12}>
              <CustomerOverviewTab
                customer={customerData}
                isLoading={isCustomerDataLoading}
              />
            </Grid.Col>
          </Tabs.Panel>

          {/** Profiles Tab */}
          <Tabs.Panel value="profile" pt="xs">
            {customerData && (
              <CustomerProfileTab customerId={customerData.id} />
            )}
          </Tabs.Panel>
        </Tabs>
      </Grid.Col>
    </Grid>
  );
}
