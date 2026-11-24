import { cn } from "@trenova/shared/lib/utils";
import { createContext, use, useEffect, useRef, type ReactNode } from "react";
import { drawCrop, outputSize, scaledArea, type Area, type CropSource } from "./image-crop";

/** The longest side a result is drawn at: twice the largest box one is shown in. */
const RESULT_PIXELS = 256;

/** What the crop dialog is holding: the picture, and the part of it inside the frame. */
export type CropDrawing = {
  /** A copy of the picture small enough to redraw on every pointer move. */
  source: CropSource;
  /** How much smaller that copy is than the picture. */
  scale: number;
  /** The part inside the frame, in the picture's own pixels. */
  area: Area;
  quarterTurns: number;
};

const CropDrawingContext = createContext<CropDrawing | null>(null);

export function CropDrawingProvider({
  drawing,
  children,
}: {
  drawing: CropDrawing | null;
  children: ReactNode;
}) {
  return <CropDrawingContext value={drawing}>{children}</CropDrawingContext>;
}

/**
 * The picture as it will be uploaded, redrawn as it is moved. A preview passed to the crop
 * dialog places one wherever the place it stands for draws the picture, with that place's own
 * classes for its box, its corners and how the picture fills them, so the preview is cut the
 * way the real thing will be.
 */
export function CropResult({ className }: { className?: string }) {
  const drawing = use(CropDrawingContext);
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const source = drawing?.source;
  const scale = drawing?.scale;
  const x = drawing?.area.x;
  const y = drawing?.area.y;
  const width = drawing?.area.width;
  const height = drawing?.area.height;
  const quarterTurns = drawing?.quarterTurns;

  useEffect(() => {
    const canvas = canvasRef.current;
    if (
      !canvas ||
      !source ||
      scale === undefined ||
      x === undefined ||
      y === undefined ||
      width === undefined ||
      height === undefined ||
      quarterTurns === undefined
    ) {
      return;
    }
    const area = scaledArea({ x, y, width, height }, scale);
    const size = outputSize(area, RESULT_PIXELS);
    canvas.width = size.width;
    canvas.height = size.height;
    const context = canvas.getContext("2d");
    if (context) {
      drawCrop(context, source, area, quarterTurns, size);
    }
  }, [source, scale, x, y, width, height, quarterTurns]);

  return <canvas ref={canvasRef} aria-hidden className={cn("block", className)} />;
}
