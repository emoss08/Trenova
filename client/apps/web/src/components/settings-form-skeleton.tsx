import {
  CheckboxFieldSkeleton,
  ImageFieldSkeleton,
  InputFieldSkeleton,
  SwitchFieldSkeleton,
  TextareaFieldSkeleton,
} from "@/components/fields/field-skeletons";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@trenova/shared/components/ui/card";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { ReactNode } from "react";

/**
 * The loading state of a settings page: stacked `Card`s of fields, described section for section
 * by the page so the first paint is the form's own shape. Each page keeps that description next
 * to itself (`skeleton.tsx`); this module only draws it. It is imported statically by the page,
 * so it must stay free of the form, its queries and react-hook-form.
 */

/**
 * `input` is any single-line control (`InputField`, `NumberField`, `SelectField`, an
 * autocomplete, a segmented control); `checkbox` is a `CheckboxField`; `image` is an upload with
 * its preview; `callout` is an inline `Alert` the form always shows among its fields.
 */
export type SettingsFieldKind = "input" | "textarea" | "switch" | "checkbox" | "image" | "callout";

export type SettingsFieldSpec =
  | SettingsFieldKind
  | {
      kind: SettingsFieldKind;
      /** How many of this field follow one another. */
      count?: number;
      /** Spans every column of its group, as `FormControl cols="full"`. */
      full?: boolean;
      /** Overrides the section's `narrow` for this field. */
      narrow?: boolean;
      /** Indented under the switch it depends on, as `FormControl className="pl-10"`. */
      nested?: boolean;
      /** A bordered option, as a checkbox drawn in an outlined row. */
      outlined?: boolean;
    };

export type SettingsColumns = 1 | 2 | 3;

export type SettingsGroupSpec = {
  cols?: SettingsColumns;
  fields: readonly SettingsFieldSpec[];
  /**
   * Draws the group once per record (a cost category, say), each under a heading of its name,
   * a badge and a figure at the end, ruled from the next.
   */
  rows?: number;
};

export type SettingsSectionSpec = {
  /** The width of the card title. */
  title?: "sm" | "md" | "lg";
  /** Lines of card description; 0 for none. */
  description?: number;
  /** The header is ruled off from the content, as `CardHeader className="border-b"`. */
  ruled?: boolean;
  /** `prose` keeps the content to a reading measure, as `CardContent className="max-w-prose"`. */
  width?: "prose" | "full";
  /** Single-line controls are capped as `FormControl className="max-w-[420px]"` caps them. */
  narrow?: boolean;
  /** The side switches sit on, as `SwitchField`'s `position`. */
  switches?: "left" | "right";
  /** Columns of `fields`; ignored when `groups` is given. */
  cols?: SettingsColumns;
  fields?: readonly SettingsFieldSpec[];
  /** Several `FormGroup`s in one card, in order. */
  groups?: readonly SettingsGroupSpec[];
};

const columnClasses: Record<SettingsColumns, string> = {
  1: "grid-cols-1",
  2: "grid-cols-1 md:grid-cols-2",
  3: "grid-cols-1 md:grid-cols-2 lg:grid-cols-3",
};

const titleClasses: Record<NonNullable<SettingsSectionSpec["title"]>, string> = {
  sm: "w-24",
  md: "w-40",
  lg: "w-56",
};

type FieldContext = {
  narrow: boolean;
  switches: "left" | "right";
};

type ResolvedField = {
  kind: SettingsFieldKind;
  full: boolean;
  narrow: boolean;
  nested: boolean;
  outlined: boolean;
};

function resolveFields(
  fields: readonly SettingsFieldSpec[],
  context: FieldContext,
): ResolvedField[] {
  const resolved: ResolvedField[] = [];
  for (const field of fields) {
    const spec: Exclude<SettingsFieldSpec, SettingsFieldKind> =
      typeof field === "string" ? { kind: field } : field;
    const entry: ResolvedField = {
      kind: spec.kind,
      full: spec.full ?? false,
      narrow: spec.narrow ?? (context.narrow && spec.kind === "input"),
      nested: spec.nested ?? false,
      outlined: spec.outlined ?? false,
    };
    for (let index = 0; index < (spec.count ?? 1); index += 1) {
      resolved.push(entry);
    }
  }
  return resolved;
}

function CalloutSkeleton() {
  return (
    <div className="rounded-lg border px-3 py-2.5">
      <span className="flex h-5 items-center">
        <Skeleton className="h-3.5 w-3/4" />
      </span>
    </div>
  );
}

