import {
    Image as UnpicImage,
    ImageProps as UnpicImageProps,
} from "@unpic/react";

export function LazyImage({ ...props }: UnpicImageProps) {
  return <UnpicImage {...props} />;
}
