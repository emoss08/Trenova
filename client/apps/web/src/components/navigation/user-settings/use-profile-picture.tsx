import { ImageCropUploadDialog } from "@/components/image-crop-upload-dialog";
import { validateCroppableImage } from "@/lib/images/crop-image";
import { IMAGE_UPLOAD_ACCEPT, profilePictureCropConfig } from "@/lib/images/upload-config";
import { user } from "@/lib/queries/user";
import { apiService } from "@/services/api";
import { useQueryClient } from "@tanstack/react-query";
import { translate } from "@trenova/shared/i18n/runtime";
import { useT } from "@trenova/shared/i18n/use-t";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import type { User } from "@trenova/shared/types/user";
import { useCallback, useRef, useState, type ChangeEvent, type ReactNode } from "react";
import { toast } from "sonner";
import { useSettingsFeedback } from "./settings-feedback";

export type ProfilePictureControls = {
  openPicker: () => void;
  remove: () => Promise<void>;
  removing: boolean;
  /** The hidden file input and the crop dialog; render once. */
  element: ReactNode;
};

/** Choosing, cropping, uploading and removing the signed-in person's photo. */
export function useProfilePicture(): ProfilePictureControls {
  const t = useT();
  const queryClient = useQueryClient();
  const feedback = useSettingsFeedback();
  const inputRef = useRef<HTMLInputElement | null>(null);
  const [pendingFile, setPendingFile] = useState<File | null>(null);
  const [cropOpen, setCropOpen] = useState(false);
  const [removing, setRemoving] = useState(false);

  const sync = useCallback(
    async (updated: User) => {
      useAuthStore.getState().setUser(updated);
      if (!updated.id) return;
      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: user.profilePicture(updated.id, "thumbnail").queryKey,
        }),
        queryClient.invalidateQueries({
          queryKey: user.profilePicture(updated.id, "full").queryKey,
        }),
      ]);
    },
    [queryClient],
  );

  const onFile = useCallback(
    (event: ChangeEvent<HTMLInputElement>) => {
      const selected = event.target.files?.[0];
      event.target.value = "";
      if (!selected) return;

      try {
        validateCroppableImage(selected, "profile pictures");
        setPendingFile(selected);
        setCropOpen(true);
      } catch (error) {
        toast.error(t("Unsupported profile picture"), {
          description:
            error instanceof Error
              ? error.message
              : translate("Please choose a JPG, PNG, or WEBP file."),
        });
      }
    },
    [t],
  );

  const upload = useCallback(
    async (file: File) => {
      const updated = await apiService.userService.uploadMyProfilePicture(file);
      await sync(updated);
      feedback.message(t("Photo updated"));
    },
    [feedback, sync, t],
  );

  const remove = useCallback(async () => {
    if (removing) return;
    setRemoving(true);
    try {
      const updated = await apiService.userService.deleteMyProfilePicture();
      await sync(updated);
      feedback.message(t("Photo removed"));
    } catch (error) {
      toast.error(t("Failed to remove profile picture"), {
        description: error instanceof Error ? error.message : translate("Please try again."),
      });
    } finally {
      setRemoving(false);
    }
  }, [feedback, removing, sync, t]);

  const element = (
    <>
      <input
        ref={inputRef}
        type="file"
        accept={IMAGE_UPLOAD_ACCEPT}
        className="hidden"
        aria-hidden
        tabIndex={-1}
        onChange={onFile}
      />
      <ImageCropUploadDialog
        open={cropOpen}
        file={pendingFile}
        title={t("Crop profile picture")}
        description={t(
          "Adjust your image before uploading. Profile pictures are cropped to a square.",
        )}
        {...profilePictureCropConfig}
        confirmLabel={t("Upload picture")}
        onClose={() => {
          setCropOpen(false);
          setPendingFile(null);
        }}
        onConfirm={upload}
      />
    </>
  );

  return {
    openPicker: () => inputRef.current?.click(),
    remove,
    removing,
    element,
  };
}
