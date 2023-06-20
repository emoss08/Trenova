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

import { ActionIcon, Menu, useMantineColorScheme } from "@mantine/core";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faDisplay, faMoon, faSun } from "@fortawesome/pro-duotone-svg-icons";
import React from "react";
import { useHeaderStyles } from "@/styles/HeaderStyles";
import { headerStore } from "@/stores/HeaderStore";

export const ThemeSwitcher: React.FC = () => {
  const { colorScheme, toggleColorScheme } = useMantineColorScheme();
  const { classes } = useHeaderStyles();
  const [themeSwitcherOpen] = headerStore.use("themeSwitcherOpen");
  const getThemeIcon = () => {
    if (colorScheme === "light") {
      return <FontAwesomeIcon icon={faSun} />;
    } else if (colorScheme === "dark") {
      return <FontAwesomeIcon icon={faMoon} />;
    } else {
      return <FontAwesomeIcon icon={faDisplay} />;
    }
  };

  return (
    <>
      <Menu
        position="bottom-end"
        width={200}
        opened={themeSwitcherOpen}
        onChange={(changeEvent) => {
          headerStore.set("themeSwitcherOpen", changeEvent);
        }}
        withinPortal
        withArrow
        arrowSize={5}
      >
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
    </>
  );
};
