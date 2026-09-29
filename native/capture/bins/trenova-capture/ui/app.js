(() => {
  "use strict";

  const app = document.getElementById("app");
  const who = document.getElementById("who");
  const settingsToggle = document.getElementById("settings-toggle");

  // What belongs to this window alone and survives a redraw: choices made,
  // text being typed, a question being asked, and buttons already pressed.
  const local = {
    settings: false,
    scanner: null,
    profile: null,
    serverDraft: null,
    confirmDiscard: null,
    messagesOpen: false,
    pending: new Map(),
  };
  let view = null;

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

  function button(label, props) {
    return el("button", { type: "button", ...props }, label);
  }

  function panel(title, count, aside, ...body) {
    return el(
      "section",
      { class: "panel", "aria-label": title },
      el(
        "div",
        { class: "panel-head" },
        el("h2", { text: title }),
        count ? el("span", { class: "count", text: String(count) }) : null,
        aside ? el("div", { class: "aside" }, aside) : null,
      ),
      ...body,
    );
  }

  function kindLabel(kind) {
    return kind === "print" ? "Print" : "Scan";
  }

  function drawStatus(v) {
    return el(
      "div",
      { class: `status tone-${v.status.tone}`, role: "status" },
      el("span", { class: "dot", "aria-hidden": "true" }),
      el(
        "div",
        null,
        el("h2", { text: v.status.title }),
        v.status.detail ? el("p", { text: v.status.detail }) : null,
      ),
    );
  }

  function serverForm(v, submitLabel) {
    const locked = v.account.serverLocked;
    const value = local.serverDraft ?? v.account.server ?? "";
    const form = el(
      "form",
      {
        class: "field",
        onsubmit: (event) => {
          event.preventDefault();
          const address = (local.serverDraft ?? v.account.server ?? "").trim();
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
          placeholder: "https://tms.example.com",
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
        : el("p", {
            class: "subtle",
            text: "The address you open Trenova at in your browser.",
          }),
    );
    return form;
  }

  function drawStage(v) {
    switch (v.stage) {
      case "needsServer":
        return el("section", { class: "panel" }, el("div", { class: "panel-body" }, serverForm(v, "Connect")));
      case "signedOut":
        return el(
          "section",
          { class: "panel" },
          el(
            "div",
            { class: "panel-body" },
            el("p", {
              class: "muted",
              text: "Signing in links this computer to your Trenova account. You approve it once, in your browser.",
            }),
            el(
              "div",
              { class: "actions" },
              button("Sign in", {
                class: "primary",
                disabled: pressed("signIn"),
                onclick: () =>
                  press("signIn", (next) => next.stage === "signedOut", { type: "signIn" }),
              }),
            ),
          ),
        );
      case "pairing":
        return el(
          "section",
          { class: "panel", "aria-label": "Approve this computer" },
          el(
            "div",
            { class: "panel-body" },
            el("p", { class: "muted", text: "Your code" }),
            el("div", { class: "code", text: v.pairingCode || "" }),
            el(
              "div",
              { class: "actions" },
              button("Open the approval page", {
                class: "primary",
                onclick: () => send({ type: "openApproval" }),
              }),
              button("Cancel", {
                class: "ghost",
                onclick: () => send({ type: "cancelSignIn" }),
              }),
            ),
          ),
        );
      default:
        return null;
    }
  }

  function drawCallouts(v) {
    const callouts = [];
    if (v.printerMissing) {
      callouts.push(
        el(
          "div",
          { class: "callout tone-warning row" },
          el("span", {
            class: "grow",
            text: "The Trenova printer is not set up on this computer, so printing into Trenova is not available.",
          }),
          button("Add the printer", {
            disabled: pressed("printer"),
            onclick: () => press("printer", (next) => next.printerMissing, { type: "addPrinter" }),
          }),
        ),
      );
    }
    if (v.update) {
      const { version, status } = v.update;
      const text = {
        available: `Trenova Capture ${version} is ready to install.`,
        askAdministrator: `Trenova Capture ${version} is available. Your administrator installs updates.`,
        installing: `Installing Trenova Capture ${version}. It restarts by itself.`,
        windowsTooOld: `Trenova Capture ${version} needs a newer version of Windows.`,
      }[status];
      const action =
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
            ? button("Download", { onclick: () => send({ type: "openDownload" }) })
            : null;
      callouts.push(
        el("div", { class: "callout tone-info row" }, el("span", { class: "grow", text }), action),
      );
    } else if (v.updateRequired) {
      callouts.push(
        el("div", {
          class: "callout tone-warning",
          text: `Your organization needs Trenova Capture ${v.updateRequired} or later.`,
        }),
      );
    }
    return callouts;
  }

  function drawScan(v) {
    if (!v.scan) {
      return null;
    }
    const scan = v.scan;
    return panel(
      scan.stopping ? "Stopping the scan" : "Scanning",
      null,
      null,
      el(
        "div",
        { class: "panel-body" },
        el(
          "div",
          { class: "row" },
          el(
            "div",
            { class: "grow" },
            el("div", { class: "big-count", text: String(scan.pages) }),
            el("p", {
              class: "muted",
              text: `${scan.pages === 1 ? "page" : "pages"} from ${scan.label}${
                scan.requested ? ", for the record you chose" : ", to Intake"
              }`,
            }),
          ),
          button(scan.stopping ? "Stopping" : "Stop", {
            disabled: scan.stopping || pressed("stop"),
            onclick: () =>
              press("stop", (next) => next.scan && !next.scan.stopping, { type: "stopScan" }),
          }),
        ),
        scan.stopping
          ? el("p", { class: "subtle", text: "The pages scanned so far will be sent." })
          : null,
      ),
    );
  }

  function drawPaused(v) {
    return v.paused.map((paused) =>
      panel(
        "The scanner stopped",
        null,
        null,
        el(
          "div",
          { class: "panel-body" },
          el("p", {
            text: `${pages(paused.pages)} from ${paused.label} are kept. ${paused.problem}`,
          }),
          el(
            "div",
            { class: "actions" },
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
        ),
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
      v.refused.length,
      null,
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
              { class: "row" },
              el("span", { class: "badge", text: kindLabel(batch.kind) }),
              el("span", { class: "name grow", text: batch.label }),
              el("span", { class: "subtle numeric", text: batch.pages ? pages(batch.pages) : "" }),
              when(batch.refusedAt),
            ),
            batch.reason
              ? el("div", { class: "callout tone-danger", text: batch.reason })
              : null,
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
                    ? button("Send again", {
                        class: "primary",
                        disabled: pressed(`retry:${batch.key}`),
                        onclick: () =>
                          press(`retry:${batch.key}`, present(batch.key), {
                            type: "retry",
                            key: batch.key,
                          }),
                      })
                    : null,
                  batch.readable
                    ? button("Save a copy", {
                        disabled: pressed(`save:${batch.key}`),
                        onclick: () =>
                          press(`save:${batch.key}`, () => true, { type: "save", key: batch.key }),
                      })
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

  function scannerKey(scanner) {
    return `${scanner.protocol}:${scanner.name}`;
  }

  function drawScanToIntake(v) {
    if (!v.account.signedIn || v.stage === "blocked") {
      return null;
    }
    const again = button("Look again", {
      class: "link",
      disabled: pressed("refresh"),
      onclick: () => press("refresh", () => false, { type: "refreshScanners" }),
    });
    if (!v.scanners.length) {
      return panel(
        "Scan to Intake",
        null,
        again,
        el("p", {
          class: "empty",
          text: "No scanners found. Check the scanner is on and connected, then look again.",
        }),
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
      null,
      again,
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
        profile ? el("p", { class: "subtle", text: profile.summary }) : null,
        el(
          "div",
          { class: "actions" },
          button("Scan", {
            class: "primary",
            disabled: !v.canScan || !chosen || pressed("scan"),
            onclick: () =>
              press("scan", (next) => next.canScan, {
                type: "scan",
                scanner: chosen.name,
                protocol: chosen.protocol,
                profile: profile ? profile.id : null,
              }),
          }),
          v.canOpenIntake
            ? button("Open Intake", { class: "ghost", onclick: () => send({ type: "openIntake" }) })
            : null,
        ),
      ),
    );
  }

  function drawWaiting(v) {
    if (!v.waiting.length) {
      return null;
    }
    return panel(
      "Waiting to send",
      v.waiting.length,
      null,
      el(
        "ul",
        { class: "list" },
        v.waiting.map((batch) =>
          el(
            "li",
            null,
            el(
              "div",
              { class: "row" },
              el("span", { class: "badge", text: kindLabel(batch.kind) }),
              el("span", { class: "name grow", text: batch.label }),
              el("span", { class: "subtle numeric", text: pages(batch.pages) }),
            ),
            el("p", { class: "subtle", text: `${batch.state} · spooled ${ago(batch.createdAt)}` }),
          ),
        ),
      ),
    );
  }

  function drawRecent(v) {
    if (!v.recent.length) {
      return null;
    }
    return panel(
      "Recently sent",
      null,
      v.canOpenIntake
        ? button("Open Intake", { class: "link", onclick: () => send({ type: "openIntake" }) })
        : null,
      el(
        "ul",
        { class: "list" },
        v.recent.map((batch) =>
          el(
            "li",
            null,
            el(
              "div",
              { class: "row" },
              el("span", { class: "name grow", text: batch.label }),
              el("span", { class: "subtle numeric", text: pages(batch.pages) }),
              when(batch.at),
              batch.link
                ? button("Open", {
                    class: "link",
                    onclick: () => send({ type: "openLink", link: batch.link }),
                  })
                : null,
            ),
            el("p", {
              class: "subtle",
              text: batch.requested ? "Filed where you asked" : "Waiting in Intake",
            }),
          ),
        ),
      ),
    );
  }

  function drawMessages(v) {
    if (!v.messages.length) {
      return null;
    }
    return el(
      "details",
      {
        class: "panel",
        open: local.messagesOpen,
        ontoggle: (event) => {
          local.messagesOpen = event.target.open;
        },
      },
      el("summary", null, "Messages ", el("span", { class: "subtle", text: String(v.messages.length) })),
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

  function drawSettings(v) {
    const account = v.account;
    return [
      panel("Server", null, null, el("div", { class: "panel-body" }, serverForm(v, "Save"))),
      panel(
        "Account",
        null,
        null,
        el(
          "div",
          { class: "panel-body" },
          el("p", {
            text: account.signedIn
              ? [account.person, account.organization].filter(Boolean).join(", ") || "Signed in"
              : "Not signed in",
          }),
          el(
            "div",
            { class: "actions" },
            account.signedIn
              ? button("Sign out", { onclick: () => send({ type: "signOut" }) })
              : v.stage === "signedOut"
                ? button("Sign in", { class: "primary", onclick: () => send({ type: "signIn" }) })
                : null,
          ),
        ),
      ),
      panel(
        "This computer",
        null,
        null,
        el(
          "div",
          { class: "panel-body" },
          el("p", { class: "muted", text: `Trenova Capture ${v.version}` }),
          el(
            "div",
            { class: "actions" },
            button("Look for scanners again", {
              disabled: !account.signedIn || pressed("refresh"),
              onclick: () => press("refresh", () => false, { type: "refreshScanners" }),
            }),
            button("Open the log folder", { onclick: () => send({ type: "openLogs" }) }),
          ),
        ),
      ),
      el(
        "div",
        { class: "actions" },
        button("Quit Trenova Capture", { class: "ghost", onclick: () => send({ type: "quit" }) }),
      ),
    ];
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
    const account = v.account;
    who.textContent = account.signedIn
      ? [account.person, account.organization].filter(Boolean).join(" · ") || "Signed in"
      : account.server || "Not connected";
    settingsToggle.setAttribute("aria-expanded", String(local.settings));

    const content = local.settings
      ? [
          el(
            "div",
            { class: "row" },
            button("Back", {
              class: "link",
              onclick: () => {
                local.settings = false;
                draw();
              },
            }),
          ),
          drawSettings(v),
        ]
      : [
          v.scan && v.stage === "online" ? null : drawStatus(v),
          drawStage(v),
          drawCallouts(v),
          drawScan(v),
          drawPaused(v),
          drawRefused(v),
          drawScanToIntake(v),
          drawWaiting(v),
          drawRecent(v),
          drawMessages(v),
          el("div", { class: "footer" }, el("span", { text: `Version ${v.version}` })),
        ];
    app.replaceChildren(...content.flat().filter(Boolean));
    restoreFocus(focus);
  }

  settingsToggle.addEventListener("click", () => {
    local.settings = !local.settings;
    draw();
  });

  document.addEventListener("keydown", (event) => {
    if (event.key !== "Escape") {
      return;
    }
    if (local.confirmDiscard) {
      local.confirmDiscard = null;
      draw();
    } else if (local.settings) {
      local.settings = false;
      draw();
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
      draw();
    },
  };

  setInterval(draw, 30000);
  send({ type: "ready" });
})();
