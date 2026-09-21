import { useT } from "@trenova/shared/i18n/use-t";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@trenova/shared/components/ui/command";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { cn } from "@trenova/shared/lib/utils";
import type { AssistantProviderOption } from "@/types/assistant";
import { CheckIcon, ChevronDownIcon } from "lucide-react";
import { useMemo, useState } from "react";
import { kindMark } from "@/routes/agent-control/_components/providers/kind-marks";
import { AssistMark } from "@trenova/shared/components/ui/assist-mark";

export type ModelPickerProps = {
  options: readonly AssistantProviderOption[];
  /** Empty means Auto: the organization's own priority order decides. */
  value: string;
  onChange: (providerId: string) => void;
  disabled?: boolean;
};

/**
 * Which model answers.
 *
 * An organization configures the endpoints; this is where a person picks one
 * of them for a conversation. Auto is first and is the default, because the
 * priority order an administrator set is the right answer for most people and
 * the only one that survives a provider going down.
 *
 * A choice is a preference, not a pin. If the chosen endpoint fails the router
 * still falls through to the rest, which is why the reply says which model
 * actually answered rather than assuming it was this one.
 */
export function ModelPicker({ options, value, onChange, disabled = false }: ModelPickerProps) {
  const t = useT();
  const [open, setOpen] = useState(false);

  const selected = useMemo(
    () => options.find((option) => option.id === value) ?? null,
    [options, value],
  );

  // Shown from the first provider onwards, not the second.
  //
  // This is an indicator as much as a picker: with one provider there is
  // nothing to choose, but which model is answering is still worth seeing, and
  // it is the only place a person can see it before the reply arrives.
  //
  // Hiding it below two also made three different situations look identical —
  // one provider configured, a second provider that was never given the
  // assistant task, and an endpoint that failed — which is a bad way to find
  // out which one you have. Nothing renders only when the organization has no
  // assistant provider at all, and in that case the assistant does not work
  // either and says so elsewhere.
  if (options.length === 0) {
    return null;
  }

  const label = selected ? selected.model : t("Auto");

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger
        disabled={disabled}
        className={cn(
          "ui-focus-ring inline-flex max-w-[14rem] items-center gap-1.5 rounded-md",
          "border border-border bg-background px-2 py-1 text-xs text-muted-foreground",
          "transition-colors hover:text-foreground disabled:opacity-50",
        )}
        aria-label={t("Choose which model answers")}
      >
        <span className="flex size-3.5 shrink-0 items-center justify-center [&_svg]:size-full">
          {selected ? kindMark(selected.kind) : <AssistMark />}
        </span>
        <span className="truncate">{label}</span>
        <ChevronDownIcon className="size-3 shrink-0" />
      </PopoverTrigger>
      <PopoverContent align="start" className="w-80 p-0">
        <Command>
          <CommandInput placeholder={t("Search models...")} className="h-8" />
          <CommandList>
            <CommandEmpty>{t("No models match.")}</CommandEmpty>
            <CommandGroup>
              <ModelRow
                icon={<AssistMark />}
                title={t("Auto")}
                detail={t("Let this organization's provider order decide")}
                selected={value === ""}
                onSelect={() => {
                  onChange("");
                  setOpen(false);
                }}
              />
              {options.map((option) => (
                <ModelRow
                  key={option.id}
                  icon={kindMark(option.kind)}
                  title={option.model}
                  detail={option.name}
                  value={`${option.model} ${option.name}`}
                  selected={option.id === value}
                  onSelect={() => {
                    onChange(option.id);
                    setOpen(false);
                  }}
                />
              ))}
            </CommandGroup>
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}

type ModelRowProps = {
  icon: React.ReactNode;
  title: string;
  detail: string;
  /** What the search box matches on, when it is more than the title. */
  value?: string;
  selected: boolean;
  onSelect: () => void;
};

function ModelRow({ icon, title, detail, value, selected, onSelect }: ModelRowProps) {
  return (
    <CommandItem value={value ?? title} onSelect={onSelect} className="gap-2">
      <span className="flex size-4 shrink-0 items-center justify-center [&_svg]:size-full">
        {icon}
      </span>
      <span className="flex min-w-0 flex-1 flex-col">
        <span className="truncate text-foreground">{title}</span>
        <span className="truncate text-xs text-muted-foreground">{detail}</span>
      </span>
      {selected ? <CheckIcon className="size-4 shrink-0 text-muted-foreground" /> : null}
    </CommandItem>
  );
}
