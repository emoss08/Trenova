import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Checkbox } from "@trenova/shared/components/ui/checkbox";
import { Input } from "@trenova/shared/components/ui/input";
import { Switch } from "@trenova/shared/components/ui/switch";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@trenova/shared/components/ui/table";
import { Textarea } from "@trenova/shared/components/ui/textarea";
import { BADGE_ACCENTS, BADGE_TONES } from "@trenova/shared/types/badge";
import type { BadgeAccent, BadgeTone } from "@trenova/shared/types/badge";
import { STATUS_PHASE_TONE, phaseTone } from "@trenova/shared/lib/status-phase";
import type { StatusPhase } from "@trenova/shared/lib/status-phase";
import type { Meta, StoryObj } from "@storybook/react-vite";
import { expect, userEvent, within } from "storybook/test";

/* The token layer, on screen.
 *
 * These stories are the visual reference for the design system and the place a
 * token change shows itself. The a11y addon runs against them, so the contrast
 * claims in docs/engineering/design-system.md are checked rather than asserted:
 * every -subtle-foreground below is rendered on its own -subtle surface.
 *
 * Class names here are written out in full rather than composed. Tailwind reads
 * source as text, so an interpolated `bg-${tone}` generates nothing and the
 * swatch silently renders transparent. */

const meta = {
  title: "Design system/Tokens",
  parameters: {
    layout: "fullscreen",
    docs: {
      description: {
        component:
          "Tones, accents, surfaces, type and focus as the components consume them. See docs/engineering/design-system.md.",
      },
    },
  },
} satisfies Meta;

export default meta;
type Story = StoryObj<typeof meta>;

function Section({
  title,
  hint,
  children,
}: {
  title: string;
  hint?: string;
  children: React.ReactNode;
}) {
  return (
    <section className="flex flex-col gap-3">
      <div>
        <h2 className="text-lg font-semibold tracking-tight">{title}</h2>
        {hint ? <p className="text-foreground-subtle text-xs">{hint}</p> : null}
      </div>
      {children}
    </section>
  );
}

function Grid({ children }: { children: React.ReactNode }) {
  return <div className="flex flex-col gap-8 p-6">{children}</div>;
}

function Swatch({ className, label }: { className: string; label: string }) {
  return (
    <div
      className={`flex h-9 flex-1 items-center justify-center rounded-md text-xs font-medium ${className}`}
    >
      {label}
    </div>
  );
}

/* ------------------------------------------------------------------ tones */

const TONE_SWATCHES: Record<BadgeTone, { solid: string; fg: string; subtle: string; border: string }> =
  {
    neutral: {
      solid: "bg-neutral text-foreground-on-solid",
      fg: "bg-card text-neutral-foreground",
      subtle: "bg-neutral-subtle text-neutral-subtle-foreground",
      border: "bg-neutral-subtle text-neutral-subtle-foreground border border-neutral-border",
    },
    brand: {
      solid: "bg-brand text-brand-foreground",
      fg: "bg-card text-brand",
      subtle: "bg-brand-subtle text-brand-subtle-foreground",
      border: "bg-brand-subtle text-brand-subtle-foreground border border-brand-border",
    },
    info: {
      solid: "bg-info text-foreground-on-solid",
      fg: "bg-card text-info-foreground",
      subtle: "bg-info-subtle text-info-subtle-foreground",
      border: "bg-info-subtle text-info-subtle-foreground border border-info-border",
    },
    success: {
      solid: "bg-success text-foreground-on-solid",
      fg: "bg-card text-success-foreground",
      subtle: "bg-success-subtle text-success-subtle-foreground",
      border: "bg-success-subtle text-success-subtle-foreground border border-success-border",
    },
    warning: {
      solid: "bg-warning text-foreground-on-solid",
      fg: "bg-card text-warning-foreground",
      subtle: "bg-warning-subtle text-warning-subtle-foreground",
      border: "bg-warning-subtle text-warning-subtle-foreground border border-warning-border",
    },
    danger: {
      solid: "bg-danger text-foreground-on-solid",
      fg: "bg-card text-danger-foreground",
      subtle: "bg-danger-subtle text-danger-subtle-foreground",
      border: "bg-danger-subtle text-danger-subtle-foreground border border-danger-border",
    },
  };

