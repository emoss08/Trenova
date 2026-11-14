import { PROFILE_PICTURE } from "@/components/image-upload/image-spec";
import { ProfilePicturePreview } from "@/components/image-upload/profile-picture-preview";
import { useImageUpload, type ImageDropTarget } from "@/components/image-upload/use-image-upload";
import { user } from "@/lib/queries/user";
import { userService } from "@trenova/shared/services/user";
import { useQueryClient } from "@tanstack/react-query";
import { translate } from "@trenova/shared/i18n/runtime";
import { useT } from "@trenova/shared/i18n/use-t";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import type { User } from "@trenova/shared/types/user";
import { useCallback, useState, type ReactNode } from "react";
import { toast } from "@trenova/shared/components/ui/toast";
import { useSettingsFeedback } from "./settings-feedback";

export type ProfilePictureControls = {
  openPicker: () => void;
  /** Reads the crop dialog's code ahead of the first picture; call it when the pointer arrives. */
  preload: () => void;
  /** Spread on the element that takes a dropped or pasted photo. */
  dropTarget: ImageDropTarget;
  /** Whether a file is being dragged over that element. */
  dragging: boolean;
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
  const [removing, setRemoving] = useState(false);
  const me = useAuthStore((state) => state.user);

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

  const upload = useImageUpload({
    spec: PROFILE_PICTURE,
    preview: (
      <ProfilePicturePreview
        name={me?.name || me?.username || ""}
        detail={me?.jobTitle || undefined}
        handle={me?.username || undefined}
      />
    ),
    title: t("Position your photo"),
    confirmLabel: t("Save photo"),
    onUpload: async (file) => {
      const updated = await userService.uploadMyProfilePicture(file);
      await sync(updated);
      feedback.message(t("Photo updated"));
    },
    // The photo's controls are an avatar and a button with no line of their own to say what
    // was wrong with a file, so a toast does.
    onProblem: (problem) => {
      if (problem) {
        toast.error(t("That file cannot be your photo"), { description: problem });
      }
    },
  });

  const remove = useCallback(async () => {
    if (removing) return;
    setRemoving(true);
    try {
      const updated = await userService.deleteMyProfilePicture();
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

  return {
    openPicker: upload.openPicker,
    preload: upload.preload,
    dropTarget: upload.dropTarget,
    dragging: upload.dragging,
    remove,
    removing,
    element: upload.element,
  };
}
