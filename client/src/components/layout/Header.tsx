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
  createStyles,
  Divider,
  Drawer,
  Group,
  Header,
  HoverCard,
  rem,
  ScrollArea,
  SimpleGrid,
  Skeleton,
  Text,
  ThemeIcon,
  UnstyledButton,
  Image,
  useMantineColorScheme,
  Menu,
  ActionIcon,
} from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import React from "react";
import { getUserId, getUserOrganizationId } from "@/lib/utils";
import { useQuery, useQueryClient } from "react-query";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import {
  faBell,
  faBuildingColumns,
  faChevronDown,
  faMagnifyingGlass,
  faTruckFast,
} from "@fortawesome/pro-solid-svg-icons";
import { Link } from "react-router-dom";
import {
  faDisplay,
  faDownload,
  faMoon,
  faSun,
} from "@fortawesome/pro-duotone-svg-icons";
import { getOrganizationDetails } from "@/requests/OrganizationRequestFactory";
import { getUserDetails } from "@/requests/UserRequestFactory";
import ActionButton from "../ActionButton";
import HeaderUserMenu from "./HeaderUserMenu";

const useStyles = createStyles((theme) => ({
  link: {
    display: "flex",
    alignItems: "center",
    height: "100%",
    paddingLeft: theme.spacing.md,
    paddingRight: theme.spacing.md,
    textDecoration: "none",
    color: theme.colorScheme === "dark" ? theme.white : theme.black,
    fontWeight: 500,
    fontSize: theme.fontSizes.sm,

    [theme.fn.smallerThan("sm")]: {
      height: rem(42),
      display: "flex",
      alignItems: "center",
      width: "100%",
    },

    ...theme.fn.hover({
      backgroundColor:
        theme.colorScheme === "dark"
          ? theme.colors.dark[6]
          : theme.colors.gray[0],
    }),
  },

  subLink: {
    width: "100%",
    padding: `${theme.spacing.xs} ${theme.spacing.md}`,
    borderRadius: theme.radius.md,

    ...theme.fn.hover({
      backgroundColor:
        theme.colorScheme === "dark"
          ? theme.colors.dark[7]
          : theme.colors.gray[0],
    }),

    "&:active": theme.activeStyles,
  },
  logoText: {
    color: theme.colorScheme === "dark" ? "white" : "black",
  },
  dropdownFooter: {
    backgroundColor:
      theme.colorScheme === "dark"
        ? theme.colors.dark[7]
        : theme.colors.gray[0],
    margin: `calc(${theme.spacing.md} * -1)`,
    marginTop: theme.spacing.sm,
    padding: `${theme.spacing.md} calc(${theme.spacing.md} * 2)`,
    paddingBottom: theme.spacing.xl,
    borderTop: `${rem(1)} solid ${
      theme.colorScheme === "dark" ? theme.colors.dark[5] : theme.colors.gray[1]
    }`,
  },

  hiddenMobile: {
    [theme.fn.smallerThan("sm")]: {
      display: "none",
    },
  },

  hiddenDesktop: {
    [theme.fn.largerThan("sm")]: {
      display: "none",
    },
  },
  hoverEffect: {
    svg: {
      color:
        theme.colorScheme === "dark"
          ? theme.colors.gray[5]
          : theme.colors.gray[9],
    },
    "&:hover svg": {
      color:
        theme.colorScheme === "dark"
          ? theme.colors.gray[0]
          : theme.colors.gray[7],
    },
  },
}));

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
  const { classes, theme } = useStyles();
  const { colorScheme, toggleColorScheme } = useMantineColorScheme();
  const [isDownloadMenuOpen, setDownloadMenuOpen] =
    React.useState<boolean>(false);
  const queryClient = useQueryClient();

  const getThemeIcon = () => {
    if (colorScheme === "light") {
      return <FontAwesomeIcon icon={faSun} />;
    } else if (colorScheme === "dark") {
      return <FontAwesomeIcon icon={faMoon} />;
    } else {
      return <FontAwesomeIcon icon={faDisplay} />;
    }
  };

  const userId = getUserId() || "";
  const { data: userData, isLoading: isUserDataLoading } = useQuery({
    queryKey: ["user", userId],
    queryFn: () => {
      if (!userId) {
        return Promise.resolve(null);
      }
      return getUserDetails(userId);
    },
    initialData: () => {
      return queryClient.getQueryData(["user", userId]);
    },
    staleTime: Infinity, // never refetch
  });

  const organizationId = getUserOrganizationId() || "";
  const { data: organizationData, isLoading: isOrganizationDataLoading } =
    useQuery({
      queryKey: ["organization", organizationId],
      queryFn: () => {
        if (!organizationId) {
          return Promise.resolve(null);
        }
        return getOrganizationDetails(organizationId);
      },
      initialData: () => {
        return queryClient.getQueryData(["organization", organizationId]);
      },
      staleTime: Infinity, // never refetch
    });

  const isLoading = isOrganizationDataLoading || isUserDataLoading;

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
          {isLoading ? (
            <Skeleton width={rem(150)} height={rem(40)} />
          ) : organizationData?.logo ? (
            <Link to="/" style={{ textDecoration: "none" }}>
              <Image
                radius="md"
                width={rem(120)}
                height={rem(40)}
                maw={rem(150)}
                src={organizationData?.logo}
                alt="Organization Logo"
              />
            </Link>
          ) : (
            <Link to="/" style={{ textDecoration: "none" }}>
              <Text size="lg" fw={600} className={classes.logoText}>
                {organizationData?.name}
              </Text>
            </Link>
          )}
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
            <Menu
              position="bottom-end"
              width={200}
              opened={isDownloadMenuOpen}
              onChange={setDownloadMenuOpen}
              withinPortal
            >
              <Menu.Target>
                <ActionIcon className={classes.hoverEffect}>
                  <FontAwesomeIcon icon={faDownload} />
                </ActionIcon>
              </Menu.Target>
              <Menu.Dropdown>
                <Menu.Label>Downloads</Menu.Label>
                <Menu.Item icon={<FontAwesomeIcon icon={faDownload} />}>
                  Weird file
                </Menu.Item>
              </Menu.Dropdown>
            </Menu>

            {/* Notifications */}
            <ActionButton icon={faBell} />

            {/* Theme Switcher */}
            <Menu position="bottom-end" width={200} withinPortal>
              <Menu.Target>
                <ActionIcon className={classes.hoverEffect}>
                  {getThemeIcon()}
                </ActionIcon>
              </Menu.Target>

              <Menu.Dropdown>
                <Menu.Label>Theme Mode</Menu.Label>
                <Menu.Item
                  onClick={() => toggleColorScheme("light")}
                  icon={<FontAwesomeIcon icon={faSun} />}
                >
                  Light Theme
                </Menu.Item>
                <Menu.Item
                  onClick={() => toggleColorScheme("dark")}
                  icon={<FontAwesomeIcon icon={faMoon} />}
                >
                  Dark Theme
                </Menu.Item>
              </Menu.Dropdown>
            </Menu>
            {isLoading ? (
              <Skeleton width={rem(150)} height={rem(40)} circle />
            ) : userData ? (
              <HeaderUserMenu user={userData} />
            ) : (
              <div>No user data available</div>
            )}
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
