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
  createStyles,
  SimpleGrid,
  rem,
  Paper,
  Group,
  Text,
  Tooltip,
} from "@mantine/core";
import {
  IconArrowDownRight,
  IconArrowUpRight,
  IconBox,
  IconClock,
  IconCurrencyDollar,
  IconTruckDelivery,
} from "@tabler/icons-react";
import { Customer } from "@/types/apps/customer";
import { truncateText, USDollarFormat } from "@/lib/utils";

export type CustomerMetricProps = {
  customer: Customer;
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

const useStatStyle = createStyles((theme) => ({
  root: {
    backgroundColor:
      theme.colorScheme === "dark" ? theme.colors.dark[7] : "white",
  },

  value: {
    fontSize: rem(24),
    fontWeight: 700,
    lineHeight: 1,
    color: theme.colorScheme === "dark" ? "white" : "black",
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

function RevenueMetric({ customer }: CustomerMetricProps) {
  const { classes } = useStyles();

  const DiffIcon =
    customer?.totalRevenueMetrics.lastMonthDiff >
    customer.totalRevenueMetrics.monthBeforeLastDiff
      ? IconArrowUpRight
      : IconArrowDownRight;

  return (
    <Paper p="md" radius="md" className={classes.root}>
      <Group position="apart">
        <Text size="xs" color="dimmed" className={classes.title}>
          Total Revenue
        </Text>
        <IconCurrencyDollar
          className={classes.icon}
          size="1.4rem"
          stroke={1.5}
        />
      </Group>

      <Group align="flex-end" spacing="xs" mt={25}>
        <Tooltip
          withArrow
          label={USDollarFormat(customer?.totalRevenueMetrics.totalRevenue)}
        >
          <Text className={classes.value}>
            {truncateText(
              USDollarFormat(customer?.totalRevenueMetrics.totalRevenue),
              9,
            )}
          </Text>
        </Tooltip>

        <Text
          color={
            customer?.totalRevenueMetrics.lastMonthDiff >
            customer.totalRevenueMetrics.monthBeforeLastDiff
              ? "teal"
              : "red"
          }
          fz="sm"
          fw={500}
          className={classes.diff}
        >
          <span>{customer?.totalRevenueMetrics.lastMonthDiff}%</span>
          <DiffIcon size="1rem" stroke={1.5} />
        </Text>
      </Group>

      <Text fz="xs" c="dimmed" mt={7}>
        Compared to previous month
      </Text>
    </Paper>
  );
}

function PerformanceMetric({ customer }: CustomerMetricProps) {
  const { classes } = useStatStyle();

  const DiffIcon =
    customer?.onTimePerformance.thisMonthOnTimePercentage >
    customer.onTimePerformance.lastMonthOnTimePercentage
      ? IconArrowUpRight
      : IconArrowDownRight;

  return (
    <Paper p="md" radius="md" className={classes.root}>
      <Group position="apart">
        <Text size="xs" color="dimmed" className={classes.title}>
          On-Time Performance
        </Text>
        <IconClock className={classes.icon} size="1.4rem" stroke={1.5} />
      </Group>

      <Group align="flex-end" spacing="xs" mt={25}>
        <Text className={classes.value}>
          {customer.onTimePerformance.thisMonthOnTimePercentage}%
        </Text>

        <Text
          color={
            customer?.onTimePerformance.thisMonthOnTimePercentage >
            customer.onTimePerformance.lastMonthOnTimePercentage
              ? "teal"
              : "red"
          }
          fz="sm"
          fw={500}
          className={classes.diff}
        >
          <span>{customer?.onTimePerformance.onTimeDiff}%</span>
          <DiffIcon size="1rem" stroke={1.5} />
        </Text>
      </Group>

      <Text fz="xs" c="dimmed" mt={7}>
        Compared to previous month
      </Text>
    </Paper>
  );
}

function OrdersMetric({ customer }: CustomerMetricProps) {
  const { classes } = useStatStyle();

  const DiffIcon =
    customer?.totalOrderMetrics.lastMonthDiff >
    customer.totalOrderMetrics.monthBeforeLastDiff
      ? IconArrowUpRight
      : IconArrowDownRight;

  return (
    <Paper p="md" radius="md" className={classes.root}>
      <Group position="apart">
        <Text size="xs" color="dimmed" className={classes.title}>
          Total Orders
        </Text>
        <IconBox className={classes.icon} size="1.4rem" stroke={1.5} />
      </Group>

      <Group align="flex-end" spacing="xs" mt={25}>
        <Text className={classes.value}>
          {customer?.totalOrderMetrics.totalOrders}
        </Text>
        <Text
          color={
            customer?.totalOrderMetrics.lastMonthDiff >
            customer.totalOrderMetrics.monthBeforeLastDiff
              ? "teal"
              : "red"
          }
          fz="sm"
          fw={500}
          className={classes.diff}
        >
          <span>{customer?.totalOrderMetrics.lastMonthDiff}%</span>
          <DiffIcon size="1rem" stroke={1.5} />
        </Text>
      </Group>

      <Text fz="xs" c="dimmed" mt={7}>
        Compared to previous month
      </Text>
    </Paper>
  );
}

function MileageMetric({ customer }: CustomerMetricProps) {
  const { classes } = useStatStyle();

  const DiffIcon =
    customer?.totalMileageMetrics.thisMonthMiles >
    customer.totalMileageMetrics.lastMonthMiles
      ? IconArrowUpRight
      : IconArrowDownRight;

  return (
    <Paper p="md" radius="md" className={classes.root}>
      <Group position="apart">
        <Text size="xs" color="dimmed" className={classes.title}>
          Total Mileage
        </Text>
        <IconTruckDelivery
          className={classes.icon}
          size="1.4rem"
          stroke={1.5}
        />
      </Group>

      <Group align="flex-end" spacing="xs" mt={25}>
        <Text className={classes.value}>
          {customer.totalMileageMetrics.thisMonthMiles}
        </Text>

        <Text
          color={
            customer.totalMileageMetrics.thisMonthMiles >
            customer.totalMileageMetrics.lastMonthMiles
              ? "teal"
              : "red"
          }
          fz="sm"
          fw={500}
          className={classes.diff}
        >
          <span>{customer.totalMileageMetrics.mileageDiff}%</span>
          <DiffIcon size="1rem" stroke={1.5} />
        </Text>
      </Group>

      <Text fz="xs" c="dimmed" mt={7}>
        Compared to previous month
      </Text>
    </Paper>
  );
}

type CustomerStatsProps = {
  customer: Customer;
};

export function CustomerStats({ customer }: CustomerStatsProps) {
  const { classes } = useStyles();

  return (
    <div className={classes.root}>
      <SimpleGrid
        cols={4}
        breakpoints={[
          { maxWidth: "md", cols: 2 },
          { maxWidth: "xs", cols: 1 },
        ]}
      >
        <OrdersMetric customer={customer} />
        <RevenueMetric customer={customer} />
        <PerformanceMetric customer={customer} />
        <MileageMetric customer={customer} />
      </SimpleGrid>
    </div>
  );
}
