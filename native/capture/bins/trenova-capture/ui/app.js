(() => {
  "use strict";

  const app = document.getElementById("app");
  const viewerHost = document.getElementById("viewer");
  const who = document.getElementById("who");
  const connection = document.getElementById("connection");
  const tabs = document.getElementById("tabs");
  const toasts = document.getElementById("toasts");

  // Trenova's own service, and the web app's dev server when Trenova runs on
  // this computer; the dev server passes /api on to the API.
  const CLOUD_ADDRESS = "https://cloud.trenova.app";
  const DEVELOPMENT_ADDRESS = "http://localhost:5173";
  // The namespace SVG elements are created in; nothing is fetched from it.
  const SVG_NAMESPACE = "http://www.w3.org/2000/svg";
  const TOAST_MS = 6000;
  const RECENT_ON_HOME = 3;
  // Pages shown of a held batch before "Show all", and in a strip.
  const HELD_PAGES_SHOWN = 12;
  const STRIP_PAGES = 4;
  const LIVE_PAGES = 6;
  // The most pictures asked for in one message, as the agent allows.
  const PICTURES_PER_ASK = 24;
  // How long before a picture asked for and not received is asked again.
  const ASK_AGAIN_MS = 20000;

  // What belongs to this window alone and survives a redraw: the tab shown,
  // choices made, text being typed, a question being asked, and buttons
  // already pressed.
  const local = {
    tab: "home",
    scanner: null,
    profile: null,
    serverDraft: null,
    confirmDiscard: null,
    copied: false,
    pending: new Map(),
    // The newest message already seen, so only later ones become toasts.
    lastMessage: null,
    // The page open full size: { key, page, label, pages, editable }.
    viewer: null,
    // Held batches showing all their pages, and the one being discarded.
    expanded: new Set(),
    confirmHeld: null,
  };

  // Pictures the agent sent, by batch, size and page, and when each was
  // last asked for. A batch whose pages change (one taken out) is dropped.
  const pictures = new Map();
  const asked = new Map();
  let shapes = new Map();

  function pictureId(key, size, page) {
    return `${key}|${size}|${page}`;
  }

  function forgetBatch(key) {
    for (const store of [pictures, asked]) {
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
    for (const [key, shape] of shapes) {
      if (next.get(key) !== shape) {
        forgetBatch(key);
      }
    }
    shapes = next;
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
  let view = null;

  const ICONS = {
    scan: "M3 7V5a2 2 0 0 1 2-2h2 M17 3h2a2 2 0 0 1 2 2v2 M21 17v2a2 2 0 0 1-2 2h-2 M7 21H5a2 2 0 0 1-2-2v-2 M7 12h10",
    print:
      "M6 9V2h12v7 M6 18H4a2 2 0 0 1-2-2v-5a2 2 0 0 1 2-2h16a2 2 0 0 1 2 2v5a2 2 0 0 1-2 2h-2 M6 14h12v8H6z",
    check: "M20 6 9 17l-5-5",
    checkCircle: "M22 11.08V12a10 10 0 1 1-5.93-9.14 M22 4 12 14.01l-3-3",
    alert:
      "M10.29 3.86 1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0Z M12 9v4 M12 17h.01",
    xCircle: "M12 22a10 10 0 1 0 0-20 10 10 0 0 0 0 20Z M15 9l-6 6 M9 9l6 6",
    cloudOff:
      "M2 2l20 20 M5.78 5.78A7 7 0 0 0 9 19h8.5a4.5 4.5 0 0 0 1.31-.2 M21.53 16.5A4.5 4.5 0 0 0 17.5 10h-1.79A7 7 0 0 0 9.66 5.3",
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
    close: "M18 6 6 18 M6 6l12 12",
    stop: "M7 7h10v10H7z",
    tray: "M4 14h16v6H4z M8 10l4-4 4 4 M12 6v8",
    rotateLeft: "M3 12a9 9 0 1 0 2.64-6.36L3 8 M3 3v5h5",
    rotateRight: "M21 12a9 9 0 1 1-2.64-6.36L21 8 M21 3v5h-5",
    trash: "M3 6h18 M8 6V4h8v2 M19 6l-1 14H6L5 6 M10 11v6 M14 11v6",
    send: "M22 2 11 13 M22 2l-7 20-4-9-9-4 20-7z",
    chevronLeft: "M15 18l-6-6 6-6",
    chevronRight: "M9 18l6-6-6-6",
    review: "M2 12s3.5-7 10-7 10 7 10 7-3.5 7-10 7S2 12 2 12Z M12 15a3 3 0 1 0 0-6 3 3 0 0 0 0 6Z",
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

  function button(label, props, iconName) {
    return el(
      "button",
      { type: "button", ...props },
      iconName ? icon(iconName) : null,
      label,
    );
  }

  function panel(title, options, ...body) {
    const { count, aside, iconName } = options || {};
    return el(
      "section",
      { class: "panel", "aria-label": title },
      el(
        "div",
        { class: "panel-head" },
        iconName ? icon(iconName) : null,
        el("h2", { text: title }),
        count ? el("span", { class: "count", text: String(count) }) : null,
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
    draw();
    window.scrollTo(0, 0);
  }

  function signedInStage(v) {
    return ["connecting", "online", "offline", "blocked"].includes(v.stage);
  }

  // The masthead: who is signed in, and a pill for the connection.
  const CONNECTION = {
    needsServer: ["neutral", "Not connected"],
    signedOut: ["neutral", "Signed out"],
    pairing: ["info", "Signing in"],
    connecting: ["neutral", "Connecting"],
    online: ["success", "Online"],
    offline: ["warning", "Offline"],
    blocked: ["danger", "Paused"],
  };

  function drawMasthead(v) {
    const account = v.account;
    who.textContent = account.signedIn
      ? [account.person, account.organization].filter(Boolean).join(" · ") || "Signed in"
      : account.server || "Not connected";
    const [tone, label] = CONNECTION[v.stage] || CONNECTION.connecting;
    connection.className = `pill tone-${tone}`;
    connection.title = v.status.detail || v.status.title;
    connection.replaceChildren(el("span", { class: "dot", "aria-hidden": "true" }), label);
  }

  function activityCount(v) {
    return v.refused.length + v.held.length + v.waiting.length;
  }

  function drawTabs(v) {
    const items = [
      ["home", "Home", null],
      ["activity", "Activity", activityCount(v)],
      ["settings", "Settings", null],
    ];
    tabs.replaceChildren(
      ...items.map(([id, label, count]) =>
        el(
          "button",
          {
            type: "button",
            class: "tab",
            role: "tab",
            id: `tab-${id}`,
            "aria-selected": String(local.tab === id),
            tabindex: local.tab === id ? "0" : "-1",
            onclick: () => goTo(id),
          },
          label,
          count
            ? el("span", {
                class: v.refused.length ? "count tone-danger" : "count",
                text: String(count),
                "aria-label": v.refused.length
                  ? `${count} need attention`
                  : `${count} waiting`,
              })
            : null,
        ),
      ),
    );
  }

  tabs.addEventListener("keydown", (event) => {
    if (event.key !== "ArrowRight" && event.key !== "ArrowLeft") {
      return;
    }
    const order = ["home", "activity", "settings"];
    const step = event.key === "ArrowRight" ? 1 : -1;
    const next = order[(order.indexOf(local.tab) + step + order.length) % order.length];
    goTo(next);
    const focused = document.getElementById(`tab-${next}`);
    if (focused) {
      focused.focus();
    }
  });

  // First run: where the person is among connect, sign in and scan.
  function drawSteps(v) {
    const current = v.stage === "needsServer" ? 0 : 1;
    const steps = ["Connect", "Sign in", "Scan and print"];
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
          el(
            "span",
            { class: "marker" },
            index < current ? icon("check") : String(index + 1),
          ),
          label,
        ),
      ),
    );
  }

  function welcome(title, text) {
    const logo = document.querySelector(".logo");
    return el(
      "div",
      { class: "welcome" },
      logo ? el("img", { src: logo.getAttribute("src"), alt: "", width: 48, height: 48 }) : null,
      el("h2", { text: title }),
      el("p", { text }),
    );
  }

  function serverForm(v, submitLabel) {
    const locked = v.account.serverLocked;
    const value = local.serverDraft ?? v.account.server ?? CLOUD_ADDRESS;
    return el(
      "form",
      {
        class: "field",
        onsubmit: (event) => {
          event.preventDefault();
          const address = (local.serverDraft ?? v.account.server ?? CLOUD_ADDRESS).trim();
          if (!address) {
            return;
          }
          local.serverDraft = null;
          press("server", () => false, { type: "setServer", address });
        },
      },
      el("label", { for: "server-address", text: "Trenova address" }),
      el(
        "div",
        { class: "row" },
        el("input", {
          id: "server-address",
          class: "grow",
          type: "url",
          inputmode: "url",
          autocomplete: "off",
          spellcheck: "false",
          placeholder: CLOUD_ADDRESS,
          value,
          disabled: locked,
          oninput: (event) => {
            local.serverDraft = event.target.value;
          },
        }),
        button(submitLabel, {
          class: "primary",
          type: "submit",
          disabled: locked || pressed("server"),
        }),
      ),
      locked
        ? el("p", { class: "subtle", text: "Your organization sets this address." })
        : el(
            "p",
            { class: "subtle" },
            "The address you open Trenova at in your browser. Running Trenova on this computer for development? ",
            button(`Use ${DEVELOPMENT_ADDRESS}`, {
              class: "link",
              onclick: () => {
                local.serverDraft = DEVELOPMENT_ADDRESS;
                draw();
              },
            }),
          ),
    );
  }

  function copyCode(code) {
    const done = () => {
      local.copied = true;
      draw();
      setTimeout(() => {
        local.copied = false;
        draw();
      }, 2000);
    };
    if (navigator.clipboard && typeof navigator.clipboard.writeText === "function") {
      navigator.clipboard.writeText(code).then(done, () => {});
    }
  }

  function drawSetup(v) {
    switch (v.stage) {
      case "needsServer":
        return [
          welcome(
            "Welcome to Trenova Capture",
            "Scan and print paper straight into Trenova from this computer. First, tell it where your Trenova is.",
          ),
          drawSteps(v),
          el("section", { class: "panel" }, el("div", { class: "panel-body" }, serverForm(v, "Connect"))),
        ];
      case "signedOut":
        return [
          welcome(
            "Sign in to Trenova",
            "Signing in links this computer to your Trenova account. You approve it once, in your browser.",
          ),
          drawSteps(v),
          el(
            "section",
            { class: "panel" },
            el(
              "div",
              { class: "panel-body" },
              el(
                "div",
                { class: "row" },
                icon("globe"),
                el("span", { class: "grow name", text: v.account.server || "" }),
                v.account.serverLocked
                  ? null
                  : button("Change", { class: "link", onclick: () => goTo("settings") }),
              ),
              button(
                "Sign in",
                {
                  class: "primary large block",
                  disabled: pressed("signIn"),
                  onclick: () =>
                    press("signIn", (next) => next.stage === "signedOut", { type: "signIn" }),
                },
                "lock",
              ),
              v.waiting.length
                ? el("p", {
                    class: "subtle",
                    text: `${pages(v.waiting.reduce((sum, b) => sum + b.pages, 0))} on this computer are sent once you sign in.`,
                  })
                : null,
            ),
          ),
        ];
      case "pairing": {
        const code = v.pairingCode || "";
        return [
          welcome(
            "Approve this computer",
            "Trenova opened in your browser. Check the code there matches this one, then approve it.",
          ),
          drawSteps(v),
          el(
            "section",
            { class: "panel", "aria-label": "Your code" },
            el(
              "div",
              { class: "panel-body" },
              el(
                "div",
                { class: "code", "aria-label": code },
                [...code].map((c) =>
                  el("span", { class: c === "-" ? "sep" : null, "aria-hidden": "true", text: c }),
                ),
              ),
              el(
                "div",
                { class: "waiting" },
                el("span", { class: "spinner", "aria-hidden": "true" }),
                "Waiting for approval",
              ),
              el(
                "div",
                { class: "actions" },
                button(
                  "Open the approval page",
                  { class: "primary grow", onclick: () => send({ type: "openApproval" }) },
                  "external",
                ),
                button(local.copied ? "Copied" : "Copy code", { onclick: () => copyCode(code) }, local.copied ? "check" : "copy"),
              ),
              el(
                "div",
                { class: "actions end" },
                button("Cancel sign-in", {
                  class: "ghost",
                  onclick: () => send({ type: "cancelSignIn" }),
                }),
              ),
            ),
          ),
        ];
      }
      case "connecting":
        return el(
          "section",
          { class: "panel" },
          el(
            "div",
            { class: "empty" },
            el("span", { class: "spinner", "aria-hidden": "true" }),
            el("h2", { text: "Connecting to Trenova" }),
            el("p", { text: v.status.detail || v.account.server || "" }),
          ),
        );
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

  function drawScanCard(v) {
    if (v.scan) {
      return drawScanning(v);
    }
    const again = button("Look again", {
      class: "link",
      disabled: pressed("refresh"),
      onclick: () => press("refresh", () => false, { type: "refreshScanners" }),
    });
    if (!v.scanners.length) {
      return panel(
        "Scan to Intake",
        { iconName: "scan", aside: again },
        el(
          "div",
          { class: "empty" },
          el("h2", { text: "No scanner found" }),
          el("p", { text: "Check the scanner is on and connected to this computer, then look again." }),
        ),
      );
    }
    if (!v.scanners.some((s) => scannerKey(s) === local.scanner)) {
      const preferred = v.scanners.find((s) => s.isDefault) || v.scanners[0];
      local.scanner = scannerKey(preferred);
    }
    if (local.profile !== "" && !v.profiles.some((p) => p.id === local.profile)) {
      const preferred = v.profiles.find((p) => p.isDefault) || v.profiles[0];
      local.profile = preferred ? preferred.id : "";
    }
    const chosen = v.scanners.find((s) => scannerKey(s) === local.scanner);
    const profile = v.profiles.find((p) => p.id === local.profile);
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
        profile
          ? el(
              "div",
              { class: "chips" },
              profile.summary.split(", ").map((part) => el("span", { class: "badge", text: part })),
            )
          : null,
        button(
          "Scan",
          {
            class: "primary large block",
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
    );
  }

  function drawPrintCard(v) {
    if (!v.printing) {
      return null;
    }
    return el(
      "section",
      { class: "panel", "aria-label": "Print to Trenova" },
      el(
        "div",
        { class: "panel-body" },
        el(
          "div",
          { class: "item" },
          el("span", { class: v.printerMissing ? "kind tone-danger" : "kind" }, icon("print")),
          el(
            "div",
            { class: "item-text" },
            el("h2", { text: "Print to Trenova" }),
            el("p", {
              class: "subtle wrap",
              text: v.printerMissing
                ? "The Trenova printer is not set up on this computer yet."
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
                class: "link",
                onclick: () => send({ type: "openLink", link: batch.link }),
              })
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
        aside: button("See all", { class: "link", onclick: () => goTo("activity") }),
      },
      el("ul", { class: "list" }, v.recent.slice(0, RECENT_ON_HOME).map(sentItem)),
    );
  }

  // A page's picture, turned as the person turned it; its number stands in
  // until the picture arrives, or for a page that has none.
  function pagePicture(key, page, rotation, size, label) {
    const src = pictureOf(key, size, page);
    return el(
      "span",
      { class: `sheet rot-${rotation || 0}` },
      src
        ? el("img", { src, alt: label, draggable: "false" })
        : el("span", { class: "sheet-empty", text: String(page) }),
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

  function chosenScan(v) {
    if (!v.scanners.some((s) => scannerKey(s) === local.scanner)) {
      const preferred = v.scanners.find((s) => s.isDefault) || v.scanners[0];
      local.scanner = preferred ? scannerKey(preferred) : null;
    }
    const scanner = v.scanners.find((s) => scannerKey(s) === local.scanner);
    const profile = v.profiles.find((p) => p.id === local.profile);
    return { scanner, profile };
  }

  function drawHeldBatch(v, batch) {
    const editable = batch.editable;
    const all = local.expanded.has(batch.key);
    const shown = all ? batch.pictures : batch.pictures.slice(0, HELD_PAGES_SHOWN);
    want(batch.key, "thumb", shown.map((p) => p.page));
    const unpictured = batch.pages - batch.pictures.length;
    const confirming = local.confirmHeld === batch.key;
    const { scanner, profile } = chosenScan(v);
    return el(
      "section",
      { class: "panel review", "aria-label": `Look over ${batch.label}` },
      el(
        "div",
        { class: "panel-head" },
        icon(batch.kind === "print" ? "print" : "scan"),
        el("h2", { class: "grow name", text: batch.label }),
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
            : el("span", { class: "spinner", "aria-hidden": "true" }),
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

  function drawHome(v) {
    const setup = drawSetup(v);
    if (setup) {
      return setup;
    }
    return [
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
    ];
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
        class: "subtle panel-note",
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
          ? button("Open Intake", { class: "link", onclick: () => send({ type: "openIntake" }) })
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
    if (!sections.length) {
      return el(
        "section",
        { class: "panel" },
        el(
          "div",
          { class: "empty" },
          icon("inbox"),
          el("h2", { text: "Nothing yet" }),
          el("p", { text: "What you scan and print shows here: waiting, sent, or needing a look." }),
        ),
      );
    }
    return sections;
  }

  function initials(name) {
    const parts = (name || "").trim().split(/\s+/).filter(Boolean);
    const letters = parts.length > 1 ? parts[0][0] + parts[parts.length - 1][0] : (parts[0] || "?").slice(0, 2);
    return letters.toUpperCase();
  }

  function drawSettings(v) {
    const account = v.account;
    return [
      signedInStage(v)
        ? null
        : el(
            "div",
            { class: "row" },
            button("Back", { class: "link", onclick: () => goTo("home") }),
          ),
      panel(
        "Account",
        null,
        el(
          "div",
          { class: "panel-body" },
          account.signedIn
            ? el(
                "div",
                { class: "item" },
                el("span", { class: "avatar", "aria-hidden": "true", text: initials(account.person) }),
                el(
                  "div",
                  { class: "item-text" },
                  el("div", { class: "name", text: account.person || "Signed in" }),
                  el("p", { class: "subtle", text: account.organization || "" }),
                ),
                button("Sign out", { onclick: () => send({ type: "signOut" }) }),
              )
            : el(
                "div",
                { class: "row" },
                el("span", { class: "grow muted", text: "Not signed in" }),
                v.stage === "signedOut"
                  ? button("Sign in", { class: "primary", onclick: () => send({ type: "signIn" }) })
                  : null,
              ),
        ),
      ),
      panel("Server", { iconName: "globe" }, el("div", { class: "panel-body" }, serverForm(v, "Save"))),
      panel(
        "Notifications",
        { iconName: "bell" },
        el(
          "div",
          { class: "panel-body" },
          el(
            "label",
            { class: "switch" },
            el("input", {
              id: "routine-notifications",
              type: "checkbox",
              role: "switch",
              checked: v.routineNotifications,
              onchange: (event) =>
                send({ type: "setNotifications", routine: Boolean(event.target.checked) }),
            }),
            el(
              "span",
              { class: "grow" },
              el("span", { class: "name", text: "Tell me when things go well" }),
              el("p", {
                class: "subtle",
                text: "Something sent, the connection back, an update installed. Problems are always shown.",
              }),
            ),
          ),
        ),
      ),
      panel(
        "Before sending",
        { iconName: "review" },
        el(
          "div",
          { class: "panel-body" },
          el(
            "label",
            { class: "switch" },
            el("input", {
              id: "review-before-sending",
              type: "checkbox",
              role: "switch",
              checked: v.reviewBeforeSending,
              disabled: v.reviewLocked,
              onchange: (event) => send({ type: "setReview", on: Boolean(event.target.checked) }),
            }),
            el(
              "span",
              { class: "grow" },
              el("span", { class: "name", text: "Let me look things over before they are sent" }),
              el("p", {
                class: "subtle",
                text: v.reviewLocked
                  ? "Your organization sets this on this computer."
                  : "Scans and prints wait here until you choose Send, so you can turn a page, take one out, or scan more.",
              }),
            ),
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
        null,
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
              text: v.scanners.length ? v.scanners.map((s) => s.title).join(", ") : "None found",
            }),
          ),
          el(
            "label",
            { class: "switch" },
            el("input", {
              id: "test-scanner",
              type: "checkbox",
              role: "switch",
              checked: v.testScanner,
              disabled: v.testScannerLocked,
              onchange: (event) =>
                send({ type: "setTestScanner", on: Boolean(event.target.checked) }),
            }),
            el(
              "span",
              { class: "grow" },
              el("span", { class: "name", text: "Show test scanners" }),
              el("p", {
                class: "subtle",
                text: v.testScannerLocked
                  ? "Your organization has turned these off on this computer."
                  : "Scanners that feed sample paperwork, including a paper jam and a double feed, for trying Trenova Capture without a scanner. What they scan goes to Intake like any scan.",
              }),
            ),
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
      el(
        "div",
        { class: "actions end" },
        button("Quit Trenova Capture", { class: "ghost", onclick: () => send({ type: "quit" }) }, "power"),
      ),
    ];
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
    if (!active || !active.id || !(app.contains(active) || viewerHost.contains(active))) {
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

  function draw() {
    if (!view) {
      return;
    }
    const v = view;
    trackShapes(v);
    prunePending();
    const focus = captureFocus();
    drawMasthead(v);
    if (signedInStage(v)) {
      drawTabs(v);
    } else {
      tabs.replaceChildren();
      if (local.tab === "activity") {
        local.tab = "home";
      }
    }
    if (!signedInStage(v) && local.tab === "settings" && v.stage !== "signedOut") {
      local.tab = "home";
    }
    const content =
      local.tab === "settings" ? drawSettings(v) : local.tab === "activity" ? drawActivity(v) : drawHome(v);
    if (signedInStage(v)) {
      app.setAttribute("role", "tabpanel");
      app.setAttribute("aria-labelledby", `tab-${local.tab}`);
    } else {
      app.removeAttribute("role");
      app.removeAttribute("aria-labelledby");
    }
    app.replaceChildren(...[content].flat(3).filter(Boolean));
    drawViewer(v);
    restoreFocus(focus);
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

  window.trenova = {
    render(next) {
      view = next;
      toastNew(next);
      draw();
    },
    pictures(delivered) {
      if (!delivered || typeof delivered.key !== "string" || !Array.isArray(delivered.pictures)) {
        return;
      }
      for (const picture of delivered.pictures) {
        if (typeof picture.src === "string" && picture.src.startsWith("data:image/jpeg;base64,")) {
          pictures.set(pictureId(delivered.key, delivered.size, picture.page), picture.src);
        }
      }
      draw();
    },
  };

  setInterval(draw, 30000);
  send({ type: "ready" });
})();
