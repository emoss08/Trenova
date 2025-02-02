import * as React from "react";

import {
  CommandDialog,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
  CommandSeparator,
} from "@/components/ui/command";
import { SidebarGroup, SidebarGroupContent } from "@/components/ui/sidebar";
import { commandRoutes } from "@/lib/nav-links";
import {
  faFileImport,
  faMagnifyingGlass,
} from "@fortawesome/pro-regular-svg-icons";
import { useCallback } from "react";
import { useNavigate } from "react-router";
import { Icon } from "./ui/icons";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "./ui/tooltip";

export function SearchForm({ ...props }: React.ComponentProps<"form">) {
  return (
    <form {...props}>
      <SidebarGroup className="py-0">
        <SidebarGroupContent className="relative">
          <SiteSearchDialog />
        </SidebarGroupContent>
      </SidebarGroup>
    </form>
  );
}

export default function SiteSearchDialog() {
  const [open, setOpen] = React.useState(false);
  const navigate = useNavigate();

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

  const handleNavigate = useCallback(
    (link: string) => {
      navigate(link);
      setOpen(false);
    },
    [navigate],
  );

  return (
    <>
      <TooltipProvider>
        <Tooltip>
          <TooltipTrigger asChild>
            <span
              aria-label="Open site search"
              aria-expanded={open}
              onClick={() => setOpen(true)}
              className="group hidden h-8 items-center justify-between rounded-md border border-muted-foreground/20 bg-background px-3 py-2 text-sm hover:cursor-pointer hover:border-muted-foreground/80 xl:flex"
            >
              <div className="flex items-center">
                <Icon
                  icon={faMagnifyingGlass}
                  className="mr-2 size-3.5 text-muted-foreground group-hover:text-foreground"
                />
                <span className="text-muted-foreground">Search...</span>
              </div>
              <div className="pointer-events-none inline-flex select-none">
                <kbd className="-me-1 ms-12 inline-flex h-5 max-h-full items-center rounded border border-border bg-background px-1 font-[inherit] text-[0.625rem] font-medium text-muted-foreground/70">
                  ⌘K
                </kbd>
              </div>
            </span>
          </TooltipTrigger>
          <TooltipContent>
            <span>Site Search</span>
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>
      <CommandDialog open={open} onOpenChange={setOpen}>
        <CommandInput
          className="rounded-none"
          placeholder="Type a command or search..."
        />
        <CommandList className="h-[600px]">
          <CommandEmpty>No results found.</CommandEmpty>
          {commandRoutes.map((group) => {
            return (
              <React.Fragment key={group.id}>
                <CommandGroup heading={group.label}>
                  {group.routes.map((route) => (
                    <CommandItem
                      className="cursor-pointer"
                      key={route.id}
                      onSelect={() => handleNavigate(route.link)}
                    >
                      <Icon
                        icon={faFileImport}
                        className="mr-2 opacity-60"
                        aria-hidden="true"
                      />
                      <span>{route.label}</span>
                    </CommandItem>
                  ))}
                </CommandGroup>
                <CommandSeparator />
              </React.Fragment>
            );
          })}
        </CommandList>
      </CommandDialog>
    </>
  );
}
