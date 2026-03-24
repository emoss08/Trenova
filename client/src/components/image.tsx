import {
  LazyLoadImage,
  type LazyLoadImageProps,
} from "react-lazy-load-image-component";
import "react-lazy-load-image-component/src/effects/blur.css";

export function LazyImage({ ...props }: LazyLoadImageProps) {
  return <LazyLoadImage {...props} effect="blur" />;
}
