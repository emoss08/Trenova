import type { DeskStartExtras } from "@/components/desk-chat/desk-home-ask";
import type { StartBinding } from "@/components/assistant/use-start-conversation";
import { useDeskAskStore } from "@/stores/desk-ask-store";
import type { DeskSubject } from "@/stores/desk-margin-store";
import { useNavigate } from "@tanstack/react-router";
import { useCallback } from "react";
import { casePath } from "../attention/attention-record";
import { casePageContext, threadSubjectOf } from "../record-case/case-subject";

/**
 * Opens a new conversation full width: Today's box, with the caret in it and
 * the record the page was about riding on the question.
 */
export function useAskNew() {
  const navigate = useNavigate();

  return useCallback(
    (subject: DeskSubject | null) => {
      useDeskAskStore.getState().askNew(subject);
      void navigate({ to: "/desk" });
    },
    [navigate],
  );
}

/**
 * What a question about a record starts with: the record named in it and,
 * where the record has a case, the conversation bound to that case and page.
 */
export function startAbout(
  subject: DeskSubject | null,
  extras?: DeskStartExtras,
): { extras: DeskStartExtras; binding?: StartBinding } {
  const mentions = extras?.mentions ?? [];
  const base = { files: extras?.files ?? [], providerId: extras?.providerId ?? "" };
  if (subject === null) {
    return { extras: { ...base, mentions } };
  }
  const caseType = threadSubjectOf(subject.kind);

  return {
    extras: {
      ...base,
      mentions: [
        { type: subject.kind, id: subject.id, label: subject.label },
        ...mentions.filter(
          (mention) => !(mention.type === subject.kind && mention.id === subject.id),
        ),
      ],
    },
    binding: {
      subjectType: caseType ?? undefined,
      subjectId: caseType ? subject.id : undefined,
      page: casePageContext(
        casePath(subject.kind, subject.id),
        subject.kind,
        subject.id,
        subject.label,
      ),
    },
  };
}
