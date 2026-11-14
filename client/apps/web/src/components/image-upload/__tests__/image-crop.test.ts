import { describe, expect, it } from "vitest";
import {
  cropArea,
  cropLayout,
  frameSize,
  outputSize,
  restingSize,
  stepToZoom,
  withinSource,
  zoomRange,
  zoomToStep,
} from "../image-crop";

// The crop dialog's stage: 488 by 288 pixels, with 20 clear on every side of the frame.
const STAGE = { width: 488, height: 288 };
const WIDE_PHOTO = { width: 4000, height: 2000 };

describe("the frame on the stage", () => {
  it("is the largest square that leaves the inset clear", () => {
    expect(frameSize(1, STAGE)).toEqual({ width: 248, height: 248 });
  });

  it("is as wide as the stage allows for a wide mark", () => {
    expect(frameSize(4, STAGE)).toEqual({ width: 448, height: 112 });
  });

  it("is as tall as the stage allows for a tall one", () => {
    expect(frameSize(0.5, STAGE)).toEqual({ width: 124, height: 248 });
  });
});

describe("a picture at rest on the stage", () => {
  it("is fitted inside the stage", () => {
    expect(restingSize(WIDE_PHOTO, STAGE)).toEqual({ width: 488, height: 244 });
  });

  it("is never enlarged to reach it", () => {
    expect(restingSize({ width: 100, height: 50 }, STAGE)).toEqual({ width: 100, height: 50 });
  });
});

describe("the zooms a picture can take", () => {
  const media = { width: 488, height: 244 };
  const frame = { width: 248, height: 248 };

  it("starts a photo filling its frame and never lets it show a gap", () => {
    const range = zoomRange(media, frame, "cover");

    expect(range.initial).toBeCloseTo(248 / 244);
    expect(range.min).toBe(range.initial);
    expect(range.max).toBeCloseTo((248 / 244) * 5);
  });

  it("starts a mark whole inside its frame and lets it shrink to half that", () => {
    const range = zoomRange(media, frame, "contain");

    expect(range.initial).toBeCloseTo(248 / 488);
    expect(range.min).toBeCloseTo(124 / 488);
    expect(range.max).toBeCloseTo((248 / 244) * 5);
  });

  it("maps the slider's ends to the range's ends and its middle to their geometric mean", () => {
    const range = { min: 0.5, max: 8, initial: 1 };

    expect(stepToZoom(0, range)).toBeCloseTo(0.5);
    expect(stepToZoom(100, range)).toBeCloseTo(8);
    expect(stepToZoom(50, range)).toBeCloseTo(2);
    expect(zoomToStep(2, range)).toBeCloseTo(50);
  });
});

describe("the area inside the frame", () => {
  const filled = cropLayout(WIDE_PHOTO, STAGE, "square", 0, "cover");

  it("is the middle square of a wide photo that fills a square frame", () => {
    const area = cropArea({
      position: { x: 0, y: 0 },
      zoom: filled.range.initial,
      quarterTurns: 0,
      source: WIDE_PHOTO,
      media: filled.media,
      frame: filled.frame,
    });

    expect(area.x).toBeCloseTo(1000);
    expect(area.y).toBeCloseTo(0);
    expect(area.width).toBeCloseTo(2000);
    expect(area.height).toBeCloseTo(2000);
  });

  it("reaches the photo's left edge once the photo is dragged right as far as it goes", () => {
    const zoom = filled.range.initial;
    const area = cropArea({
      position: { x: ((filled.media.width - filled.frame.width / zoom) / 2) * zoom, y: 0 },
      zoom,
      quarterTurns: 0,
      source: WIDE_PHOTO,
      media: filled.media,
      frame: filled.frame,
    });

    expect(area.x).toBeCloseTo(0);
    expect(area.width).toBeCloseTo(2000);
  });

  it("reaches past a wide mark above and below when the mark sits whole in a square", () => {
    const whole = cropLayout(WIDE_PHOTO, STAGE, "square", 0, "contain");
    const area = cropArea({
      position: { x: 0, y: 0 },
      zoom: whole.range.initial,
      quarterTurns: 0,
      source: WIDE_PHOTO,
      media: whole.media,
      frame: whole.frame,
    });

    expect(area.x).toBeCloseTo(0);
    expect(area.y).toBeCloseTo(-1000);
    expect(area.width).toBeCloseTo(4000);
    expect(area.height).toBeCloseTo(4000);
  });

  it("is the whole picture, in its turned shape, after a quarter turn in its own frame", () => {
    const turnedLayout = cropLayout(WIDE_PHOTO, STAGE, "source", 1, "contain");
    const area = cropArea({
      position: { x: 0, y: 0 },
      zoom: turnedLayout.range.initial,
      quarterTurns: 1,
      source: WIDE_PHOTO,
      media: turnedLayout.media,
      frame: turnedLayout.frame,
    });

    expect(turnedLayout.frame).toEqual({ width: 124, height: 248 });
    expect(area.x).toBeCloseTo(0);
    expect(area.y).toBeCloseTo(0);
    expect(area.width).toBeCloseTo(2000);
    expect(area.height).toBeCloseTo(4000);
  });

  it("is pulled back inside the picture when rounding left it overhanging", () => {
    expect(withinSource({ x: -0.3, y: 1999.8, width: 2000.4, height: 0.5 }, WIDE_PHOTO)).toEqual({
      x: 0,
      y: 1999.5,
      width: 2000.4,
      height: 0.5,
    });
    expect(withinSource({ x: 3000, y: 0, width: 4000.2, height: 2000.1 }, WIDE_PHOTO)).toEqual({
      x: 0,
      y: 0,
      width: 4000,
      height: 2000,
    });
  });
});

describe("the size a result is saved at", () => {
  it("is reduced until its longest side fits", () => {
    expect(outputSize({ width: 2000, height: 2000 }, 512)).toEqual({ width: 512, height: 512 });
    expect(outputSize({ width: 4000, height: 1000 }, 1024)).toEqual({ width: 1024, height: 256 });
  });

  it("is never enlarged", () => {
    expect(outputSize({ width: 100, height: 50 }, 512)).toEqual({ width: 100, height: 50 });
  });
});
