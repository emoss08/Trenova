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
import { Navbar, Code, Group, rem, Skeleton } from "@mantine/core";
import { HeaderUserMenu } from "@/components/layout/HeaderUserMenu";
import { getUserId } from "@/lib/utils";
import { useQuery, useQueryClient } from "react-query";
import { getUserDetails } from "@/requests/UserRequestFactory";
import { UserDownloads } from "@/components/layout/Header/_Partials/UserDownloads";
import { UserNotifications } from "@/components/layout/Header/_Partials/UserNotifications";
import { ThemeSwitcher } from "@/components/layout/Header/_Partials/ThemeSwitcher";
import { useNavbarStyles } from "@/styles/NavbarStyles";
import { navbarScroll } from "@/components/layout/Navbar/_partials/NavbarScroll";
import { BillingLinks } from "@/components/layout/Navbar/_partials/BillingLinks";
import { OrganizationLogo } from "@/components/layout/Navbar/_partials/OrganizationLogo";
import { AdminLinks } from "@/components/layout/Navbar/_partials/SystemHealthLinks";
import { SearchModal } from "@/components/layout/Navbar/_partials/SearchModal";

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
    initialData: () => {
      return queryClient.getQueryData(["user", userId]);
    },
    staleTime: Infinity, // never refetch
  });

  return (
    <Navbar height="100%" width={{ sm: 300 }} p="md" className={classes.navbar}>
      <Group className={classes.header} position="apart">
        <OrganizationLogo />
        <Code sx={{ fontWeight: 700 }}>v0.0.1</Code>
      </Group>

      <Navbar.Section className={classes.section}>
        <>
          {isUserDataLoading ? (
            <Skeleton width={rem(300)} height={rem(70)} />
          ) : (
            userData && <HeaderUserMenu user={userData} />
          )}
        </>
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
