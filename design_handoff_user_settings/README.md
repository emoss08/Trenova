# Handoff: User settings (fullscreen dialog)

## About the design files
`design/User settings.html` is a **high-fidelity HTML prototype**. Open it in a browser to see layout, copy, states and behavior. It is a design reference, not code to ship. Recreate it in `client/apps/web` with the existing React, Tailwind, Base UI and field-component patterns.

- `design/fields.jsx` + `fields.css` are stand-ins that copy Trenova's real form components. **Do not port them.** Use the real components in the table below.
- `design/backdrop.jsx` is a fake Shipments page that only shows the dialog opening over the app.
- `design/tweaks-panel.jsx` is prototype-only.

Fidelity: **hifi** for structure, copy and states. Visual values come from `packages/shared/src/styles/tokens.css`.

### Files
- `app.jsx`: the dialog shell (top bar, left nav, scroll-spy, ⌘K setting search, autosave and undo toast, preview toasts) and the host that opens and closes it
- `sections.jsx`: Profile, Preferences, Security
- `notify.jsx`: Notifications
- `avatar.jsx`: gradient avatar generator and picker
- `data.jsx`: notification catalog (mirrors `notification-registry.ts`), sessions, sign-ins, time zones and languages
- `settings.css`, `notify.css`: layout. The token values at the top of `settings.css` are copied from `tokens.css`.

## Layout
- **Dialog:** fullscreen, `position: fixed; inset: 0`. Opens with a 240ms fade + scale (.985 → 1) + 6px rise, easing `cubic-bezier(.16,1,.3,1)`. Closes in 180ms ease-in.
- **Top bar:** 52px tall with a bottom border.
  - Left: logo tile, then "Settings" with "over {current page}" in muted text.
  - Right: "All changes saved" status with a green dot. It flashes "Saved" on a success-subtle pill for 900ms after each change. Then a ghost icon-sm ✕.
- **Body:** grid `248px | 1fr`.
  - **Nav** (12px padding, right border): "Find a setting" field with a ⌘K hint, then four 34px items with a 15px icon. The active item is `bg-muted`/hover, weight 500. Notifications has a "New" brand-subtle pill, or a warning dot while paused. The footer has a ⌘⇧S hint and Log out.
  - **Content:** centered column, max-width 800px, padding 36px 32px 120px, smooth scrolling.
- **Hero:**
  - 64px round gradient avatar. Hovering shows a camera overlay.
  - h1, 28px/500, tracking −0.025em: "{Good morning|afternoon|evening}, *{first name}*." The name is Instrument Serif italic 32px. Pick a serif already in the codebase, or drop the italic.
  - Meta line: title · org · live local time.
  - Right: a status pill (Available green / Focus brand / Away warning) that jumps to Availability.
- **Sections:** h2 20px/500 plus a 13px muted description, then cards.
  - **Cards:** `bg-card border rounded-lg` (8px), 14px gap between cards.
  - **Card header:** 14px/500 title plus a 12px muted line, padding 14px 18px 12px, bottom border.
  - **Rows:** padding 14px 18px with a top border. Sub-rows (revealed by a parent setting) use `bg-muted` (sunken).
- **Responsive:** under 900px the nav hides. Under 620px inline fields stack.

## States to build
- **Setting search** (⌘K): matches labels and every notification type. Arrow keys move, Enter jumps. The target row flashes brand-subtle for 1.6s.
- **Autosave:** each change shows a "✓ Saved · Undo" ink toast, bottom-center, for 4.2s. Undo restores the previous value.
- **Notification preview:** where a test is offered, a toast slides in from the top-right in the real notification-center style.
- **Password change:** expands inline. All four checks must pass and the confirmation must match before Update is enabled.
- **Sessions:** a signed-out row fades and slides out over 300ms.
- **Reduce motion:** setting it to on removes all animation (respect it app-wide).

## What changes
- Settings stays an **app-level dialog**, but it is now **fullscreen**. Open it from anywhere exactly as today: `useAppDialogsStore().openDialog("settings")` from the user menu or command palette, or ⌘⇧S, all through `UserSettingsHost`.
- It is **not a route**. The page underneath stays mounted. Closing (✕, Esc, or ⌘⇧S again) returns you to exactly where you were.
- Use `Dialog`/`DialogContent` with a fullscreen size (`inset-0`, no radius, no backdrop blur on the content itself). Keep the lazy-loaded chunk and skeleton from `user-settings-host.tsx`.
- Optional: `openDialog("settings", { section: "notifications" })` scrolls to a section. Use it from notification-center rows ("Notification settings").
- Four sections on one scrolling page, with a left nav that tracks scroll position: **Profile, Preferences, Notifications, Security**.
- A "Find a setting" search (⌘K while on the page) jumps to and highlights a row.
- Changes save as they're made. A toast says "Saved" and offers **Undo**. There is no Save button. Text fields save on blur.
- No shadows. Cards are `bg-card border rounded-lg` (radius-surface, 8px). Every color comes from `packages/shared/src/styles/tokens.css`.

## Field components: use these, not hand-rolled controls
Every settings row uses `layout="inline"` on the field (the `FieldWrapper` inline grid: label + description on the left, 2fr; control on the right, 3fr).

