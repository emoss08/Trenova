/**
 * The Safety tab's entry, read lazily by the page with `SafetySkeleton` as its fallback. Safety
 * is one view, so its code is this module's own chunk: no second lazy step after the first.
 */
export { default } from "./worker-safety-tab";
