import type { SyntheticEvent } from "react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { createCroppedWebPFile, loadImageFromFile } from "@/lib/images/crop-image";
import { cn } from "@/lib/utils";
import { Loader2 } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import ReactCrop, {
  centerCrop,
  makeAspectCrop,
  type Crop,
  type PixelCrop,
} from "react-image-crop";
import "react-image-crop/dist/ReactCrop.css";

type ImageCropUploadDialogProps = {
  open: boolean;
  file: File | null;
  title: string;
  description: string;
  maxDimension?: number;
  targetWidth?: number;
  targetHeight?: number;
  aspect?: number;
  onClose: () => void;
  onConfirm: (file: File) => Promise<void>;
  confirmLabel?: string;
  className?: string;
};

const DEFAULT_CROP: Crop = {
  unit: "%",
  width: 90,
  height: 90,
  x: 5,
  y: 5,
};

function makeCenteredAspectCrop(mediaWidth: number, mediaHeight: number, aspect?: number): Crop {
  if (!aspect) {
    return DEFAULT_CROP;
  }

  return centerCrop(
    makeAspectCrop(
      {
        unit: "%",
        width: 85,
      },
      aspect,
      mediaWidth,
      mediaHeight,
    ),
    mediaWidth,
    mediaHeight,
  );
}

export function ImageCropUploadDialog({
  open,
  file,
  title,
  description,
  maxDimension,
  targetWidth,
  targetHeight,
  aspect,
  onClose,
  onConfirm,
  confirmLabel = "Upload",
  className,
}: ImageCropUploadDialogProps) {
  const imageRef = useRef<HTMLImageElement | null>(null);
  const [previewUrl, setPreviewUrl] = useState<string | null>(null);
  const [crop, setCrop] = useState<Crop>();
  const [completedCrop, setCompletedCrop] = useState<PixelCrop>();
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  useEffect(() => {
    if (!file) {
      setPreviewUrl(null);
      setCrop(undefined);
      setCompletedCrop(undefined);
      setErrorMessage(null);
      return;
    }

    const objectUrl = URL.createObjectURL(file);
    setPreviewUrl(objectUrl);
    setCrop(undefined);
    setCompletedCrop(undefined);
    setErrorMessage(null);

    return () => {
      URL.revokeObjectURL(objectUrl);
    };
  }, [file]);

  const canSubmit = useMemo(
    () => Boolean(file && completedCrop?.width && completedCrop?.height && !isSubmitting),
    [completedCrop?.height, completedCrop?.width, file, isSubmitting],
  );

  const handleConfirm = async () => {
    if (!file || !completedCrop) {
      return;
    }

    try {
      setIsSubmitting(true);
      setErrorMessage(null);

      const image = imageRef.current ?? (await loadImageFromFile(file));
      const croppedFile = await createCroppedWebPFile(image, completedCrop, file, {
        maxDimension,
        targetWidth,
        targetHeight,
      });

      await onConfirm(croppedFile);
      onClose();
    } catch (error) {
      setErrorMessage(error instanceof Error ? error.message : "Failed to prepare image");
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={(nextOpen) => !nextOpen && !isSubmitting && onClose()}>
      <DialogContent className={cn("sm:max-w-3xl", className)}>
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>{description}</DialogDescription>
        </DialogHeader>

        {previewUrl ? (
          <div className="space-y-3">
            <div className="overflow-hidden rounded-lg border bg-muted/20">
              <ReactCrop
                crop={crop}
                onChange={(_, percentCrop) => setCrop(percentCrop)}
                onComplete={(nextCrop) => setCompletedCrop(nextCrop)}
                aspect={aspect}
                className="max-h-[65vh] w-full"
              >
                <div className="flex min-h-[420px] w-full items-center justify-center bg-muted/30 sm:min-h-[460px]">
                  <img
                    ref={imageRef}
                    src={previewUrl}
                    alt="Selected upload"
                    className="block max-h-[65vh] max-w-full object-contain"
                    onLoad={(event: SyntheticEvent<HTMLImageElement>) => {
                      const { width, height } = event.currentTarget;
                      const nextCrop = makeCenteredAspectCrop(width, height, aspect);
                      setCrop(nextCrop);
                    }}
                  />
                </div>
              </ReactCrop>
            </div>

            {errorMessage ? <p className="text-sm text-destructive">{errorMessage}</p> : null}
          </div>
        ) : (
          <div className="rounded-lg border border-dashed p-8 text-center text-sm text-muted-foreground">
            Select an image to continue.
          </div>
        )}

        <DialogFooter>
          <Button type="button" variant="outline" onClick={onClose} disabled={isSubmitting}>
            Cancel
          </Button>
          <Button type="button" onClick={() => void handleConfirm()} disabled={!canSubmit}>
            {isSubmitting ? <Loader2 className="size-4 animate-spin" /> : null}
            {confirmLabel}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
