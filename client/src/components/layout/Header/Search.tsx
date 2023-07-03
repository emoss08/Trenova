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
import { faMagnifyingGlass } from "@fortawesome/pro-solid-svg-icons";
import { SpotlightAction, SpotlightProvider } from "@mantine/spotlight";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { routes } from "@/routing/AppRoutes";
import { useNavigate } from "react-router-dom";
import { ActionsWrapper } from "./_Partials/SpotlightActionsWrapper";
import { SearchControl } from "./_Partials/SpotlightSearchControl";
import { useUserPermissions } from "@/hooks/useUserPermissions";

interface RouteSpotlightAction extends SpotlightAction {
  path: string;
}

export const SearchSpotlight: React.FC = () => {
  const navigate = useNavigate();
  const { isAuthenticated, userHasPermission } = useUserPermissions();
  const onTrigger = (path: string) => {
    console.info("Navigating to", path);
    navigate(path);
  };

  // Converting the routes into actions
  const actions: RouteSpotlightAction[] = routes
    .filter((route) => {
      // Exclude the route if `excludeFromMenu` is true
      if (route.excludeFromMenu) {
        return false;
      }

      // If the route requires a permission, check if the user has it
      if (route.permission) {
        return userHasPermission(route.permission);
      }

      // If the user is authenticated, include the route
      return isAuthenticated;
    })
    .map((route) => ({
      title: route.title,
      group: route.group,
      path: route.path,
      description: route.description,
      onTrigger: () => onTrigger(route.path),
    }));

  return (
    <SpotlightProvider
      actions={actions}
      actionsWrapperComponent={ActionsWrapper}
      searchIcon={<FontAwesomeIcon icon={faMagnifyingGlass} />}
      searchPlaceholder="Search..."
      shortcut="mod + k"
      nothingFoundMessage="Nothing found..."
      limit={10}
      transitionProps={{ duration: 300, transition: "slide-down" }}
    >
      <SearchControl />
    </SpotlightProvider>
  );
};
