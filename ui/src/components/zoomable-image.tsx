/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import type { DetailedHTMLProps, ImgHTMLAttributes } from "react";
import { Dialog, DialogContent, DialogTrigger } from "./ui/dialog";
import { LazyImage } from "./ui/image";
import { ScrollArea } from "./ui/scroll-area";

export function ZoomableImage({
  src,
  width,
  height,
  alt,
  className,
}: DetailedHTMLProps<ImgHTMLAttributes<HTMLImageElement>, HTMLImageElement>) {
  if (!src) return null;

  return (
    <Dialog>
      <DialogTrigger asChild>
        <LazyImage
          src={src}
          alt={alt || ""}
          className={className}
          style={{
            width: "100%",
            height: "auto",
          }}
          width={width}
          height={height}
        />
      </DialogTrigger>
      <DialogContent className="max-w-7xl border-0 bg-transparent p-0">
        <ScrollArea className="flex relative h-[calc(100vh-100px)] w-full overflow-clip rounded-md bg-transparent shadow-md">
          <LazyImage
            src={src}
            alt={alt || ""}
            className="size-full object-contain"
          />
        </ScrollArea>
      </DialogContent>
    </Dialog>
  );
}
