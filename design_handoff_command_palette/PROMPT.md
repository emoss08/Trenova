Implement the redesigned **command palette** in the Trenova web client (`client/apps/web/src/components/command-palette/`).

Read `design_handoff_command_palette/README.md` in full, then open `design_handoff_command_palette/design/Command Palette.html` in a browser. Try:
- "ai"
- "acme"
- "leon"
- "bol-88213"
- "bq"
- "@doc"
- ">theme"
- "any loads at risk today?" then Enter
- ⌘I on a record
- ⌘K on any row

Use the Tweaks panel to check light and dark.

That file is an HTML design reference: copy its structure, copy and behavior, but rebuild it with the codebase's own components. Don't port the prototype code. In particular:
- `logic.jsx` `novaAnswer` is canned. Answers, their steps and their record references come from `useAsk` and the assistant agent.
- `data.jsx` is fixtures. Every count, amount, ETA, HOS value and extraction comes from real queries.
- `app.jsx` is a stand-in app shell. Ignore it.

## Requirements
1. **Same component, same data.**
   - Rebuild `command-palette.tsx` and its parts in place.
   - Keep `CommandDialog` with `shouldFilter={false}`, the store, `usePaletteCatalog` / `usePaletteHome` / `usePaletteRemoteSearch` / `usePinnedPages`, `usePaletteIntentRunner` and the `PaletteIntent` model.
   - Nothing is fetched until the palette opens.
2. **Matching and highlighting** (README, "Matching and highlighting"):
   - Replace substring highlighting with the matcher in the README:
     - a term matches the start of a field, the start of a word, or a word's initials;
     - only 3+ character terms may match inside a word.
   - It returns the winning field and the character ranges. `MatchText` renders only those ranges, only on that field.
   - Matches through hidden fields (keywords, driver) show the match-source tag.
   - Add tests to `fuzzy-score` and `palette-sections.test.ts`. "ai" must not match "Service failures", "Trailers" or "Email logs". It must match "AI control", "Aisha Grant" and "Document intelligence" (by keyword).
3. **Layout:**
   - Dialog `min(1080px,100%)` × `min(680px,100vh−12vh)`, body `minmax(0,1fr) | 420px`.
   - Below 900px the preview is 340px; below 780px it hides.
   - Rows are 46px (36px single-line) with 28px tiles.
   - The selection is one sliding fill. **No accent bars anywhere**: remove the `before:bg-brand` bar.
4. **Restrained color** (README, "Color: restrained"):
   - Neutral tiles for every type, so drop the categorical tile accents in `palette-entities.ts`.
   - Outlined status labels with a state dot.
   - Neutral count chips.
   - The highlight is our yellow `#ffa31a` (the `highlight` token) as a fill (26% dark / 32% light), not an underline.
   - The only gradients are Nova's ring mark and the Triage spectrum border.
5. **Preview pane** (README, "Preview pane"): rebuild every preview in the Desk artifact vocabulary (`desk-art-kinds.tsx`):
   - source line, 22px title with a status label, key/value rows, mono section headings, related-record rows;
   - an action bar with an ink primary button, Ask, and copy link / new tab / all actions icon buttons.
   - **Shipment:** the Desk route card replaces the Google map.
   - **Customer:** stat strip, aging bar, open loads.
   - **Worker:** HOS strip, current load, compliance dates.
   - **Document:** first-page thumbnail, Nova's extracted fields with confidence, attached record.
   - **Page:** live glance counts, related create commands, recent records of that type.
   - Previews keep the 90ms settle and the prefetch of the first record.
6. **Home:**
   - Sections: Jump back in (with relative times) · Needs you · Pinned · Quick actions.
   - With an empty query, a scope browses that type (Recent first). Commands browse grouped.
7. **Actions** (README, "Actions menu"):
   - ⌘K or → opens a filterable actions popover over the preview, not a replacement list.
   - Extend `palette-actions.ts` with the README's per-type actions. Each shortcut works from the result list without opening the menu: ⌘↵ new tab, ⌘L copy link, ⌥C copy ID, ⌘I ask, ⌘P pin.
   - Copy, pin and theme keep the palette open and confirm in the footer.
8. **Nova** (README, "Nova"):
   - `?` or a question-word query gets an Ask row first. Every non-empty search ends with an Ask row.
   - The answer card shows the turn's tool steps, then streams the answer.
   - Cited entities render as inline record chips. **Hovering a chip previews the record in the right pane**; clicking opens it.
   - Done actions: Continue in Desk (⌘↵) · Copy · Follow up, on one line.
   - After an answer with nothing selected, the preview lists the records Nova read.
   - Asking about a record (⌘I) puts a subject chip in the input and offers 3 starter questions per type.
   - Every amount and fact in an answer must come from tools, never from the model.
9. **Keyboard and Esc:**
   - Esc steps back one layer at a time: @ popover → actions → answer → subject → query → scope → close.
   - Tab cycles scopes.
   - `>` is command mode.
   - The `@type` popover commits as a chip.
   - Footer hints follow the selection.
10. **Styling:**
    - Use the `tokens.css` utilities only. Primary buttons are `bg-primary` (ink), not brand blue.
    - Key caps are `Kbd`. Every scrolling region is a `ScrollArea`.
    - The Nova ring is `AssistMark`, with a 6px gap to its label, optically centered.
    - Support light and dark.

Work in this order:
1. Matcher and highlighting (with tests)
2. Layout, rows, color pass and selection
3. Preview pane per type
4. Actions menu and shortcuts
5. Home and empty states
6. Nova card, subject mode and citation hover

Run typecheck, lint and the existing command-palette tests when you're done (`palette-sections`, `palette-actions`, `palette-commands`, `use-palette-intents`, `command-palette`). Update their fixtures for the new sections and actions. Add the new strings to the command-palette locale files for es, zh-CN and zh-TW. Placeholder numbers in the mock (attention counts, glance counts, HOS, aging, extraction confidence) must come from real data, and hide a section when its data is missing.
