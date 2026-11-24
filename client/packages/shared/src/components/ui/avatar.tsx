import * as React from "react";

import { photoPath, reportExpiredPhoto } from "@trenova/shared/lib/photo-expiry";
import { cn } from "@trenova/shared/lib/utils";

type AvatarImageStatus = "idle" | "loading" | "loaded" | "error";

type AvatarStatus = {
  status: AvatarImageStatus;
  setStatus: (status: AvatarImageStatus) => void;
};

const AvatarStatusContext = React.createContext<AvatarStatus | null>(null);

/**
 * A face or a mark in a fixed box. The picture (`AvatarImage`) is a native lazy image laid over the
 * fallback in the same box, so nothing moves when it arrives; the fallback (initials, a gradient)
 * shows until the picture has loaded, and again if it fails.
 */
function Avatar({
  className,
  size = "default",
  ...props
}: React.ComponentProps<"span"> & {
  size?: "default" | "sm" | "lg";
}) {
  const [status, setStatus] = React.useState<AvatarImageStatus>("idle");
  return (
    <AvatarStatusContext value={{ status, setStatus }}>
      <span
        data-slot="avatar"
        data-size={size}
        data-image-status={status}
        className={cn(
          "group/avatar relative flex size-8 shrink-0 rounded-md select-none after:absolute after:inset-0 after:rounded-md after:border after:border-border after:mix-blend-darken data-[size=lg]:size-10 data-[size=sm]:size-6 dark:after:mix-blend-lighten",
          className,
        )}
        {...props}
      />
    </AvatarStatusContext>
  );
}

/**
 * The picture, fetched only as it nears the screen (`loading="lazy"`) and decoded off the main
 * thread. A preloading avatar (`new Image()`) fetches every face in a long list at once; this one
 * leaves that to the browser.
 *
 * A new `src` loads behind the picture already shown, which stays until the new one has loaded,
 * so a link signed again (the same path, a new query string) swaps without a flicker. Until a
 * picture has loaded the fallback is what is seen and read. A link that fails is reported once
 * (`reportExpiredPhoto`), so the data that carried it is read again; the initials show only when
 * nothing has loaded, or when a different picture failed.
 */
function AvatarImage({
  className,
  src,
  alt = "",
  onLoad,
  onError,
  "aria-hidden": ariaHidden,
  ...props
}: Omit<React.ComponentProps<"img">, "loading" | "decoding">) {
  const avatar = React.use(AvatarStatusContext);
  const setStatus = avatar?.setStatus;
  const [shown, setShown] = React.useState<string | null>(null);
  const [failed, setFailed] = React.useState<string | null>(null);
  const current = src || null;

  let visible: string | null = null;
  let pending: string | null = null;
  if (current) {
    if (shown === current) {
      visible = shown;
    } else if (failed === current) {
      visible = shown && photoPath(shown) === photoPath(current) ? shown : null;
    } else {
      visible = shown;
      pending = current;
    }
  }
  const status: AvatarImageStatus = visible ? "loaded" : pending ? "loading" : "error";

  React.useEffect(() => {
    setStatus?.(status);
  }, [setStatus, status]);
  React.useEffect(() => () => setStatus?.("idle"), [setStatus]);

  const imageClass = "absolute inset-0 aspect-square size-full rounded-md object-cover";
  return (
    <>
      {visible ? (
        <img
          {...props}
          key="shown"
          data-slot="avatar-image"
          src={visible}
          alt={alt}
          aria-hidden={ariaHidden}
          loading="lazy"
          decoding="async"
          className={cn(imageClass, className)}
        />
      ) : null}
      {pending ? (
        <img
          key={pending}
          data-slot="avatar-image"
          data-pending=""
          src={pending}
          alt=""
          aria-hidden
          loading="lazy"
          decoding="async"
          ref={(image) => {
            // A picture already in the cache can finish before React listens for its load.
            if (image?.complete && image.naturalWidth > 0) {
              setShown(pending);
            }
          }}
          onLoad={(event) => {
            setShown(pending);
            onLoad?.(event);
          }}
          onError={(event) => {
            setFailed(pending);
            reportExpiredPhoto(pending);
            onError?.(event);
          }}
          className={cn(imageClass, "opacity-0", className)}
        />
      ) : null}
    </>
  );
}

/** What the avatar shows while it has no picture: until one loads, or when there is none. */
function AvatarFallback({ className, ...props }: React.ComponentProps<"span">) {
  const avatar = React.use(AvatarStatusContext);
  if (avatar?.status === "loaded") {
    return null;
  }
  return (
    <span
      data-slot="avatar-fallback"
      className={cn(
        "flex size-full items-center justify-center rounded-md bg-muted text-sm text-muted-foreground group-data-[size=sm]/avatar:text-xs",
        className,
      )}
      {...props}
    />
  );
}

function AvatarBadge({ className, ...props }: React.ComponentProps<"span">) {
  return (
    <span
      data-slot="avatar-badge"
      className={cn(
        "absolute right-0 bottom-0 z-10 inline-flex items-center justify-center rounded-full bg-primary text-primary-foreground bg-blend-color ring-2 ring-background select-none",
        "group-data-[size=sm]/avatar:size-2 group-data-[size=sm]/avatar:[&>svg]:hidden",
        "group-data-[size=default]/avatar:size-2.5 group-data-[size=default]/avatar:[&>svg]:size-2",
        "group-data-[size=lg]/avatar:size-3 group-data-[size=lg]/avatar:[&>svg]:size-2",
        className,
      )}
      {...props}
    />
  );
}

function AvatarGroup({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="avatar-group"
      className={cn(
        "group/avatar-group flex -space-x-2 *:data-[slot=avatar]:ring-2 *:data-[slot=avatar]:ring-background",
        className,
      )}
      {...props}
    />
  );
}

function AvatarGroupCount({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="avatar-group-count"
      className={cn(
        "relative flex size-8 shrink-0 items-center justify-center rounded-full bg-muted text-sm text-muted-foreground ring-2 ring-background group-has-data-[size=lg]/avatar-group:size-10 group-has-data-[size=sm]/avatar-group:size-6 [&>svg]:size-4 group-has-data-[size=lg]/avatar-group:[&>svg]:size-5 group-has-data-[size=sm]/avatar-group:[&>svg]:size-3",
        className,
      )}
      {...props}
    />
  );
}

export { Avatar, AvatarBadge, AvatarFallback, AvatarGroup, AvatarGroupCount, AvatarImage };
