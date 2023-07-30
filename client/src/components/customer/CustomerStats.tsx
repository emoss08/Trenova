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

import { createStyles, SimpleGrid, rem } from "@mantine/core";
import { OrdersMetric } from "@/components/customer/_partials/OrdersMetric";
import { useQuery, useQueryClient } from "react-query";
import { getCustomerMetrics } from "@/requests/CustomerRequestFactory";
import { CustomerOrderMetrics } from "@/types/apps/customer";
import { RevenueMetric } from "@/components/customer/_partials/RevenueMetric";
import { PerformanceMetric } from "@/components/customer/_partials/PerformanceMetric";
import { MetricsSkeleton } from "@/components/customer/_partials/MetricsSkeleton";
import { MileageMetric } from "@/components/customer/_partials/MileageMetric";

export type CustomerMetricProps = {
  metrics: CustomerOrderMetrics;
};

const useStyles = createStyles((theme) => ({
  root: {
    paddingBottom: rem(20),
  },

  value: {
    fontSize: rem(24),
    fontWeight: 700,
    lineHeight: 1,
  },

  diff: {
    lineHeight: 1,
    display: "flex",
    alignItems: "center",
  },

  icon: {
    color:
      theme.colorScheme === "dark"
        ? theme.colors.dark[3]
        : theme.colors.gray[4],
  },

  title: {
    fontWeight: 700,
    textTransform: "uppercase",
  },
}));

type CustomerStatsProps = {
  id: string;
};

export function CustomerStats({ id }: CustomerStatsProps) {
  const { classes } = useStyles();
  const queryClient = useQueryClient();

  const { data: customerMetrics, isLoading: isCustomerMetricsLoading } =
    useQuery({
      queryKey: ["customerMetrics", id],
      queryFn: () => {
        if (!id) {
          return Promise.resolve(null);
        }
        return getCustomerMetrics(id);
      },
      initialData: () => {
        return queryClient.getQueryData(["customerMetrics", id]);
      },
      staleTime: Infinity,
    });

  return (
    <div className={classes.root}>
      <SimpleGrid
        cols={4}
        breakpoints={[
          { maxWidth: "md", cols: 2 },
          { maxWidth: "xs", cols: 1 },
        ]}
      >
        {isCustomerMetricsLoading ? (
          <MetricsSkeleton />
        ) : (
          customerMetrics && (
            <>
              <OrdersMetric metrics={customerMetrics} />
              <RevenueMetric metrics={customerMetrics} />
              <PerformanceMetric metrics={customerMetrics} />
              <MileageMetric metrics={customerMetrics} />
            </>
          )
        )}
      </SimpleGrid>
    </div>
  );
}
