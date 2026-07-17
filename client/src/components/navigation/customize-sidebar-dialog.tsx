import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { Switch } from "@/components/ui/switch";
import { QUICK_ACTION_ICONS } from "@/config/quick-action-icons";
import {
  useSidebarCustomizationOptions,
  useSidebarPreferences,
  useUpdateSidebarPreferences,
} from "@/hooks/use-sidebar-preferences";
import { graphQLErrorMessage } from "@/lib/graphql";
import type {
  EffectiveSidebarPreferences,
  SidebarCustomizationOptions,
} from "@/lib/graphql/sidebar-preferences";
import { DEFAULT_SIDEBAR_PREFERENCES } from "@/lib/graphql/sidebar-preferences";
import { cn } from "@/lib/utils";
import {
  closestCenter,
  DndContext,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
  type DragEndEvent,
} from "@dnd-kit/core";
import { restrictToVerticalAxis } from "@dnd-kit/modifiers";
import {
  arrayMove,
  SortableContext,
  useSortable,
  verticalListSortingStrategy,
} from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { GripVerticalIcon, SlidersHorizontalIcon } from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { toast } from "sonner";

interface SectionDraft {
  key: string;
  hidden: boolean;
}

interface PreferencesDraft {
  sections: SectionDraft[];
  attentionMetrics: string[];
  quickActionIds: string[];
  activity: {
    pageSize: number;
    defaultOpen: boolean;
  };
}

function draftFromPreferences(preferences: EffectiveSidebarPreferences): PreferencesDraft {
  return {
    sections: preferences.sections.map(({ key, hidden }) => ({ key, hidden })),
    attentionMetrics: [...preferences.attentionMetrics],
    quickActionIds: [...preferences.quickActionIds],
    activity: { ...preferences.activity },
  };
}

function useVerticalDndSensors() {
  return useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 6 } }),
    useSensor(KeyboardSensor),
  );
}

function GroupLabel({ children, hint }: { children: React.ReactNode; hint?: string }) {
  return (
    <div className="flex items-baseline justify-between">
      <span className="text-2xs font-semibold tracking-wider text-muted-foreground uppercase select-none">
        {children}
      </span>
      {hint && <span className="text-2xs text-muted-foreground/70 tabular-nums">{hint}</span>}
    </div>
  );
}

function SortableRow({
  id,
  disabled,
  children,
}: {
  id: string;
  disabled?: boolean;
  children: React.ReactNode;
}) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id,
    disabled,
  });

  return (
    <div
      ref={setNodeRef}
      style={{ transform: CSS.Transform.toString(transform), transition }}
      className={cn(
        "flex h-8 items-center gap-2 rounded-md border border-border bg-background px-2",
        isDragging && "z-10 opacity-60",
      )}
    >
      {children}
      <button
        type="button"
        aria-label="Reorder"
        className={cn(
          "ml-auto flex size-6 shrink-0 cursor-grab touch-none items-center justify-center rounded text-muted-foreground/60 transition-colors hover:text-foreground",
          disabled && "invisible",
        )}
        {...attributes}
        {...listeners}
      >
        <GripVerticalIcon className="size-3.5" />
      </button>
    </div>
  );
}

function SectionsEditor({
  options,
  sections,
  onChange,
}: {
  options: SidebarCustomizationOptions;
  sections: SectionDraft[];
  onChange: (sections: SectionDraft[]) => void;
}) {
  const sensors = useVerticalDndSensors();
  const sectionsByKey = useMemo(
    () => new Map(options.sections.map((section) => [section.key, section])),
    [options.sections],
  );

  const handleDragEnd = useCallback(
    (event: DragEndEvent) => {
      const { active, over } = event;
      if (!over || active.id === over.id) return;
      const oldIndex = sections.findIndex((section) => section.key === active.id);
      const newIndex = sections.findIndex((section) => section.key === over.id);
      onChange(arrayMove(sections, oldIndex, newIndex));
    },
    [sections, onChange],
  );

  return (
    <div className="flex flex-col gap-1.5">
      <GroupLabel>Sections</GroupLabel>
      <DndContext
        sensors={sensors}
        collisionDetection={closestCenter}
        onDragEnd={handleDragEnd}
        modifiers={[restrictToVerticalAxis]}
      >
        <SortableContext
          items={sections.map((section) => section.key)}
          strategy={verticalListSortingStrategy}
        >
          <div className="flex flex-col gap-1">
            {sections.map((section) => {
              const definition = sectionsByKey.get(section.key);
              if (!definition) return null;
              return (
                <SortableRow key={section.key} id={section.key}>
                  <Switch
                    size="sm"
                    checked={!section.hidden}
                    disabled={!definition.hideable}
                    onCheckedChange={(checked) =>
                      onChange(
                        sections.map((entry) =>
                          entry.key === section.key ? { ...entry, hidden: !checked } : entry,
                        ),
                      )
                    }
                  />
                  <span className="truncate text-xs font-medium">{definition.label}</span>
                  {!definition.hideable && (
                    <span className="text-2xs text-muted-foreground/70">Always visible</span>
                  )}
                </SortableRow>
              );
            })}
          </div>
        </SortableContext>
      </DndContext>
    </div>
  );
}

