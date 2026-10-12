import { cn, getNameInitials } from "@trenova/shared/lib/utils";

const SIZES = {
  xs: "size-4 text-3xs",
  sm: "size-5 text-3xs",
  md: "size-6 text-2xs",
  lg: "size-8.5 text-xs",
} as const;

/**
 * A person in Triage as a round tile of their initials: the owner, someone
 * with the message open, whoever sent a reply. `present` adds the green dot
 * that says they are here now.
 */
export function PersonAvatar({
  name,
  size = "sm",
  present = false,
  className,
}: {
  name: string;
  size?: keyof typeof SIZES;
  present?: boolean;
  className?: string;
}) {
  return (
    <span
      aria-hidden
      className={cn(
        "bg-surface-active text-foreground relative inline-flex shrink-0 items-center justify-center rounded-full font-semibold",
        SIZES[size],
        className,
      )}
    >
      {getNameInitials(name, "?")}
      {present && (
        <span className="bg-success ring-canvas animate-breathe absolute -right-px -bottom-px size-2 rounded-full ring-2" />
      )}
    </span>
  );
}
