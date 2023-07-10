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
  Burger,
  Collapse,
  Divider,
  Drawer,
  Group,
  Header,
  rem,
  ScrollArea,
  Text,
  ThemeIcon,
  UnstyledButton,
} from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import React from "react";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import {
  faBuildingColumns,
  faTruckFast,
} from "@fortawesome/pro-solid-svg-icons";
import { Link } from "react-router-dom";
import ActionButton from "../ActionButton";
import { useHeaderStyles } from "@/styles/HeaderStyles";
import { UserDownloads } from "@/components/layout/Header/_Partials/UserDownloads";
import { ThemeSwitcher } from "./Header/_Partials/ThemeSwitcher";
import { HeaderUserMenu } from "./HeaderUserMenu";
import { HeaderLogo } from "@/components/layout/Header/_Partials/HeaderLogo";
import { UserNotifications } from "./Header/UserNotifications";
import { faGrid2 } from "@fortawesome/pro-duotone-svg-icons";
import { SearchSpotlight } from "@/components/layout/Header/Search";
import { CustomerServiceMenuItem } from "@/components/layout/Header/_Partials/CustomerServiceMenuItem";
import { BillingMenuItem } from "@/components/layout/Header/_Partials/BillingMenuItem";
import { EquipmentMenuItem } from "@/components/layout/Header/_Partials/EquipmentMenuItem";
import { AdministratorMenuItem } from "@/components/layout/Header/_Partials/AdminMenuItem";
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

export function HeaderMegaMenu() {
  const [drawerOpened, { toggle: toggleDrawer, close: closeDrawer }] =
    useDisclosure(false);
  const [linksOpened] = useHeaderStore.use("linksOpen");
  // const [linksOpened, { toggle: toggleLinks }] = useDisclosure(false);
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
    <>
      <Header height={60} px="md">
        <Group position="apart" sx={{ height: "100%" }}>
          <HeaderLogo />
          <Group
            sx={{
              height: "100%",
            }}
            spacing={0}
            className={classes.hiddenMobile}
          >
            <Link to="/" className={classes.link}>
              Home
            </Link>

            {/* Customer Service */}
            <CustomerServiceMenuItem />

            {/* Billing & AR */}
            <BillingMenuItem />

            {/* Equipment Management */}
            <EquipmentMenuItem />

            {/* Administrator*/}
            <AdministratorMenuItem />
          </Group>

          <Group className={classes.hiddenMobile}>
            {/* Search */}
            <SearchSpotlight />

            {/* User Downloads */}
            <UserDownloads />

            {/* Notifications */}
            <UserNotifications />

            {/* Applications */}
            <ActionButton icon={faGrid2} />

            {/* Theme Switcher */}
            <ThemeSwitcher />

            {/* User Menu */}
            <HeaderUserMenu />
          </Group>

          <Burger
            opened={drawerOpened}
            onClick={toggleDrawer}
            className={classes.hiddenDesktop}
          />
        </Group>
      </Header>
      <Drawer
        opened={drawerOpened}
        onClose={closeDrawer}
        size="100%"
        padding="md"
        title="Navigation"
        className={classes.hiddenDesktop}
        zIndex={1000000}
      >
        <ScrollArea h={`calc(100vh - ${rem(60)})`} mx="-md">
          <Divider
            my="sm"
            color={theme.colorScheme === "dark" ? "dark.5" : "gray.1"}
          />

          <a href="#" className={classes.link}>
            Home
          </a>
          {/*<UnstyledButton className={classes.link} onClick={toggleLinks}>*/}
          {/*  <Center inline>*/}
          {/*    <Box component="span" mr={5}>*/}
          {/*      Features*/}
          {/*    </Box>*/}
          {/*    <FontAwesomeIcon*/}
          {/*      icon={faChevronDown}*/}
          {/*      size="xs"*/}
          {/*      color={theme.fn.primaryColor()}*/}
          {/*    />*/}
          {/*  </Center>*/}
          {/*</UnstyledButton>*/}
          <Collapse in={linksOpened}>{customerServiceLinks}</Collapse>
          <a href="#" className={classes.link}>
            Learn
          </a>
          <a href="#" className={classes.link}>
            Academy
          </a>

          <Divider
            my="sm"
            color={theme.colorScheme === "dark" ? "dark.5" : "gray.1"}
          />
        </ScrollArea>
      </Drawer>
    </>
  );
}
