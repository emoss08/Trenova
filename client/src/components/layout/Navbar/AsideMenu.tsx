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
  Navbar,
  Code,
  Group,
  rem,
  Skeleton,
  createStyles,
} from "@mantine/core";
import { useQuery, useQueryClient } from "react-query";
import { HeaderUserMenu } from "@/components/layout/HeaderUserMenu";
import { getUserId } from "@/lib/utils";
import { getUserDetails } from "@/requests/UserRequestFactory";
import { UserDownloads } from "@/components/layout/Header/_Partials/UserDownloads";
import { UserNotifications } from "@/components/layout/Header/_Partials/UserNotifications";
import { ThemeSwitcher } from "@/components/layout/Header/_Partials/ThemeSwitcher";
import { navbarScroll } from "@/components/layout/Navbar/_partials/NavbarScroll";
import { BillingLinks } from "@/components/layout/Navbar/_partials/BillingLinks";
import { OrganizationLogo } from "@/components/layout/Navbar/_partials/OrganizationLogo";
import { AdminLinks } from "@/components/layout/Navbar/_partials/SystemHealthLinks";
import { SearchModal } from "@/components/layout/Navbar/_partials/SearchModal";

const useNavbarStyles = createStyles((theme) => ({
  navbar: {
    paddingTop: 0,
  },

  section: {
    marginLeft: `calc(${theme.spacing.md} * -1)`,
    marginRight: `calc(${theme.spacing.md} * -1)`,
    marginBottom: theme.spacing.md,

    "&:not(:last-of-type)": {
      borderBottom: `${rem(1)} solid ${
        theme.colorScheme === "dark"
          ? theme.colors.dark[4]
          : theme.colors.gray[3]
      }`,
    },
  },

  searchCode: {
    fontWeight: 700,
    fontSize: rem(10),
    backgroundColor:
      theme.colorScheme === "dark"
        ? theme.colors.dark[7]
        : theme.colors.gray[0],
    border: `${rem(1)} solid ${
      theme.colorScheme === "dark" ? theme.colors.dark[7] : theme.colors.gray[2]
    }`,
  },

  header: {
    paddingBottom: theme.spacing.md,
    borderBottom: `${rem(1)} solid ${
      theme.colorScheme === "dark" ? theme.colors.dark[4] : theme.colors.gray[2]
    }`,
  },

  mainLinks: {
    paddingLeft: `calc(${theme.spacing.md} - ${theme.spacing.xs})`,
    paddingRight: `calc(${theme.spacing.md} - ${theme.spacing.xs})`,
    paddingBottom: theme.spacing.md,
  },

  mainLink: {
    display: "flex",
    alignItems: "center",
    width: "100%",
    fontSize: theme.fontSizes.xs,
    padding: `${rem(8)} ${theme.spacing.xs}`,
    borderRadius: theme.radius.sm,
    fontWeight: 500,
    color:
      theme.colorScheme === "dark"
        ? theme.colors.dark[0]
        : theme.colors.gray[7],

    "&:hover": {
      backgroundColor:
        theme.colorScheme === "dark"
          ? theme.colors.dark[6]
          : theme.colors.gray[0],
      color: theme.colorScheme === "dark" ? theme.white : theme.black,
    },
  },

  mainLinkInner: {
    display: "flex",
    alignItems: "center",
    flex: 1,
  },

  mainLinkIcon: {
    marginRight: theme.spacing.sm,
    color:
      theme.colorScheme === "dark"
        ? theme.colors.dark[2]
        : theme.colors.gray[6],
  },

  mainLinkBadge: {
    padding: 0,
    width: rem(20),
    height: rem(20),
    pointerEvents: "none",
  },

  collections: {
    paddingLeft: `calc(${theme.spacing.md} - ${rem(6)})`,
    paddingRight: `calc(${theme.spacing.md} - ${rem(6)})`,
    paddingBottom: theme.spacing.md,
  },

  collectionsHeader: {
    paddingLeft: `calc(${theme.spacing.md} + ${rem(2)})`,
    paddingRight: theme.spacing.md,
    marginBottom: rem(5),
  },

  collectionLink: {
    display: "block",
    padding: `${rem(8)} ${theme.spacing.xs}`,
    textDecoration: "none",
    borderRadius: theme.radius.sm,
    fontSize: theme.fontSizes.xs,
    color:
      theme.colorScheme === "dark"
        ? theme.colors.dark[0]
        : theme.colors.gray[7],
    lineHeight: 1,
    fontWeight: 500,

    "&:hover": {
      backgroundColor:
        theme.colorScheme === "dark"
          ? theme.colors.dark[6]
          : theme.colors.gray[0],
      color: theme.colorScheme === "dark" ? theme.white : theme.black,
    },
  },

  links: {
    marginLeft: `calc(${theme.spacing.md} * -1)`,
    marginRight: `calc(${theme.spacing.md} * -1)`,
  },

  linksInner: {
    paddingBottom: theme.spacing.xl,
  },
}));

export function AsideMenu() {
  const { classes } = useNavbarStyles();
  const queryClient = useQueryClient();
  const userId = getUserId() || "";

  const { data: userData, isLoading: isUserDataLoading } = useQuery({
    queryKey: ["user", userId],
    queryFn: () => {
      if (!userId) {
        return Promise.resolve(null);
      }
      return getUserDetails(userId);
    },
    initialData: () => queryClient.getQueryData(["user", userId]),
    staleTime: Infinity, // never refetch
  });

  return (
    <Navbar
      hiddenBreakpoint="sm"
      height="100%"
      width={{ sm: 300 }}
      p="md"
      zIndex={10}
      className={classes.navbar}
    >
      <Group className={classes.header} position="apart">
        <OrganizationLogo />
        <Code sx={{ fontWeight: 700 }}>v0.0.1</Code>
      </Group>

      <Navbar.Section className={classes.section}>
        {isUserDataLoading ? (
          <Skeleton width={rem(300)} height={rem(70)} />
        ) : (
          userData && <HeaderUserMenu user={userData} />
        )}
      </Navbar.Section>

      <SearchModal />

      <Navbar.Section className={classes.section}>
        <UserDownloads />
        <UserNotifications />
        <ThemeSwitcher />
      </Navbar.Section>
      <Navbar.Section grow className={classes.links} component={navbarScroll}>
        <div className={classes.linksInner}>
          {/* Billing Links */}
          <BillingLinks />

          {/* Admin Links */}
          <AdminLinks />
        </div>
      </Navbar.Section>
    </Navbar>
  );
}