export const Tones: Story = {
  render: () => (
    <Grid>
      <Section
        title="Tones"
        hint="Severity. Six tones, five rungs each. Text on a tinted fill uses -subtle-foreground, which the a11y addon checks for contrast."
      >
        <div className="flex flex-col gap-2">
          {BADGE_TONES.map((tone) => {
            const s = TONE_SWATCHES[tone];
            return (
              <div key={tone} className="flex items-center gap-2">
                <span className="text-foreground-subtle w-20 shrink-0 text-xs">{tone}</span>
                <Swatch className={s.solid} label="solid" />
                <Swatch className={s.fg} label="foreground" />
                <Swatch className={s.subtle} label="subtle" />
                <Swatch className={s.border} label="border" />
              </div>
            );
          })}
        </div>
      </Section>
    </Grid>
  ),
};

/* ---------------------------------------------------------------- accents */

const ACCENT_SWATCHES: Record<BadgeAccent, { solid: string; subtle: string }> = {
  "accent-indigo": {
    solid: "bg-accent-indigo",
    subtle:
      "bg-accent-indigo-subtle text-accent-indigo-on-subtle border border-accent-indigo-border",
  },
  "accent-teal": {
    solid: "bg-accent-teal",
    subtle: "bg-accent-teal-subtle text-accent-teal-on-subtle border border-accent-teal-border",
  },
  "accent-amber": {
    solid: "bg-accent-amber",
    subtle: "bg-accent-amber-subtle text-accent-amber-on-subtle border border-accent-amber-border",
  },
  "accent-rose": {
    solid: "bg-accent-rose",
    subtle: "bg-accent-rose-subtle text-accent-rose-on-subtle border border-accent-rose-border",
  },
  "accent-emerald": {
    solid: "bg-accent-emerald",
    subtle:
      "bg-accent-emerald-subtle text-accent-emerald-on-subtle border border-accent-emerald-border",
  },
  "accent-sky": {
    solid: "bg-accent-sky",
    subtle: "bg-accent-sky-subtle text-accent-sky-on-subtle border border-accent-sky-border",
  },
  "accent-violet": {
    solid: "bg-accent-violet",
    subtle:
      "bg-accent-violet-subtle text-accent-violet-on-subtle border border-accent-violet-border",
  },
  "accent-slate": {
    solid: "bg-accent-slate",
    subtle: "bg-accent-slate-subtle text-accent-slate-on-subtle border border-accent-slate-border",
  },
};

export const Accents: Story = {
  render: () => (
    <Grid>
      <Section
        title="Categorical accents"
        hint="Identity, not severity. The same eight hues drive chart series and agent tiles, so a teal series and a teal agent are the same teal."
      >
        <div className="flex flex-col gap-2">
          {BADGE_ACCENTS.map((accent) => (
            <div key={accent} className="flex items-center gap-2">
              <span className="text-foreground-subtle w-32 shrink-0 text-xs">{accent}</span>
              <div className={`h-9 flex-1 rounded-md ${ACCENT_SWATCHES[accent].solid}`} />
              <Swatch className={ACCENT_SWATCHES[accent].subtle} label="on subtle" />
            </div>
          ))}
        </div>
      </Section>
    </Grid>
  ),
};

/* --------------------------------------------------------------- surfaces */

const SURFACES: [string, string][] = [
  ["bg-canvas", "the page ground"],
  ["bg-card", "panels on the canvas"],
  ["bg-sunken", "wells, table headers"],
  ["bg-raised", "popovers, menus"],
  ["bg-overlay", "dialogs, sheets"],
  ["bg-surface-hover", "hover fill"],
  ["bg-surface-active", "active fill"],
  ["bg-surface-selected", "selected fill"],
];


export const Surfaces: Story = {
  render: () => (
    <Grid>
      <Section
        title="Surfaces"
        hint="A ladder, each rung with one job. Three of these held the same value in light mode and three different ones in dark before."
      >
        <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
          {SURFACES.map(([cls, role]) => (
            <div key={cls} className={`border-border rounded-md border p-3 ${cls}`}>
              <div className="text-foreground text-xs font-medium">{cls}</div>
              <div className="text-foreground-subtle text-2xs mt-0.5">{role}</div>
            </div>
          ))}
        </div>
      </Section>
    </Grid>
  ),
};

