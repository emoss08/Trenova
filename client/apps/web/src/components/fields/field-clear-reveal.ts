/**
 * When a field's clear control shows: while the pointer is over the field or the focus is
 * inside it, and always on a touch screen, which has no hover. It keeps its space while
 * hidden, so nothing in the field moves when it appears, and takes no clicks until it is
 * seen. A form of optional fields would otherwise carry a column of crosses nobody is
 * reaching for.
 *
 * The field is whatever wears `group/field`: `FieldWrapper` does, and so does a control
 * that can be used without one.
 */
export const fieldClearRevealClass =
  "pointer-events-none opacity-0 transition-opacity duration-150 group-hover/field:pointer-events-auto group-hover/field:opacity-100 group-focus-within/field:pointer-events-auto group-focus-within/field:opacity-100 pointer-coarse:pointer-events-auto pointer-coarse:opacity-100";

/**
 * The same, for a clear control that shares its place with the field's unit ("%",
 * "minutes"). The two are never shown together: the unit gives way while the field is
 * hovered or focused. On a touch screen that is while the field is focused, so the unit is
 * not hidden for good.
 */
export const fieldClearOverUnitClass =
  "pointer-events-none opacity-0 transition-opacity duration-150 group-hover/field:pointer-events-auto group-hover/field:opacity-100 group-focus-within/field:pointer-events-auto group-focus-within/field:opacity-100";

/** A unit that gives its place to the clear control while the field is hovered or focused. */
export const fieldUnitYieldClass =
  "transition-opacity duration-150 group-hover/field:opacity-0 group-focus-within/field:opacity-0";

/**
 * The one place a unit and the clear control share. It is as wide as the wider of the two
 * and both sit at its right edge, so the clear control is always the last thing in the
 * field, never in the middle of where a long unit was, and never left of the unit.
 */
export const fieldEndSlotClass = "grid items-center justify-items-end *:col-start-1 *:row-start-1";
