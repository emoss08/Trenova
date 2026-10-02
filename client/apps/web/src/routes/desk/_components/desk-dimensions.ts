/**
 * The Desk's fixed measures, shared by the room and by the screen that
 * stands in for it while it loads, so the skeleton has the room's shape.
 */

/** The rail's width on a wide screen, in pixels; the fold animates to and from it. */
export const DESK_RAIL_WIDTH = 264;

/** The rail folded to a strip of its places, in pixels. */
export const DESK_RAIL_STRIP_WIDTH = 52;

/**
 * The workspace's width on a wide screen: never narrower than a readable
 * table, a little under half the room, and capped so a wide window gives
 * the rest to the conversation.
 */
export const DESK_WORKSPACE_WIDTH = "clamp(26rem, 42%, 44rem)";

/** The viewport from which the rail stands beside the room rather than over it. */
export const DESK_WIDE_QUERY = "(min-width: 1024px)";
