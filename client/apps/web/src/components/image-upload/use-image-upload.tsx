import { lazyComponent } from "@/lib/lazy-component";
import { LazyComponent } from "@trenova/shared/components/error-boundary";
import { useT } from "@trenova/shared/i18n/use-t";
import { useRef, useState, type ClipboardEvent, type DragEvent, type ReactNode } from "react";
import {
  IMAGE_UPLOAD_ACCEPT,
  MAX_IMAGE_MEGABYTES,
  imageRejection,
  offeredFile,
} from "./image-file";
import type { ImageSpec } from "./image-spec";

// The crop dialog brings the cropper with it, so its code is read when a picture is first
// offered (or a moment earlier, when the pointer reaches the control) and not with the page.
const CropDialog = lazyComponent(
  () => import("./image-crop-dialog"),
  (module) => module.ImageCropDialog,
);

export type ImageUploadOptions = {
  spec: ImageSpec;
  /**
   * The places the result is drawn, shown in the crop dialog as the picture moves, with a
   * `CropResult` wherever one draws it.
   */
  preview?: ReactNode;
  /** The crop dialog's title. */
  title: string;
  /** The crop dialog's primary button. */
  confirmLabel: string;
  disabled?: boolean;
  /** Uploads the positioned picture. A rejection keeps the dialog open and is shown in it. */
  onUpload: (file: File) => Promise<void>;
  /** Why the file last offered was turned away, in words, or null once one is taken. */
  onProblem: (problem: string | null) => void;
};

export type ImageDropTarget = {
  onDragEnter: (event: DragEvent<HTMLElement>) => void;
  onDragOver: (event: DragEvent<HTMLElement>) => void;
  onDragLeave: (event: DragEvent<HTMLElement>) => void;
  onDrop: (event: DragEvent<HTMLElement>) => void;
  onPaste: (event: ClipboardEvent<HTMLElement>) => void;
};

export type ImageUpload = {
  openPicker: () => void;
  /** Reads the crop dialog's code ahead of the first picture. */
  preload: () => void;
  /** Whether a file is being dragged over the drop target. */
  dragging: boolean;
  /** Spread on the element that takes a dropped or pasted picture. */
  dropTarget: ImageDropTarget;
  /** The hidden file input and the crop dialog; render once. */
  element: ReactNode;
};

function carriesFiles(event: DragEvent<HTMLElement>): boolean {
  return Array.from(event.dataTransfer.types).includes("Files");
}

function preloadCropDialog(): void {
  void CropDialog.preload().catch(() => undefined);
}

/**
 * Choosing a picture, by the file picker, a drop or a paste, then positioning it in the crop
 * dialog and handing the result to `onUpload`. It draws nothing of its own but the dialog, so
 * any control can open it: `ImageUploadField` is the ready-made one.
 */
export function useImageUpload({
  spec,
  preview,
  title,
  confirmLabel,
  disabled = false,
  onUpload,
  onProblem,
}: ImageUploadOptions): ImageUpload {
  const t = useT();
  const inputRef = useRef<HTMLInputElement>(null);
  const dragDepth = useRef(0);
  const [dragging, setDragging] = useState(false);
  const [file, setFile] = useState<File | null>(null);
  const [open, setOpen] = useState(false);
  const [replaceProblem, setReplaceProblem] = useState<string | null>(null);

  const offer = (files: FileList | readonly File[] | null | undefined) => {
    const offered = offeredFile(files);
    if (!offered) return;

    const rejection = imageRejection(offered);
    const problem =
      rejection === "type"
        ? t("Choose a PNG, JPG, WebP or SVG picture.")
        : rejection === "size"
          ? t("That picture is over {0} MB. Choose a smaller one.", MAX_IMAGE_MEGABYTES)
          : null;

    if (open) {
      setReplaceProblem(problem);
    } else {
      onProblem(problem);
    }
    if (!problem) {
      setFile(offered);
      setOpen(true);
    }
  };

  const dropTarget: ImageDropTarget = {
    onDragEnter: (event) => {
      if (disabled || !carriesFiles(event)) return;
      event.preventDefault();
      dragDepth.current += 1;
      setDragging(true);
      preloadCropDialog();
    },
    onDragOver: (event) => {
      if (disabled || !carriesFiles(event)) return;
      event.preventDefault();
      event.dataTransfer.dropEffect = "copy";
    },
    onDragLeave: (event) => {
      if (!carriesFiles(event)) return;
      dragDepth.current = Math.max(0, dragDepth.current - 1);
      if (dragDepth.current === 0) {
        setDragging(false);
      }
    },
    onDrop: (event) => {
      if (!carriesFiles(event)) return;
      event.preventDefault();
      dragDepth.current = 0;
      setDragging(false);
      if (!disabled) {
        offer(event.dataTransfer.files);
      }
    },
    onPaste: (event) => {
      if (disabled || event.clipboardData.files.length === 0) return;
      event.preventDefault();
      offer(event.clipboardData.files);
    },
  };

  const openPicker = () => {
    if (!disabled) {
      inputRef.current?.click();
    }
  };

  const element = (
    <>
      <input
        ref={inputRef}
        type="file"
        accept={IMAGE_UPLOAD_ACCEPT}
        className="hidden"
        aria-hidden
        tabIndex={-1}
        onChange={(event) => {
          const files = Array.from(event.target.files ?? []);
          event.target.value = "";
          offer(files);
        }}
      />
      {file ? (
        <LazyComponent fallback={null}>
          <CropDialog.Component
            open={open}
            file={file}
            spec={spec}
            preview={preview}
            title={title}
            confirmLabel={confirmLabel}
            problem={replaceProblem}
            onReplace={() => inputRef.current?.click()}
            onClose={() => {
              setOpen(false);
              setReplaceProblem(null);
            }}
            onConfirm={onUpload}
          />
        </LazyComponent>
      ) : null}
    </>
  );

  return { openPicker, preload: preloadCropDialog, dragging, dropTarget, element };
}
