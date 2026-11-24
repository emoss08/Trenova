import { worker as workerQueries } from "@/lib/queries/worker";
import type { WorkerRecord } from "@/routes/worker/use-worker-state";
import { workerService } from "@/services/worker";
import { useQueryClient } from "@tanstack/react-query";
import { PROFILE_PICTURE } from "@trenova/shared/components/image-upload/image-spec";
import { ProfilePicturePreview } from "@trenova/shared/components/image-upload/profile-picture-preview";
import {
  useImageUpload,
  type ImageDropTarget,
} from "@trenova/shared/components/image-upload/use-image-upload";
import { toast } from "@trenova/shared/components/ui/toast";
import { useT } from "@trenova/shared/i18n/use-t";
import { describeError } from "@trenova/shared/lib/error-presentation";
import { photoAfterChange, type WorkerPhotoChange } from "@trenova/shared/types/person-photo";
import { useRef, useState, type ReactNode } from "react";
import { useWorkerInvalidation } from "./use-worker-invalidation";

export type WorkerPhotoControls = {
  openPicker: () => void;
  /** Reads the crop dialog's code ahead of the first photo; call it when the pointer arrives. */
  preload: () => void;
  /** Spread on the element that takes a dropped or pasted photo. */
  dropTarget: ImageDropTarget;
  /** Whether a file is being dragged over that element. */
  dragging: boolean;
  /** A positioned photo is on its way to the server. */
  uploading: boolean;
  remove: () => Promise<void>;
  removing: boolean;
  /** The hidden file input and the crop dialog; render once. */
  element: ReactNode;
};

/**
 * Choosing, cropping, uploading and removing a worker's photo from their record. Each change moves
 * the worker's version, so the answer's version and photo are written straight onto the cached
 * record (the Profile form's next save carries the new version and is not refused as stale), and
 * then the record and the roster are read again.
 */
export function useWorkerPhoto(
  record: WorkerRecord,
  { disabled = false }: { disabled?: boolean } = {},
): WorkerPhotoControls {
  const t = useT();
  const queryClient = useQueryClient();
  const invalidate = useWorkerInvalidation(record.id, "photo");
  const recordKey = workerQueries.record(record.id).queryKey;
  const [uploading, setUploading] = useState(false);
  const [removing, setRemoving] = useState(false);
  const busy = useRef(false);
  const name = record.wholeName || `${record.firstName} ${record.lastName}`.trim();

  /** The version the server holds now, as far as this page knows: the cache moves before props do. */
  const currentVersion = () =>
    queryClient.getQueryData<WorkerRecord | null>(recordKey)?.version ?? record.version ?? 0;

  const apply = async (change: WorkerPhotoChange) => {
    queryClient.setQueryData<WorkerRecord | null>(recordKey, (current) =>
      current ? { ...current, version: change.version, photo: photoAfterChange(change) } : current,
    );
    await invalidate();
  };

  /** A refused change may have been refused for a stale version: read the record again. */
  const refreshAfterFailure = () => {
    void queryClient.invalidateQueries({ queryKey: recordKey });
  };

  const upload = useImageUpload({
    spec: PROFILE_PICTURE,
    preview: <ProfilePicturePreview name={name} detail={record.position?.title ?? undefined} />,
    title: t("Position {0}'s photo", name),
    confirmLabel: t("Save photo"),
    disabled: disabled || removing,
    onUpload: async (file) => {
      if (busy.current) return;
      busy.current = true;
      setUploading(true);
      try {
        await apply(await workerService.uploadPhoto(record.id, file, currentVersion()));
      } catch (error) {
        refreshAfterFailure();
        throw error;
      } finally {
        busy.current = false;
        setUploading(false);
      }
    },
    // The photo is an avatar with no line of its own to say what was wrong with a file.
    onProblem: (problem) => {
      if (problem) {
        toast.error(t("That file cannot be their photo"), { description: problem });
      }
    },
  });

  const remove = async () => {
    if (busy.current || disabled) return;
    busy.current = true;
    setRemoving(true);
    try {
      await apply(await workerService.removePhoto(record.id, currentVersion()));
    } catch (error) {
      refreshAfterFailure();
      toast.error(t("The photo could not be removed"), {
        description: describeError(error).detail ?? t("Please try again."),
      });
    } finally {
      busy.current = false;
      setRemoving(false);
    }
  };

  return {
    openPicker: upload.openPicker,
    preload: upload.preload,
    dropTarget: upload.dropTarget,
    dragging: upload.dragging,
    uploading,
    remove,
    removing,
    element: upload.element,
  };
}
