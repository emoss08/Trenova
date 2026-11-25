import { Avatar, AvatarFallback, AvatarImage } from "@trenova/shared/components/ui/avatar";
import { user } from "@/lib/queries/user";
import { avatarBackground } from "@trenova/shared/lib/avatar-gradient";
import { personPhotoSource } from "@trenova/shared/components/worker-avatar";
import { cn } from "@trenova/shared/lib/utils";
import type { PersonPhoto } from "@trenova/shared/types/person-photo";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import type { User } from "@trenova/shared/types/user";
import { useQuery } from "@tanstack/react-query";
import type React from "react";

const AVATAR_STALE_MS = 5 * 60 * 1000;

type ResolvedUserAvatarProps = React.ComponentProps<typeof Avatar> & {
  userId?: User["id"];
  name?: string;
  /** The person's signed photo, as the read that names them carries it. */
  photo?: PersonPhoto | null;
  imageClassName?: string;
  fallbackClassName?: string;
  alt?: string;
  variant?: "thumbnail" | "full";
};

/**
 * Who a gradient is drawn for: the signed-in person from their own record, anyone else
 * from the batched avatar lookup. Until a teammate's answer arrives the tile stays the
 * plain sunken fill, so a gradient never flashes to a different one.
 */
function useAvatarGradient(userId: string | undefined, name: string | undefined) {
  const me = useAuthStore((state) => state.user);
  const isMe = Boolean(userId && me?.id === userId);
  const { data: summary } = useQuery({
    ...user.avatar(userId ?? ""),
    enabled: Boolean(userId) && !isMe,
    staleTime: AVATAR_STALE_MS,
    retry: false,
  });

  if (isMe && me) {
    return avatarBackground(me.username || me.id || "", me.avatarStyle);
  }
  if (summary) {
    return avatarBackground(summary.username || summary.id, summary.avatarStyle);
  }
  if (!userId && name) {
    return avatarBackground(name);
  }
  return undefined;
}

export function ResolvedUserAvatar({
  userId,
  name,
  photo,
  imageClassName,
  fallbackClassName,
  alt,
  variant = "thumbnail",
  className,
  ...avatarProps
}: ResolvedUserAvatarProps) {
  const imageSrc = personPhotoSource(photo, variant);
  const gradient = useAvatarGradient(userId, name);

  return (
    <Avatar {...avatarProps} className={cn("rounded-full after:rounded-full", className)}>
      <AvatarImage
        className={cn("rounded-full", imageClassName)}
        src={imageSrc}
        alt={alt ?? name}
      />
      <AvatarFallback
        aria-label={name}
        role={name ? "img" : undefined}
        className={cn(fallbackClassName, "bg-muted rounded-full bg-cover")}
        style={gradient ? { backgroundImage: gradient } : undefined}
      />
    </Avatar>
  );
}
