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
  IconClock,
} from "@tabler/icons-react";
import { CustomerMetricProps } from "@/components/customer/CustomerStats";

const useStyles = createStyles((theme) => ({
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

export function PerformanceMetric({ customer }: CustomerMetricProps) {
  const { classes } = useStyles();

  const DiffIcon =
    customer?.on_time_performance.this_month_on_time_percentage >
    customer.on_time_performance.last_month_on_time_percentage
      ? IconArrowUpRight
      : IconArrowDownRight;

  return (
    <>
      <Paper withBorder p="md" radius="md" className={classes.root}>
        <Group position="apart">
          <Text size="xs" color="dimmed" className={classes.title}>
            On-Time Performance
          </Text>
          <IconClock className={classes.icon} size="1.4rem" stroke={1.5} />
        </Group>

        <Group align="flex-end" spacing="xs" mt={25}>
          <Text className={classes.value}>
            {customer.on_time_performance.this_month_on_time_percentage}%
          </Text>

          <Text
            color={
              customer?.on_time_performance.this_month_on_time_percentage >
              customer.on_time_performance.last_month_on_time_percentage
                ? "teal"
                : "red"
            }
            fz="sm"
            fw={500}
            className={classes.diff}
          >
            <span>{customer?.on_time_performance.on_time_diff}%</span>
            <DiffIcon size="1rem" stroke={1.5} />
          </Text>
        </Group>

        <Text fz="xs" c="dimmed" mt={7}>
          Compared to previous month
        </Text>
      </Paper>
    </>
  );
}
