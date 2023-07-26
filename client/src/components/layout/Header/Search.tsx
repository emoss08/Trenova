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
import React, { useEffect } from "react";
import { faMagnifyingGlass } from "@fortawesome/pro-solid-svg-icons";
import { SpotlightAction, SpotlightProvider } from "@mantine/spotlight";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { routes } from "@/routing/AppRoutes";
import { useNavigate } from "react-router-dom";
import { ActionsWrapper } from "./_Partials/SpotlightActionsWrapper";
import { SearchControl } from "./_Partials/SpotlightSearchControl";
import { useUserPermissions } from "@/hooks/useUserPermissions";
import { Badge } from "@mantine/core";
import axios from "@/lib/AxiosConfig";

interface RouteSpotlightAction extends SpotlightAction {
  path: string;
}

export const SearchSpotlight: React.FC = () => {
  const navigate = useNavigate();
  const { isAuthenticated, userHasPermission } = useUserPermissions();
  const [input, setInput] = React.useState("");
  const [badge, setBadge] = React.useState<string | null>(null);
  const [actions, setActions] = React.useState<RouteSpotlightAction[]>([]);

  const onTrigger = (path: string) => {
    console.info("Navigating to", path);
    navigate(path);
  };
  const models = ["Order"];

  // Validate the user's input
  useEffect(() => {
    const [model, _] = input.split(":").map((s) => s.trim());
    if (models.includes(model)) {
      setBadge(model);
      // Perform the search on the backend
      axios
        .get(`search/?term=${encodeURIComponent(input)}`)
        .then((response) => {
          // Generate actions based on the search results
          const searchActions = response.data.results.map((result: any) => ({
            title: result.display,
            group: model,
            description: "",
            onTrigger: () => onTrigger("test"), // Adjust this based on your requirements
          }));
          setActions(searchActions);
        })
        .catch((error) => {
          console.error("Error fetching search results:", error);
        });
    } else {
      setBadge(null);
      // Generate actions for page navigation
      const navigationActions: RouteSpotlightAction[] = routes
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
      setActions(navigationActions);
    }
  }, [input]); // Only re-run the effect if `input` changes

  return (
    <SpotlightProvider
      actions={actions}
      onActionsChange={(newActions) =>
        setActions(
          newActions.map((action) => ({
            ...action,
            path: (action as any).path || "", // Provide a default value for `path`
          }))
        )
      }
      actionsWrapperComponent={ActionsWrapper}
      searchIcon={<FontAwesomeIcon icon={faMagnifyingGlass} />}
      searchPlaceholder="Search..."
      shortcut="mod + k"
      highlightQuery
      searchInputProps={{
        styles: {
          rightSection: {
            pointerEvents: "none",
            // make sure that is full size
            width: "auto",
            // add some padding to the right
            paddingRight: 10,
          },
        },
        // add a badge
        rightSection: badge && (
          <Badge
            variant="dot"
            color="red"
            radius="sm"
            style={{
              // background color light red
              backgroundColor: "#ffcccc",
              // border color dark red
              borderColor: "#fa5252",
              // Text color dark red
              color: "#fa5252",
            }}
          >
            {badge}
          </Badge>
        ),
        onInput: (e) => {
          setInput(e.currentTarget.value);
          // change actions based on the input
        },
      }}
      nothingFoundMessage="Nothing found..."
      limit={10}
      transitionProps={{ duration: 300, transition: "slide-down" }}
    >
      <SearchControl />
    </SpotlightProvider>
  );
};