/* ------------------------------------------------------------------- type */

const STEPS: [string, string][] = [
  ["text-3xs", "9px"],
  ["text-2xs", "10px"],
  ["text-xs", "11px"],
  ["text-sm", "12px"],
  ["text-base", "13px"],
  ["text-lg", "15px"],
  ["text-xl", "17px"],
  ["text-2xl", "20px"],
  ["text-3xl", "24px"],
  ["text-4xl", "30px"],
  ["text-5xl", "36px"],
];

export const TypeScale: Story = {
  render: () => (
    <Grid>
      <Section
        title="Type scale"
        hint="Whole-pixel steps, each carrying its line-height. The old ramp stepped 0.8px at a time, carried no line-heights and hit no round number, so an arbitrary 11px size was written 424 times to get around it."
      >
        <div className="flex flex-col gap-1">
          {STEPS.map(([cls, px]) => (
            <div
              key={cls}
              className="border-border-subtle flex items-baseline gap-4 border-b py-1.5"
            >
              <span className="text-foreground-subtle w-20 shrink-0 text-xs">{cls}</span>
              <span className="text-foreground-subtle w-12 shrink-0 text-xs tabular-nums">{px}</span>
              <span className={cls}>Shipment 4417 departed Laredo at 06:12</span>
            </div>
          ))}
        </div>
      </Section>

      <Section
        title="Numerals"
        hint="font-mono resolves to a real monospace now; it was a proportional sans on 452 call sites, so digits never aligned."
      >
        <div className="flex flex-col gap-1 text-sm">
          <div className="font-mono tabular-nums">1,284.50 &middot; 0000417 &middot; 06:12:44</div>
          <div className="font-mono tabular-nums">9,911.05 &middot; 1180022 &middot; 23:07:01</div>
        </div>
      </Section>
    </Grid>
  ),
};

/* ------------------------------------------------------------------ badge */

export const Badges: Story = {
  render: () => (
    <Grid>
      <Section
        title="Badge"
        hint="A tone or a categorical accent, crossed with an appearance. No variant names a colour."
      >
        {(["subtle", "solid", "outline"] as const).map((appearance) => (
          <div key={appearance} className="flex flex-wrap items-center gap-2">
            <span className="text-foreground-subtle w-16 shrink-0 text-xs">{appearance}</span>
            {BADGE_TONES.map((tone) => (
              <Badge key={tone} variant={tone} appearance={appearance}>
                {tone}
              </Badge>
            ))}
          </div>
        ))}
        <div className="flex flex-wrap items-center gap-2 pt-2">
          <span className="text-foreground-subtle w-16 shrink-0 text-xs">accents</span>
          {BADGE_ACCENTS.map((accent) => (
            <Badge key={accent} variant={accent}>
              {accent.replace("accent-", "")}
            </Badge>
          ))}
        </div>
      </Section>

      <Section
        title="Status phases"
        hint="Status metadata names a lifecycle phase and the tone follows, so a new status cannot pick a colour."
      >
        <div className="flex flex-wrap items-center gap-2">
          {(Object.keys(STATUS_PHASE_TONE) as StatusPhase[]).map((phase) => (
            <Badge key={phase} variant={phaseTone(phase)}>
              {phase} &rarr; {phaseTone(phase)}
            </Badge>
          ))}
        </div>
      </Section>
    </Grid>
  ),
};

/* ------------------------------------------------------------------ focus */

