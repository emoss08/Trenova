import { useUserPermissions } from "@/context/user-permissions";
import { upperFirst } from "@/lib/utils";
import { routes } from "@/routing/AppRoutes";
import { AlertCircle } from "lucide-react";
import React from "react";
import { useNavigate } from "react-router-dom";
import {
  CommandDialog,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "./ui/command";

const prepareRouteGroups = (routeList: typeof routes) => {
  return routeList.reduce(
    (acc, route) => {
      if (!acc[route.group]) {
        acc[route.group] = [];
      }
      acc[route.group].push(route);
      return acc;
    },
    {} as Record<string, typeof routes>,
  );
};

export function SiteSearch() {
  const navigate = useNavigate();
  const { isAuthenticated, userHasPermission } = useUserPermissions();
  const [open, setOpen] = React.useState(false);
  const [searchText, setSearchText] = React.useState("");

  React.useEffect(() => {
    const down = (e: KeyboardEvent) => {
      if (e.key === "k" && (e.metaKey || e.ctrlKey)) {
        e.preventDefault();
        setOpen((open) => !open);
      }
    };
    document.addEventListener("keydown", down);
    return () => document.removeEventListener("keydown", down);
  }, []);

  // Filtering and preparing route groups
  const filteredRoutes = routes.filter((route) => {
    if (route.excludeFromMenu) return false;
    if (route.path.endsWith("/:id")) return false;
    if (route.permission && !userHasPermission(route.permission)) return false;
    return isAuthenticated;
  });

  const groupedRoutes = prepareRouteGroups(filteredRoutes);

  const filteredGroups = Object.entries(groupedRoutes).reduce(
    (acc, [group, groupRoutes]) => {
      const filtered = groupRoutes.filter((route) =>
        route.title.toLowerCase().includes(searchText.toLowerCase()),
      );

      if (filtered.length) {
        acc[group] = filtered;
      }

      return acc;
    },
    {} as Record<string, typeof routes>,
  );

  return (
    <CommandDialog open={open} onOpenChange={setOpen}>
      <CommandInput
        placeholder="Search..."
        value={searchText}
        onValueChange={setSearchText}
      />
      <CommandList>
        {Object.entries(filteredGroups).length === 0 && (
          <CommandEmpty>
            <AlertCircle className="mx-auto h-6 w-6 text-accent-foreground" />
            <p className="mt-4 font-semibold text-accent-foreground">
              No results found
            </p>
            <p className="mt-2 text-muted-foreground">
              No pages found for this search term. Please try again.
            </p>
          </CommandEmpty>
        )}
        {Object.entries(filteredGroups).map(([group, groupCommands]) => (
          <CommandGroup key={group} heading={upperFirst(group)}>
            {groupCommands.map((cmd) => (
              <CommandItem
                key={cmd.path}
                onSelect={() => {
                  navigate(cmd.path);
                  setOpen(false);
                }}
                className="hover:cursor-pointer"
              >
                {cmd.title}
              </CommandItem>
            ))}
          </CommandGroup>
        ))}
      </CommandList>
      <div className="sticky bg-muted-foreground/5 flex items-center justify-between border-t py-2">
        <div>
          <p className="text-xs ml-2">Search powered by Monta</p>
        </div>
        <div>
          <a
            href="https://monta.so"
            target="_blank"
            rel="noopener noreferrer"
            className="text-xs text-foreground mr-2"
          >
            Learn More
          </a>
        </div>
      </div>
    </CommandDialog>
  );
}
