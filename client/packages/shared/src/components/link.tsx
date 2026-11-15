import { Link as RouterLink, type LinkComponent } from "@tanstack/react-router";

const EXACT = { exact: true } as const;
const NO_ACTIVE_PROPS = {} as const;

/**
 * The router's link, with the app's rule for where the person is: a link counts as current only
 * on the exact address it opens (its path and its query), never on the pages under it, and it
 * adds no class of its own. The router marks a current link `aria-current="page"` and
 * `data-status="active"`; left to its default it would mark "/desk" current on every Desk page,
 * lighting Today beside the place actually open. A list that decides its current item itself
 * passes `aria-current`, and a navigation that wants the pages under a link too passes its own
 * `activeOptions`.
 */
export const Link: LinkComponent<"a"> = (props) => (
  <RouterLink activeOptions={EXACT} activeProps={NO_ACTIVE_PROPS} {...props} />
);
