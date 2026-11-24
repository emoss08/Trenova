import type { ImageFrame, ImageSpec } from "./image-spec";

export type Size = { width: number; height: number };
export type Point = { x: number; y: number };
export type Area = Point & Size;

/** The clear margin between the frame and the edge of the stage, in pixels. */
const STAGE_INSET = 20;

/** How far past filling the frame a picture can be enlarged. */
const MAX_ZOOM_FACTOR = 5;

/** How far below sitting whole in the frame a contained picture can be reduced. */
const MIN_CONTAIN_FACTOR = 0.5;

const OUTPUT_TYPE = "image/webp";
const OUTPUT_QUALITY = 0.9;

/** A size after a number of quarter turns: an odd number swaps its sides. */
export function turned(size: Size, quarterTurns: number): Size {
  return quarterTurns % 2 === 0 ? size : { width: size.height, height: size.width };
}

/** Width over height of a frame, for a picture turned as it is. */
export function frameAspect(frame: ImageFrame, source: Size, quarterTurns: number): number {
  if (frame === "square") {
    return 1;
  }
  const { width, height } = turned(source, quarterTurns);
  return width / height;
}

/** The largest box of an aspect that fits the stage with the inset clear on every side. */
export function frameSize(aspect: number, stage: Size): Size {
  const maxWidth = Math.max(1, stage.width - STAGE_INSET * 2);
  const maxHeight = Math.max(1, stage.height - STAGE_INSET * 2);
  return maxWidth > maxHeight * aspect
    ? { width: maxHeight * aspect, height: maxHeight }
    : { width: maxWidth, height: maxWidth / aspect };
}

/**
 * The picture's size on the stage at zoom 1: fitted inside it and never enlarged, which is
 * how the cropper lays a picture out before any zoom.
 */
export function restingSize(source: Size, stage: Size): Size {
  const scale = Math.min(1, stage.width / source.width, stage.height / source.height);
  return { width: source.width * scale, height: source.height * scale };
}

export type ZoomRange = { min: number; max: number; initial: number };

/** The zooms a picture can take in a frame. `media` is its resting size, turned as it is. */
export function zoomRange(media: Size, frame: Size, fit: ImageSpec["fit"]): ZoomRange {
  const fill = Math.max(frame.width / media.width, frame.height / media.height);
  const whole = Math.min(frame.width / media.width, frame.height / media.height);
  const max = fill * MAX_ZOOM_FACTOR;
  return fit === "cover"
    ? { min: fill, initial: fill, max }
    : { min: whole * MIN_CONTAIN_FACTOR, initial: whole, max };
}

export type CropLayout = {
  /** The frame's box on the stage. */
  frame: Size;
  /** The picture's resting size on the stage, before any turn. */
  media: Size;
  range: ZoomRange;
};

/** Where a picture, its frame and its zooms sit on a stage of a given size. */
export function cropLayout(
  source: Size,
  stage: Size,
  frame: ImageFrame,
  quarterTurns: number,
  fit: ImageSpec["fit"],
): CropLayout {
  const box = frameSize(frameAspect(frame, source, quarterTurns), stage);
  const media = restingSize(source, stage);
  return { frame: box, media, range: zoomRange(turned(media, quarterTurns), box, fit) };
}

/** Where a zoom sits in its range, from 0 to 100, on a scale where each step multiplies. */
export function zoomToStep(zoom: number, range: ZoomRange): number {
  return (Math.log(zoom / range.min) / Math.log(range.max / range.min)) * 100;
}

export function stepToZoom(step: number, range: ZoomRange): number {
  return range.min * (range.max / range.min) ** (step / 100);
}

export type CropView = {
  /** The picture's offset from the centre of the stage, in stage pixels. */
  position: Point;
  zoom: number;
  quarterTurns: number;
  /** The picture's own size in pixels, before any turn. */
  source: Size;
  /** The picture's resting size on the stage, before any turn. */
  media: Size;
  frame: Size;
};

/**
 * The part of the turned picture that sits inside the frame, in the picture's own pixels,
 * measured from the turned picture's top left corner. It reaches past the picture wherever
 * the frame does.
 */
export function cropArea({ position, zoom, quarterTurns, source, media, frame }: CropView): Area {
  const box = turned(media, quarterTurns);
  const scale = turned(source, quarterTurns).width / box.width;
  const width = frame.width / zoom;
  const height = frame.height / zoom;
  return {
    x: ((box.width - width) / 2 - position.x / zoom) * scale,
    y: ((box.height - height) / 2 - position.y / zoom) * scale,
    width: width * scale,
    height: height * scale,
  };
}

