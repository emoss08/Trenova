import { usePermission } from "@/hooks/use-permission";
import type { WorkerRecord } from "@/routes/worker/use-worker-state";
import { Camera01Icon, Trash01Icon, Upload01Icon } from "@trenova/shared/components/icons";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@trenova/shared/components/ui/dropdown-menu";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import { WorkerAvatar } from "@trenova/shared/components/worker-avatar";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useWorkerPhoto } from "./use-worker-photo";

function Face({ worker }: { worker: WorkerRecord }) {
  return (
    <WorkerAvatar
      aria-hidden
      firstName={worker.firstName}
      lastName={worker.lastName}
      photo={worker.photo}
      variant="full"
      className="size-10"
      fallbackClassName="text-sm"
    />
  );
}

/**
 * The worker's face at the head of their record: their photo, or their initials. Someone who may
 * update the worker presses it to choose a photo (or drops or pastes one on it); with a photo in
 * place it opens a menu to replace or remove it. Anyone else sees the face alone.
 */
export function WorkerPhotoMark({ worker }: { worker: WorkerRecord }) {
  const { allowed } = usePermission(Resource.Worker, Operation.Update);
  return allowed ? <EditableWorkerPhoto worker={worker} /> : <Face worker={worker} />;
}

function EditableWorkerPhoto({ worker }: { worker: WorkerRecord }) {
  const t = useT();
  const control = useWorkerPhoto(worker);
  const working = control.uploading || control.removing;
  const hasPhoto = Boolean(worker.photo);
  const label = hasPhoto ? t("Change photo") : t("Add a photo");

  const buttonClass = cn(
    "group/photo relative grid size-10 shrink-0 cursor-pointer place-items-center rounded-full ui-focus-ring outline-1 outline-offset-2 outline-border-strong transition-[outline-color] outline-dashed hover:outline-brand disabled:cursor-default data-popup-open:outline-brand",
    control.dragging && "outline-brand",
  );
  const overlay = (
    <span
      aria-hidden
      data-state={working ? "working" : control.dragging ? "dragging" : "idle"}
      className={cn(
        "absolute inset-0 grid place-items-center rounded-full bg-scrim text-foreground-on-solid opacity-0 transition-opacity duration-150 group-hover/photo:opacity-100 group-focus-visible/photo:opacity-100",
        (working || control.dragging) && "opacity-100",
      )}
    >
      {working ? (
        <Spinner className="size-4" />
      ) : control.dragging ? (
        <Upload01Icon className="size-4" />
      ) : (
        <Camera01Icon className="size-4" />
      )}
    </span>
  );

  return (
    <>
      {hasPhoto ? (
        <DropdownMenu>
          <DropdownMenuTrigger
            render={
              <button
                type="button"
                aria-label={label}
                title={label}
                aria-busy={working || undefined}
                disabled={working}
                className={buttonClass}
                onPointerEnter={control.preload}
                onFocus={control.preload}
                {...control.dropTarget}
              />
            }
          >
            <Face worker={worker} />
            {overlay}
          </DropdownMenuTrigger>
          <DropdownMenuContent align="start" sideOffset={8} className="w-44">
            <DropdownMenuItem
              title={t("Replace photo")}
              startContent={<Upload01Icon className="size-4" />}
              onClick={control.openPicker}
            />
            <DropdownMenuItem
              title={t("Remove photo")}
              color="danger"
              startContent={<Trash01Icon className="size-4" />}
              onClick={() => void control.remove()}
            />
          </DropdownMenuContent>
        </DropdownMenu>
      ) : (
        <button
          type="button"
          aria-label={label}
          title={label}
          aria-busy={working || undefined}
          disabled={working}
          className={buttonClass}
          onClick={control.openPicker}
          onPointerEnter={control.preload}
          onFocus={control.preload}
          {...control.dropTarget}
        >
          <Face worker={worker} />
          {overlay}
        </button>
      )}
      {control.element}
    </>
  );
}
