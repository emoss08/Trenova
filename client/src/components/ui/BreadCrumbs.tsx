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
import { useLocation } from "react-router-dom";
import { useEffect } from "react";
import { routes } from "@/routing/AppRoutes";
import { Text, Flex, Skeleton } from "@mantine/core";
import { upperFirst } from "@/utils/utils";
import { pathToRegexp } from "path-to-regexp";
import { usePageStyles } from "@/styles/PageStyles";
import { breadcrumbStore } from "@/stores/BreadcrumbStore";

export function Breadcrumb() {
  const location = useLocation();
  const [currentRoute] = breadcrumbStore.use("currentRoute");
  const [loading] = breadcrumbStore.use("loading");
  const { classes } = usePageStyles();

  useEffect(() => {
    breadcrumbStore.set("loading", true);
    const route = routes.find((route) => {
      const re = pathToRegexp(route.path);
      return re.test(location.pathname);
    });

    if (route) {
      breadcrumbStore.set("currentRoute", route);
    }
    breadcrumbStore.set("loading", false);
  }, [location.pathname]);

  useEffect(() => {
    if (currentRoute) {
      document.title = currentRoute.title;
    }
  }, [currentRoute]);

  return (
    <div style={{ flex: 1, marginBottom: 10 }}>
      {loading ? (
        <>
          <Skeleton width={200} height={30} />
          <Skeleton width={250} height={20} mt={5} />
        </>
      ) : (
        <>
          <Text className={classes.text} fz={20} weight={600}>
            {currentRoute?.title}
          </Text>
          <Flex>
            <Text color="dimmed" size="sm">
              {currentRoute?.group && `${upperFirst(currentRoute.group)} - `}
              {currentRoute?.subMenu &&
                `${upperFirst(currentRoute.subMenu)} - `}
              {currentRoute?.title}
            </Text>
          </Flex>
        </>
      )}
    </div>
  );
}
