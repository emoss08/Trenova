Implement the new **User Settings** in the Trenova web client (`client/apps/web`).

Read `design_handoff_user_settings/README.md` in full, then open `design_handoff_user_settings/design/User settings.html` in a browser. That file is an HTML design reference: copy its structure, copy and behavior, but rebuild it with the codebase's own components. Don't port the prototype code.

## Requirements
1. **A fullscreen dialog, not a page.** Keep Settings as the app-level dialog opened through `UserSettingsHost`, `useAppDialogsStore().openDialog("settings")` and ⌘⇧S, so it opens over any page.
   - Rebuild `components/navigation/user-settings-dialog.tsx` as a fullscreen `Dialog`.
   - Do not add a route.
   - ✕, Esc and ⌘⇧S close it and return the user to where they were.
   - Support `openDialog("settings", { section })` to scroll to a section.
2. **Use the existing field components only**, with `layout="inline"` on every settings row. No raw `<input>`, `<select>` or custom toggles.
   - Fields from `@/components/fields/*`: `InputField`, `PhoneNumberField`, `SensitiveField`, `SelectField`, `TimezoneField`, `DatePickerField`, `SegmentedField`, `SwitchField`, `ToggleChipsField`
   - From `@trenova/shared/components/ui/*`: `Switch`, `SegmentedControl`, `Button`, `Badge`
   - Descriptions go through each field's `description` prop.
   - The README's table maps every control to its component.
3. **Styling:** use the `tokens.css` utilities only. No shadows. Cards are `bg-card border rounded-lg`. Support light and dark.
4. **Autosave:** every change saves immediately, with a sonner "Saved · Undo" toast. Text fields save on blur.
5. **Notification preferences (new; backend + frontend):**
   - Add a per-user preferences model and API with the fields in the README's "Backend needed".
   - Enforce it in the notification dispatcher for the desktop and email channels, including quiet hours, pause and the urgent-only modes. The in-app bell always receives everything.
   - Build the "What you hear about" list from `notification-registry.ts`, grouped by `category`.
   - Lock the required types the README lists.
   - Keep the four calm cards: Delivery, Quiet hours, Pause, What you hear about. No channel matrix.
6. **Profile additions:**
   - Availability (Available / Focus / Away, with back-on date, hand-off person and away note).
   - Gradient avatar fallback (algorithm in the README). Use it in `ResolvedUserAvatar` everywhere.
7. **Security:**
   - Reuse the `two-factor-section.tsx` logic.
   - Add an inline password change with strength checks.
   - Add a sessions list with per-session sign-out and "Sign out everywhere else", and recent sign-ins. Add API endpoints if they're missing.

Work in this order:
1. Dialog shell and nav
2. Profile and Preferences
3. Notification preferences backend
4. Notifications UI
5. Security

Run typecheck and lint when you're done. Placeholder numbers in the mock (counts, the sessions list) must come from real data.
