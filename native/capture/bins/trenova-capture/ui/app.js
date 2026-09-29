(() => {
  "use strict";

  const app = document.getElementById("app");
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
  };
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
    return v.refused.length + v.waiting.length;
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
        el("p", {
          class: "subtle",
          text: scan.stopping
            ? "The pages scanned so far will be sent."
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

  function drawHome(v) {
    const setup = drawSetup(v);
    if (setup) {
      return setup;
    }
    return [
      drawNotices(v),
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
    const sections = [drawRefused(v), drawWaiting(v), drawSent(v), drawMessages(v)].filter(Boolean);
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
    if (!active || !active.id || !app.contains(active)) {
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
    restoreFocus(focus);
  }

  document.addEventListener("keydown", (event) => {
    if (event.key !== "Escape") {
      return;
    }
    if (local.confirmDiscard) {
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
  };

  setInterval(draw, 30000);
  send({ type: "ready" });
})();
