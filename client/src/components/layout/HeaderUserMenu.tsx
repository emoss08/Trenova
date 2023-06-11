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

import { User } from "@/types/user";
import React, { useState } from "react";
import {
  Container,
  Avatar,
  Group,
  Text,
  Menu,
  Burger,
  rem,
  createStyles,
  ActionIcon,
  Indicator,
} from "@mantine/core";
import {
  IconHeart,
  IconStar,
  IconMessage,
  IconPlayerPause,
  IconTrash,
  IconSwitchHorizontal,
} from "@tabler/icons-react";
import { useDisclosure } from "@mantine/hooks";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faGear, faRightFromBracket } from "@fortawesome/pro-regular-svg-icons";
import { Link } from "react-router-dom";

interface HeaderUserMenuProps {
  user: User;
}

const useStyles = createStyles((theme) => ({
  header: {
    paddingTop: theme.spacing.sm,
    backgroundColor:
      theme.colorScheme === "dark"
        ? theme.colors.dark[6]
        : theme.colors.gray[0],
    borderBottom: `${rem(1)} solid ${
      theme.colorScheme === "dark" ? "transparent" : theme.colors.gray[2]
    }`,
    marginBottom: rem(120),
  },

  user: {
    color: theme.colorScheme === "dark" ? theme.colors.dark[0] : theme.black,
    padding: `${theme.spacing.xs} ${theme.spacing.sm}`,
    borderRadius: theme.radius.sm,
    transition: "background-color 100ms ease",

    "&:hover": {
      backgroundColor:
        theme.colorScheme === "dark" ? theme.colors.dark[8] : theme.white,
    },

    [theme.fn.smallerThan("xs")]: {
      display: "none",
    },
  },

  burger: {
    [theme.fn.largerThan("xs")]: {
      display: "none",
    },
  },

  userActive: {
    backgroundColor:
      theme.colorScheme === "dark" ? theme.colors.dark[8] : theme.white,
  },

  tabs: {
    [theme.fn.smallerThan("sm")]: {
      display: "none",
    },
  },

  tabsList: {
    borderBottom: "0 !important",
  },

  tab: {
    fontWeight: 500,
    height: rem(38),
    backgroundColor: "transparent",

    "&:hover": {
      backgroundColor:
        theme.colorScheme === "dark"
          ? theme.colors.dark[5]
          : theme.colors.gray[1],
    },

    "&[data-active]": {
      backgroundColor:
        theme.colorScheme === "dark" ? theme.colors.dark[7] : theme.white,
      borderColor:
        theme.colorScheme === "dark"
          ? theme.colors.dark[7]
          : theme.colors.gray[2],
    },
  },
}));

interface HeaderUserMenuProps {
  user: User;
}

const HeaderUserMenu = ({ user }: HeaderUserMenuProps) => {
  const { classes, theme } = useStyles();
  const [opened, { toggle }] = useDisclosure(false);
  const [, setUserMenuOpened] = useState(false);

  return (
    <Container>
      <Group position="apart">
        <Burger
          opened={opened}
          onClick={toggle}
          className={classes.burger}
          size="sm"
        />

        <Menu
          width={260}
          position="bottom-end"
          transitionProps={{ transition: "pop-top-right" }}
          onClose={() => setUserMenuOpened(false)}
          onOpen={() => setUserMenuOpened(true)}
          withinPortal
        >
          <Menu.Target>
            <ActionIcon>
              <Group spacing={7}>
                <Indicator
                  inline
                  withBorder
                  processing
                  size={10}
                  offset={3}
                  position="bottom-end"
                  color="green"
                >
                  {user.profile?.profile_picture ? (
                    <Avatar
                      src={user.profile?.profile_picture}
                      alt={"Test"}
                      radius="xl"
                      size={30}
                    />
                  ) : (
                    <Avatar color="blue" radius="xl" size={30}>
                      {user.profile?.first_name.charAt(0)}
                      {user.profile?.last_name.charAt(0)}
                    </Avatar>
                  )}
                </Indicator>
              </Group>
            </ActionIcon>
          </Menu.Target>
          <Menu.Dropdown>
            {/* User Information */}
            <Group my={10}>
              {user.profile?.profile_picture ? (
                <Avatar
                  src={user.profile?.profile_picture}
                  alt={"Test"}
                  radius="xl"
                  size={40}
                  ml={5}
                  mb={2}
                />
              ) : (
                <Avatar color="blue" radius="xl" ml={5} mb={2} size={40}>
                  {user.profile?.first_name.charAt(0)}
                  {user.profile?.last_name.charAt(0)}
                </Avatar>
              )}

              <div style={{ flex: 1 }}>
                <Text size="sm" weight={500}>
                  {user.profile?.first_name} {user.profile?.last_name}
                </Text>
                <Text color="dimmed" size="xs">
                  {user.email}
                </Text>
              </div>
            </Group>
            <Menu.Divider />
            <Menu.Item
              icon={
                <IconHeart
                  size="0.9rem"
                  color={theme.colors.red[6]}
                  stroke={1.5}
                />
              }
            >
              Liked posts
            </Menu.Item>
            <Menu.Item
              icon={
                <IconStar
                  size="0.9rem"
                  color={theme.colors.yellow[6]}
                  stroke={1.5}
                />
              }
            >
              Saved posts
            </Menu.Item>
            <Menu.Item
              icon={
                <IconMessage
                  size="0.9rem"
                  color={theme.colors.blue[6]}
                  stroke={1.5}
                />
              }
            >
              Your comments
            </Menu.Item>

            <Menu.Label>Settings</Menu.Label>
            <Link
              to={`/account/settings/${user.id}/`}
              style={{ textDecoration: "none" }}
            >
              <Menu.Item icon={<FontAwesomeIcon icon={faGear} stroke="1.5" />}>
                Account settings
              </Menu.Item>
            </Link>
            <Menu.Item
              icon={<IconSwitchHorizontal size="0.9rem" stroke={1.5} />}
            >
              Change account
            </Menu.Item>

            <Link to="/logout/" style={{ textDecoration: "none" }}>
              <Menu.Item
                icon={
                  <FontAwesomeIcon
                    size="sm"
                    icon={faRightFromBracket}
                    stroke="1.5"
                  />
                }
              >
                Logout
              </Menu.Item>
            </Link>

            <Menu.Divider />

            <Menu.Label>Danger zone</Menu.Label>
            <Menu.Item icon={<IconPlayerPause size="0.9rem" stroke={1.5} />}>
              Pause subscription
            </Menu.Item>
            <Menu.Item
              color="red"
              icon={<IconTrash size="0.9rem" stroke={1.5} />}
            >
              Delete account
            </Menu.Item>
          </Menu.Dropdown>
        </Menu>
      </Group>
    </Container>
  );
};

export default HeaderUserMenu;
