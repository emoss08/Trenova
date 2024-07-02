import { Button } from "@/components/ui/button";
import * as AvatarPrimitive from "@radix-ui/react-avatar";
import * as React from "react";
import { useRef } from "react";
import { toast } from "sonner";

import { cn } from "@/lib/utils";
import { UploadIcon } from "@radix-ui/react-icons";

const Avatar = React.forwardRef<
  React.ElementRef<typeof AvatarPrimitive.Root>,
  React.ComponentPropsWithoutRef<typeof AvatarPrimitive.Root>
>(({ className, ...props }, ref) => (
  <AvatarPrimitive.Root
    ref={ref}
    className={cn(
      "relative flex h-10 w-10 shrink-0 overflow-hidden rounded-full",
      className,
    )}
    {...props}
  />
));
Avatar.displayName = AvatarPrimitive.Root.displayName;

const AvatarImage = React.forwardRef<
  React.ElementRef<typeof AvatarPrimitive.Image>,
  React.ComponentPropsWithoutRef<typeof AvatarPrimitive.Image>
>(({ className, ...props }, ref) => (
  <AvatarPrimitive.Image
    ref={ref}
    className={cn("aspect-square h-full w-full", className)}
    {...props}
  />
));
AvatarImage.displayName = AvatarPrimitive.Image.displayName;

const AvatarFallback = React.forwardRef<
  React.ElementRef<typeof AvatarPrimitive.Fallback>,
  React.ComponentPropsWithoutRef<typeof AvatarPrimitive.Fallback>
>(({ className, ...props }, ref) => (
  <AvatarPrimitive.Fallback
    ref={ref}
    className={cn(
      "flex h-full w-full items-center justify-center rounded-full bg-muted",
      className,
    )}
    {...props}
  />
));
AvatarFallback.displayName = AvatarPrimitive.Fallback.displayName;

export { Avatar, AvatarFallback, AvatarImage };

export function ImageUploader({
  callback,
  successCallback,
  removeFileCallback,
  removeSuccessCallback,
  iconText = "Change Avatar",
}: {
  callback: (file: File) => Promise<any>;
  successCallback: (data: any) => string;
  removeFileCallback?: () => Promise<any>;
  removeSuccessCallback?: (data: any) => string;
  iconText?: string;
}) {
  const fileInputRef = useRef<HTMLInputElement>(null);

  // Handle file change event
  const handleFileChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const files = event.target.files;
    if (files && files.length > 0) {
      const file = files[0];
      toast.promise(callback(file), {
        loading: "Uploading your image...",
        success: successCallback,
        error: "Failed to upload image.",
      });
    }
  };

  // Function to trigger file input
  const handleClick = () => {
    if (fileInputRef.current) {
      // Check if the ref is not null
      fileInputRef.current.click();
    }
  };

  return (
    <div className="flex flex-col items-center">
      <div className="flex gap-x-2">
        <Button size="sm" type="button" onClick={handleClick}>
          <UploadIcon className="mr-2 size-4" />
          {iconText}
        </Button>
        <Button
          size="sm"
          type="button"
          variant="outline"
          onClick={() => {
            if (removeFileCallback) {
              toast.promise(removeFileCallback, {
                loading: "Removing your image...",
                success: removeSuccessCallback || "Image removed successfully.",
                error: "Failed to remove image.",
              });
            }
          }}
        >
          Remove
        </Button>
      </div>
      <div className="flex gap-x-2">
        <input
          ref={fileInputRef}
          type="file"
          accept=".jpg, .gif, .png, .webp"
          style={{ display: "none" }}
          onChange={handleFileChange}
        />
        <p className="text-muted-foreground mt-2 text-xs leading-5">
          JPG, GIF, WEBP or PNG. Max size 1MB.
        </p>
      </div>
    </div>
  );
}
