import { isOffered, scopeLabel, usageLine } from "@/components/desk-chat/memory/memory-format";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import {
  DESK_MEMORIES_KEY,
  createDeskMemory,
  fetchDeskMemories,
  reviseDeskMemory,
  setDeskMemoryStatus,
  type DeskMemory,
  type DeskMemorySubject,
} from "@/lib/graphql/desk-memories";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { Textarea } from "@trenova/shared/components/ui/textarea";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { resolveUserTimezone } from "@trenova/shared/lib/date";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useState } from "react";
import { RecordMemoriesSkeleton } from "./record-memories-skeleton";

/** The most memories about one record the panel reads; a record with more is rare. */
const PAGE_SIZE = 50;

type WrittenKind = "Fact" | "Instruction";
type WrittenFor = "Organization" | "User";

export function recordMemoriesQueryKey(subject: DeskMemorySubject) {
  return [DESK_MEMORIES_KEY, "record", subject.type, subject.id] as const;
}

/**
 * Whether the person may read what agents remember. Hiding the assistant that
 * floats on every page is a preference about that assistant; it does not hide
 * what agents were told about a record.
 */
export function useRecordMemoriesAvailable(): boolean {
  return usePermission(Resource.Assistant, Operation.Read).allowed;
}

/** What a memory is, in the words the panel uses for the two kinds a person writes. */
export function memoryKindWord(kind: DeskMemory["kind"], t: TranslateFn): string {
  switch (kind) {
    case "Instruction":
      return t("Rule");
    case "Fact":
      return t("Fact");
    case "Procedure":
      return t("Steps");
    default:
      return t("Correction");
  }
}

/**
 * The panel sits inside each record's own form, so nothing in it is a form or
 * a submit button: saving a memory must never save the record.
 *
 * What agents remember about one customer, location, driver or carrier, on
 * that record's own page. Agents read these whenever the record comes up, so
 * the people who know the record best should see them, fix them and add to
 * them here, without going to the Desk or to AI Control. A memory someone may
 * not change is still shown: it shapes what agents do with the record.
 */
export function RecordMemories({ subject }: { subject: DeskMemorySubject }) {
  const t = useT();
  const available = useRecordMemoriesAvailable();
  const memoriesQuery = useQuery({
    queryKey: recordMemoriesQueryKey(subject),
    queryFn: ({ signal }) =>
      fetchDeskMemories({ scope: null, roleId: null, query: "", subject }, null, PAGE_SIZE, signal),
    enabled: available,
  });

  if (!available) {
    return null;
  }

  const memories = memoriesQuery.data?.items ?? [];

  return (
    <section className="flex flex-col gap-3" aria-label={t("What agents remember")}>
      <p className="text-foreground-muted text-sm">
        {t(
          "Agents read these whenever this record comes up. Write down what they should know or always do, and correct anything that is no longer true.",
        )}
      </p>
      <AddMemory subject={subject} />
      {memoriesQuery.isPending ? (
        <RecordMemoriesSkeleton />
      ) : memoriesQuery.isError ? (
        <p className="text-danger text-sm" role="alert">
          {t("The memories could not be read. Try again in a moment.")}
        </p>
      ) : memories.length === 0 ? (
        <p className="text-foreground-muted text-sm">{t("Nothing is remembered about it yet.")}</p>
      ) : (
        <ul className="flex flex-col gap-2">
          {memories.map((memory) => (
            <MemoryRow key={memory.id} memory={memory} subject={subject} />
          ))}
        </ul>
      )}
    </section>
  );
}

function useRefresh(subject: DeskMemorySubject) {
  const queryClient = useQueryClient();

  return () => queryClient.invalidateQueries({ queryKey: recordMemoriesQueryKey(subject) });
}

