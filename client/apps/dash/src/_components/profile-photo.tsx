import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Camera01Icon, Upload01Icon } from "@trenova/shared/components/icons";
import { PROFILE_PICTURE } from "@trenova/shared/components/image-upload/image-spec";
import { ProfilePicturePreview } from "@trenova/shared/components/image-upload/profile-picture-preview";
import { useImageUpload } from "@trenova/shared/components/image-upload/use-image-upload";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import { toast } from "@trenova/shared/components/ui/toast";
import { WorkerAvatar } from "@trenova/shared/components/worker-avatar";
import { translate } from "@trenova/shared/i18n/runtime";
import { useT } from "@trenova/shared/i18n/use-t";
import {
  fetchMyProfileChangeRequests,
  withdrawMyProfileChange,
  type MyProfileChangeRequest,
} from "@trenova/shared/lib/graphql/driver-portal";
import { submitMyPhoto } from "@trenova/shared/lib/portal";
import { changeRequestTone, isPhotoChange } from "@trenova/shared/lib/self-service";
import { cn } from "@trenova/shared/lib/utils";
import type { PersonPhoto } from "@trenova/shared/types/person-photo";
import { useEffect, useRef, useState, type ReactNode } from "react";
import { useDashFeatures } from "./use-dash-features";

export const PROFILE_CHANGE_REQUESTS_KEY = ["dash-profile-change-requests"] as const;
const PROFILE_KEY = ["dash-profile"] as const;

export type ProfilePhotoPerson = {
  firstName: string;
  lastName: string;
  photo?: PersonPhoto | null;
};

/** The driver's newest photo request, which is the one that decides what they are shown. */
export function latestPhotoRequest(
  requests: readonly MyProfileChangeRequest[] | undefined,
): MyProfileChangeRequest | null {
  return requests?.find((request) => isPhotoChange(request.changes)) ?? null;
}

/**
 * The driver's face at the head of their profile. Where the carrier lets drivers edit their
 * details, it is pressed (or a picture dropped on it) to send a new photo, positioned first; the
 * office decides, and until it does the photo sent waits beside the one on the record.
 */
export function ProfilePhoto({
  person,
  profileReadAt,
  children,
}: {
  person: ProfilePhotoPerson;
  /** When the profile was read (TanStack's `dataUpdatedAt`), to tell a photo approved since. */
  profileReadAt: number;
  /** What stands beside the face: the driver's name and carrier. */
  children: ReactNode;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const features = useDashFeatures();
  const busy = useRef(false);
  const [sending, setSending] = useState(false);
  const name = `${person.firstName} ${person.lastName}`.trim();

  const requests = useQuery({
    queryKey: PROFILE_CHANGE_REQUESTS_KEY,
    queryFn: ({ signal }) => fetchMyProfileChangeRequests({ signal }),
    staleTime: 60 * 1000,
  });
  const latest = latestPhotoRequest(requests.data);
  const waiting = latest?.status === "Pending" ? latest : null;
  const turnedDown = latest?.status === "Rejected" ? latest : null;

  // An approval writes the photo onto the record: once one is seen that is newer than the profile
  // on screen, read the profile again so the new face shows.
  const approvedAt = latest?.status === "Approved" ? (latest.decidedAt ?? 0) : 0;
  useEffect(() => {
    if (approvedAt > 0 && approvedAt * 1000 > profileReadAt) {
      void queryClient.invalidateQueries({ queryKey: PROFILE_KEY });
    }
  }, [approvedAt, profileReadAt, queryClient]);

  const canSend = features.allowContactInfoEdit && !waiting;

  const upload = useImageUpload({
    spec: PROFILE_PICTURE,
    preview: <ProfilePicturePreview name={name} />,
    title: t("Position your photo"),
    confirmLabel: t("Send for approval"),
    disabled: !canSend,
    onUpload: async (file) => {
      if (busy.current) return;
      busy.current = true;
      setSending(true);
      try {
        await submitMyPhoto(file);
        await queryClient.invalidateQueries({ queryKey: PROFILE_CHANGE_REQUESTS_KEY });
      } finally {
        busy.current = false;
        setSending(false);
      }
    },
    onProblem: (problem) => {
      if (problem) {
        toast.error(t("That file cannot be your photo"), { description: problem });
      }
    },
  });

  const withdraw = useMutation({
    mutationFn: (id: string) => withdrawMyProfileChange(id),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: PROFILE_CHANGE_REQUESTS_KEY });
    },
    onError: (error: Error) => toast.error(error.message || translate("Could not withdraw it.")),
  });

  const face = (
    <WorkerAvatar
      aria-hidden
      firstName={person.firstName}
      lastName={person.lastName}
      photo={person.photo}
      variant="full"
      className="size-16"
      fallbackClassName="text-lg"
    />
  );

  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center gap-4">
        {canSend ? (
          <button
            type="button"
            aria-label={person.photo ? t("Change your photo") : t("Add your photo")}
            className={cn(
              "relative size-16 shrink-0 rounded-full ui-focus-ring outline-1 outline-offset-2 outline-border-strong transition-[outline-color] outline-dashed",
              upload.dragging && "outline-brand",
            )}
            onClick={upload.openPicker}
            onPointerEnter={upload.preload}
            onFocus={upload.preload}
            {...upload.dropTarget}
          >
            {face}
            <span
              aria-hidden
              className="absolute -right-0.5 -bottom-0.5 grid size-6 place-items-center rounded-full border-2 border-card bg-primary text-primary-foreground"
            >
              {sending ? (
                <Spinner className="size-3" />
              ) : upload.dragging ? (
                <Upload01Icon className="size-3" />
              ) : (
                <Camera01Icon className="size-3" />
              )}
            </span>
          </button>
        ) : (
          face
        )}
        <div className="min-w-0 flex-1">{children}</div>
      </div>
      {features.allowContactInfoEdit ? upload.element : null}

      {waiting ? (
        <div className="flex items-center gap-3 rounded-lg border border-border bg-muted/30 p-3">
          {waiting.proposedPhoto ? (
            <WorkerAvatar
              firstName={person.firstName}
              lastName={person.lastName}
              photo={waiting.proposedPhoto}
              alt={t("The photo you sent")}
              className="size-10"
              fallbackClassName="text-sm"
            />
          ) : null}
          <div className="flex min-w-0 flex-1 flex-col gap-1">
            <Badge variant={changeRequestTone(waiting.status).variant} className="self-start">
              {t(changeRequestTone(waiting.status).label)}
            </Badge>
            <p className="text-xs text-muted-foreground">
              {t("Your new photo shows here once the office approves it.")}
            </p>
          </div>
          <Button
            variant="ghost"
            size="sm"
            className="h-7"
            isLoading={withdraw.isPending}
            disabled={withdraw.isPending}
            onClick={() => withdraw.mutate(waiting.id)}
          >
            {t("Withdraw")}
          </Button>
        </div>
      ) : turnedDown ? (
        <div className="flex flex-col gap-1 rounded-lg border border-border bg-muted/30 p-3 text-xs">
          <div className="flex items-center gap-2">
            <Badge variant={changeRequestTone(turnedDown.status).variant}>
              {t(changeRequestTone(turnedDown.status).label)}
            </Badge>
            <span className="text-muted-foreground">{t("Your last photo")}</span>
          </div>
          {turnedDown.decisionNote ? (
            <p className="text-muted-foreground">
              {t("Your carrier said: {0}", turnedDown.decisionNote)}
            </p>
          ) : null}
        </div>
      ) : null}
    </div>
  );
}