| UI in the design | Component |
|---|---|
| Display name, Job title, Away note | `InputField` (`@/components/fields/input-field`) |
| Username (read-only, copy button) | `InputField` with `readOnly` + `rightElement={<Button variant="ghost" size="icon-xs">}` |
| Email (read-only, "Verified") | `InputField` `readOnly` + `rightElement={<Badge>}` + `description` |
| Phone | `PhoneNumberField` |
| Status, Focus for, Time format, Sidebar, Desktop mode, Email mode | `SegmentedField` (wraps `SegmentedControl`, `fullWidth` by default) |
| Back on (Away) | `DatePickerField` (outline button, calendar popover, "PPP" format) |
| Hand off to, Language, Pause notifications, Quiet hours from/to | `SelectField` |
| Time zone | `TimezoneField` (grouped, shows the UTC offset as the option description) |
| Reduce motion, Sound, Daily digest, Quiet hours, Let urgent through, each notification type | `SwitchField` (`position="right"`) |
| Category on/off in "What you hear about" | `Switch` (`@trenova/shared/components/ui/switch`) on its own |
| Quiet days | `ToggleChipsField` (neutral tone) |
| Current / new / confirm password | `SensitiveField` |
| All buttons | `Button`: `default` (ink) for primary, `outline` for secondary, `ghost`/`quiet` for tertiary. Sizes `sm`/`xs`/`icon-xs`/`icon-sm` |
| Section/row descriptions | the field's `description` prop (renders the `FieldDescription` from `field-components.tsx`) |

The only custom pieces are the **theme picker cards** (three 72px mini previews: Light / Dark / System split diagonally; the selected card gets a 2px brand ring), the **gradient avatar picker** (32px round swatches; the selected one gets a 2px brand outline with 2px offset), the **session rows**, and the **sign-in log**.

## Notifications (new)
Today nothing that sends notifications can be turned off. This page fixes that. Keep it calm: four cards, no matrix.

1. **Delivery**
   - Desktop: `Off / Urgent only / Everything`. Its description handles browser permission ("Allow notifications" link, or "Send a test" once allowed).
   - Email: `Off / Urgent only / Everything`. When set to Everything, show a sub-row `SwitchField` "Bundle into a daily digest" (8:00 AM, urgent ones still go out immediately).
   - Sound: `SwitchField`.
   - The in-app bell always receives everything. It is not configurable.
2. **Quiet hours.** A `SwitchField`. When on, sub-rows appear: From/To (`SelectField`, 30-minute steps, formatted by the user's time format), Days (`ToggleChipsField`), and "Let urgent ones through" (`SwitchField`).
3. **Pause.** `SelectField`: Not paused / 1 hour / Until 8:00 AM tomorrow / Until Monday. While paused, the nav shows a dot and the section header shows a "Paused" warning badge.
4. **What you hear about.** One row per category from `notification-registry.ts`: icon tile using the registry's tile/icon classes, category name, one-line blurb, a "n of m" quiet button that expands the list, and a `Switch` for the whole category. Expanded, each event type is a `SwitchField`. Urgent types get a `danger` "Urgent" badge.
   - **Required types** (switch disabled, "Required" badge, reason as the description):
     - `ai_audit.chain_mismatch`, for admins
     - System/security notices in the bell

"Urgent" means the descriptors that use the danger tone (e.g. `tender_waterfall_exhausted`, `carrier_intel_block`, `edi.message.dead_lettered`, `credential_expired`, `report_run_failed`).

### Backend needed
- A per-user preferences record with:
  - `disabledEventTypes: string[]`
  - `desktopMode` and `emailMode` (`off|urgent|all`)
  - `emailDigest: bool`
  - `sound: bool`
  - `quietHours {enabled, start, end, days[], allowUrgent}`
  - `pausedUntil: timestamp|null`
- The notification dispatcher must check this record before fanning out to each channel.
- Category grouping comes from the registry's `category` field.

## Profile & availability (new)
- Status: `available | focus | away`.
  - **Focus:** only urgent notifications and direct @mentions get through.
  - **Away:** takes `backOn` (date), `delegateUserId` (receives approvals, tender reviews and @mentions) and `awayNote` (shown to people who @mention the user).
- **Gradient avatars** (see `design/avatar.jsx`). With no photo, `ResolvedUserAvatar` falls back to a mesh gradient instead of a grey initials tile.
  - Default `auto`: three hues taken from an FNV-1a hash of the username, so the avatar is stable and unique, like avatar.vercel.sh. Hues: `a = h % 360`, `b = a + 40 + (h>>9)%50`, `c = a + 300 − (h>>17)%60`.
  - Rendering: four radial gradients (at 18% 22%, 86% 18%, 70% 88%, 12% 92%) over a 135° linear base. Lightness 0.66–0.88, chroma ≈0.17 (Slate 0.06). See `avMesh()`.
  - Avatar is a circle with an inset 1px border at 8% black.
  - The user can instead pick one of 7 presets (Dusk, Lagoon, Citrus, Orchid, Ember, Mint, Slate) or shuffle.
  - Stored as `avatarStyle` (`auto` or a preset id). No initials on the gradient.
  - Use the same fallback everywhere avatars appear (comments, assignments, user menu).

## Preferences
Theme (moves out of the user menu submenu), language, time zone with a live local clock and a "Use this device's" link, time format, sidebar layout, reduce motion.

## Security
- **Password:** an inline expand with a strength meter and live requirement checks.
- **Two-factor:** status, recovery codes (view / copy / regenerate), and turn off. Reuse the logic in `two-factor-section.tsx`.
- **Sessions:** each with device, location, IP and last active, plus per-session sign-out and "Sign out everywhere else".
- **Recent sign-ins:** the last 5.
