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

import { createStyles, Group, Paper, rem, Text } from "@mantine/core";
import {
  IconArrowDownRight,
  IconArrowUpRight,
  IconBox,
} from "@tabler/icons-react";
import { useQuery, useQueryClient } from "react-query";
import { getCustomerOrderMetrics } from "@/requests/CustomerRequestFactory";
import { MetricsSkeleton } from "@/components/customer/_partials/MetricsSkeleton";

type TotalOrdersProps = {
  id: string;
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

export function TotalOrders({ id }: TotalOrdersProps) {
  const { classes } = useStyles();
  const queryClient = useQueryClient();

  const { data: orderMetrics, isLoading: isOrderMetricsLoading } = useQuery({
    queryKey: ["customerOrderMetrics", id],
    queryFn: () => {
      if (!id) {
        return Promise.resolve(null);
      }
      return getCustomerOrderMetrics(id);
    },
    initialData: () => {
      return queryClient.getQueryData(["customerOrderMetrics", id]);
    },
    staleTime: Infinity,
  });

  const DiffIcon =
    orderMetrics && orderMetrics?.total_order_metrics.last_month_diff > 0
      ? IconArrowUpRight
      : IconArrowDownRight;

  return (
    <>
      {isOrderMetricsLoading ? (
        <MetricsSkeleton />
      ) : (
        <Paper withBorder p="md" radius="md">
          <Group position="apart">
            <Text size="xs" color="dimmed" className={classes.title}>
              Total Orders
            </Text>
            <IconBox className={classes.icon} size="1.4rem" stroke={1.5} />
          </Group>

          <Group align="flex-end" spacing="xs" mt={25}>
            <Text className={classes.value}>
              {orderMetrics?.total_order_metrics.total_orders}
            </Text>
            <Text
              color={
                orderMetrics &&
                orderMetrics?.total_order_metrics.last_month_diff >
                  orderMetrics.total_order_metrics.month_before_last_diff
                  ? "teal"
                  : "red"
              }
              fz="sm"
              fw={500}
              className={classes.diff}
            >
              <span>
                {orderMetrics &&
                  orderMetrics?.total_order_metrics.last_month_diff}
                %
              </span>
              <DiffIcon size="1rem" stroke={1.5} />
            </Text>
          </Group>

          <Text fz="xs" c="dimmed" mt={7}>
            Compared to previous month
          </Text>
        </Paper>
      )}
    </>
  );
}
