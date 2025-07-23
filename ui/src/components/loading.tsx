/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import { faSpinnerThird } from "@fortawesome/pro-regular-svg-icons";
import { Icon } from "./ui/icons";

export default function LoadingSkeleton() {
  return (
    <div className="flex min-h-screen flex-row items-center justify-center text-center">
      <div className="flex w-[700px] flex-col rounded-md border border-border bg-card sm:flex-row sm:items-center sm:justify-center">
        <div className="space-y-4 p-8">
          <Icon
            icon={faSpinnerThird}
            size="2x"
            className="motion-safe:animate-spin"
          />
          <p className="font-xl mb-2 font-semibold">
            Hang tight!{" "}
            <u className="font-bold underline decoration-blue-600">Trenova</u>{" "}
            is gearing up for you.
          </p>
          <p className="mt-1 text-sm text-muted-foreground">
            We&apos;re working at lightning speed to get things ready. If this
            takes longer than a coffee break (10 seconds), please check your
            internet connection. <br />
            <u className="text-foreground decoration-blue-600">
              Still stuck?
            </u>{" "}
            Your friendly system administrator is just a call away for a swift
            rescue!
          </p>
        </div>
      </div>
    </div>
  );
}
