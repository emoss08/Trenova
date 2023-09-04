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
  AppShell,
  Container,
  useMantineTheme,
  createStyles,
  rem,
} from "@mantine/core";
import { Breadcrumb } from "@/components/common/BreadCrumbs";
import { AsideMenu } from "@/components/layout/Navbar/AsideMenu";

type LayoutProps = {
  children: React.ReactNode;
};

const useStyles = createStyles((theme) => ({
  header: {
    backgroundImage: `${theme.fn.linearGradient(
      90,
      "rgba(129,26,188,0.9)",
      "rgba(219,52,52,0.9)",
      "rgba(241, 196, 15,  .9)",
      "rgba(34,230,171,0.9)",
      "rgba(0,60,211,0.9)",
    )}`,
    height: rem(2),
    minHeight: rem(2),
    padding: rem(2),
    zIndex: 100,
    top: 0,
    left: 0,
    right: 0,
    position: "fixed",
    boxSizing: "border-box",
  },
}));

export function Layout({ children }: LayoutProps): React.ReactElement {
  const theme = useMantineTheme();
  const { classes } = useStyles();

  return (
    <AppShell
      styles={{
        main: {
          background:
            theme.colorScheme === "dark"
              ? theme.colors.dark[8]
              : theme.colors.gray[0],
        },
      }}
      header={<header className={classes.header} />}
      navbar={<AsideMenu />}
    >
      <Container size="xl">
        <Breadcrumb />
        {/* {shouldRenderBreadcrumbs && <Breadcrumb />} */}
        {children}
      </Container>
    </AppShell>
  );
}