function AddMemory({ subject }: { subject: DeskMemorySubject }) {
  const t = useT();
  const refresh = useRefresh(subject);
  const [content, setContent] = useState("");
  const [kind, setKind] = useState<WrittenKind>("Fact");
  const [keptFor, setKeptFor] = useState<WrittenFor>("Organization");

  const add = useApiMutation({
    mutationFn: () =>
      createDeskMemory(content.trim(), { scope: keptFor, roleId: null }, { kind, subject }),
    onSuccess: async () => {
      setContent("");
      await refresh();
    },
    resourceName: "Memory",
  });

  return (
    <div className="border-border flex flex-col gap-2 rounded-md border p-3">
      <Textarea
        value={content}
        onChange={(event) => setContent(event.target.value)}
        rows={2}
        aria-label={t("Something agents should remember about this record")}
        placeholder={t("The receiving dock closes at 2pm on Fridays.")}
      />
      <div className="flex flex-wrap items-center gap-2">
        <SegmentedControl<WrittenKind>
          aria-label={t("What it is")}
          value={kind}
          onValueChange={setKind}
          items={[
            { value: "Fact", label: t("Fact") },
            { value: "Instruction", label: t("Rule") },
          ]}
        />
        <SegmentedControl<WrittenFor>
          aria-label={t("Who it is for")}
          value={keptFor}
          onValueChange={setKeptFor}
          items={[
            { value: "Organization", label: t("Everyone") },
            { value: "User", label: t("Just me") },
          ]}
        />
        <span className="flex-1" />
        <Button
          type="button"
          size="sm"
          isLoading={add.isPending}
          disabled={content.trim() === ""}
          onClick={() => add.mutate()}
        >
          {t("Remember")}
        </Button>
      </div>
    </div>
  );
}

function MemoryRow({ memory, subject }: { memory: DeskMemory; subject: DeskMemorySubject }) {
  const t = useT();
  const refresh = useRefresh(subject);
  const timezone = resolveUserTimezone(useAuthStore((state) => state.user?.timezone));
  const [draft, setDraft] = useState<string | null>(null);
  const offered = isOffered(memory);

  const revise = useApiMutation({
    mutationFn: (content: string) =>
      reviseDeskMemory(memory.id, { content, version: memory.version }),
    onSuccess: async () => {
      setDraft(null);
      await refresh();
    },
    resourceName: "Memory",
  });
  const forget = useApiMutation({
    mutationFn: () => setDeskMemoryStatus(memory.id, "Retired"),
    onSuccess: refresh,
    resourceName: "Memory",
  });
  const busy = revise.isPending || forget.isPending;

  return (
    <li className="border-border flex flex-col gap-1.5 rounded-md border px-3 py-2.5">
      {draft === null ? (
        <p className="text-sm whitespace-pre-wrap">{memory.content}</p>
      ) : (
        <Textarea
          value={draft}
          onChange={(event) => setDraft(event.target.value)}
          rows={3}
          aria-label={t("What agents should remember")}
        />
      )}
      <div className="flex flex-wrap items-center gap-2">
        <Badge variant={memory.kind === "Fact" ? "accent-teal" : "accent-indigo"}>
          {memoryKindWord(memory.kind, t)}
        </Badge>
        {offered ? <Badge variant="warning">{t("Waiting for an administrator")}</Badge> : null}
        <span className="text-foreground-muted text-xs">
          {t("{0} · {1}", scopeLabel(memory, t), usageLine(memory, timezone, t))}
        </span>
        <span className="flex-1" />
        {memory.editable && draft === null ? (
          <>
            <Button
              type="button"
              size="xs"
              variant="ghost"
              disabled={busy}
              onClick={() => setDraft(memory.content)}
            >
              {t("Edit")}
            </Button>
            <Button
              type="button"
              size="xs"
              variant="ghost"
              isLoading={forget.isPending}
              disabled={busy}
              onClick={() => forget.mutate()}
            >
              {t("Forget")}
            </Button>
          </>
        ) : null}
        {draft !== null ? (
          <>
            <Button
              type="button"
              size="xs"
              variant="ghost"
              disabled={busy}
              onClick={() => setDraft(null)}
            >
              {t("Cancel")}
            </Button>
            <Button
              type="button"
              size="xs"
              isLoading={revise.isPending}
              disabled={busy || draft.trim() === "" || draft.trim() === memory.content}
              onClick={() => revise.mutate(draft.trim())}
            >
              {t("Save")}
            </Button>
          </>
        ) : null}
      </div>
    </li>
  );
}
