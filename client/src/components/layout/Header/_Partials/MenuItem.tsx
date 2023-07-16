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
  SimpleGrid,
  Text,
  ThemeIcon,
  UnstyledButton,
} from "@mantine/core";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import React from "react";
import { faChevronDown } from "@fortawesome/pro-solid-svg-icons";
import { useHeaderStyles } from "@/styles/HeaderStyles";
import { Link } from "react-router-dom";
import { useUserPermissions } from "@/hooks/useUserPermissions";
import { TNavigationLink } from "@/types";
import { useHeaderStore } from "@/stores/HeaderStore";

type MenuItemProps = {
  menuLinks: Record<string, TNavigationLink[]>;
  name: string;
  store: any;
  numOfColumns?: number;
  width?: number;
};

export const MenuItem = ({
  menuLinks,
  name,
  store,
  numOfColumns,
  width,
}: MenuItemProps) => {
  const { userHasPermission } = useUserPermissions();
  const { classes, theme } = useHeaderStyles();
  const [currentMenu, setCurrentMenu] = store.use("currentMenu");
  const [clickCount, setClickCount] = useHeaderStore.use("clickCount");

  const links = React.useMemo(() => {
    return Object.entries(menuLinks).reduce(
      (acc: React.ReactElement[], [groupTitle, links]) => {
        const linksForGroup = links.reduce(
          (groupAcc: React.ReactElement[], item) => {
            if (userHasPermission(item.permission)) {
              const linkContent = (
                <UnstyledButton
                  className={classes.subLink}
                  key={`${groupTitle}-${item.title}`}
                  onClick={() => item.subLinks && setCurrentMenu(item.title)}
                >
                  <Group noWrap align="flex-start">
                    <ThemeIcon size={34} variant="default" radius="md">
                      <FontAwesomeIcon
                        icon={item.icon}
                        color={
                          theme.colorScheme === "dark" ? "dark.0" : "dark.9"
                        }
                      />
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
              );

              if (currentMenu === item.title) {
                // If this is the current menu, render its sublinks
                groupAcc.push(
                  <React.Fragment key={groupTitle}>
                    {item.subLinks &&
                      item.subLinks.map((sublink) => {
                        const subLinkContent = (
                          <UnstyledButton
                            className={classes.subLink}
                            key={sublink.title}
                          >
                            <Group noWrap align="flex-start">
                              <ThemeIcon
                                size={34}
                                variant="default"
                                radius="md"
                              >
                                <FontAwesomeIcon
                                  icon={sublink.icon}
                                  color={
                                    theme.colorScheme === "dark"
                                      ? "dark.0"
                                      : "dark.9"
                                  }
                                />
                              </ThemeIcon>
                              <div>
                                <Text size="sm" fw={500}>
                                  {sublink.title}
                                </Text>
                                <Text size="xs" color="dimmed">
                                  {sublink.description}
                                </Text>
                              </div>
                            </Group>
                          </UnstyledButton>
                        );

                        return (
                          <Link
                            to={sublink.href || "#"}
                            key={sublink.title}
                            onClick={() => {
                              setClickCount((prevCount) => prevCount + 1); // Increment clickCount
                              useHeaderStore.set("linksOpen", false); // Close the menu
                            }}
                          >
                            {subLinkContent}
                          </Link>
                        );
                      })}
                  </React.Fragment>
                );
              } else if (!currentMenu) {
                // If there is no current menu, render the top-level links
                groupAcc.push(
                  item.href ? (
                    <Link
                      to={item.href}
                      onClick={() => {
                        setClickCount((prevCount) => prevCount + 1); // Increment clickCount
                        useHeaderStore.set("linksOpen", false); // Close the menu
                      }}
                    >
                      {linkContent}
                    </Link>
                  ) : (
                    linkContent
                  )
                );
              }
            }
            return groupAcc;
          },
          [] as React.ReactElement[]
        );

        if (linksForGroup.length > 0) {
          acc.push(
            <React.Fragment key={groupTitle}>
              <Text fz="xs" mt={3} color="dimmed">
                {groupTitle}
              </Text>
              <SimpleGrid cols={numOfColumns ? numOfColumns : 3} spacing={0}>
                {linksForGroup}
              </SimpleGrid>
            </React.Fragment>
          );
        }
        return acc;
      },
      [] as React.ReactElement[]
    );
  }, [menuLinks, userHasPermission, currentMenu]);

  if (links.length === 0) return null;

  return (
    <HoverCard
      width={width ? width : currentMenu ? "lg" : numOfColumns ? "md" : "sm"}
      position="bottom"
      radius="md"
      // since mantine does not provide a way to control the open/close state of the hover-card, we have to force it to re-render by changing the key
      key={clickCount}
      shadow="md"
      withinPortal
      withArrow
    >
      <HoverCard.Target>
        <Link to="#" className={classes.link}>
          <Center inline>
            <Box component="span" mr={5}>
              {name}
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
          <Text fw={500}>{name}</Text>
          <Anchor href="#" fz="xs">
            {currentMenu && (
              <Text fz="xs" onClick={() => setCurrentMenu(null)}>
                Back
              </Text>
            )}
          </Anchor>
        </Group>

        <Divider
          my="sm"
          mx="-md"
          color={theme.colorScheme === "dark" ? "dark.5" : "gray.1"}
        />

        {links}

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
