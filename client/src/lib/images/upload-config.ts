export const IMAGE_UPLOAD_ACCEPT = ".jpg,.jpeg,.png,.webp";

export const profilePictureCropConfig = {
  aspect: 1,
  targetWidth: 512,
  targetHeight: 512,
} as const;

export const organizationLogoCropConfig = {
  maxDimension: 1024,
} as const;
