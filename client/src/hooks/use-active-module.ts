import type { NavModule } from "@/config/navigation.types";
import { useMemo } from "react";
import { useLocation } from "react-router";
import { useFilteredNavigation } from "./use-filtered-navigation";

export function useActiveModule(): NavModule | null {
  const location = useLocation();
  const filteredModules = useFilteredNavigation();

  return useMemo(() => {
    const { pathname } = location;

    for (const module of filteredModules) {
      if (module.basePath === "/") {
        if (pathname === "/") {
          return module;
        }
        continue;
      }

      if (pathname.startsWith(module.basePath)) {
        return module;
      }
    }

    return filteredModules[0] ?? null;
  }, [location, filteredModules]);
}
