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

import React, { useCallback, useMemo, useState } from "react";
import {
  Group,
  Box,
  Collapse,
  Text,
  UnstyledButton,
  createStyles,
  rem,
} from "@mantine/core";
import { IconChevronLeft, IconChevronRight } from "@tabler/icons-react";
import { Link } from "react-router-dom";
import { useUserPermissions } from "@/hooks/useUserPermissions";
import { LinkItem } from "./LinksGroup";

const useStyles = createStyles((theme) => ({
  control: {
    fontWeight: 500,
    display: "block",
    width: "100%",
    padding: `${theme.spacing.xs}`,
    color: theme.colorScheme === "dark" ? theme.colors.dark[0] : theme.black,
    fontSize: theme.fontSizes.xs,

    "&:hover": {
      backgroundColor:
        theme.colorScheme === "dark"
          ? theme.colors.dark[7]
          : theme.colors.gray[0],
      color: theme.colorScheme === "dark" ? theme.white : theme.black,
    },
  },

  link: {
    fontWeight: 500,
    display: "block",
    textDecoration: "none",
    padding: `${theme.spacing.xs} ${theme.spacing.md}`,
    paddingLeft: rem(31),
    marginLeft: rem(30),
    fontSize: theme.fontSizes.xs,
    color:
      theme.colorScheme === "dark"
        ? theme.colors.dark[0]
        : theme.colors.gray[7],
    borderLeft: `${rem(1)} solid ${
      theme.colorScheme === "dark" ? theme.colors.dark[4] : theme.colors.gray[3]
    }`,

    "&:hover": {
      backgroundColor:
        theme.colorScheme === "dark"
          ? theme.colors.dark[7]
          : theme.colors.gray[0],
      color: theme.colorScheme === "dark" ? theme.white : theme.black,
    },
  },

  chevron: {
    transition: "transform 200ms ease",
  },
}));

interface SubLinksGroupProps {
  label: string;
  subLinks: LinkItem[];
}
export function SubLinksGroup({ label, subLinks }: SubLinksGroupProps) {
  const { classes, theme } = useStyles();
  const { userHasPermission } = useUserPermissions();

  const hasSubLinks = useMemo(
    () =>
      Array.isArray(subLinks) &&
      subLinks.some(
        (subLink) =>
          !subLink.permission || userHasPermission(subLink.permission)
      ),
    [subLinks, userHasPermission]
  );

  const [opened, setOpened] = useState(false);

  const handleOpenedToggle = useCallback(() => setOpened((o) => !o), []);

  const ChevronIcon = useMemo(
    () => (theme.dir === "ltr" ? IconChevronRight : IconChevronLeft),
    [theme.dir]
  );

  const subLinkItems = useMemo(() => {
    return subLinks
      ?.filter(
        (subLink) =>
          !subLink.permission || userHasPermission(subLink.permission)
      )
      .map((subLink) => (
        <Link
          to={subLink.link}
          style={{
            textDecoration: "none",
          }}
          key={subLink.label}
        >
          <Text className={classes.link}>{subLink.label}</Text>
        </Link>
      ));
  }, [subLinks, userHasPermission]);

  if (!hasSubLinks) {
    return null;
  }

  return (
    <>
      <UnstyledButton onClick={handleOpenedToggle} className={classes.control}>
        <Group position="apart" spacing={0}>
          <Box sx={{ display: "flex", alignItems: "center" }}>
            <Box ml="md">{label}</Box>
          </Box>
          <ChevronIcon
            className={classes.chevron}
            size="1rem"
            stroke={1.5}
            style={{
              transform: opened
                ? `rotate(${theme.dir === "rtl" ? -90 : 90}deg)`
                : "none",
            }}
          />
        </Group>
      </UnstyledButton>
      <Collapse in={opened}>{subLinkItems}</Collapse>
    </>
  );
}
