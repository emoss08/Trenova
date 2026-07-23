import type { PixelCrop } from "react-image-crop";

const ALLOWED_IMAGE_MIME_TYPES = new Set([
  "image/jpeg",
  "image/jpg",
  "image/png",
  "image/webp",
]);

const OUTPUT_MIME_TYPE = "image/webp";

function getBaseFileName(name: string): string {
  const lastDot = name.lastIndexOf(".");
  if (lastDot <= 0) {
    return name;
  }

  return name.slice(0, lastDot);
}

export function validateCroppableImage(file: File, label: string) {
  if (!ALLOWED_IMAGE_MIME_TYPES.has(file.type.toLowerCase())) {
    throw new Error(`Only JPG, PNG, and WEBP files are supported for ${label}`);
  }
}

export async function loadImageFromFile(file: File): Promise<HTMLImageElement> {
  const objectURL = URL.createObjectURL(file);

  try {
    const image = new Image();
    image.decoding = "async";
    image.src = objectURL;
    await image.decode();
    return image;
  } finally {
    URL.revokeObjectURL(objectURL);
  }
}

function canvasToBlob(canvas: HTMLCanvasElement, quality: number): Promise<Blob> {
  return new Promise((resolve, reject) => {
    canvas.toBlob(
      (blob) => {
        if (!blob) {
          reject(new Error("Failed to create cropped image"));
          return;
        }

        resolve(blob);
      },
      OUTPUT_MIME_TYPE,
      quality,
    );
  });
}

export async function createCroppedWebPFile(
  image: HTMLImageElement,
  crop: PixelCrop,
  file: File,
  options: {
    maxDimension?: number;
    targetWidth?: number;
    targetHeight?: number;
    quality?: number;
    fileNamePrefix?: string;
  },
): Promise<File> {
  const scaleX = image.naturalWidth / image.width;
  const scaleY = image.naturalHeight / image.height;
  const cropWidth = Math.max(1, Math.round(crop.width * scaleX));
  const cropHeight = Math.max(1, Math.round(crop.height * scaleY));
  const targetWidth = options.targetWidth
    ? Math.max(1, Math.round(options.targetWidth))
    : Math.max(
        1,
        Math.round(
          cropWidth * Math.min(1, (options.maxDimension ?? cropWidth) / Math.max(cropWidth, cropHeight)),
        ),
      );
  const targetHeight = options.targetHeight
    ? Math.max(1, Math.round(options.targetHeight))
    : Math.max(
        1,
        Math.round(
          cropHeight * Math.min(1, (options.maxDimension ?? cropHeight) / Math.max(cropWidth, cropHeight)),
        ),
      );

  const canvas = document.createElement("canvas");
  canvas.width = targetWidth;
  canvas.height = targetHeight;

  const context = canvas.getContext("2d");
  if (!context) {
    throw new Error("Failed to initialize cropped image canvas");
  }

  context.imageSmoothingQuality = "high";
  context.drawImage(
    image,
    Math.round(crop.x * scaleX),
    Math.round(crop.y * scaleY),
    cropWidth,
    cropHeight,
    0,
    0,
    targetWidth,
    targetHeight,
  );

  const blob = await canvasToBlob(canvas, options.quality ?? 0.86);
  const baseFileName = getBaseFileName(file.name) || options.fileNamePrefix || "image";

  return new File([blob], `${baseFileName}.webp`, {
    type: OUTPUT_MIME_TYPE,
    lastModified: Date.now(),
  });
}
