/**
 * The shape a picture is positioned in before it is uploaded: a square, or the picture's own
 * shape, so nothing is cut off a wide mark.
 */
export type ImageFrame = "square" | "source";

/** What a picture has to be for the place it is uploaded to. */
export type ImageSpec = {
  /**
   * How the frame is drawn while the picture is positioned: a circle for a person, a rounded
   * tile for an icon, a plain rectangle for a full mark. The uploaded file is always a
   * rectangle; the place that shows it rounds it.
   */
  mask: "round" | "tile" | "rect";
  frame: ImageFrame;
  /**
   * `cover` keeps the frame filled. `contain` starts with the whole picture inside the frame and
   * lets it be made smaller still, the ground around it left clear.
   */
  fit: "cover" | "contain";
  /** The longest side of the uploaded file, in pixels. A smaller selection is never enlarged. */
  maxDimension: number;
};

/** A person's photo: cut to a circle wherever it is shown. */
export const PROFILE_PICTURE: ImageSpec = {
  mask: "round",
  frame: "square",
  fit: "cover",
  maxDimension: 512,
};
