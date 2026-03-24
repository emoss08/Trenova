const ALLOWED_LOGO_MIME_TYPES = new Set([
  "image/jpeg",
  "image/jpg",
  "image/png",
  "image/webp",
]);

const OUTPUT_MIME_TYPE = "image/webp";
const DEFAULT_MAX_DIMENSION = 1024;
const DEFAULT_QUALITY = 0.82;

function getBaseFileName(name: string): string {
  const lastDot = name.lastIndexOf(".");
  if (lastDot <= 0) {
    return name;
  }

  return name.slice(0, lastDot);
}

async function loadImage(file: File): Promise<HTMLImageElement> {
  const objectURL = URL.createObjectURL(file);

  try {
    const img = new Image();
    img.decoding = "async";
    img.src = objectURL;
    await img.decode();
    return img;
  } finally {
    URL.revokeObjectURL(objectURL);
  }
}

function getTargetDimensions(
  width: number,
  height: number,
  maxDimension: number,
) {
  if (width <= maxDimension && height <= maxDimension) {
    return { width, height };
  }

  const ratio = Math.min(maxDimension / width, maxDimension / height);
  return {
    width: Math.max(1, Math.round(width * ratio)),
    height: Math.max(1, Math.round(height * ratio)),
  };
}

function canvasToBlob(
  canvas: HTMLCanvasElement,
  quality: number,
): Promise<Blob> {
  return new Promise((resolve, reject) => {
    canvas.toBlob(
      (blob) => {
        if (!blob) {
          reject(new Error("Failed to compress image"));
          return;
        }

        resolve(blob);
      },
      OUTPUT_MIME_TYPE,
      quality,
    );
  });
}

export async function convertOrganizationLogoToWebP(
  file: File,
  options: { maxDimension?: number; quality?: number } = {},
): Promise<File> {
  if (!ALLOWED_LOGO_MIME_TYPES.has(file.type.toLowerCase())) {
    throw new Error("Only JPG, PNG, and WEBP files are supported for logos");
  }

  const maxDimension = options.maxDimension ?? DEFAULT_MAX_DIMENSION;
  const quality = options.quality ?? DEFAULT_QUALITY;

  const image = await loadImage(file);
  const target = getTargetDimensions(
    image.naturalWidth,
    image.naturalHeight,
    maxDimension,
  );
  const canvas = document.createElement("canvas");
  canvas.width = target.width;
  canvas.height = target.height;

  const context = canvas.getContext("2d");
  if (!context) {
    throw new Error("Failed to initialize image compression");
  }

  context.drawImage(image, 0, 0, target.width, target.height);
  const blob = await canvasToBlob(canvas, quality);
  const baseFileName = getBaseFileName(file.name) || "organization-logo";

  return new File([blob], `${baseFileName}.webp`, {
    type: OUTPUT_MIME_TYPE,
    lastModified: Date.now(),
  });
}
