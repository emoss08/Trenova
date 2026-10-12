// The REST routes Settings calls: the person's profile and availability, notification
// preferences, two-factor status, sessions, sign-ins and teammates' avatars.
import { TEAM } from "../fixtures/settings.mjs";

const LIMITS = { jobTitle: 100, phoneNumber: 30, awayNote: 280 };

function validation(field, message) {
  return {
    __status: 422,
    body: {
      type: "validation-error",
      title: "Validation Failed",
      status: 422,
      detail: message,
      errors: [{ field, code: "INVALID", message, location: "body" }],
    },
  };
}

function updateProfile(state, _path, body = {}) {
  const user = state.settings.user;
  for (const [field, max] of Object.entries(LIMITS)) {
    if (typeof body[field] === "string" && body[field].length > max) {
      return validation(field, `Must be ${max} characters or fewer`);
    }
  }
  if (
    typeof body.phoneNumber === "string" &&
    body.phoneNumber &&
    !/^[\d\s+().-]{7,30}$/.test(body.phoneNumber)
  ) {
    return validation(
      "phoneNumber",
      "Phone number must contain only digits, spaces and + ( ) - . characters",
    );
  }
  for (const field of ["jobTitle", "phoneNumber", "avatarStyle"]) {
    if (field in body) user[field] = String(body[field]).trim();
  }
  user.version += 1;
  return user;
}

function updateAvailability(state, _path, body = {}) {
  const user = state.settings.user;
  const status = body.status ?? "available";
  user.availabilityStatus = status;
  user.focusUntil = status === "focus" ? (body.focusUntil ?? null) : null;
  user.awayBackOn = status === "away" ? (body.backOn ?? null) : null;
  user.awayDelegateUserId = status === "away" ? (body.delegateUserId ?? null) : null;
  user.awayNote = status === "away" ? String(body.awayNote ?? "").trim() : "";
  user.version += 1;
  return user;
}

function updateSettings(state, _path, body = {}) {
  const user = state.settings.user;
  for (const field of ["timezone", "timeFormat", "locale"]) {
    if (body[field]) user[field] = body[field];
  }
  user.version += 1;
  return user;
}

function changePassword(state, _path, body = {}) {
  if (body.currentPassword !== "correct-horse") {
    return validation("currentPassword", "Invalid password. Please try again");
  }
  state.settings.user.passwordChangedAt = state.now();
  return state.settings.user;
}

function avatars(state, _path, _body, query) {
  const ids = (query?.get("ids") ?? "").split(",").filter(Boolean);
  const people = [state.settings.user, ...TEAM];
  return people
    .filter((person) => ids.includes(person.id))
    .map((person) => ({
      id: person.id,
      name: person.name,
      username: person.username,
      avatarStyle: person.avatarStyle ?? "auto",
      availability: person.availabilityStatus ?? "available",
      awayBackOn: null,
      awayNote: "",
    }));
}

function revokeSession(state, path) {
  const id = path.split("/").filter(Boolean).pop();
  const target = state.settings.sessions.find((session) => session.id === id);
  if (!target)
    return {
      __status: 404,
      body: { type: "not-found", title: "Not found", status: 404, detail: "Session not found" },
    };
  if (target.current)
    return validation("sessionId", "Use Log out to end the session you are using");
  state.settings.sessions = state.settings.sessions.filter((session) => session.id !== id);
  return { __status: 204, body: undefined };
}

function revokeOthers(state) {
  const before = state.settings.sessions.length;
  state.settings.sessions = state.settings.sessions.filter((session) => session.current);
  return { revoked: before - state.settings.sessions.length };
}

export const SETTINGS_ROUTES = [
  ["GET", /^\/api\/v1\/users\/me\/?$/, (state) => state.settings.user],
  ["PATCH", /^\/api\/v1\/users\/me\/profile\/?$/, updateProfile],
  ["PUT", /^\/api\/v1\/users\/me\/availability\/?$/, updateAvailability],
  ["PATCH", /^\/api\/v1\/users\/me\/settings\/?$/, updateSettings],
  ["POST", /^\/api\/v1\/users\/me\/change-password\/?$/, changePassword],
  [
    "GET",
    /^\/api\/v1\/users\/me\/notification-preferences\/?$/,
    (state) => state.settings.preferences,
  ],
  [
    "PUT",
    /^\/api\/v1\/users\/me\/notification-preferences\/?$/,
    (state, _path, body = {}) => {
      state.settings.preferences = { ...state.settings.preferences, ...body };
      return state.settings.preferences;
    },
  ],
  ["GET", /^\/api\/v1\/users\/me\/mfa\/?$/, (state) => state.settings.mfa],
  ["GET", /^\/api\/v1\/users\/me\/sessions\/?$/, (state) => state.settings.sessions],
  ["POST", /^\/api\/v1\/users\/me\/sessions\/revoke-others\/?$/, revokeOthers],
  ["DELETE", /^\/api\/v1\/users\/me\/sessions\/[^/]+\/?$/, revokeSession],
  ["GET", /^\/api\/v1\/users\/me\/sign-ins\/?$/, (state) => state.settings.signIns],
  ["GET", /^\/api\/v1\/users\/avatars\/?$/, avatars],
];

export function userSelectOptions(state, input = {}) {
  const query = String(input.query ?? "").toLowerCase();
  const people = TEAM.filter(
    (person) =>
      !query || person.name.toLowerCase().includes(query) || person.username.includes(query),
  ).filter((person) => !input.ids?.length || input.ids.includes(person.id));
  return {
    selectOptions: {
      totalCount: people.length,
      pageInfo: { hasNextPage: false, endCursor: null },
      edges: people.map((person) => ({
        node: { id: person.id, label: person.name, description: person.jobTitle, meta: null },
      })),
    },
  };
}
