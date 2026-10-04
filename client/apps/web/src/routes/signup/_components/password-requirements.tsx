import {
  passwordRequirements,
  passwordStrength,
  type PasswordStrength,
} from "@/types/cloud-signup";
import { CheckIcon, XCloseIcon } from "@trenova/shared/components/icons";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";

const STRENGTH_LABEL: Record<PasswordStrength, string> = {
  0: "Too weak",
  1: "Fair",
  2: "Good",
  3: "Strong",
  4: "Very strong",
};

const STRENGTH_FILL: Record<PasswordStrength, string> = {
  0: "bg-danger",
  1: "bg-warning",
  2: "bg-success",
  3: "bg-success",
  4: "bg-success",
};

const SEGMENTS = [1, 2, 3, 4] as const;

/**
 * The rule the server enforces, shown as it is met, with a strength meter that only
 * advises. A requirement turns from muted to met as the person types, so the
 * failing one is the one still in the quieter colour.
 */
export function PasswordRequirements({
  password,
  emailAddress,
  id,
}: {
  password: string;
  emailAddress: string;
  id?: string;
}) {
  const t = useT();
  const requirements = passwordRequirements(password, emailAddress);
  const strength = passwordStrength(password, emailAddress);
  const started = password.length > 0;

  return (
    <div id={id} className="flex flex-col gap-1.5" aria-live="polite">
      <div className="flex items-center gap-2">
        <div className="grid flex-1 grid-cols-4 gap-1" aria-hidden="true">
          {SEGMENTS.map((segment) => (
            <span
              key={segment}
              className={cn(
                "h-1 rounded-full transition-colors duration-150",
                started && strength >= segment ? STRENGTH_FILL[strength] : "bg-border",
              )}
            />
          ))}
        </div>
        <span className="text-subtle-foreground w-20 text-right text-xs">
          {started ? t(STRENGTH_LABEL[strength]) : ""}
        </span>
      </div>
      <ul className="m-0 flex list-none flex-col gap-0.5 p-0">
        {requirements.map((requirement) => (
          <li
            key={requirement.id}
            data-met={requirement.met}
            className={cn(
              "flex items-center gap-1.5 text-xs",
              requirement.met ? "text-success" : "text-subtle-foreground",
            )}
          >
            {requirement.met ? (
              <CheckIcon className="size-3" aria-hidden="true" />
            ) : (
              <XCloseIcon className="size-3" aria-hidden="true" />
            )}
            <span>{t(requirement.label)}</span>
            <span className="sr-only">{requirement.met ? t("met") : t("not met")}</span>
          </li>
        ))}
      </ul>
    </div>
  );
}
