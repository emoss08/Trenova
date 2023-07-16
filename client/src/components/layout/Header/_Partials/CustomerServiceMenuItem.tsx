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
  Anchor,
  Box,
  Button,
  Center,
  Divider,
  Group,
  HoverCard,
  Menu,
  SimpleGrid,
  Text,
  ThemeIcon,
  UnstyledButton,
} from "@mantine/core";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import React from "react";
import {
  faBuildingColumns,
  faChevronDown,
  faTruckFast,
} from "@fortawesome/pro-solid-svg-icons";
import { useHeaderStyles } from "@/styles/HeaderStyles";
import { Link } from "react-router-dom";
import { useHeaderStore } from "@/stores/HeaderStore";

const navigationLinks = [
  {
    icon: faTruckFast,
    title: "Shipment Management",
    description: "Manage your shipments with ease and efficiency",
    href: "/shipments",
  },
  {
    icon: faBuildingColumns,
    title: "Billing Management",
    description: "Manage your billing and invoices with ease and efficiency",
    href: "/billing",
  },
];

export const CustomerServiceMenuItem = () => {
  const { classes, theme } = useHeaderStyles();

  const customerServiceLinks = navigationLinks.map((item) => (
    <UnstyledButton className={classes.subLink} key={item.title}>
      <Group noWrap align="flex-start">
        <ThemeIcon size={34} variant="default" radius="md">
          <FontAwesomeIcon icon={item.icon} color={theme.fn.primaryColor()} />
        </ThemeIcon>
        <div>
          <Text size="sm" fw={500}>
            {item.title}
          </Text>
          <Text size="xs" color="dimmed">
            {item.description}
          </Text>
        </div>
      </Group>
    </UnstyledButton>
  ));

  return (
    <HoverCard
      width={600}
      position="bottom"
      radius="md"
      shadow="md"
      withinPortal
    >
      <HoverCard.Target>
        <Link to="#" className={classes.link}>
          <Center inline>
            <Box component="span" mr={5}>
              Customer Service
            </Box>
            <FontAwesomeIcon
              icon={faChevronDown}
              size="xs"
              color={theme.fn.primaryColor()}
            />
          </Center>
        </Link>
      </HoverCard.Target>

      <HoverCard.Dropdown sx={{ overflow: "hidden" }}>
        <Group position="apart" px="md">
          <Text fw={500}>Customer Service</Text>
          <Anchor href="#" fz="xs">
            View all
          </Anchor>
        </Group>

        <Divider
          my="sm"
          mx="-md"
          color={theme.colorScheme === "dark" ? "dark.5" : "gray.1"}
        />

        <SimpleGrid cols={2} spacing={0}>
          {customerServiceLinks}
        </SimpleGrid>

        <div className={classes.dropdownFooter}>
          <Group position="apart">
            <div>
              <Text fw={500} fz="sm">
                Get started
              </Text>
              <Text size="xs" color="dimmed">
                Their food sources have decreased, and their numbers
              </Text>
            </div>
            <Button variant="default">Get started</Button>
          </Group>
        </div>
      </HoverCard.Dropdown>
    </HoverCard>
  );
};
