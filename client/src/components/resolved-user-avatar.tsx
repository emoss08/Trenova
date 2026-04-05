import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { queries } from "@/lib/queries";
import { getNameInitials, isAbsoluteUrl } from "@/lib/utils";
import type { User } from "@/types/user";
import { useQuery } from "@tanstack/react-query";
import type React from "react";

type ResolvedUserAvatarProps = React.ComponentProps<typeof Avatar> & {
  userId?: User["id"];
  name?: string;
  profilePicUrl?: string | null;
  thumbnailUrl?: string | null;
  imageClassName?: string;
  fallbackClassName?: string;
  alt?: string;
  variant?: "thumbnail" | "full";
};

export function ResolvedUserAvatar({
  userId,
  name,
  profilePicUrl,
  thumbnailUrl,
  imageClassName,
  fallbackClassName,
  alt,
  variant = "thumbnail",
  ...avatarProps
}: ResolvedUserAvatarProps) {
  const preferredSource =
    variant === "thumbnail"
      ? thumbnailUrl || profilePicUrl || undefined
      : profilePicUrl || thumbnailUrl || undefined;
  const shouldResolve = Boolean(userId && preferredSource && !isAbsoluteUrl(preferredSource));

  const { data: resolvedUrl } = useQuery({
    ...queries.user.profilePicture(userId ?? "", variant),
    enabled: shouldResolve,
    retry: false,
  });

  const imageSrc = isAbsoluteUrl(preferredSource) ? preferredSource : resolvedUrl ?? undefined;

  return (
    <Avatar {...avatarProps}>
      <AvatarImage className={imageClassName} src={imageSrc} alt={alt ?? name} />
      <AvatarFallback className={fallbackClassName}>{getNameInitials(name)}</AvatarFallback>
    </Avatar>
  );
}
