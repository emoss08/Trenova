# Implement the Desk memory card redesign

Read `README.md` and open `design/Memory cards.html` in a browser before you change any code. The HTML is a reference only; rebuild the design in the existing code.

## Scope
Edit these files in place. Do not add parallel components.
- `client/apps/web/src/components/desk-chat/memory/desk-memory-notes.tsx`: `DeskMemorySaved`, for the ask, saved, edit, removed, declined and replaced states.
- `client/apps/web/src/components/desk-chat/memory/desk-memory.css`: the `.dk-mem*` conversation styles. Leave the Memory page `.dk-mm-*` styles alone.
- `client/apps/web/src/components/desk-chat/memory/memory-format.ts`: add `scopeWord()` and `splitSteps()`.
- `client/apps/web/src/components/desk-chat/memory/desk-memory-why.tsx`: restyle only, keeping its API.

Do not touch `DeskMemoryRecall` ("Used N memories").

## Steps
1. **`memory-format.ts`**
   - Add `scopeWord(memory, t)`: returns "just you", the role name or "everyone". Write it in the same style as `scopeLabel`.
   - Add `splitSteps(content)`: returns `string[]`. Split on newlines or on `/\s*\d+\.\s+/`, and drop empty strings. Add tests next to the existing `memory-format.test.ts`.
2. **Shared parts** inside `desk-memory-notes.tsx` (small and private, single responsibility):
   - `MemoryHeader`: the icon, title and action slots.
   - `MemorySteps`: the numbered list inside `<ScrollArea>` (`@trenova/shared/components/ui/scroll-area`), `max-height: 172px`, with edge fades. It switches to an auto-sized textarea when editing; the textarea has the card background and no border or ring.
   - `MemoryNote`: the note line plus the Why disclosure, wrapping `DeskMemoryWhy`.
   - `ScopeWord`: the inline word that triggers the shared `Popover` scope radiogroup.
   Both cards are built from these four parts, so don't duplicate markup between them.
3. **Ask card**: one sentence, "Keep these steps for {who}?", with the actions "Not now" (ghost) and "Keep" (ink). There is no chevron; the word is the trigger. Keep calls `confirmDeskMemory` with the chosen audience, and Not now calls `dismissDeskMemory`. Both collapse the body and swap the header to "Kept for {who} · Desk will follow these next time" or "Not kept", with Undo.
4. **Saved card**:
   - Header: "Learned steps" followed by "{Who} · {day}".
   - Edit and Undo are icon buttons that appear on hover or focus. Edit edits in place and saves through `reviseDeskMemory`; Undo runs the existing remove.
   - Note line: "Replaces earlier steps · Why", or "Saved {day} from “{source}” · Why".
5. **Styles**
   - Replace the old `.dk-mem-card`, `.dk-mem-done`, `.dk-mem-seg`, `.dk-mem-save` and `.dk-mem-f` rules with the new ones, using only `--dsk-*`, `--accent-violet*` and `--lift-float`.
   - Point `--dk-mem` at `var(--accent-violet)`.
   - Use 6px and 8px radii, no shadows on the card, and the motion values in the README. Respect `prefers-reduced-motion`.
6. **i18n**: wrap every new string in `t()`.
7. **Tests**: update the `desk-memory-notes` tests to cover:
   - the inline scope word opening the popover and changing the audience
   - Keep, Not now and Undo
   - editing in place with Enter and Esc
   - the Why toggle
   - long content rendering inside `ScrollArea`

## Principles
- **DRY**: the four shared parts and the format helpers are the only places their logic lives.
- **SOLID**: each part does one thing. Mutations stay in `DeskMemorySaved`, and the parts receive data and callbacks as props.
- **Remove dead code**: once nothing references the old segmented "Visible to" control, its CSS, and the separate `edit` render branch, delete them.
- Run `pnpm lint`, `pnpm typecheck` and the desk-chat tests before finishing.
