import { cn } from "@/lib/utils";
import type { SequenceType } from "@/types/sequence-config";
import { memo } from "react";
import { useFormState } from "react-hook-form";
import {
  sequenceIcons,
  sequenceTitles,
  sidebarGroups,
} from "./sequence-config-constants";

type SidebarProps = {
  value: SequenceType;
  onChange: (next: SequenceType) => void;
  indexByType: Record<SequenceType, number>;
};

export function SequenceConfigSidebar({ value, onChange, indexByType }: SidebarProps) {
  return (
    <nav
      aria-label="Sequence configuration sections"
      className="sticky top-4 hidden w-60 shrink-0 self-start md:block"
    >
      <div className="rounded-md border border-border bg-card p-3">
        <ul className="flex flex-col gap-5">
        {sidebarGroups.map((group) => (
          <li key={group.label}>
            <div className="mb-1.5 px-2 text-[11px] font-medium tracking-wide text-muted-foreground uppercase">
              {group.label}
            </div>
            <ul className="flex flex-col gap-0.5">
              {group.items.map((type) => (
                <SidebarItem
                  key={type}
                  type={type}
                  index={indexByType[type]}
                  active={value === type}
                  onSelect={onChange}
                />
              ))}
            </ul>
          </li>
          ))}
        </ul>
      </div>
    </nav>
  );
}

type SidebarItemProps = {
  type: SequenceType;
  index: number;
  active: boolean;
  onSelect: (next: SequenceType) => void;
};

const SidebarItem = memo(function SidebarItem({
  type,
  index,
  active,
  onSelect,
}: SidebarItemProps) {
  const Icon = sequenceIcons[type];
  const { dirtyFields } = useFormState({ name: `configs.${index}` });
  const isDirty = Boolean(
    (dirtyFields as { configs?: unknown[] })?.configs?.[index],
  );

  return (
    <li>
      <button
        type="button"
        onClick={() => onSelect(type)}
        aria-current={active ? "page" : undefined}
        className={cn(
          "group relative flex w-full items-center gap-2.5 rounded-md px-2.5 py-2 text-left text-sm transition-colors",
          active
            ? "bg-muted font-medium text-foreground"
            : "text-muted-foreground hover:bg-muted/60 hover:text-foreground",
        )}
      >
        <Icon className="size-4 shrink-0" aria-hidden />
        <span className="flex-1 truncate">{sequenceTitles[type]}</span>
        {isDirty ? (
          <span
            aria-label="Unsaved changes"
            className="size-1.5 shrink-0 rounded-full bg-amber-500"
          />
        ) : null}
      </button>
    </li>
  );
});
