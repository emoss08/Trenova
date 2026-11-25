export const SVG_TYPE = "image/svg+xml";

export const IMAGE_UPLOAD_ACCEPT = `image/png,image/jpeg,image/webp,${SVG_TYPE}`; // i18n-ignore: media types

export const MAX_IMAGE_MEGABYTES = 20;

const MAX_IMAGE_BYTES = MAX_IMAGE_MEGABYTES * 1024 * 1024;

const IMAGE_TYPES = new Set(IMAGE_UPLOAD_ACCEPT.split(","));

export type ImageRejection = "type" | "size";

/** Why a file cannot be used as a picture, or null when it can. */
export function imageRejection(file: File): ImageRejection | null {
  if (!IMAGE_TYPES.has(file.type.toLowerCase())) {
    return "type";
  }
  if (file.size > MAX_IMAGE_BYTES) {
    return "size";
  }
  return null;
}

/**
 * The file a drop, a paste or a picker offered. A drop of several files takes the first
 * picture among them, and the first file of any kind when there is none, so the person is
 * told what was wrong with what they dropped.
 */
export function offeredFile(files: FileList | readonly File[] | null | undefined): File | null {
  if (!files || files.length === 0) {
    return null;
  }
  const list = Array.from(files);
  return list.find((file) => imageRejection(file) !== "type") ?? list[0] ?? null;
}