export const Focus: Story = {
  render: () => (
    <Grid>
      <Section
        title="Focus"
        hint="One ring across every control. There were 34 distinct focus treatments across 16 primitives before this."
      >
        <div className="flex flex-wrap items-center gap-3">
          <Button>Button</Button>
          <Button variant="outline">Outline</Button>
          <Button variant="destructive">Destructive</Button>
          <Checkbox aria-label="Checkbox" />
          <Switch aria-label="Switch" />
        </div>
        <div className="flex max-w-sm flex-col gap-3">
          <Input aria-label="Input" placeholder="Focus me" />
          <Input aria-label="Invalid input" aria-invalid placeholder="Invalid: the ring turns red" />
          <Textarea aria-label="Textarea" placeholder="Focus me too" />
        </div>
      </Section>
    </Grid>
  ),
  play: async ({ canvasElement, step }) => {
    const canvas = within(canvasElement);

    await step("focus reaches the first control and draws a ring", async () => {
      const button = canvas.getByRole("button", { name: "Button" });
      await userEvent.tab();
      await expect(button).toHaveFocus();
      // A missing @utility would leave the element focused with no visible
      // indicator and fail nothing else, so assert the rule actually applied.
      await expect(getComputedStyle(button).boxShadow).not.toBe("none");
    });

    await step("an invalid control still draws one", async () => {
      const invalid = canvas.getByLabelText("Invalid input");
      invalid.focus();
      await expect(invalid).toHaveFocus();
      await expect(getComputedStyle(invalid).boxShadow).not.toBe("none");
    });
  },
};

/* --------------------------------------------------------- shape and rhythm */

/* The three things that changed shape rather than colour. A radius story is
   worth having because the collapse from seven values to two is invisible in a
   diff — `rounded-2xl` still reads as `rounded-2xl` at every call site, and only
   the rendered corner shows that it now lands on the surface radius. */

export const Radius: Story = {
  render: () => (
    <Grid>
      <Section
        title="Radius"
        hint="Two values and a pill. rounded-sm/md resolve to --radius-control, rounded-lg through 4xl to --radius-surface."
      >
        <div className="flex flex-wrap items-end gap-4">
          {[
            { cls: "rounded-sm", note: "control" },
            { cls: "rounded-md", note: "control" },
            { cls: "rounded-lg", note: "surface" },
            { cls: "rounded-xl", note: "surface" },
            { cls: "rounded-2xl", note: "surface" },
            { cls: "rounded-full", note: "pill" },
          ].map(({ cls, note }) => (
            <div key={cls} className="flex flex-col items-center gap-1.5">
              <div className={`bg-sunken border-border size-16 border ${cls}`} />
              <code className="text-2xs">{cls}</code>
              <span className="text-foreground-subtle text-3xs uppercase">{note}</span>
            </div>
          ))}
        </div>
      </Section>

      <Section
        title="Weight"
        hint="400 body, 500 label, 600 heading. font-medium was written 1,794 times against 94 font-normal, so nothing could be emphasised by weight."
      >
        <div className="flex flex-col gap-1">
          <p className="text-base font-semibold">Heading — 600, card and section titles</p>
          <p className="text-base font-medium">Label — 500, column heads, field labels, badges</p>
          <p className="text-base font-normal">Body — 400, cells and values</p>
        </div>
      </Section>
    </Grid>
  ),
};

export const Density: Story = {
  render: () => (
    <Grid>
      <Section
        title="Density"
        hint="Row height is a token, not whatever padding a cell carried. The compact table below repoints --row-h; nothing else differs."
      >
        <div className="flex flex-wrap gap-6">
          {[
            { label: "comfortable — --row-h, 30px", cls: "" },
            { label: "compact — --row-h-compact, 26px", cls: "[--row-h:var(--row-h-compact)]" },
          ].map(({ label, cls }) => (
            <div key={label} className="flex flex-col gap-2">
              <span className="text-foreground-subtle text-2xs uppercase">{label}</span>
              <div className="border-border overflow-hidden rounded-lg border">
                <Table className={cls}>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Pro</TableHead>
                      <TableHead>Lane</TableHead>
                      <TableHead>Status</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {[
                      { pro: "118-0224", lane: "Laredo → Dallas", tone: "info" as const, at: "In transit" },
                      { pro: "118-0231", lane: "Dallas → Memphis", tone: "warning" as const, at: "Detained" },
                      { pro: "118-0245", lane: "Memphis → Atlanta", tone: "success" as const, at: "Delivered" },
                    ].map((row) => (
                      <TableRow key={row.pro}>
                        <TableCell className="font-mono tabular-nums">{row.pro}</TableCell>
                        <TableCell>{row.lane}</TableCell>
                        <TableCell>
                          <Badge variant={row.tone}>{row.at}</Badge>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            </div>
          ))}
        </div>
      </Section>
    </Grid>
  ),
};
