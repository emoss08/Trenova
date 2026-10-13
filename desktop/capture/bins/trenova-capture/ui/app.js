(() => {
  "use strict";

  const shell = document.getElementById("shell");
  const rail = document.getElementById("rail");
  const app = document.getElementById("app");
  const viewerHost = document.getElementById("viewer");
  const toasts = document.getElementById("toasts");
  const logoSource = document.getElementById("logo");

  // The address this build suggests, filled in when the page is built and
  // empty in public builds: every Trenova is somebody's own, so a person
  // enters theirs. The example only shows the shape of one. Trenova Cloud,
  // the hosted service, is offered beside a self-hosted address so nobody
  // has to know an address to use it; a build may offer none. The web app's
  // dev server is offered when Trenova runs on this computer; it passes
  // /api on to the API.
  const SUGGESTED_ADDRESS = "/*SUGGESTED_SERVER*/";
  const CLOUD_ADDRESS = "/*CLOUD_SERVER*/";
  const EXAMPLE_ADDRESS = "https://trenova.example.com";
  const DEVELOPMENT_ADDRESS = "http://localhost:5173";
  // The namespace SVG elements are created in; nothing is fetched from it.
  const SVG_NAMESPACE = "http://www.w3.org/2000/svg";
  const TOAST_MS = 6000;
  const RECENT_ON_HOME = 3;
  // Pages shown of a held batch before "Show all", and in a strip.
  const HELD_PAGES_SHOWN = 18;
  const STRIP_PAGES = 5;
  const LIVE_PAGES = 8;
  // The most pictures asked for in one message, as the agent allows.
  const PICTURES_PER_ASK = 24;
  // How long before a picture asked for and not received is asked again.
  const ASK_AGAIN_MS = 20000;
  // Full-size pictures kept for going back and forth in the viewer; each is
  // a few hundred kilobytes, so older ones are let go and asked for again.
  const VIEWS_KEPT = 12;
  // The shape of the batch being scanned, which only grows.
  const LIVE_SHAPE = "live";
  // What this window remembers between runs, in its own storage: the
  // appearance, and the scanner and settings last chosen.
  const REMEMBERED = "trenova-capture";

  function remembered() {
    try {
      const raw = window.localStorage.getItem(REMEMBERED);
      const parsed = raw ? JSON.parse(raw) : {};
      return parsed && typeof parsed === "object" ? parsed : {};
    } catch {
      return {};
    }
  }

  function remember(changes) {
    try {
      window.localStorage.setItem(REMEMBERED, JSON.stringify({ ...remembered(), ...changes }));
    } catch {
      // Storage may be off; the choice holds for this run.
    }
  }

  const kept = remembered();

  // What belongs to this window alone and survives a redraw: the tab shown,
  // choices made, text being typed, a question being asked, and buttons
  // already pressed.
  const local = {
    tab: "home",
    scanner: typeof kept.scanner === "string" ? kept.scanner : null,
    profile: typeof kept.profile === "string" ? kept.profile : null,
    theme: ["system", "light", "dark"].includes(kept.theme) ? kept.theme : "system",
    serverDraft: null,
    // The first run's self-hosted form, or Settings changing the address.
    selfHosted: false,
    changingServer: false,
    confirmDiscard: null,
    copied: false,
    copiedDiagnostics: false,
    pending: new Map(),
    // The newest message already seen, so only later ones become toasts.
    lastMessage: null,
    // The page open full size: { key, page, label, pages, editable }.
    viewer: null,
    // Held batches showing all their pages, and the one being discarded.
    expanded: new Set(),
    confirmHeld: null,
    confirmSignOut: false,
  };

  function applyTheme() {
    if (local.theme === "system") {
      document.documentElement.removeAttribute("data-theme");
    } else {
      document.documentElement.setAttribute("data-theme", local.theme);
    }
  }
  applyTheme();

  // Pictures the agent sent, by batch, size and page, and when each was
  // last asked for. A batch whose pages change (one taken out) is dropped,
  // and so is one that has gone. Each picture keeps one <img>, moved from
  // one drawing to the next, so a redraw never decodes a picture again.
  const pictures = new Map();
  const images = new Map();
  // Pictures placed in the drawing under way; one shown twice gets a copy.
  const placed = new Set();
  // A held page's tile, kept while nothing it shows changes, so redrawing a
  // long batch moves the tiles already made instead of making them again.
  // Tiles not drawn are let go.
  const tiles = new Map();
  const tilesDrawn = new Set();
  const asked = new Map();
  let shapes = new Map();

  function pictureId(key, size, page) {
    return `${key}|${size}|${page}`;
  }

  function forgetBatch(key) {
    for (const store of [pictures, images, asked, tiles]) {
      for (const id of [...store.keys()]) {
        if (id.startsWith(`${key}|`)) {
          store.delete(id);
        }
      }
    }
  }

  function trackShapes(v) {
    const next = new Map();
    for (const batch of [...v.held, ...v.waiting, ...v.refused]) {
      next.set(batch.key, batch.pictures.map((p) => p.page).join(","));
    }
    if (v.scan) {
      next.set(v.scan.key, LIVE_SHAPE);
    }
    for (const [key, shape] of shapes) {
      const now = next.get(key);
      const grew = shape === LIVE_SHAPE && now !== undefined;
      if (now !== shape && !grew) {
        forgetBatch(key);
      }
    }
    shapes = next;
  }

  function keep(id, src) {
    pictures.delete(id);
    images.delete(id);
    pictures.set(id, src);
    if (!id.includes("|view|")) {
      return;
    }
    const views = [...pictures.keys()].filter((kept) => kept.includes("|view|"));
    for (const old of views.slice(0, Math.max(0, views.length - VIEWS_KEPT))) {
      pictures.delete(old);
      images.delete(old);
      asked.delete(old);
    }
  }

  // Asks the agent for the pictures of `pages` not already here or asked
  // for lately.
  function want(key, size, pages) {
    const now = Date.now();
    const missing = pages.filter((page) => {
      const id = pictureId(key, size, page);
      return !pictures.has(id) && now - (asked.get(id) || 0) > ASK_AGAIN_MS;
    });
    for (let at = 0; at < missing.length; at += PICTURES_PER_ASK) {
      const chunk = missing.slice(at, at + PICTURES_PER_ASK);
      for (const page of chunk) {
        asked.set(pictureId(key, size, page), now);
      }
      send({ type: "pictures", key, size, pages: chunk });
    }
  }

  function pictureOf(key, size, page) {
    return pictures.get(pictureId(key, size, page)) || null;
  }

  // The one <img> of a picture, made the first time it is shown.
  function imageOf(key, size, page, label) {
    const id = pictureId(key, size, page);
    const src = pictures.get(id);
    if (!src) {
      return null;
    }
    let image = images.get(id);
    if (!image) {
      image = el("img", { src, alt: label, draggable: "false", decoding: "async" });
      images.set(id, image);
    } else if (image.alt !== label) {
      image.alt = label;
    }
    if (placed.has(id)) {
      return image.cloneNode(false);
    }
    placed.add(id);
    return image;
  }
  let view = null;

  // Untitled UI's line icons, drawn on a 24-pixel grid with a round stroke,
  // as the web app draws them.
  const ICONS = {
    scan: "M3 7V5a2 2 0 0 1 2-2h2 M17 3h2a2 2 0 0 1 2 2v2 M21 17v2a2 2 0 0 1-2 2h-2 M7 21H5a2 2 0 0 1-2-2v-2 M7 12h10",
    print:
      "M6 9V2h12v7 M6 18H4a2 2 0 0 1-2-2v-5a2 2 0 0 1 2-2h16a2 2 0 0 1 2 2v5a2 2 0 0 1-2 2h-2 M6 14h12v8H6z",
    home: "M3 10.5 12 3l9 7.5 M5 9.5V21h14V9.5 M10 21v-6h4v6",
    activity: "M22 12h-4l-3 9L9 3l-3 9H2",
    settings:
      "M12 15a3 3 0 1 0 0-6 3 3 0 0 0 0 6Z M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 1 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 1 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 1 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 1 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1Z",
    check: "M20 6 9 17l-5-5",
    checkCircle: "M22 11.08V12a10 10 0 1 1-5.93-9.14 M22 4 12 14.01l-3-3",
    alert:
      "M10.29 3.86 1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0Z M12 9v4 M12 17h.01",
    xCircle: "M12 22a10 10 0 1 0 0-20 10 10 0 0 0 0 20Z M15 9l-6 6 M9 9l6 6",
    cloud: "M17.5 19a4.5 4.5 0 0 0 .5-8.97A7 7 0 0 0 4.5 12 4 4 0 0 0 6 19.9h11.5",
    cloudOff:
      "M2 2l20 20 M5.78 5.78A7 7 0 0 0 9 19h8.5a4.5 4.5 0 0 0 1.31-.2 M21.53 16.5A4.5 4.5 0 0 0 17.5 10h-1.79A7 7 0 0 0 9.66 5.3",
    server:
      "M4 4h16v6H4z M4 14h16v6H4z M8 7h.01 M8 17h.01",
    clock: "M12 22a10 10 0 1 0 0-20 10 10 0 0 0 0 20Z M12 6v6l4 2",
    inbox:
      "M22 12h-6l-2 3h-4l-2-3H2 M5.45 5.11 2 12v6a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2v-6l-3.45-6.89A2 2 0 0 0 16.76 4H7.24a2 2 0 0 0-1.79 1.11Z",
    bell: "M18 8a6 6 0 0 0-12 0c0 7-3 9-3 9h18s-3-2-3-9 M13.73 21a2 2 0 0 1-3.46 0",
    refresh: "M21 12a9 9 0 1 1-2.64-6.36L21 8 M21 3v5h-5",
    folder:
      "M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z",
    external: "M15 3h6v6 M10 14 21 3 M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6",
    download: "M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4 M7 10l5 5 5-5 M12 15V3",
    power: "M18.36 6.64a9 9 0 1 1-12.73 0 M12 2v10",
    globe:
      "M12 22a10 10 0 1 0 0-20 10 10 0 0 0 0 20Z M2 12h20 M12 2a15 15 0 0 1 0 20 M12 2a15 15 0 0 0 0 20",
    pause: "M10 4H6v16h4z M18 4h-4v16h4z",
    copy: "M20 9h-9a2 2 0 0 0-2 2v9a2 2 0 0 0 2 2h9a2 2 0 0 0 2-2v-9a2 2 0 0 0-2-2Z M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1",
    lock: "M5 11h14v10H5z M8 11V7a4 4 0 0 1 8 0v4",
    logIn: "M15 3h4a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2h-4 M10 17l5-5-5-5 M15 12H3",
    close: "M18 6 6 18 M6 6l12 12",
    stop: "M7 7h10v10H7z",
    tray: "M4 14h16v6H4z M8 10l4-4 4 4 M12 6v8",
    rotateLeft: "M3 12a9 9 0 1 0 2.64-6.36L3 8 M3 3v5h5",
    rotateRight: "M21 12a9 9 0 1 1-2.64-6.36L21 8 M21 3v5h-5",
    trash: "M3 6h18 M8 6V4h8v2 M19 6l-1 14H6L5 6 M10 11v6 M14 11v6",
    send: "M22 2 11 13 M22 2l-7 20-4-9-9-4 20-7z",
    chevronLeft: "M15 18l-6-6 6-6",
    chevronRight: "M9 18l6-6-6-6",
    arrowRight: "M5 12h14 M13 6l6 6-6 6",
    review: "M2 12s3.5-7 10-7 10 7 10 7-3.5 7-10 7S2 12 2 12Z M12 15a3 3 0 1 0 0-6 3 3 0 0 0 0 6Z",
    monitor: "M4 4h16a1 1 0 0 1 1 1v10a1 1 0 0 1-1 1H4a1 1 0 0 1-1-1V5a1 1 0 0 1 1-1Z M8 20h8 M12 16v4",
    sun: "M12 17a5 5 0 1 0 0-10 5 5 0 0 0 0 10Z M12 1v2 M12 21v2 M4.22 4.22l1.42 1.42 M18.36 18.36l1.42 1.42 M1 12h2 M21 12h2 M4.22 19.78l1.42-1.42 M18.36 5.64l1.42-1.42",
    moon: "M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79Z",
    users: "M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2 M9 11a4 4 0 1 0 0-8 4 4 0 0 0 0 8Z M23 21v-2a4 4 0 0 0-3-3.87 M16 3.13a4 4 0 0 1 0 7.75",
    lifeBuoy:
      "M12 22a10 10 0 1 0 0-20 10 10 0 0 0 0 20Z M12 16a4 4 0 1 0 0-8 4 4 0 0 0 0 8Z M4.93 4.93l4.24 4.24 M14.83 14.83l4.24 4.24 M14.83 9.17l4.24-4.24 M4.93 19.07l4.24-4.24",
  };

  function send(message) {
    if (window.ipc && typeof window.ipc.postMessage === "function") {
      window.ipc.postMessage(JSON.stringify(message));
    }
  }

  // Every string from the agent goes in as text, never as markup.
  function el(tag, props, ...children) {
    const node = document.createElement(tag);
    if (props) {
      for (const [name, value] of Object.entries(props)) {
        if (value === undefined || value === null || value === false) {
          continue;
        }
        if (name === "class") {
          node.className = value;
        } else if (name === "text") {
          node.textContent = value;
        } else if (name.startsWith("on")) {
          node.addEventListener(name.slice(2), value);
        } else if (value === true) {
          node.setAttribute(name, "");
        } else {
          node.setAttribute(name, String(value));
        }
      }
    }
    for (const child of children.flat()) {
      if (child === null || child === undefined || child === false) {
        continue;
      }
      node.append(typeof child === "string" ? document.createTextNode(child) : child);
    }
    return node;
  }

  function icon(name, extra) {
    const svg = document.createElementNS(SVG_NAMESPACE, "svg");
    svg.setAttribute("viewBox", "0 0 24 24");
    svg.setAttribute("class", extra ? `icon ${extra}` : "icon");
    svg.setAttribute("aria-hidden", "true");
    const path = document.createElementNS(SVG_NAMESPACE, "path");
    path.setAttribute("d", ICONS[name]);
    svg.append(path);
    return svg;
  }

  function logo(size) {
    const src = logoSource ? logoSource.getAttribute("src") : "";
    return src ? el("img", { src, alt: "", width: size, height: size }) : null;
  }

  function pages(count) {
    return count === 1 ? "1 page" : `${count} pages`;
  }

  const relative = new Intl.RelativeTimeFormat(undefined, { numeric: "auto" });
  const dateTime = new Intl.DateTimeFormat(undefined, { dateStyle: "medium", timeStyle: "short" });

  function ago(millis) {
    if (!millis) {
      return "";
    }
    const seconds = (millis - Date.now()) / 1000;
    const abs = Math.abs(seconds);
    if (abs < 45) {
      return "just now";
    }
    if (abs < 45 * 60) {
      return relative.format(Math.round(seconds / 60), "minute");
    }
    if (abs < 22 * 3600) {
      return relative.format(Math.round(seconds / 3600), "hour");
    }
    if (abs < 7 * 86400) {
      return relative.format(Math.round(seconds / 86400), "day");
    }
    return dateTime.format(new Date(millis));
  }

  function when(millis) {
    return el("time", {
      class: "subtle numeric",
      datetime: millis ? new Date(millis).toISOString() : undefined,
      title: millis ? dateTime.format(new Date(millis)) : undefined,
      text: ago(millis),
    });
  }

  // A button pressed stays pressed until the view shows its effect, or for
  // fifteen seconds when nothing visible changes.
  function press(id, stillWaiting, message) {
    local.pending.set(id, { at: Date.now(), stillWaiting });
    send(message);
    draw();
  }

  function pressed(id) {
    return local.pending.has(id);
  }

  function prunePending() {
    const now = Date.now();
    for (const [id, entry] of local.pending) {
      if (now - entry.at > 15000 || !entry.stillWaiting(view)) {
        local.pending.delete(id);
      }
    }
  }

  function button(label, props, iconName, trailingIcon) {
    return el(
      "button",
      { type: "button", ...props },
      iconName ? icon(iconName) : null,
      label,
      trailingIcon ? icon(trailingIcon) : null,
    );
  }

  function panel(title, options, ...body) {
    const { count, aside, iconName, hint, extra } = options || {};
    return el(
      "section",
      { class: extra ? `panel ${extra}` : "panel", "aria-label": title },
      el(
        "div",
        { class: "panel-head" },
        iconName ? icon(iconName) : null,
        el("h2", { text: title }),
        count ? el("span", { class: "count", text: String(count) }) : null,
        hint ? el("span", { class: "hint", text: hint }) : null,
        aside ? el("div", { class: "aside" }, aside) : null,
      ),
      ...body,
    );
  }

  function kindIcon(kind, tone) {
    return el(
      "span",
      { class: tone ? `kind tone-${tone}` : "kind", title: kind === "print" ? "Printed" : "Scanned" },
      icon(kind === "print" ? "print" : "scan"),
    );
  }

  function goTo(tab) {
    local.tab = tab;
    local.confirmDiscard = null;
    local.confirmSignOut = false;
    local.changingServer = false;
    draw();
    const body = app.querySelector(".page-body");
    if (body) {
      body.scrollTo(0, 0);
    }
  }

  function signedInStage(v) {
    return ["connecting", "online", "offline", "blocked"].includes(v.stage);
  }

  function initials(name) {
    const parts = (name || "").trim().split(/\s+/).filter(Boolean);
    const letters = parts.length > 1 ? parts[0][0] + parts[parts.length - 1][0] : (parts[0] || "?").slice(0, 2);
    return letters.toUpperCase();
  }

  function serverTitle(account) {
    return account.serverName || "Trenova";
  }

  // The rail: the brand, where to go, and who is signed in where.
  const CONNECTION = {
    needsServer: ["neutral", "Not connected"],
    checking: ["neutral", "Checking"],
    signedOut: ["neutral", "Signed out"],
    pairing: ["info", "Signing in"],
    connecting: ["neutral", "Connecting"],
    online: ["success", "Online"],
    offline: ["warning", "Offline"],
    blocked: ["danger", "Paused"],
  };

  function connectionPill(v) {
    const [tone, label] = CONNECTION[v.stage] || CONNECTION.connecting;
    return el(
      "span",
      { class: `pill tone-${tone}`, role: "status", title: v.status.detail || v.status.title },
      el("span", { class: "dot", "aria-hidden": "true" }),
      el("span", { class: "label", text: label }),
    );
  }

  function activityCount(v) {
    return v.refused.length + v.held.length + v.waiting.length;
  }

  function drawRail(v) {
    const account = v.account;
    const count = activityCount(v);
    const items = [
      ["home", "Home", "home", null],
      ["activity", "Activity", "activity", count],
      ["settings", "Settings", "settings", null],
    ];
    rail.replaceChildren(
      el(
        "div",
        { class: "brand" },
        logo(22),
        el("div", { class: "wordmark" }, "Trenova ", el("span", { text: "Capture" })),
      ),
      el(
        "div",
        { class: "nav" },
        items.map(([id, label, iconName, badge]) =>
          el(
            "button",
            {
              type: "button",
              class: "nav-item",
              id: `nav-${id}`,
              "aria-current": local.tab === id ? "page" : null,
              title: label,
              onclick: () => goTo(id),
            },
            icon(iconName),
            el("span", { class: "label", text: label }),
            badge
              ? el("span", {
                  class: v.refused.length ? "count tone-danger" : "count",
                  text: String(badge),
                  "aria-label": v.refused.length ? `${badge} need attention` : `${badge} waiting`,
                })
              : null,
          ),
        ),
      ),
      el("div", { class: "rail-spacer" }),
      el(
        "div",
        { class: "rail-foot" },
        connectionPill(v),
        el(
          "button",
          {
            type: "button",
            class: "account-chip",
            title: [account.person, account.organization].filter(Boolean).join(" · ") || "Account",
            onclick: () => goTo("settings"),
          },
          el("span", { class: "avatar", "aria-hidden": "true", text: initials(account.person) }),
          el(
            "span",
            { class: "who" },
            el("span", { class: "name", text: account.person || "Signed in" }),
            el("span", { class: "subtle", text: account.organization || serverTitle(account) }),
          ),
        ),
      ),
    );
  }

  // The page's title bar and its scrolling body.
  function page(title, actions, ...body) {
    return [
      el(
        "header",
        { class: "page-head" },
        el("h1", { text: title }),
        actions && actions.filter(Boolean).length ? el("div", { class: "actions" }, actions) : null,
      ),
      el("div", { class: "page-body" }, el("div", { class: "stack" }, ...body)),
    ];
  }

  // Before sign-in: where the person is among connect, sign in and scan.
  function drawSteps(v) {
    const current = v.stage === "needsServer" || v.stage === "checking" ? 0 : 1;
    const steps = ["Connect", "Sign in", "Scan"];
    return el(
      "ol",
      { class: "steps", "aria-label": "Setting up" },
      steps.map((label, index) =>
        el(
          "li",
          {
            class: index < current ? "done" : index === current ? "current" : null,
            "aria-current": index === current ? "step" : null,
          },
          el("span", { class: "marker" }, index < current ? icon("check") : String(index + 1)),
          label,
        ),
      ),
    );
  }

  function heading(title, text) {
    return el("div", { class: "auth-heading" }, el("h1", { text: title }), text ? el("p", { text }) : null);
  }

  function auth(v, ...card) {
    return el(
      "div",
      { class: "auth" },
      el("div", { class: "auth-brand" }, logo(24), "Trenova ", el("span", { text: "Capture" })),
      el("div", { class: "auth-stage" }, el("div", { class: "auth-card" }, ...card)),
      el(
        "div",
        { class: "auth-foot" },
        drawSteps(v),
        el("span", { class: "numeric", text: `Version ${v.version}` }),
      ),
    );
  }

  // The field for a self-hosted address, with what the agent said about the
  // last one tried.
  function addressForm(v, submitLabel, onsubmit) {
    const account = v.account;
    const locked = account.serverLocked;
    const value = local.serverDraft ?? account.server ?? SUGGESTED_ADDRESS;
    const problem = local.serverDraft === null ? account.serverProblem : null;
    return el(
      "form",
      {
        class: "field",
        onsubmit: (event) => {
          event.preventDefault();
          const address = (local.serverDraft ?? account.server ?? SUGGESTED_ADDRESS).trim();
          if (!address) {
            return;
          }
          local.serverDraft = null;
          onsubmit(address);
        },
      },
      el("label", { for: "server-address", text: "Trenova address" }),
      el("input", {
        id: "server-address",
        type: "url",
        inputmode: "url",
        autocomplete: "off",
        spellcheck: "false",
        placeholder: SUGGESTED_ADDRESS || EXAMPLE_ADDRESS,
        required: true,
        value,
        disabled: locked,
        "aria-invalid": problem ? "true" : null,
        "aria-describedby": problem ? "server-problem" : "server-help",
        oninput: (event) => {
          local.serverDraft = event.target.value;
          draw();
        },
      }),
      problem
        ? el("p", { id: "server-problem", class: "callout tone-danger", role: "alert", text: problem })
        : el(
            "p",
            { id: "server-help", class: "help" },
            locked
              ? "Your organization sets this address."
              : "The address you open Trenova at in your browser. Trenova Capture finds the rest.",
          ),
      button(submitLabel, { class: "primary large block", type: "submit", disabled: pressed("server") }, null, "arrowRight"),
      !locked
        ? el(
            "p",
            { class: "subtle" },
            "Running Trenova on this computer for development? ",
            button(`Use ${DEVELOPMENT_ADDRESS}`, {
              class: "link",
              onclick: () => {
                local.serverDraft = DEVELOPMENT_ADDRESS;
                draw();
              },
            }),
          )
        : null,
    );
  }

  // Connecting shows its effect as the stage changing (checking, then signed
  // out or pairing) or as a problem with the address.
  function connect(address) {
    const stage = view ? view.stage : null;
    const problem = view ? view.account.serverProblem : null;
    press(
      "server",
      (next) => next.stage === stage && next.account.serverProblem === problem,
      { type: "connect", address },
    );
  }

  function cloudOption() {
    if (!CLOUD_ADDRESS) {
      return null;
    }
    return el(
      "button",
      {
        type: "button",
        class: "option",
        id: "connect-cloud",
        disabled: pressed("server"),
        onclick: () => connect(CLOUD_ADDRESS),
      },
      el("span", { class: "tile ink" }, icon("cloud")),
      el(
        "span",
        { class: "text" },
        el("span", { class: "name", text: "Trenova Cloud" }),
        el("p", { text: "Sign in with your Trenova Cloud account." }),
      ),
      icon("arrowRight", "trail"),
    );
  }

  function selfHostedOption() {
    return el(
      "button",
      {
        type: "button",
        class: "option",
        id: "connect-self-hosted",
        onclick: () => {
          local.selfHosted = true;
          draw();
          const field = document.getElementById("server-address");
          if (field) {
            field.focus();
          }
        },
      },
      el("span", { class: "tile" }, icon("server")),
      el(
        "span",
        { class: "text" },
        el("span", { class: "name", text: "Self-hosted Trenova" }),
        el("p", { text: "Enter the address your organization runs Trenova at." }),
      ),
      icon("arrowRight", "trail"),
    );
  }

  function drawConnect(v) {
    const account = v.account;
    // A locked address, an address that failed, or a draft being typed goes
    // straight to the field; otherwise the choice between Cloud and
    // self-hosted comes first.
    const showForm =
      account.serverLocked ||
      local.selfHosted ||
      local.serverDraft !== null ||
      Boolean(account.serverProblem) ||
      !CLOUD_ADDRESS;
    if (account.serverLocked) {
      return auth(
        v,
        heading("Connect to Trenova", "Scan and print paper straight into Trenova from this computer."),
        el(
          "div",
          { class: "server-card" },
          el("span", { class: "kind" }, icon("lock")),
          el(
            "span",
            { class: "text" },
            el("span", { class: "name", text: "Your organization's Trenova" }),
            el("span", { class: "subtle", text: account.server || "" }),
          ),
        ),
        account.serverProblem
          ? el("p", { class: "callout tone-danger", role: "alert", text: account.serverProblem })
          : null,
        button(
          "Continue",
          {
            class: "primary large block",
            disabled: !account.server || pressed("server"),
            onclick: () => connect(account.server),
          },
          null,
          "arrowRight",
        ),
      );
    }
    if (!showForm) {
      return auth(
        v,
        heading("Connect to Trenova", "Scan and print paper straight into Trenova from this computer. Where is your Trenova?"),
        el("div", { class: "options" }, cloudOption(), selfHostedOption()),
      );
    }
    return auth(
      v,
      heading(
        "Connect to Trenova",
        account.serverProblem
          ? "That address could not be used. Check it and try again."
          : "Enter the address your organization runs Trenova at.",
      ),
      addressForm(v, "Continue", connect),
      CLOUD_ADDRESS
        ? el(
            "div",
            { class: "auth-foot" },
            el("span", { class: "subtle" }, "Using the hosted service? "),
            button("Sign in with Trenova Cloud instead", {
              class: "link",
              onclick: () => {
                local.selfHosted = false;
                local.serverDraft = null;
                connect(CLOUD_ADDRESS);
              },
            }),
          )
        : null,
    );
  }

  function drawChecking(v) {
    return auth(
      v,
      heading("Connecting to Trenova", `Looking for Trenova at ${v.account.server || ""}.`),
      el("div", { class: "waiting" }, el("span", { class: "spinner", "aria-hidden": "true" }), "One moment"),
    );
  }

  function drawSignedOut(v) {
    const account = v.account;
    const cloud = account.serverShared;
    const waitingPages = v.waiting.reduce((sum, b) => sum + b.pages, 0);
    return auth(
      v,
      heading(
        cloud ? "Sign in to Trenova Cloud" : `Sign in to ${serverTitle(account)}`,
        "Your browser opens Trenova, where you approve this computer with a code. It then scans and prints as you.",
      ),
      el(
        "div",
        { class: "server-card" },
        el("span", { class: "kind" }, icon(cloud ? "cloud" : "server")),
        el(
          "span",
          { class: "text" },
          el("span", { class: "name", text: serverTitle(account) }),
          el("span", { class: "subtle", text: account.server || "" }),
        ),
        account.serverLocked
          ? null
          : button("Change", {
              class: "ghost small",
              onclick: () => {
                local.changingServer = true;
                local.selfHosted = false;
                local.serverDraft = null;
                draw();
              },
            }),
      ),
      button(
        cloud ? "Sign in with Trenova Cloud" : "Sign in",
        {
          class: "primary large block",
          id: "sign-in",
          disabled: pressed("signIn"),
          onclick: () => press("signIn", (next) => next.stage === "signedOut", { type: "signIn" }),
        },
        "logIn",
      ),
      waitingPages
        ? el("p", {
            class: "callout tone-neutral",
            text: `${pages(waitingPages)} on this computer ${waitingPages === 1 ? "is" : "are"} sent once you sign in.`,
          })
        : null,
    );
  }

  function drawChangeServer(v) {
    return auth(
      v,
      heading("Change the Trenova address", "Pages waiting to be sent go to the address they were scanned for, so change it only once nothing is waiting."),
      addressForm(v, "Use this address", (address) => {
        local.changingServer = false;
        connect(address);
      }),
      el(
        "div",
        { class: "actions" },
        button("Keep the current address", {
          class: "ghost",
          onclick: () => {
            local.changingServer = false;
            local.serverDraft = null;
            draw();
          },
        }),
      ),
    );
  }

  function copyText(text, mark) {
    const done = () => {
      local[mark] = true;
      draw();
      setTimeout(() => {
        local[mark] = false;
        draw();
      }, 2000);
    };
    if (navigator.clipboard && typeof navigator.clipboard.writeText === "function") {
      navigator.clipboard.writeText(text).then(done, () => {});
    }
  }

  function drawPairing(v) {
    const code = v.pairingCode || "";
    return auth(
      v,
      heading("Approve this computer", "Trenova opened in your browser. Check the code there matches this one, then approve it."),
      el(
        "div",
        { class: "code", "aria-label": code },
        [...code].map((c) => el("span", { class: c === "-" ? "sep" : null, "aria-hidden": "true", text: c })),
      ),
      el("div", { class: "waiting" }, el("span", { class: "spinner", "aria-hidden": "true" }), "Waiting for approval"),
      el(
        "div",
        { class: "actions" },
        button(
          "Open the approval page",
          { class: "primary grow", onclick: () => send({ type: "openApproval" }) },
          "external",
        ),
        button(local.copied ? "Copied" : "Copy code", { onclick: () => copyText(code, "copied") }, local.copied ? "check" : "copy"),
      ),
      v.account.serverShared
        ? el("p", {
            class: "subtle",
            text: "Approve it from the organization you want this computer to scan into. Switch organizations in Trenova first if you belong to more than one.",
          })
        : null,
      el(
        "div",
        { class: "actions end" },
        button("Cancel sign-in", { class: "ghost", onclick: () => send({ type: "cancelSignIn" }) }),
      ),
    );
  }

  function drawConnecting(v) {
    return auth(
      v,
      heading("Connecting to Trenova", v.status.detail || v.account.server || ""),
      el("div", { class: "waiting" }, el("span", { class: "spinner", "aria-hidden": "true" }), "One moment"),
    );
  }

  function drawBeforeSignIn(v) {
    switch (v.stage) {
      case "needsServer":
        return drawConnect(v);
      case "checking":
        return drawChecking(v);
      case "signedOut":
        return local.changingServer ? drawChangeServer(v) : drawSignedOut(v);
      case "pairing":
        return drawPairing(v);
      case "connecting":
        return drawConnecting(v);
      default:
        return null;
    }
  }

  // Notices at the top of Home: what needs the person, most urgent first.
  function notice(tone, iconName, title, text, ...actions) {
    return el(
      "div",
      { class: `notice tone-${tone}`, role: tone === "danger" ? "alert" : "status" },
      icon(iconName, "lg"),
      el(
        "div",
        { class: "grow" },
        el("h2", { text: title }),
        text ? el("p", { text }) : null,
        actions.filter(Boolean).length ? el("div", { class: "actions" }, actions) : null,
      ),
    );
  }

  function drawNotices(v) {
    const notices = [];
    if (v.stage === "offline") {
      notices.push(notice("warning", "cloudOff", v.status.title, v.status.detail));
    } else if (v.stage === "blocked") {
      notices.push(notice("danger", "pause", v.status.title, v.status.detail));
    }
    for (const paused of v.paused) {
      notices.push(
        notice(
          "warning",
          "alert",
          "The scanner stopped",
          `${pages(paused.pages)} from ${paused.label} are kept. ${paused.problem}`,
          button("Continue scanning", {
            class: "primary",
            disabled: pressed(`continue:${paused.key}`),
            onclick: () =>
              press(
                `continue:${paused.key}`,
                (next) => next.paused.some((p) => p.key === paused.key),
                { type: "continue", key: paused.key },
              ),
          }),
          button(`Finish and send ${pages(paused.pages)}`, {
            disabled: pressed(`finish:${paused.key}`),
            onclick: () =>
              press(
                `finish:${paused.key}`,
                (next) => next.paused.some((p) => p.key === paused.key),
                { type: "finish", key: paused.key },
              ),
          }),
        ),
      );
    }
    if (v.refused.length) {
      const count = v.refused.length;
      notices.push(
        notice(
          "danger",
          "xCircle",
          count === 1 ? "Something was not sent" : `${count} things were not sent`,
          "Trenova refused it. Send it again, save a copy, or discard it.",
          button("Review", { class: "primary", onclick: () => goTo("activity") }),
        ),
      );
    }
    if (v.update) {
      const { version, status } = v.update;
      const text = {
        available: "It installs in a minute and restarts by itself. Nothing waiting to be sent is lost.",
        askAdministrator: "Your administrator installs updates on this computer.",
        installing: "It restarts by itself when the update is done.",
        windowsTooOld: "It needs a newer version of Windows.",
      }[status];
      const title =
        status === "installing"
          ? `Installing Trenova Capture ${version}`
          : `Trenova Capture ${version} is available`;
      notices.push(
        notice(
          "info",
          "download",
          title,
          text,
          status === "available"
            ? button("Update now", {
                class: "primary",
                disabled: pressed("update"),
                onclick: () =>
                  press("update", (next) => next.update && next.update.status === "available", {
                    type: "update",
                  }),
              })
            : status === "askAdministrator"
              ? button("Download", { onclick: () => send({ type: "openDownload" }) }, "external")
              : null,
        ),
      );
    } else if (v.updateRequired) {
      notices.push(
        notice(
          "warning",
          "download",
          "An update is needed",
          `Your organization needs Trenova Capture ${v.updateRequired} or later.`,
        ),
      );
    }
    return notices;
  }

  function scannerKey(scanner) {
    return `${scanner.protocol}:${scanner.name}`;
  }

  // The last pages scanned, as they arrive.
  function liveStrip(scan) {
    if (!scan.pages) {
      return null;
    }
    const first = Math.max(1, scan.pages - LIVE_PAGES + 1);
    const recent = [];
    for (let page = first; page <= scan.pages; page += 1) {
      recent.push(page);
    }
    want(scan.key, "thumb", recent);
    const live = { key: scan.key, label: scan.label };
    return el(
      "div",
      { class: "strip live", "aria-label": "Pages scanned so far" },
      recent.map((page) =>
        el(
          "button",
          {
            type: "button",
            class: "strip-page",
            "aria-label": `Open page ${page}`,
            onclick: () => openViewer(live, page, false),
          },
          pagePicture(scan.key, page, 0, "thumb", `Page ${page}`),
        ),
      ),
    );
  }

  function drawScanning(v) {
    const scan = v.scan;
    return el(
      "section",
      { class: "panel", "aria-label": "Scanning" },
      el(
        "div",
        { class: "panel-body" },
        el(
          "div",
          { class: "hero" },
          el("span", { class: "kind" }, icon("scan")),
          el(
            "div",
            { class: "grow" },
            el("div", { class: "big-count", text: String(scan.pages) }),
            el("p", {
              class: "muted",
              text: `${scan.pages === 1 ? "page" : "pages"} from ${scan.label}`,
            }),
          ),
          button(
            scan.stopping ? "Stopping" : "Stop",
            {
              disabled: scan.stopping || pressed("stop"),
              onclick: () =>
                press("stop", (next) => next.scan && !next.scan.stopping, { type: "stopScan" }),
            },
            "stop",
          ),
        ),
        el("div", { class: scan.stopping ? "progress done" : "progress", role: "progressbar", "aria-label": "Scanning" }),
        liveStrip(scan),
        el("p", {
          class: "subtle",
          text: scan.stopping
            ? v.reviewBeforeSending
              ? "The pages scanned so far wait here for you to look over."
              : "The pages scanned so far will be sent."
            : v.reviewBeforeSending
              ? "The pages wait here for you to look over before they are sent."
              : scan.requested
                ? "Going to the record you chose in Trenova. Pages are sent as they are scanned."
                : "Going to Intake. Pages are sent as they are scanned.",
        }),
      ),
    );
  }

  function chooseScanner(v) {
    if (!v.scanners.some((s) => scannerKey(s) === local.scanner)) {
      const preferred = v.scanners.find((s) => s.isDefault) || v.scanners[0];
      local.scanner = preferred ? scannerKey(preferred) : null;
    }
    if (local.profile !== "" && !v.profiles.some((p) => p.id === local.profile)) {
      const preferred = v.profiles.find((p) => p.isDefault) || v.profiles[0];
      local.profile = preferred ? preferred.id : "";
    }
    return {
      scanner: v.scanners.find((s) => scannerKey(s) === local.scanner),
      profile: v.profiles.find((p) => p.id === local.profile),
    };
  }

  function drawScanCard(v) {
    if (v.scan) {
      return drawScanning(v);
    }
    const again = button("Look again", {
      class: "ghost small",
      disabled: pressed("refresh"),
      onclick: () => press("refresh", () => false, { type: "refreshScanners" }),
    }, "refresh");
    if (!v.scanners.length) {
      return panel(
        "Scan to Intake",
        { iconName: "scan", aside: again },
        el(
          "div",
          { class: "empty" },
          icon("scan"),
          el("h2", { text: "No scanner found" }),
          el("p", { text: "Check the scanner is on and connected to this computer, then look again. Settings lists test scanners for trying it without one." }),
        ),
      );
    }
    const { scanner: chosen, profile } = chooseScanner(v);
    return panel(
      "Scan to Intake",
      { iconName: "scan", aside: again },
      el(
        "div",
        { class: "panel-body" },
        el(
          "div",
          { class: "fields" },
          el(
            "div",
            { class: "field" },
            el("label", { for: "scanner", text: "Scanner" }),
            el(
              "select",
              {
                id: "scanner",
                onchange: (event) => {
                  local.scanner = event.target.value;
                  remember({ scanner: local.scanner });
                  draw();
                },
              },
              v.scanners.map((s) =>
                el("option", { value: scannerKey(s), selected: scannerKey(s) === local.scanner, text: s.title }),
              ),
            ),
          ),
          el(
            "div",
            { class: "field" },
            el("label", { for: "profile", text: "Settings" }),
            el(
              "select",
              {
                id: "profile",
                disabled: !v.profiles.length,
                onchange: (event) => {
                  local.profile = event.target.value;
                  remember({ profile: local.profile });
                  draw();
                },
              },
              v.profiles.length
                ? v.profiles.map((p) =>
                    el("option", {
                      value: p.id,
                      selected: p.id === local.profile,
                      text: p.isDefault ? `${p.name} (default)` : p.name,
                    }),
                  )
                : el("option", { value: "", text: "Organization default" }),
            ),
          ),
        ),
        el(
          "div",
          { class: "row" },
          profile
            ? el(
                "div",
                { class: "chips grow" },
                profile.summary.split(", ").map((part) => el("span", { class: "badge", text: part })),
              )
            : el("span", { class: "grow" }),
          button(
            "Scan",
            {
              class: "primary large",
              id: "scan",
              disabled: !v.canScan || !chosen || pressed("scan"),
              onclick: () =>
                press("scan", (next) => next.canScan, {
                  type: "scan",
                  scanner: chosen.name,
                  protocol: chosen.protocol,
                  profile: profile ? profile.id : null,
                }),
            },
            "scan",
          ),
        ),
      ),
    );
  }

  function drawPrintCard(v) {
    if (!v.printing) {
      return null;
    }
    return panel(
      "Print to Trenova",
      { iconName: "print" },
      el(
        "div",
        { class: "panel-body" },
        el(
          "div",
          { class: "item" },
          el("span", { class: v.printerMissing ? "kind tone-danger" : "kind tone-success" }, icon(v.printerMissing ? "alert" : "check")),
          el(
            "div",
            { class: "item-text" },
            el("div", { class: "name", text: v.printerMissing ? "The Trenova printer is not set up" : "The Trenova printer is ready" }),
            el("p", {
              class: "subtle wrap",
              text: v.printerMissing
                ? "Adding it asks for administrator rights once."
                : "Choose the Trenova printer in any program. What you print goes to Intake.",
            }),
          ),
          v.printerMissing
            ? button("Add the printer", {
                class: "primary",
                disabled: pressed("printer"),
                onclick: () => press("printer", (next) => next.printerMissing, { type: "addPrinter" }),
              })
            : null,
        ),
      ),
    );
  }

  function sentItem(batch) {
    return el(
      "li",
      null,
      el(
        "div",
        { class: "item" },
        el("span", { class: "kind tone-success" }, icon("checkCircle")),
        el(
          "div",
          { class: "item-text" },
          el("div", { class: "name", text: batch.label }),
          el("p", {
            class: "subtle",
            text: `${pages(batch.pages)} · ${batch.requested ? "Filed where you asked" : "Waiting in Intake"}`,
          }),
        ),
        el(
          "div",
          { class: "item-meta" },
          when(batch.at),
          batch.link
            ? button("Open", {
                class: "ghost small",
                onclick: () => send({ type: "openLink", link: batch.link }),
              }, null, "external")
            : null,
        ),
      ),
    );
  }

  function drawRecentOnHome(v) {
    if (!v.recent.length) {
      return null;
    }
    return panel(
      "Recently sent",
      {
        iconName: "checkCircle",
        aside: button("See all", { class: "ghost small", onclick: () => goTo("activity") }),
      },
      el("ul", { class: "list" }, v.recent.slice(0, RECENT_ON_HOME).map(sentItem)),
    );
  }

  // A page's picture, turned as the person turned it; its number stands in
  // until the picture arrives, or for a page that has none.
  function pagePicture(key, page, rotation, size, label) {
    return el(
      "span",
      { class: `sheet rot-${rotation || 0}` },
      imageOf(key, size, page, label) || el("span", { class: "sheet-empty", text: String(page) }),
    );
  }

  function openViewer(batch, page, editable) {
    local.viewer = {
      key: batch.key,
      page,
      label: batch.label,
      editable,
    };
    draw();
  }

  function heldPresent(key) {
    return (next) => next.held.some((batch) => batch.key === key);
  }

  function rotate(batch, page, degrees) {
    const before = (batch.pictures.find((p) => p.page === page) || {}).rotation || 0;
    press(
      `rotate:${batch.key}:${page}`,
      (next) => {
        const now = next.held.find((b) => b.key === batch.key);
        const picture = now && now.pictures.find((p) => p.page === page);
        return Boolean(picture) && picture.rotation === before;
      },
      { type: "rotatePage", key: batch.key, page, degrees },
    );
  }

  function removePage(batch, page) {
    forgetBatch(batch.key);
    press(
      `delete:${batch.key}`,
      (next) => {
        const now = next.held.find((b) => b.key === batch.key);
        return Boolean(now) && now.pages === batch.pages;
      },
      { type: "deletePage", key: batch.key, page },
    );
  }

  function pageTile(batch, picture, editable) {
    const busy = pressed(`delete:${batch.key}`) || pressed(`rotate:${batch.key}:${picture.page}`);
    const pictured = pictures.has(pictureId(batch.key, "thumb", picture.page));
    const id = [batch.key, picture.page, picture.rotation, editable, busy, pictured].join("|");
    let tile = tiles.get(id);
    if (!tile) {
      tile = makeTile(batch, picture, editable, busy);
      tiles.set(id, tile);
    }
    tilesDrawn.add(id);
    return tile;
  }

  function makeTile(batch, picture, editable, busy) {
    const label = `Page ${picture.page}`;
    return el(
      "li",
      { class: "tile" },
      el(
        "button",
        {
          type: "button",
          class: "tile-open",
          "aria-label": `Open ${label.toLowerCase()}`,
          onclick: () => openViewer(batch, picture.page, editable),
        },
        pagePicture(batch.key, picture.page, picture.rotation, "thumb", label),
      ),
      el(
        "div",
        { class: "tile-bar" },
        el("span", { class: "subtle numeric", text: String(picture.page) }),
        editable
          ? el(
              "span",
              { class: "tile-actions" },
              button("", {
                class: "icon-button",
                "aria-label": `Turn ${label.toLowerCase()} left`,
                title: "Turn left",
                disabled: busy,
                onclick: () => rotate(batch, picture.page, -90),
              }, "rotateLeft"),
              button("", {
                class: "icon-button",
                "aria-label": `Turn ${label.toLowerCase()} right`,
                title: "Turn right",
                disabled: busy,
                onclick: () => rotate(batch, picture.page, 90),
              }, "rotateRight"),
              button("", {
                class: "icon-button danger-text",
                "aria-label": `Take out ${label.toLowerCase()}`,
                title: "Take out",
                disabled: busy,
                onclick: () => removePage(batch, picture.page),
              }, "trash"),
            )
          : null,
      ),
    );
  }

  function drawHeldBatch(v, batch) {
    const editable = batch.editable;
    const all = local.expanded.has(batch.key);
    const shown = all ? batch.pictures : batch.pictures.slice(0, HELD_PAGES_SHOWN);
    want(batch.key, "thumb", shown.map((p) => p.page));
    const unpictured = batch.pages - batch.pictures.length;
    const confirming = local.confirmHeld === batch.key;
    const { scanner, profile } = chooseScanner(v);
    return el(
      "section",
      { class: "panel review", "aria-label": `Look over ${batch.label}` },
      el(
        "div",
        { class: "panel-head" },
        icon(batch.kind === "print" ? "print" : "scan"),
        el("h2", { class: "grow name", text: batch.label }),
        el("span", { class: "badge tone-brand", text: "Ready to send" }),
        el("span", { class: "badge", text: pages(batch.pages) }),
      ),
      el(
        "div",
        { class: "panel-body" },
        el("p", {
          class: "subtle",
          text: editable
            ? "Nothing is sent until you choose Send. Turn a page, take one out, or scan more first."
            : "Nothing is sent until you choose Send. Pages of a print are turned or split in Intake.",
        }),
        shown.length
          ? el(
              "ul",
              { class: "tiles", "aria-label": "Pages" },
              shown.map((picture) => pageTile(batch, picture, editable)),
            )
          : el("p", {
              class: "callout tone-neutral",
              text: "There are no pictures of these pages on this computer. You see them in Intake once they are sent.",
            }),
        batch.pictures.length > HELD_PAGES_SHOWN
          ? button(all ? "Show fewer" : `Show all ${batch.pictures.length} pages`, {
              class: "link",
              onclick: () => {
                if (all) {
                  local.expanded.delete(batch.key);
                } else {
                  local.expanded.add(batch.key);
                }
                draw();
              },
            })
          : null,
        unpictured > 0 && batch.pictures.length
          ? el("p", {
              class: "subtle",
              text: `${pages(unpictured)} ${unpictured === 1 ? "has" : "have"} no picture here.`,
            })
          : null,
        confirming
          ? el(
              "div",
              { class: "row" },
              el("span", {
                class: "grow",
                text: `Delete these ${pages(batch.pages)} from this computer for good?`,
              }),
              button("Discard", {
                class: "danger",
                disabled: pressed(`discardHeld:${batch.key}`),
                onclick: () => {
                  local.confirmHeld = null;
                  press(`discardHeld:${batch.key}`, heldPresent(batch.key), {
                    type: "discardHeld",
                    key: batch.key,
                  });
                },
              }),
              button("Keep", {
                class: "ghost",
                onclick: () => {
                  local.confirmHeld = null;
                  draw();
                },
              }),
            )
          : el(
              "div",
              { class: "actions" },
              button(
                `Send ${pages(batch.pages)}`,
                {
                  class: "primary",
                  disabled: pressed(`sendHeld:${batch.key}`),
                  onclick: () =>
                    press(`sendHeld:${batch.key}`, heldPresent(batch.key), {
                      type: "sendHeld",
                      key: batch.key,
                    }),
                },
                "send",
              ),
              editable && scanner
                ? button(
                    "Scan more",
                    {
                      disabled: Boolean(v.scan) || pressed(`more:${batch.key}`),
                      title: `Adds pages from ${scanner.title}`,
                      onclick: () =>
                        press(`more:${batch.key}`, (next) => !next.scan, {
                          type: "scanMore",
                          key: batch.key,
                          scanner: scanner.name,
                          protocol: scanner.protocol,
                          profile: profile ? profile.id : null,
                        }),
                    },
                    "scan",
                  )
                : null,
              button("Discard", {
                class: "ghost",
                onclick: () => {
                  local.confirmHeld = batch.key;
                  draw();
                },
              }),
            ),
      ),
    );
  }

  function drawHeld(v) {
    return v.held.map((batch) => drawHeldBatch(v, batch));
  }

  // A few of a waiting or refused batch's pages, to see what it is.
  function strip(batch) {
    if (!batch.pictures.length) {
      return null;
    }
    const shown = batch.pictures.slice(0, STRIP_PAGES);
    want(batch.key, "thumb", shown.map((p) => p.page));
    const more = batch.pictures.length - shown.length;
    return el(
      "div",
      { class: "strip" },
      shown.map((picture) =>
        el(
          "button",
          {
            type: "button",
            class: "strip-page",
            "aria-label": `Open page ${picture.page}`,
            onclick: () => openViewer(batch, picture.page, false),
          },
          pagePicture(batch.key, picture.page, picture.rotation, "thumb", `Page ${picture.page}`),
        ),
      ),
      more > 0 ? el("span", { class: "subtle numeric", text: `+${more}` }) : null,
    );
  }

  // The page open full size, over the window.
  function viewerBatch(v) {
    if (!local.viewer) {
      return null;
    }
    const { key } = local.viewer;
    if (v.scan && v.scan.key === key) {
      const first = Math.max(1, v.scan.pages - LIVE_PAGES + 1);
      const scanned = [];
      for (let page = first; page <= v.scan.pages; page += 1) {
        scanned.push({ page, rotation: 0 });
      }
      return { key, label: v.scan.label, pictures: scanned, editable: false };
    }
    const found =
      v.held.find((b) => b.key === key) ||
      v.waiting.find((b) => b.key === key) ||
      v.refused.find((b) => b.key === key);
    if (!found) {
      return null;
    }
    const editable = Boolean(v.held.find((b) => b.key === key && b.editable));
    return { ...found, editable };
  }

  function drawViewer(v) {
    const batch = viewerBatch(v);
    if (!batch || !batch.pictures.some((p) => p.page === local.viewer.page)) {
      local.viewer = null;
      viewerHost.replaceChildren();
      viewerHost.hidden = true;
      return;
    }
    const index = batch.pictures.findIndex((p) => p.page === local.viewer.page);
    const picture = batch.pictures[index];
    want(batch.key, "view", [picture.page]);
    const neighbours = [batch.pictures[index - 1], batch.pictures[index + 1]].filter(Boolean);
    want(batch.key, "view", neighbours.map((p) => p.page));
    const go = (step) => {
      const next = batch.pictures[index + step];
      if (next) {
        local.viewer.page = next.page;
        draw();
      }
    };
    const opening = viewerHost.hidden;
    const large = pictureOf(batch.key, "view", picture.page);
    const small = pictureOf(batch.key, "thumb", picture.page);
    viewerHost.hidden = false;
    viewerHost.replaceChildren(
      el(
        "div",
        {
          class: "viewer",
          role: "dialog",
          "aria-modal": "true",
          "aria-label": `${batch.label}, page ${picture.page}`,
        },
        el(
          "div",
          { class: "viewer-bar" },
          el(
            "div",
            { class: "grow" },
            el("div", { class: "name", text: batch.label }),
            el("div", {
              class: "viewer-count numeric",
              text: `Page ${index + 1} of ${batch.pictures.length}`,
            }),
          ),
          button("", {
            id: "viewer-close",
            class: "icon-button",
            "aria-label": "Close",
            onclick: () => {
              local.viewer = null;
              draw();
            },
          }, "close"),
        ),
        el(
          "div",
          { class: "viewer-stage" },
          large || small
            ? el(
                "span",
                { class: `viewer-sheet rot-${picture.rotation || 0}` },
                el("img", {
                  src: large || small,
                  alt: `Page ${picture.page}`,
                  draggable: "false",
                }),
              )
            : el("span", { class: "spinner lg", "aria-hidden": "true" }),
        ),
        el(
          "div",
          { class: "viewer-bar" },
          button("", {
            id: "viewer-previous",
            class: "icon-button",
            "aria-label": "Previous page",
            disabled: index === 0,
            onclick: () => go(-1),
          }, "chevronLeft"),
          button("", {
            id: "viewer-next",
            class: "icon-button",
            "aria-label": "Next page",
            disabled: index === batch.pictures.length - 1,
            onclick: () => go(1),
          }, "chevronRight"),
          el("span", { class: "grow" }),
          batch.editable
            ? [
                button("Turn left", {
                  id: "viewer-left",
                  disabled: pressed(`rotate:${batch.key}:${picture.page}`),
                  onclick: () => rotate(batch, picture.page, -90),
                }, "rotateLeft"),
                button("Turn right", {
                  id: "viewer-right",
                  disabled: pressed(`rotate:${batch.key}:${picture.page}`),
                  onclick: () => rotate(batch, picture.page, 90),
                }, "rotateRight"),
                button("Take out", {
                  id: "viewer-delete",
                  class: "danger-text",
                  disabled: pressed(`delete:${batch.key}`),
                  onclick: () => {
                    const after = batch.pictures[index + 1] || batch.pictures[index - 1];
                    local.viewer.page = after ? (after.page > picture.page ? picture.page : after.page) : picture.page;
                    removePage(batch, picture.page);
                  },
                }, "trash"),
              ]
            : null,
        ),
      ),
    );
    if (opening) {
      const close = document.getElementById("viewer-close");
      if (close) {
        close.focus();
      }
    }
  }

  function openTrenovaButton(v) {
    if (!v.account.webBase) {
      return null;
    }
    return button("Open Trenova", { class: "ghost small", onclick: () => send({ type: "openTrenova" }) }, null, "external");
  }

  function drawHome(v) {
    return page(
      "Home",
      [
        v.canOpenIntake
          ? button("Open Intake", { class: "small", onclick: () => send({ type: "openIntake" }) }, null, "external")
          : openTrenovaButton(v),
      ],
      drawNotices(v),
      drawHeld(v),
      v.stage === "blocked" ? null : drawScanCard(v),
      drawPrintCard(v),
      drawRecentOnHome(v),
      el(
        "p",
        { class: "footnote" },
        icon("tray"),
        "Closing this window keeps Trenova Capture running in the notification area, where scans and prints still reach Trenova.",
      ),
    );
  }

  function drawRefused(v) {
    if (!v.refused.length) {
      return null;
    }
    const present = (key) => (next) => next.refused.some((batch) => batch.key === key);
    return panel(
      "Not sent",
      { count: v.refused.length, iconName: "xCircle" },
      el(
        "ul",
        { class: "list" },
        v.refused.map((batch) => {
          const confirming = local.confirmDiscard === batch.key;
          return el(
            "li",
            null,
            el(
              "div",
              { class: "item" },
              kindIcon(batch.kind, "danger"),
              el(
                "div",
                { class: "item-text" },
                el("div", { class: "name", text: batch.label }),
                el("p", {
                  class: "subtle",
                  text: [batch.pages ? pages(batch.pages) : null, ago(batch.refusedAt)]
                    .filter(Boolean)
                    .join(" · "),
                }),
              ),
            ),
            batch.reason ? el("div", { class: "callout tone-danger", text: batch.reason }) : null,
            strip(batch),
            confirming
              ? el(
                  "div",
                  { class: "row" },
                  el("span", {
                    class: "grow",
                    text: batch.pages
                      ? `Delete these ${pages(batch.pages)} from this computer for good?`
                      : "Delete this from this computer for good?",
                  }),
                  button("Discard", {
                    class: "danger",
                    disabled: pressed(`discard:${batch.key}`),
                    onclick: () => {
                      local.confirmDiscard = null;
                      press(`discard:${batch.key}`, present(batch.key), {
                        type: "discard",
                        key: batch.key,
                      });
                    },
                  }),
                  button("Keep", {
                    class: "ghost",
                    onclick: () => {
                      local.confirmDiscard = null;
                      draw();
                    },
                  }),
                )
              : el(
                  "div",
                  { class: "actions" },
                  batch.readable
                    ? button(
                        "Send again",
                        {
                          class: "primary",
                          disabled: pressed(`retry:${batch.key}`),
                          onclick: () =>
                            press(`retry:${batch.key}`, present(batch.key), {
                              type: "retry",
                              key: batch.key,
                            }),
                        },
                        "refresh",
                      )
                    : null,
                  batch.readable
                    ? button(
                        "Save a copy",
                        {
                          disabled: pressed(`save:${batch.key}`),
                          onclick: () =>
                            press(`save:${batch.key}`, () => true, { type: "save", key: batch.key }),
                        },
                        "download",
                      )
                    : null,
                  button("Discard", {
                    class: "ghost",
                    onclick: () => {
                      local.confirmDiscard = batch.key;
                      draw();
                    },
                  }),
                ),
          );
        }),
      ),
      el("p", {
        class: "panel-quiet",
        text: "Sending again puts it in Intake. Save a copy writes its pages as PDFs to your Downloads folder.",
      }),
    );
  }

  function drawWaiting(v) {
    if (!v.waiting.length) {
      return null;
    }
    return panel(
      "Waiting to send",
      { count: v.waiting.length, iconName: "clock" },
      el(
        "ul",
        { class: "list" },
        v.waiting.map((batch) =>
          el(
            "li",
            null,
            el(
              "div",
              { class: "item" },
              kindIcon(batch.kind),
              el(
                "div",
                { class: "item-text" },
                el("div", { class: "name", text: batch.label }),
                el("p", { class: "subtle", text: `${batch.state} · ${pages(batch.pages)}` }),
              ),
              el("div", { class: "item-meta" }, when(batch.createdAt)),
            ),
            strip(batch),
          ),
        ),
      ),
    );
  }

  function drawSent(v) {
    if (!v.recent.length) {
      return null;
    }
    return panel(
      "Sent",
      {
        iconName: "checkCircle",
        aside: v.canOpenIntake
          ? button("Open Intake", { class: "ghost small", onclick: () => send({ type: "openIntake" }) }, null, "external")
          : null,
      },
      el("ul", { class: "list" }, v.recent.map(sentItem)),
    );
  }

  function drawMessages(v) {
    if (!v.messages.length) {
      return null;
    }
    return panel(
      "Notifications",
      { iconName: "bell", count: v.messages.length },
      el(
        "ul",
        { class: "list" },
        v.messages.map((message) =>
          el(
            "li",
            { class: `message tone-${message.tone}-text` },
            el(
              "div",
              { class: "row" },
              el("span", { class: "tone-mark", "aria-hidden": "true" }),
              el("span", { class: "message-title grow", text: message.title }),
              when(message.at),
            ),
            message.body ? el("p", { class: "muted", text: message.body }) : null,
            message.link
              ? el(
                  "div",
                  null,
                  button("Open in Trenova", {
                    class: "link",
                    onclick: () => send({ type: "openLink", link: message.link }),
                  }),
                )
              : null,
          ),
        ),
      ),
    );
  }

  function drawActivity(v) {
    const sections = [
      ...drawHeld(v),
      drawRefused(v),
      drawWaiting(v),
      drawSent(v),
      drawMessages(v),
    ].filter(Boolean);
    return page(
      "Activity",
      [
        v.canOpenIntake
          ? button("Open Intake", { class: "small", onclick: () => send({ type: "openIntake" }) }, null, "external")
          : null,
      ],
      sections.length
        ? sections
        : el(
            "section",
            { class: "panel" },
            el(
              "div",
              { class: "empty" },
              icon("inbox"),
              el("h2", { text: "Nothing yet" }),
              el("p", { text: "What you scan and print shows here: waiting, sent, or needing a look." }),
            ),
          ),
    );
  }

  function diagnostics(v) {
    const account = v.account;
    const lines = [
      `Trenova Capture ${v.version}`,
      `Connection: ${(CONNECTION[v.stage] || CONNECTION.connecting)[1]} (${v.status.title})`,
      `Server: ${account.server || "none"}${account.serverName ? ` (${account.serverName})` : ""}`,
      account.serverApi ? `API: ${account.serverApi}` : null,
      account.webBase ? `Web app: ${account.webBase}` : null,
      account.person ? `Signed in as: ${account.person}${account.organization ? ` at ${account.organization}` : ""}` : null,
      account.deviceName ? `Computer: ${account.deviceName}` : null,
      account.deviceId ? `Device ID: ${account.deviceId}` : null,
      `Scanners: ${v.scanners.length ? v.scanners.map((s) => s.title).join(", ") : "none found"}`,
      `Printing: ${!v.printing ? "not installed" : v.printerMissing ? "printer missing" : "ready"}`,
      `Waiting: ${pages(v.waiting.reduce((sum, b) => sum + b.pages, 0))}, not sent: ${v.refused.length}`,
      `Review before sending: ${v.reviewBeforeSending ? "on" : "off"}${v.reviewLocked ? " (by policy)" : ""}`,
      v.update ? `Update: ${v.update.version} ${v.update.status}` : null,
      v.updateRequired ? `Required version: ${v.updateRequired}` : null,
    ];
    return lines.filter(Boolean).join("\n");
  }

  function settingSwitch(id, checked, disabled, name, text, onchange) {
    return el(
      "label",
      { class: "switch" },
      el("input", { id, type: "checkbox", role: "switch", checked, disabled, onchange }),
      el(
        "span",
        { class: "grow" },
        el("span", { class: "name", text: name }),
        el("p", { class: "subtle", text }),
      ),
    );
  }

  function drawSettings(v) {
    const account = v.account;
    const cloud = account.serverShared;
    return page(
      "Settings",
      [],
      panel(
        "Account",
        { iconName: "users" },
        el(
          "div",
          { class: "panel-body" },
          account.signedIn
            ? el(
                "div",
                { class: "item" },
                el("span", { class: "avatar lg", "aria-hidden": "true", text: initials(account.person) }),
                el(
                  "div",
                  { class: "item-text" },
                  el("div", { class: "name", text: account.person || "Signed in" }),
                  el("p", { class: "subtle", text: [account.organization, account.deviceName ? `this computer is ${account.deviceName}` : null].filter(Boolean).join(" · ") }),
                ),
                local.confirmSignOut
                  ? el(
                      "div",
                      { class: "actions" },
                      button("Sign out", {
                        class: "danger",
                        onclick: () => {
                          local.confirmSignOut = false;
                          send({ type: "signOut" });
                        },
                      }),
                      button("Keep me signed in", {
                        class: "ghost",
                        onclick: () => {
                          local.confirmSignOut = false;
                          draw();
                        },
                      }),
                    )
                  : button("Sign out", {
                      onclick: () => {
                        local.confirmSignOut = true;
                        draw();
                      },
                    }),
              )
            : el(
                "div",
                { class: "row" },
                el("span", { class: "grow muted", text: "Not signed in" }),
                v.stage === "signedOut"
                  ? button("Sign in", { class: "primary", onclick: () => send({ type: "signIn" }) })
                  : null,
              ),
          account.signedIn
            ? el("p", {
                class: "subtle",
                text: local.confirmSignOut
                  ? "Signing out removes this computer from Trenova. Pages not yet sent are kept here until you sign in again."
                  : cloud
                    ? "This computer scans into the organization you approved it from. To scan into another of your organizations, sign out and sign in again from that one."
                    : "This computer scans and prints as you, with your permissions, until it is signed out or revoked in Trenova.",
              })
            : null,
        ),
      ),
      panel(
        "Server",
        { iconName: cloud ? "cloud" : "server" },
        el(
          "div",
          { class: "panel-body" },
          el(
            "dl",
            { class: "facts" },
            el("dt", { text: "Trenova" }),
            el("dd", { text: serverTitle(account) }),
            el("dt", { text: "Address" }),
            el("dd", { class: "mono", text: account.server || "Not set" }),
            account.serverApi ? el("dt", { text: "API" }) : null,
            account.serverApi ? el("dd", { class: "mono", text: account.serverApi }) : null,
          ),
          account.serverProblem
            ? el("p", { class: "callout tone-danger", role: "alert", text: account.serverProblem })
            : null,
          el(
            "div",
            { class: "actions" },
            account.webBase
              ? button("Open Trenova", { onclick: () => send({ type: "openTrenova" }) }, null, "external")
              : null,
            account.serverLocked
              ? el("span", { class: "subtle", text: "Your organization sets this address." })
              : button("Change address", {
                  class: "ghost",
                  onclick: () => {
                    local.changingServer = true;
                    local.serverDraft = null;
                    draw();
                  },
                }),
          ),
        ),
      ),
      panel(
        "Appearance",
        { iconName: local.theme === "dark" ? "moon" : "sun" },
        el(
          "div",
          { class: "panel-body" },
          el(
            "div",
            { class: "row" },
            el(
              "span",
              { class: "grow" },
              el("span", { class: "name", text: "Theme" }),
              el("p", { class: "subtle", text: "Follow Windows, or choose one for this window." }),
            ),
            el(
              "div",
              { class: "segmented", role: "group", "aria-label": "Theme" },
              [
                ["system", "System"],
                ["light", "Light"],
                ["dark", "Dark"],
              ].map(([value, label]) =>
                button(label, {
                  "aria-pressed": String(local.theme === value),
                  onclick: () => {
                    local.theme = value;
                    remember({ theme: value });
                    applyTheme();
                    draw();
                  },
                }),
              ),
            ),
          ),
        ),
      ),
      panel(
        "Notifications",
        { iconName: "bell" },
        el(
          "div",
          { class: "panel-body" },
          settingSwitch(
            "routine-notifications",
            v.routineNotifications,
            false,
            "Tell me when things go well",
            "Something sent, the connection back, an update installed. Problems are always shown.",
            (event) => send({ type: "setNotifications", routine: Boolean(event.target.checked) }),
          ),
        ),
      ),
      panel(
        "Before sending",
        { iconName: "review" },
        el(
          "div",
          { class: "panel-body" },
          settingSwitch(
            "review-before-sending",
            v.reviewBeforeSending,
            v.reviewLocked,
            "Let me look things over before they are sent",
            v.reviewLocked
              ? "Your organization sets this on this computer."
              : "Scans and prints wait here until you choose Send, so you can turn a page, take one out, or scan more.",
            (event) => send({ type: "setReview", on: Boolean(event.target.checked) }),
          ),
        ),
      ),
      v.printing
        ? panel(
            "Printing",
            { iconName: "print" },
            el(
              "div",
              { class: "panel-body" },
              el(
                "div",
                { class: "row" },
                el("span", {
                  class: "grow",
                  text: v.printerMissing
                    ? "The Trenova printer is not set up on this computer."
                    : "The Trenova printer is set up. Print to it from any program.",
                }),
                v.printerMissing
                  ? button("Add the printer", {
                      class: "primary",
                      disabled: pressed("printer"),
                      onclick: () =>
                        press("printer", (next) => next.printerMissing, { type: "addPrinter" }),
                    })
                  : null,
              ),
            ),
          )
        : null,
      panel(
        "This computer",
        { iconName: "monitor" },
        el(
          "div",
          { class: "panel-body" },
          el(
            "dl",
            { class: "facts" },
            el("dt", { text: "Version" }),
            el("dd", { class: "numeric", text: v.version }),
            el("dt", { text: "Scanners" }),
            el("dd", {
              class: "wrap",
              text: v.scanners.length ? v.scanners.map((s) => s.title).join(", ") : "None found",
            }),
          ),
          settingSwitch(
            "test-scanner",
            v.testScanner,
            v.testScannerLocked,
            "Show test scanners",
            v.testScannerLocked
              ? "Your organization has turned these off on this computer."
              : "Scanners that feed sample paperwork, including a paper jam and a double feed, for trying Trenova Capture without a scanner. What they scan goes to Intake like any scan.",
            (event) => send({ type: "setTestScanner", on: Boolean(event.target.checked) }),
          ),
          el(
            "div",
            { class: "actions" },
            button(
              "Look for scanners again",
              {
                disabled: !account.signedIn || pressed("refresh"),
                onclick: () => press("refresh", () => false, { type: "refreshScanners" }),
              },
              "refresh",
            ),
            button("Open the log folder", { onclick: () => send({ type: "openLogs" }) }, "folder"),
          ),
        ),
      ),
      panel(
        "Support",
        { iconName: "lifeBuoy", hint: "What to send when asking for help" },
        el(
          "div",
          { class: "panel-body" },
          el("pre", { class: "diagnostics", id: "diagnostics", text: diagnostics(v) }),
          el(
            "div",
            { class: "actions" },
            button(local.copiedDiagnostics ? "Copied" : "Copy", {
              onclick: () => copyText(diagnostics(v), "copiedDiagnostics"),
            }, local.copiedDiagnostics ? "check" : "copy"),
            el("span", { class: "subtle", text: "The log folder has the rest." }),
          ),
        ),
      ),
      el(
        "div",
        { class: "actions end" },
        button("Quit Trenova Capture", { class: "ghost", onclick: () => send({ type: "quit" }) }, "power"),
      ),
    );
  }

  // New messages become toasts while the window is in front, since Windows
  // is not asked to show them then.
  function toastNew(v) {
    const newest = v.messages[0];
    const mark = newest ? `${newest.at}:${newest.title}` : "";
    if (local.lastMessage === null) {
      local.lastMessage = mark;
      return;
    }
    if (!newest || mark === local.lastMessage) {
      return;
    }
    const fresh = [];
    for (const message of v.messages) {
      if (`${message.at}:${message.title}` === local.lastMessage) {
        break;
      }
      fresh.push(message);
    }
    local.lastMessage = mark;
    if (!document.hasFocus()) {
      return;
    }
    for (const message of fresh.slice(0, 2).reverse()) {
      showToast(message);
    }
  }

  function showToast(message) {
    const iconName = { danger: "xCircle", warning: "alert" }[message.tone] || "checkCircle";
    const toast = el(
      "div",
      { class: `toast tone-${message.tone}-text`, role: message.tone === "danger" ? "alert" : "status" },
      icon(iconName),
      el(
        "div",
        { class: "grow" },
        el("div", { class: "name", text: message.title }),
        message.body ? el("p", { text: message.body }) : null,
        message.link
          ? button("Open in Trenova", {
              class: "link",
              onclick: () => send({ type: "openLink", link: message.link }),
            })
          : null,
      ),
      button("", {
        class: "icon-button",
        "aria-label": "Dismiss",
        onclick: () => toast.remove(),
      }, "close"),
    );
    toasts.append(toast);
    while (toasts.children.length > 3) {
      toasts.firstElementChild.remove();
    }
    setTimeout(() => toast.remove(), message.tone === "danger" ? TOAST_MS * 2 : TOAST_MS);
  }

  // Keeps the field being typed in, and where in it, across a redraw.
  function captureFocus() {
    const active = document.activeElement;
    if (!active || !active.id || !(app.contains(active) || viewerHost.contains(active) || rail.contains(active))) {
      return null;
    }
    const focus = { id: active.id };
    if (typeof active.selectionStart === "number") {
      focus.start = active.selectionStart;
      focus.end = active.selectionEnd;
    }
    return focus;
  }

  function restoreFocus(focus) {
    if (!focus) {
      return;
    }
    const node = document.getElementById(focus.id);
    if (!node) {
      return;
    }
    node.focus();
    if (typeof focus.start === "number" && typeof node.setSelectionRange === "function") {
      try {
        node.setSelectionRange(focus.start, focus.end);
      } catch {
        // Not every input keeps a selection.
      }
    }
  }

  // Keeps the page where it was scrolled to across a redraw.
  function captureScroll() {
    const body = app.querySelector(".page-body");
    return body ? body.scrollTop : 0;
  }

  function restoreScroll(top) {
    const body = app.querySelector(".page-body");
    if (body && top) {
      body.scrollTop = top;
    }
  }

  function draw() {
    if (!view) {
      return;
    }
    const v = view;
    placed.clear();
    tilesDrawn.clear();
    trackShapes(v);
    prunePending();
    const focus = captureFocus();
    const scrolled = captureScroll();
    const signedIn = signedInStage(v);
    shell.classList.toggle("alone", !signedIn);
    if (signedIn) {
      drawRail(v);
    } else {
      rail.replaceChildren();
      local.tab = "home";
      local.selfHosted = local.selfHosted && v.stage === "needsServer";
    }
    if (v.stage !== "needsServer" && v.stage !== "signedOut") {
      local.changingServer = false;
    }
    const content = !signedIn
      ? drawBeforeSignIn(v)
      : local.tab === "settings"
        ? drawSettings(v)
        : local.tab === "activity"
          ? drawActivity(v)
          : drawHome(v);
    app.replaceChildren(...[content].flat(3).filter(Boolean));
    for (const id of tiles.keys()) {
      if (!tilesDrawn.has(id)) {
        tiles.delete(id);
      }
    }
    drawViewer(v);
    restoreFocus(focus);
    restoreScroll(scrolled);
  }

  document.addEventListener("keydown", (event) => {
    if (local.viewer) {
      const control = {
        Escape: "viewer-close",
        ArrowLeft: "viewer-previous",
        ArrowRight: "viewer-next",
      }[event.key];
      const target = control && document.getElementById(control);
      if (target && !target.disabled) {
        event.preventDefault();
        target.click();
      }
      return;
    }
    if (event.key !== "Escape") {
      return;
    }
    if (local.confirmHeld) {
      local.confirmHeld = null;
      draw();
    } else if (local.confirmDiscard) {
      local.confirmDiscard = null;
      draw();
    } else if (local.confirmSignOut) {
      local.confirmSignOut = false;
      draw();
    } else if (local.changingServer) {
      local.changingServer = false;
      local.serverDraft = null;
      draw();
    } else if (local.selfHosted && local.serverDraft === null && CLOUD_ADDRESS) {
      local.selfHosted = false;
      draw();
    } else if (local.tab !== "home") {
      goTo("home");
    } else {
      send({ type: "close" });
    }
  });

  document.addEventListener("contextmenu", (event) => {
    if (!(event.target instanceof HTMLInputElement)) {
      event.preventDefault();
    }
  });

  // Draws once for everything that arrived since the last frame: a scan
  // reports every page, and pictures arrive a few dozen at a time.
  let drawQueued = false;
  function drawSoon() {
    if (drawQueued) {
      return;
    }
    drawQueued = true;
    requestAnimationFrame(() => {
      drawQueued = false;
      draw();
    });
  }

  window.trenova = {
    render(next) {
      view = next;
      toastNew(next);
      drawSoon();
    },
    pictures(delivered) {
      if (!delivered || typeof delivered.key !== "string" || !Array.isArray(delivered.pictures)) {
        return;
      }
      for (const picture of delivered.pictures) {
        if (typeof picture.src === "string" && picture.src.startsWith("data:image/jpeg;base64,")) {
          keep(pictureId(delivered.key, delivered.size, picture.page), picture.src);
        }
      }
      drawSoon();
    },
  };

  setInterval(draw, 30000);
  send({ type: "ready" });
})();
