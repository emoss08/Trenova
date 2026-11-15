/** The query parameter that carries, to the sign-in page, the address the person was going to. */
export const RETURN_PATH_PARAM = "redirect";

/** The longest return path followed; no address a page links to comes near it. */
export const MAX_RETURN_PATH_LENGTH = 2048;

/** How many layers of percent-encoding are looked through before a path is refused. */
const MAX_DECODE_ROUNDS = 4;

const PARSE_ORIGIN = "https://return-path.invalid";

/** Where an app signs people in, and where it starts once they have. */
export interface SignInPaths {
  readonly loginPath: string;
  readonly home: string;
}

function hasControlCharacter(value: string): boolean {
  for (let index = 0; index < value.length; index += 1) {
    const code = value.charCodeAt(index);
    if (code < 0x20 || (code >= 0x7f && code <= 0x9f)) {
      return true;
    }
  }
  return false;
}

function leavesTheOrigin(path: string): boolean {
  return path.startsWith("//") || path.includes("\\") || hasControlCharacter(path);
}

/**
 * The path and each form it takes as it is decoded, so a separator hidden behind
 * percent-encoding ("/%2F%2Fevil.com") is judged as a router or browser that decodes it would
 * read it. Null when it does not decode, or is still decoding after several rounds.
 */
function decodedForms(path: string): readonly string[] | null {
  const forms = [path];
  let current = path;
  for (let round = 0; round < MAX_DECODE_ROUNDS; round += 1) {
    let next: string;
    try {
      next = decodeURIComponent(current);
    } catch {
      return null;
    }
    if (next === current) {
      return forms;
    }
    forms.push(next);
    current = next;
  }
  return null;
}

function isWithin(pathname: string, base: string): boolean {
  if (base === "/") {
    return true;
  }
  return pathname === base || pathname.startsWith(`${base}/`);
}

/**
 * The address a sign-in may send the person on to, or null when `value` is not one to follow.
 * Only a path on this origin inside the app (under `home`) qualifies: it starts with a single
 * "/", carries no scheme, backslash or control character, hides no "//" behind
 * percent-encoding, and is not the sign-in page itself. The path is returned as it was
 * judged, normalized and without a fragment, so what is followed is what was checked.
 */
export function safeReturnPath(
  value: string | null | undefined,
  paths: SignInPaths,
): string | null {
  if (
    !value ||
    value.length > MAX_RETURN_PATH_LENGTH ||
    !value.startsWith("/") ||
    leavesTheOrigin(value)
  ) {
    return null;
  }

  let url: URL;
  try {
    url = new URL(value, PARSE_ORIGIN);
  } catch {
    return null;
  }
  if (url.origin !== PARSE_ORIGIN) {
    return null;
  }

  const forms = decodedForms(url.pathname);
  if (forms === null || forms.some(leavesTheOrigin)) {
    return null;
  }

  const pathname = (forms.at(-1) ?? url.pathname).toLowerCase();
  if (
    isWithin(pathname, paths.loginPath.toLowerCase()) ||
    !isWithin(pathname, paths.home.toLowerCase())
  ) {
    return null;
  }

  return `${url.pathname}${url.search}`;
}

/**
 * The return path to carry to the sign-in page for the address the person was going to, or
 * null when there is nothing worth carrying: the app's home, which a sign-in reaches anyway,
 * or an address that is not safe to follow.
 */
export function returnPathFor(path: string, paths: SignInPaths): string | null {
  const safe = safeReturnPath(path, paths);
  return safe === null || safe === paths.home ? null : safe;
}

/** Where a finished sign-in goes: the safe return path the page was opened with, else home. */
export function returnPathFrom(search: URLSearchParams, paths: SignInPaths): string {
  return safeReturnPath(search.get(RETURN_PATH_PARAM), paths) ?? paths.home;
}