function FieldSkeleton({ field, switches }: { field: ResolvedField; switches: "left" | "right" }) {
  switch (field.kind) {
    case "switch":
      return <SwitchFieldSkeleton position={switches} />;
    case "textarea":
      return <TextareaFieldSkeleton />;
    case "checkbox":
      return <CheckboxFieldSkeleton outlined={field.outlined} />;
    case "image":
      return <ImageFieldSkeleton />;
    case "callout":
      return <CalloutSkeleton />;
    case "input":
      return <InputFieldSkeleton />;
  }
}

function GroupSkeleton({ group, context }: { group: SettingsGroupSpec; context: FieldContext }) {
  const fields = resolveFields(group.fields, context);

  return (
    <div className={cn("grid gap-x-4 gap-y-2", columnClasses[group.cols ?? 1])}>
      {fields.map((field, index) => (
        <div
          key={index}
          data-slot="field-skeleton"
          data-kind={field.kind}
          className={cn(
            field.full && "col-span-full",
            field.narrow && "max-w-105",
            field.nested && "pl-10",
          )}
        >
          <FieldSkeleton field={field} switches={context.switches} />
        </div>
      ))}
    </div>
  );
}

function RecordRowsSkeleton({
  group,
  context,
}: {
  group: SettingsGroupSpec;
  context: FieldContext;
}) {
  return (
    <div className="flex flex-col">
      {Array.from({ length: group.rows ?? 1 }, (_, row) => (
        <div
          key={row}
          data-slot="record-row-skeleton"
          className={cn("flex flex-col gap-3 py-4", row < (group.rows ?? 1) - 1 && "border-b")}
        >
          <div className="flex items-center justify-between gap-2">
            <div className="flex items-center gap-2">
              <Skeleton className="h-4 w-32" />
              <Skeleton className="h-4 w-16 rounded-badge" />
            </div>
            <Skeleton className="h-4 w-24" />
          </div>
          <GroupSkeleton group={group} context={context} />
        </div>
      ))}
    </div>
  );
}

/** One settings card: its title, description and fields. Draws no loading region of its own. */
export function SettingsSectionSkeleton({ section }: { section: SettingsSectionSpec }) {
  const groups = section.groups ?? [{ cols: section.cols, fields: section.fields ?? [] }];
  const context: FieldContext = {
    narrow: section.narrow ?? false,
    switches: section.switches ?? "left",
  };
  const descriptionLines = section.description ?? 1;
  const hasFields = groups.some((group) => group.fields.length > 0);

  return (
    <Card>
      <CardHeader className={cn(section.ruled && "border-b pb-3")}>
        <CardTitle className="flex h-5.5 items-center">
          <Skeleton className={cn("h-4", titleClasses[section.title ?? "md"])} />
        </CardTitle>
        {descriptionLines > 0 && (
          <CardDescription className="flex flex-col">
            {Array.from({ length: descriptionLines }, (_, line) => (
              <span key={line} className="flex h-5 items-center">
                <Skeleton
                  className={cn("h-3.5", line === descriptionLines - 1 ? "w-2/3" : "w-full")}
                />
              </span>
            ))}
          </CardDescription>
        )}
      </CardHeader>
      {hasFields && (
        <CardContent
          className={cn(
            "flex flex-col gap-2",
            (section.width ?? "prose") === "prose" && "max-w-prose",
          )}
        >
          {groups.map((group, index) =>
            group.rows ? (
              <RecordRowsSkeleton key={index} group={group} context={context} />
            ) : (
              <GroupSkeleton key={index} group={group} context={context} />
            ),
          )}
        </CardContent>
      )}
    </Card>
  );
}

/**
 * The root of a loading state: announced once as busy under `label`, with everything drawn
 * inside hidden from assistive technology so a screen reader hears "loading", not a tour of
 * empty shapes.
 */
export function SkeletonRegion({
  label,
  className,
  children,
}: {
  label: string;
  className?: string;
  children: ReactNode;
}) {
  return (
    <div className={className} aria-busy aria-label={label}>
      <div className="contents" aria-hidden>
        {children}
      </div>
    </div>
  );
}

/** The cards of a settings form, laid out as the form stacks them. */
export function SettingsSectionsSkeleton({
  sections,
}: {
  sections: readonly SettingsSectionSpec[];
}) {
  return (
    <>
      {sections.map((section, index) => (
        <SettingsSectionSkeleton key={index} section={section} />
      ))}
    </>
  );
}

/** A settings page's loading state: its cards, stacked as the form stacks them. */
export function SettingsFormSkeleton({ sections }: { sections: readonly SettingsSectionSpec[] }) {
  const t = useT();

  return (
    <SkeletonRegion label={t("Loading settings")} className="flex flex-col gap-4">
      <SettingsSectionsSkeleton sections={sections} />
    </SkeletonRegion>
  );
}
