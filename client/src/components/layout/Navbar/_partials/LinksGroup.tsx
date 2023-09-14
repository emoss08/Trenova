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
  Box,
  Collapse,
  createStyles,
  Group,
  rem,
  Text,
  UnstyledButton,
} from "@mantine/core";
import { IconChevronLeft, IconChevronRight } from "@tabler/icons-react";
import { Link } from "react-router-dom";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { IconProp } from "@fortawesome/fontawesome-svg-core";
import { useUserPermissions } from "@/hooks/useUserPermissions";
import { SubLinksGroup } from "@/components/layout/Navbar/_partials/SubLinkGroups";

const useStyles = createStyles((theme) => ({
  control: {
    fontWeight: 500,
    display: "block",
    width: "100%",
    paddingBottom: `${theme.spacing.xs}`,
    paddingTop: `${theme.spacing.xs}`,
    marginBottom: rem(1),
    fontSize: theme.fontSizes.xs,
    "& svg": {
      color:
        theme.colorScheme === "dark"
          ? theme.colors.dark[2]
          : theme.colors.gray[9],
    },
    color:
      theme.colorScheme === "dark" ? theme.colors.dark[0] : theme.colors.black,
    "&:hover svg": {
      color:
        theme.colorScheme === "dark"
          ? theme.colors.gray[0]
          : theme.colors.gray[7],
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

export interface SubLink {
  label: string;
  link: string;
  permission?: string;
}

export interface LinkItem {
  label: string;
  link: string;
  permission?: string;
  subLinks?: SubLink[];
}

LinksGroup.defaultProps = {
  initiallyOpened: false,
  links: [],
  permission: undefined,
  link: undefined,
};

export interface LinksGroupProps {
  icon: IconProp;
  label: string;
  link?: string;
  initiallyOpened?: boolean;
  links?: LinkItem[];
  permission?: string;
}

export function LinksGroup({
  icon,
  label,
  initiallyOpened,
  links,
  link, // New direct link prop
  permission,
}: LinksGroupProps) {
  const { classes, theme } = useStyles();
  const hasLinks = Array.isArray(links) && links.length > 0;
  const [opened, setOpened] = useState(initiallyOpened || false);

  const handleOpenedToggle = useCallback(() => setOpened((o) => !o), []);

  const ChevronIcon = useMemo(
    () => (theme.dir === "ltr" ? IconChevronRight : IconChevronLeft),
    [theme.dir],
  );

  const { userHasPermission } = useUserPermissions();

  const linkItems = useMemo(
    () =>
      links
        ?.map((lk) => {
          if (lk.subLinks) {
            return (
              <SubLinksGroup
                key={lk.label}
                label={lk.label}
                subLinks={lk.subLinks}
              />
            );
          }

          if (lk.permission && userHasPermission(lk.permission)) {
            return (
              <Link
                to={lk.link}
                style={{
                  textDecoration: "none",
                }}
                key={lk.label}
              >
                <Text className={classes.link}>{lk.label}</Text>
              </Link>
            );
          }
          if (!lk.permission) {
            return (
              <Link
                to={lk.link}
                style={{
                  textDecoration: "none",
                }}
                key={lk.label}
              >
                <Text className={classes.link}>{lk.label}</Text>
              </Link>
            );
          }
          return null;
        })
        .filter(Boolean), // remove null items
    [links, userHasPermission],
  );

  // If the `LinksGroup` doesn't have permission, and doesn't have any visible link items, and there's no direct link, don't render it.
  if (
    (permission && !userHasPermission(permission)) ||
    (linkItems && linkItems.length === 0 && !link) // Adjusted this condition
  ) {
    return null;
  }

  // Component content
  const componentContent = (
    <>
      <UnstyledButton onClick={handleOpenedToggle} className={classes.control}>
        <Group position="apart" spacing={0}>
          <Box sx={{ display: "flex", alignItems: "center" }}>
            <FontAwesomeIcon icon={icon} size="lg" />
            <Box ml="md">{label}</Box>
          </Box>
          {hasLinks && (
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
          )}
        </Group>
      </UnstyledButton>
      {hasLinks && <Collapse in={opened}>{linkItems}</Collapse>}
    </>
  );

  return link && !hasLinks ? (
    <Link to={link} style={{ textDecoration: "none" }}>
      {componentContent}
    </Link>
  ) : (
    componentContent
  );
}
