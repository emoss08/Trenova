import { cn } from "@trenova/shared/lib/utils";

export function versionTriggerClass(old: boolean): string {
  return cn(
    "h-7.5 gap-1.25 rounded-lg px-2.25 font-plex-mono text-xs font-medium text-dsk-fg2 ring-1 ring-dsk-b-sub ring-inset transition-colors duration-120 hover:bg-dsk-hover aria-expanded:bg-dsk-hover [&_svg]:text-dsk-subtle",
    old && "text-dsk-warn-fg ring-dsk-warn/40",
  );
}

export const VERSION_OPTION_CLASS =
  "dk-axv-r flex w-full gap-2.5 rounded-lg px-2 py-1.75 text-left hover:bg-dsk-hover";
