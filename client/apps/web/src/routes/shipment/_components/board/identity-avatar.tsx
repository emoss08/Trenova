import { AvatarImage } from "@trenova/shared/components/ui/avatar";
import { personPhotoSource } from "@trenova/shared/components/worker-avatar";
import { identityAccentClass } from "@trenova/shared/lib/identity-accent";
import { cn } from "@trenova/shared/lib/utils";
import type { PersonPhoto } from "@trenova/shared/types/person-photo";

type IdentityAvatarProps = {
  id: string;
  initials: string;
  /** A person is a circle, a company a rounded square. */
  shape: "circle" | "square";
  /**
   * A person's photo, drawn as a lazy thumbnail over the initials once it has loaded; the initials
   * stay underneath while it loads or if it fails. A company has none.
   */
  photo?: PersonPhoto | null;
  className?: string;
  dashed?: boolean;
};

export function IdentityAvatar({
  id,
  initials,
  shape,
  photo,
  className,
  dashed,
}: IdentityAvatarProps) {
  const source = shape === "circle" ? personPhotoSource(photo) : undefined;
  return (
    <span
      aria-hidden
      className={cn(
        "relative inline-grid shrink-0 place-items-center leading-none font-medium select-none",
        shape === "circle" ? "rounded-full" : "rounded-sm",
        identityAccentClass(id),
        dashed && "border border-dashed border-border-strong bg-transparent opacity-70",
        className,
      )}
    >
      {initials}
      {source ? <AvatarImage src={source} className="rounded-full" /> : null}
    </span>
  );
}
