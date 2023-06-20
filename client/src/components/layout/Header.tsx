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
  Burger,
  Button,
  Center,
  Collapse,
  Divider,
  Drawer,
  Group,
  Header,
  HoverCard,
  Indicator,
  rem,
  ScrollArea,
  SimpleGrid,
  Text,
  ThemeIcon,
  UnstyledButton,
} from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import React from "react";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import {
  faBell,
  faBuildingColumns,
  faChevronDown,
  faMagnifyingGlass,
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
import { faGrid, faGrid2 } from "@fortawesome/pro-duotone-svg-icons";

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
  const [linksOpened, { toggle: toggleLinks }] = useDisclosure(false);
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
            sx={{ height: "100%" }}
            spacing={0}
            className={classes.hiddenMobile}
          >
            <Link to="/" className={classes.link}>
              Home
            </Link>
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

            <Link to="/admin/users/" className={classes.link}>
              Admin Panel
            </Link>
            <a href="#" className={classes.link}>
              Academy
            </a>
          </Group>

          <Group className={classes.hiddenMobile}>
            {/* Search */}
            <ActionButton icon={faMagnifyingGlass} />

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
          <UnstyledButton className={classes.link} onClick={toggleLinks}>
            <Center inline>
              <Box component="span" mr={5}>
                Features
              </Box>
              <FontAwesomeIcon
                icon={faChevronDown}
                size="xs"
                color={theme.fn.primaryColor()}
              />
            </Center>
          </UnstyledButton>
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