function ChecklistEditor({
  label,
  hint,
  items,
  selected,
  maxSelected,
  onChange,
  renderIcon,
}: {
  label: string;
  hint?: string;
  items: { id: string; label: string }[];
  selected: string[];
  maxSelected?: number;
  onChange: (selected: string[]) => void;
  renderIcon?: (id: string) => React.ReactNode;
}) {
  const sensors = useVerticalDndSensors();
  const itemsById = useMemo(() => new Map(items.map((item) => [item.id, item])), [items]);
  const selectedSet = useMemo(() => new Set(selected), [selected]);
  const unselected = items.filter((item) => !selectedSet.has(item.id));
  const atLimit = maxSelected != null && selected.length >= maxSelected;

  const handleDragEnd = useCallback(
    (event: DragEndEvent) => {
      const { active, over } = event;
      if (!over || active.id === over.id) return;
      const oldIndex = selected.indexOf(active.id as string);
      const newIndex = selected.indexOf(over.id as string);
      onChange(arrayMove(selected, oldIndex, newIndex));
    },
    [selected, onChange],
  );

  const toggle = useCallback(
    (id: string, checked: boolean) => {
      if (checked) {
        onChange([...selected, id]);
      } else {
        onChange(selected.filter((entry) => entry !== id));
      }
    },
    [selected, onChange],
  );

  const renderRowContent = (id: string, itemLabel: string, checked: boolean) => (
    <>
      <Checkbox
        checked={checked}
        disabled={!checked && atLimit}
        onCheckedChange={(value) => toggle(id, value === true)}
      />
      {renderIcon?.(id)}
      <span className={cn("truncate text-xs font-medium", !checked && "text-foreground/70")}>
        {itemLabel}
      </span>
    </>
  );

  return (
    <div className="flex flex-col gap-1.5">
      <GroupLabel hint={hint}>{label}</GroupLabel>
      <DndContext
        sensors={sensors}
        collisionDetection={closestCenter}
        onDragEnd={handleDragEnd}
        modifiers={[restrictToVerticalAxis]}
      >
        <SortableContext items={selected} strategy={verticalListSortingStrategy}>
          <div className="flex flex-col gap-1">
            {selected.map((id) => {
              const item = itemsById.get(id);
              if (!item) return null;
              return (
                <SortableRow key={id} id={id}>
                  {renderRowContent(id, item.label, true)}
                </SortableRow>
              );
            })}
          </div>
        </SortableContext>
      </DndContext>
      {unselected.length > 0 && (
        <div className="flex flex-col gap-1">
          {unselected.map((item) => (
            <div
              key={item.id}
              className={cn(
                "flex h-8 items-center gap-2 rounded-md border border-dashed border-border/70 px-2",
                atLimit && "opacity-50",
              )}
            >
              {renderRowContent(item.id, item.label, false)}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function ActivityEditor({
  options,
  activity,
  onChange,
}: {
  options: SidebarCustomizationOptions;
  activity: PreferencesDraft["activity"];
  onChange: (activity: PreferencesDraft["activity"]) => void;
}) {
  return (
    <div className="flex flex-col gap-1.5">
      <GroupLabel>Recent Activity</GroupLabel>
      <div className="flex h-8 items-center justify-between rounded-md border border-border bg-background px-2">
        <span className="text-xs font-medium">Entries per page</span>
        <Select
          value={String(activity.pageSize)}
          onValueChange={(value) => {
            if (value == null) return;
            onChange({ ...activity, pageSize: Number(value) });
          }}
          items={options.activityPageSizes.map((size) => ({
            value: String(size),
            label: String(size),
          }))}
        >
          <SelectTrigger size="sm" className="w-16">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {options.activityPageSizes.map((size) => (
              <SelectItem key={size} value={String(size)}>
                {size}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
      <div className="flex h-8 items-center justify-between rounded-md border border-border bg-background px-2">
        <span className="text-xs font-medium">Expanded by default</span>
        <Switch
          size="sm"
          checked={activity.defaultOpen}
          onCheckedChange={(checked) => onChange({ ...activity, defaultOpen: checked })}
        />
      </div>
    </div>
  );
}

function CustomizeSidebarForm({
  preferences,
  options,
  onSaved,
}: {
  preferences: EffectiveSidebarPreferences;
  options: SidebarCustomizationOptions;
  onSaved: () => void;
}) {
  const [draft, setDraft] = useState<PreferencesDraft>(() => draftFromPreferences(preferences));
  const updateMutation = useUpdateSidebarPreferences();

  const permittedMetricKeys = useMemo(
    () => new Set(options.attentionMetrics.map((metric) => metric.key)),
    [options.attentionMetrics],
  );
  const permittedActionIds = useMemo(
    () => new Set(options.quickActions.map((action) => action.id)),
    [options.quickActions],
  );

  const handleReset = () => {
    const defaults = draftFromPreferences(DEFAULT_SIDEBAR_PREFERENCES);
    defaults.attentionMetrics = defaults.attentionMetrics.filter((key) =>
      permittedMetricKeys.has(key),
    );
    defaults.quickActionIds = defaults.quickActionIds.filter((id) => permittedActionIds.has(id));
    setDraft(defaults);
  };

  const handleSave = () => {
    updateMutation.mutate(
      {
        version: preferences.version,
        sections: draft.sections,
        attentionMetrics: draft.attentionMetrics,
        quickActionIds: draft.quickActionIds,
        activity: draft.activity,
      },
      {
        onSuccess: () => {
          toast.success("Sidebar preferences saved");
          onSaved();
        },
        onError: (error) => {
          toast.error("Failed to save sidebar preferences", {
            description: graphQLErrorMessage(error, "Your changes could not be saved."),
          });
        },
      },
    );
  };

  return (
    <>
      <div className="-mx-1 flex max-h-[60vh] flex-col gap-4 overflow-y-auto px-1 py-0.5">
        <SectionsEditor
          options={options}
          sections={draft.sections}
          onChange={(sections) => setDraft((previous) => ({ ...previous, sections }))}
        />
        <ChecklistEditor
          label="Needs Attention"
          items={options.attentionMetrics.map((metric) => ({
            id: metric.key,
            label: metric.label,
          }))}
          selected={draft.attentionMetrics}
          onChange={(attentionMetrics) =>
            setDraft((previous) => ({ ...previous, attentionMetrics }))
          }
        />
        <ChecklistEditor
          label="Quick Actions"
          hint={`${draft.quickActionIds.length}/${options.maxQuickActions}`}
          items={options.quickActions.map((action) => ({ id: action.id, label: action.label }))}
          selected={draft.quickActionIds}
          maxSelected={options.maxQuickActions}
          onChange={(quickActionIds) => setDraft((previous) => ({ ...previous, quickActionIds }))}
          renderIcon={(id) => {
            const Icon = QUICK_ACTION_ICONS[id];
            return Icon ? (
              <Icon className="size-3.5 shrink-0 text-muted-foreground" strokeWidth={1.75} />
            ) : null;
          }}
        />
        <ActivityEditor
          options={options}
          activity={draft.activity}
          onChange={(activity) => setDraft((previous) => ({ ...previous, activity }))}
        />
      </div>
      <DialogFooter>
        <Button
          variant="ghost"
          className="sm:mr-auto"
          onClick={handleReset}
          disabled={updateMutation.isPending}
        >
          Reset to defaults
        </Button>
        <DialogClose render={<Button variant="outline">Cancel</Button>} />
        <Button onClick={handleSave} disabled={updateMutation.isPending}>
          {updateMutation.isPending ? "Saving…" : "Save changes"}
        </Button>
      </DialogFooter>
    </>
  );
}

function FormSkeleton() {
  return (
    <div className="flex flex-col gap-2">
      {Array.from({ length: 6 }, (_, index) => (
        <Skeleton key={index} className="h-8 w-full rounded-md" />
      ))}
    </div>
  );
}

export function CustomizeSidebarDialog() {
  const [open, setOpen] = useState(false);
  const { data: preferences, isPlaceholderData } = useSidebarPreferences();
  const { data: options } = useSidebarCustomizationOptions(open);
  const isReady = preferences != null && !isPlaceholderData && options != null;

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger
        render={
          <button
            type="button"
            aria-label="Customize sidebar"
            title="Customize sidebar"
            className="flex size-7 shrink-0 items-center justify-center rounded-md border border-border bg-background text-muted-foreground transition-colors hover:border-ring/40 hover:text-foreground"
          />
        }
      >
        <SlidersHorizontalIcon className="size-3.5" strokeWidth={1.75} />
      </DialogTrigger>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Customize sidebar</DialogTitle>
          <DialogDescription>
            Choose which sections appear, reorder them, and tune what each one shows. Your layout
            follows you across devices.
          </DialogDescription>
        </DialogHeader>
        {isReady ? (
          <CustomizeSidebarForm
            preferences={preferences}
            options={options}
            onSaved={() => setOpen(false)}
          />
        ) : (
          <FormSkeleton />
        )}
      </DialogContent>
    </Dialog>
  );
}
