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
  Avatar,
  Group,
  Text,
  Menu,
  Burger,
  createStyles,
  UnstyledButton,
  Indicator,
} from "@mantine/core";
import {
  IconHeart,
  IconStar,
  IconMessage,
  IconPlayerPause,
  IconTrash,
  IconSwitchHorizontal,
  IconChevronRight,
} from "@tabler/icons-react";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faGear, faRightFromBracket } from "@fortawesome/pro-regular-svg-icons";
import { Link } from "react-router-dom";
import { useNavbarStore } from "@/stores/HeaderStore";
import { User } from "@/types/apps/accounts";

const pageStyles = createStyles((theme) => ({
  user: {
    display: "block",
    width: "100%",
    padding: theme.spacing.md,
    color: theme.colorScheme === "dark" ? theme.colors.dark[0] : theme.black,

    "&:hover": {
      backgroundColor:
        theme.colorScheme === "dark"
          ? theme.colors.dark[8]
          : theme.colors.gray[0],
    },
  },
  burger: {
    [theme.fn.largerThan("xs")]: {
      display: "none",
    },
  },
}));

type Props = {
  user: User;
};

export const HeaderUserMenu: React.FC<Props> = ({ user }) => {
  const { classes, theme } = pageStyles();
  const [userMenuOpen] = useNavbarStore.use("userMenuOpen");

  if (!user) {
    return <div>No user data available</div>;
  }

  return (
    <Group position="apart">
      <Burger
        opened={userMenuOpen}
        onClick={() => useNavbarStore.set("userMenuOpen", !userMenuOpen)}
        className={classes.burger}
        size="sm"
      />

      <Menu
        width={260}
        position="right-start"
        transitionProps={{ transition: "pop-top-right" }}
        onClose={() => useNavbarStore.set("userMenuOpen", false)}
        onOpen={() => useNavbarStore.set("userMenuOpen", true)}
        withinPortal
      >
        <Menu.Target>
          <UnstyledButton className={classes.user}>
            <Group>
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
                  />
                ) : (
                  <Avatar color="blue" radius="xl">
                    {user.profile?.first_name.charAt(0)}
                    {user.profile?.last_name.charAt(0)}
                  </Avatar>
                )}
              </Indicator>
              {/*<Avatar src={user.profile?.profile_picture} radius="xl" />*/}

              <div style={{ flex: 1 }}>
                <Text size="sm" weight={500}>
                  {user.profile?.first_name} {user.profile?.last_name}
                </Text>

                <Text color="dimmed" size="xs">
                  {user.email}
                </Text>
              </div>
              <IconChevronRight size="0.9rem" stroke={1.5} />
            </Group>
          </UnstyledButton>
        </Menu.Target>
        <Menu.Dropdown>
          {/* User Information */}
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
          <Menu.Item icon={<IconSwitchHorizontal size="0.9rem" stroke={1.5} />}>
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
  );
};
