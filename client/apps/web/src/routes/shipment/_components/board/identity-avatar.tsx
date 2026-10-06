import { identityAccentClass } from "@trenova/shared/lib/identity-accent";
import { cn } from "@trenova/shared/lib/utils";

type IdentityAvatarProps = {
  id: string;
  initials: string;
  /** A person is a circle, a company a rounded square. */
  shape: "circle" | "square";
  className?: string;
  dashed?: boolean;
};

export function IdentityAvatar({ id, initials, shape, className, dashed }: IdentityAvatarProps) {
  return (
    <span
      aria-hidden
      className={cn(
        "inline-grid shrink-0 place-items-center leading-none font-medium select-none",
        shape === "circle" ? "rounded-full" : "rounded-sm",
        identityAccentClass(id),
        dashed && "border-border-strong border border-dashed bg-transparent opacity-70",
        className,
      )}
    >
      {initials}
    </span>
  );
}
