/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import logoRainbow from "@/assets/logo.webp";
import { MetaTags } from "@/components/meta-tags";
import { LazyImage } from "@/components/ui/image";
import { AuthForm } from "./_components/auth-form";

export function LoginPage() {
  return (
    <>
      <MetaTags title="Login" description="Login to your account" />
      <div className="flex min-h-svh flex-col items-center justify-center gap-6 p-6 md:p-10">
        <div className="flex w-full max-w-sm flex-col items-center gap-6">
          <LazyImage
            src={logoRainbow}
            alt="Trenova Logo"
            className="size-14 object-contain"
          />
          <AuthForm />
        </div>
      </div>
    </>
  );
}
