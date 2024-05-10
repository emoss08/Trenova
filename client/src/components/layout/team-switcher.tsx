import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
  CommandSeparator,
} from "@/components/ui/command";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { cn } from "@/lib/utils";
import { faCheck } from "@fortawesome/pro-duotone-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { CaretSortIcon } from "@radix-ui/react-icons";
import React from "react";
import { Button } from "../ui/button";

const organizations = [
  {
    label: "Current Organization",
    teams: [
      {
        label: "Trenova Transportation",
        value: "personal",
      },
    ],
  },
  {
    label: "Other Organizations",
    teams: [
      {
        label: "Acme Inc.",
        value: "acme-inc",
      },
      {
        label: "Monsters Inc.",
        value: "monsters",
      },
    ],
  },
];

type Team = (typeof organizations)[number]["teams"][number];

type PopoverTriggerProps = React.ComponentPropsWithoutRef<
  typeof PopoverTrigger
>;

interface OrganizationSwitcherProps extends PopoverTriggerProps {}

export default function OrganizationSwitcher({
  className,
}: OrganizationSwitcherProps) {
  const [open, setOpen] = React.useState(false);
  const [selectedTeam, setSelectedTeam] = React.useState<Team>(
    organizations[0].teams[0],
  );

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          variant="outline"
          role="combobox"
          aria-expanded={open}
          aria-label="Select a team"
          className={cn(
            "w-[230px] justify-between border-muted-foreground/40 hover:border-muted-foreground/80 h-8 hidden md:flex",
            className,
          )}
        >
          <Avatar className="mr-2 size-5">
            <AvatarImage
              src={`https://avatar.vercel.sh/${selectedTeam.value}.png`}
              alt={selectedTeam.label}
              className="grayscale"
            />
            <AvatarFallback>SC</AvatarFallback>
          </Avatar>
          <span className="truncate">{selectedTeam.label}</span>
          <CaretSortIcon className="ml-auto size-4 shrink-0" />
        </Button>
      </PopoverTrigger>
      <PopoverContent align="start" className="w-[250px] p-0">
        <Command>
          <CommandList>
            <CommandInput placeholder="Search organization..." />
            <CommandEmpty>No team found.</CommandEmpty>
            {organizations.map((org) => (
              <CommandGroup key={org.label} heading={org.label}>
                {org.teams.map((team) => (
                  <CommandItem
                    key={team.value}
                    onSelect={() => {
                      setSelectedTeam(team);
                      setOpen(false);
                    }}
                    className="text-sm"
                  >
                    <Avatar className="mr-2 size-5">
                      <AvatarImage
                        src={`https://avatar.vercel.sh/${team.value}.png`}
                        alt={team.label}
                        className="grayscale"
                      />
                      <AvatarFallback>SC</AvatarFallback>
                    </Avatar>
                    <span className="truncate">{team.label}</span>
                    <FontAwesomeIcon
                      icon={faCheck}
                      className={cn(
                        "ml-auto h-4 w-4",
                        selectedTeam.value === team.value
                          ? "opacity-100"
                          : "opacity-0",
                      )}
                    />
                  </CommandItem>
                ))}
              </CommandGroup>
            ))}
          </CommandList>
          <CommandSeparator />
        </Command>
      </PopoverContent>
    </Popover>
  );
}
