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
  Divider,
  Menu,
  UnstyledButton,
  useMantineColorScheme,
} from "@mantine/core";
import React, { useRef } from "react";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faLaptop, faMoon, faSun } from "@fortawesome/pro-duotone-svg-icons";
import { useNavbarStore } from "@/stores/HeaderStore";
import { useAsideStyles } from "@/assets/styles/AsideStyles";

export function ThemeSwitcher(): React.ReactElement {
  const { colorScheme, toggleColorScheme } = useMantineColorScheme();
  const { classes } = useAsideStyles();
  const [themeSwitcherOpen] = useNavbarStore.use("themeSwitcherOpen");
  const ref = useRef<HTMLButtonElement>(null);

  const getThemeIcon = () => {
    if (colorScheme === "light") {
      return (
        <FontAwesomeIcon
          size="lg"
          icon={faSun}
          className={classes.mainLinkIcon}
        />
      );
    }
    if (colorScheme === "dark") {
      return (
        <FontAwesomeIcon
          size="lg"
          icon={faMoon}
          className={classes.mainLinkIcon}
        />
      );
    }
    return (
      <FontAwesomeIcon
        size="lg"
        icon={faLaptop}
        className={classes.mainLinkIcon}
      />
    );
  };

  return (
    <Menu
      position="right-start"
      width={200}
      opened={themeSwitcherOpen}
      onChange={(changeEvent) => {
        useNavbarStore.set("themeSwitcherOpen", changeEvent);
      }}
      withinPortal
      withArrow
      arrowSize={5}
    >
      <Menu.Target>
        <div className={classes.mainLinks}>
          <UnstyledButton className={classes.mainLink} mb={5} ref={ref}>
            <div className={classes.mainLinkInner}>
              {getThemeIcon()}
              <span>Switch Theme</span>
            </div>
          </UnstyledButton>
        </div>
      </Menu.Target>

      <Menu.Dropdown>
        <Menu.Label>Theme Modes</Menu.Label>
        <Divider />
        <Menu.Item
          className={classes.menuItem}
          onClick={() => toggleColorScheme("light")}
          icon={<FontAwesomeIcon size="lg" icon={faSun} />}
        >
          Light Theme
        </Menu.Item>
        <Menu.Item
          className={classes.menuItem}
          onClick={() => toggleColorScheme("dark")}
          icon={<FontAwesomeIcon size="lg" icon={faMoon} />}
        >
          Dark Theme
        </Menu.Item>
      </Menu.Dropdown>
    </Menu>
  );
}
