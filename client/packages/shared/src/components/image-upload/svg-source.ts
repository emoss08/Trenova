import { SVG_TYPE } from "./image-file";
import type { Size } from "./image-crop";

/**
 * The longest side a vector picture is drawn at before it is positioned: enough that the
 * uploaded file, at most half this, is cut from more pixels than it keeps.
 */
const RASTER_PIXELS = 2048;

function positive(value: number): boolean {
  return Number.isFinite(value) && value > 0;
}

function viewBoxSize(root: Element): Size | null {
  const parts = (root.getAttribute("viewBox") ?? "").trim().split(/[\s,]+/).map(Number);
  const [, , width, height] = parts;
  return parts.length === 4 && width !== undefined && height !== undefined && positive(width) && positive(height)
    ? { width, height }
    : null;
}

function pixels(value: string | null): number | null {
  const match = /^\s*(\d*\.?\d+)(px)?\s*$/.exec(value ?? "");
  const size = match?.[1] ? Number(match[1]) : Number.NaN;
  return positive(size) ? size : null;
}

function attributeSize(root: Element): Size | null {
  const width = pixels(root.getAttribute("width"));
  const height = pixels(root.getAttribute("height"));
  return width && height ? { width, height } : null;
}

/**
 * A vector picture given the pixel size it is drawn at. An SVG states its size loosely or not
 * at all: one with only a view box has no size of its own, which some browsers refuse to draw
 * onto a canvas, and one sized for a 24 pixel icon would be positioned and uploaded as 24
 * blurred pixels. Stating a large size on the root, in the picture's own shape, settles both.
 * A file with neither a view box nor a size is left as it is.
 */
export async function sizedSvg(file: File): Promise<File> {
  const parsed = new DOMParser().parseFromString(await file.text(), SVG_TYPE);
  const root = parsed.documentElement;
  if (root.localName !== "svg" || parsed.querySelector("parsererror")) {
    throw new Error("not an svg document");
  }

  const size = viewBoxSize(root) ?? attributeSize(root);
  if (!size) {
    return file;
  }

  const scale = RASTER_PIXELS / Math.max(size.width, size.height);
  if (!root.hasAttribute("viewBox")) {
    root.setAttribute("viewBox", `0 0 ${size.width} ${size.height}`); // i18n-ignore: an attribute value
  }
  root.setAttribute("width", String(Math.max(1, Math.round(size.width * scale))));
  root.setAttribute("height", String(Math.max(1, Math.round(size.height * scale))));

  return new File([new XMLSerializer().serializeToString(parsed)], file.name, {
    type: SVG_TYPE,
    lastModified: file.lastModified,
  });
}