/**
 * An area pulled back inside a picture. A frame that must stay filled can still overhang by
 * a fraction of a pixel after rounding, which would leave a clear hairline on that edge.
 */
export function withinSource(area: Area, source: Size): Area {
  const width = Math.min(area.width, source.width);
  const height = Math.min(area.height, source.height);
  return {
    x: Math.min(Math.max(area.x, 0), source.width - width),
    y: Math.min(Math.max(area.y, 0), source.height - height),
    width,
    height,
  };
}

/** The pixel size an area is saved at: its own, reduced until its longest side fits. */
export function outputSize(area: Size, maxDimension: number): Size {
  const scale = Math.min(1, maxDimension / Math.max(area.width, area.height));
  return {
    width: Math.max(1, Math.round(area.width * scale)),
    height: Math.max(1, Math.round(area.height * scale)),
  };
}

export function scaledArea(area: Area, scale: number): Area {
  return {
    x: area.x * scale,
    y: area.y * scale,
    width: area.width * scale,
    height: area.height * scale,
  };
}

/** A picture to draw from, with the size it is drawn at. */
export type CropSource = { image: CanvasImageSource; size: Size };

/**
 * Draws an area of a turned picture across a canvas of the output size. What the area holds
 * beyond the picture stays clear.
 */
export function drawCrop(
  context: CanvasRenderingContext2D,
  source: CropSource,
  area: Area,
  quarterTurns: number,
  output: Size,
): void {
  const box = turned(source.size, quarterTurns);
  context.save();
  context.clearRect(0, 0, output.width, output.height);
  context.imageSmoothingEnabled = true;
  context.imageSmoothingQuality = "high";
  context.scale(output.width / area.width, output.height / area.height);
  context.translate(box.width / 2 - area.x, box.height / 2 - area.y);
  context.rotate((quarterTurns * Math.PI) / 2);
  context.drawImage(
    source.image,
    -source.size.width / 2,
    -source.size.height / 2,
    source.size.width,
    source.size.height,
  );
  context.restore();
}

/**
 * A copy of a picture no longer than `maxDimension` on its longest side, with the scale it
 * was reduced by. Previews redraw on every pointer move, and drawing each from a
 * twelve-megapixel photo makes the drag stutter.
 */
export function reducedSource(
  image: HTMLImageElement,
  size: Size,
  maxDimension: number,
): { source: CropSource; scale: number } {
  const reduced = outputSize(size, maxDimension);
  const scale = reduced.width / size.width;
  if (scale >= 1) {
    return { source: { image, size }, scale: 1 };
  }
  const canvas = window.document.createElement("canvas");
  canvas.width = reduced.width;
  canvas.height = reduced.height;
  const context = canvas.getContext("2d");
  if (!context) {
    return { source: { image, size }, scale: 1 };
  }
  context.imageSmoothingQuality = "high";
  context.drawImage(image, 0, 0, reduced.width, reduced.height);
  return { source: { image: canvas, size: reduced }, scale };
}

function baseName(name: string): string {
  const lastDot = name.lastIndexOf(".");
  return lastDot > 0 ? name.slice(0, lastDot) : name;
}

function encode(canvas: HTMLCanvasElement): Promise<Blob> {
  return new Promise((resolve, reject) => {
    canvas.toBlob(
      (blob) => (blob ? resolve(blob) : reject(new Error("canvas produced no image"))),
      OUTPUT_TYPE,
      OUTPUT_QUALITY,
    );
  });
}

/**
 * The file that is uploaded: the area drawn at its output size. A browser that cannot write
 * WebP writes a PNG instead, and the file is named and typed for what it holds.
 */
export async function croppedFile(
  source: CropSource,
  area: Area,
  quarterTurns: number,
  maxDimension: number,
  name: string,
): Promise<File> {
  const output = outputSize(area, maxDimension);
  const canvas = window.document.createElement("canvas");
  canvas.width = output.width;
  canvas.height = output.height;
  const context = canvas.getContext("2d");
  if (!context) {
    throw new Error("canvas has no 2d context");
  }
  drawCrop(context, source, area, quarterTurns, output);
  const blob = await encode(canvas);
  const type = blob.type || OUTPUT_TYPE;
  const extension = type === OUTPUT_TYPE ? "webp" : "png";
  return new File([blob], `${baseName(name) || "image"}.${extension}`, {
    type,
    lastModified: Date.now(),
  });
}
