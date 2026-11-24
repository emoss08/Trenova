import { Avatar, AvatarFallback, AvatarImage } from "@trenova/shared/components/ui/avatar";
import { cn, getNameInitials, initials } from "@trenova/shared/lib/utils";
import type { PersonPhoto } from "@trenova/shared/types/person-photo";
import type { ComponentProps, ReactNode } from "react";

export type WorkerAvatarProps = Omit<ComponentProps<typeof Avatar>, "children"> & {
  firstName?: string | null;
  lastName?: string | null;
  /** The whole name, for a read that carries no first and last name; used when they are absent. */
  name?: string | null;
  photo?: PersonPhoto | null;
  /**
   * Which picture to draw: the thumbnail for a face in a row or a list, the full picture where the
   * face is large (a record's header). Either falls back to the other.
   */
  variant?: "thumbnail" | "full";
  /** The image's alternative text; empty by default, since the name is written beside it. */
  alt?: string;
  imageClassName?: string;
  fallbackClassName?: string;
  /** A badge on the corner, such as an `AvatarBadge`. */
  children?: ReactNode;
};

/** The source the avatar draws for a photo, or undefined when there is none. */
export function personPhotoSource(
  photo: PersonPhoto | null | undefined,
  variant: "thumbnail" | "full" = "thumbnail",
): string | undefined {
  if (!photo) return undefined;
  const source =
    variant === "full" ? photo.url || photo.thumbnailUrl : photo.thumbnailUrl || photo.url;
  return source || undefined;
}

/**
 * A worker's face, cut to a circle: their photo, or their initials until it loads or when they
 * have none. The one drawing of a worker in a row, a list, a lane or a header.
 */
export function WorkerAvatar({
  firstName,
  lastName,
  name,
  photo,
  variant = "thumbnail",
  alt = "",
  className,
  imageClassName,
  fallbackClassName,
  children,
  ...avatarProps
}: WorkerAvatarProps) {
  const source = personPhotoSource(photo, variant);
  return (
    <Avatar {...avatarProps} className={cn("rounded-full after:rounded-full", className)}>
      {source ? (
        <AvatarImage src={source} alt={alt} className={cn("rounded-full", imageClassName)} />
      ) : null}
      <AvatarFallback className={cn("rounded-full font-medium", fallbackClassName)}>
        {firstName || lastName || !name
          ? initials(firstName ?? undefined, lastName ?? undefined)
          : getNameInitials(name, "•")}
      </AvatarFallback>
      {children}
    </Avatar>
  );
}
