import { useAgentProfile } from "@/hooks/use-agent-profile";
import { describeApiError } from "@/lib/api-error-message";
import { MAX_PROFILE_GUIDANCE_CHARS } from "@/lib/graphql/agent-profile";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { useState, type FormEvent } from "react";
import { DeskIcon } from "@/components/desk-chat/desk-icons";

/**
 * What agents are told about the person's own work: a short write-up of what their
 * records show they work on, kept up each week, and the lines of guidance they gave.
 * They never edit the write-up; a line of guidance ("I only handle the Midwest
 * accounts") is how they correct it, and the next version honours it.
 */
export function DeskProfileCard() {
  const t = useT();
  const { profileQuery, addGuidance, retireGuidance, regenerate } = useAgentProfile("mine");
  const [draft, setDraft] = useState("");

  const profile = profileQuery.data;
  if (profileQuery.isPending) {
    return <DeskProfileCardSkeleton />;
  }
  if (!profile) {
    return (
      <section className="dk-mm-me" aria-label={t("What agents know about your work")}>
        <div className="dk-mm-me-h">
          <span>
            <b>{t("What agents know about your work")}</b>
          </span>
        </div>
        <p className="dk-mm-me-d" role="alert">
          {describeApiError(profileQuery.error, t("This could not be loaded"))}
        </p>
      </section>
    );
  }
  const content = profile.current?.content;

  const submit = (event: FormEvent) => {
    event.preventDefault();
    const line = draft.trim();
    if (line === "") {
      return;
    }
    addGuidance.mutate(line, { onSuccess: () => setDraft("") });
  };

  return (
    <section className="dk-mm-me" aria-label={t("What agents know about your work")}>
      <div className="dk-mm-me-h">
        <span>
          <b>{t("What agents know about your work")}</b>
          <em>
            {t(
              "Worked out from your own records and what you've told them, so a short request means what you meant.",
            )}
          </em>
        </span>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          disabled={regenerate.isPending}
          onClick={() => regenerate.mutate()}
        >
          {t("Write it again")}
        </Button>
      </div>

      {content ? (
        <p className="dk-mm-me-c">{content}</p>
      ) : (
        <p className="dk-mm-me-n">
          {t(
            "Nothing yet. After you've worked with agents for a while, a short note about your work appears here.",
          )}
        </p>
      )}
      {profile.latestFailure && (
        <p className="dk-mm-me-d">
          {t(
            "The last attempt to write it again did not finish: {0}",
            profile.latestFailure.errorMessage ?? "",
          )}
        </p>
      )}

      {profile.guidance.length > 0 && (
        <ul className="dk-mm-me-g" aria-label={t("What you told agents")}>
          {profile.guidance.map((line) => (
            <li key={line.id}>
              <span>{line.content}</span>
              <Button
                type="button"
                variant="ghost"
                size="sm"
                disabled={retireGuidance.isPending}
                onClick={() => retireGuidance.mutate(line.id)}
                aria-label={t("Retire “{0}”", line.content)}
              >
                {t("Retire")}
              </Button>
            </li>
          ))}
        </ul>
      )}

      <form className="dk-mm-add" onSubmit={submit}>
        <DeskIcon name="plus" size={14} />
        <input
          value={draft}
          onChange={(event) => setDraft(event.target.value)}
          placeholder={t("Tell agents about your work, e.g. “I only handle the Midwest accounts”")}
          aria-label={t("Tell agents about your work")}
          maxLength={MAX_PROFILE_GUIDANCE_CHARS}
        />
        <Button
          type="submit"
          className="rounded-lg px-3.25 disabled:opacity-35"
          disabled={draft.trim() === "" || addGuidance.isPending}
        >
          {t("Add")}
        </Button>
      </form>
    </section>
  );
}

function DeskProfileCardSkeleton() {
  const t = useT();

  return (
    <section
      className="dk-mm-me dk-mm-me-sk"
      aria-label={t("What agents know about your work")}
      aria-busy="true"
    >
      <div className="dk-mm-me-h">
        <span>
          <b>{t("What agents know about your work")}</b>
          <i className="dk-sk dk-mm-me-sk-e" />
        </span>
        <i className="dk-sk dk-mm-me-sk-b" />
      </div>
      <div className="dk-mm-me-sk-c" aria-hidden="true">
        <i className="dk-sk" />
        <i className="dk-sk" />
        <i className="dk-sk" />
      </div>
      <i className="dk-sk dk-mm-me-sk-f" aria-hidden="true" />
    </section>
  );
}
