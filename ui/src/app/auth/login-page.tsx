import logoRainbow from "@/assets/logo.webp";
import { MetaTags } from "@/components/meta-tags";
import { LazyImage } from "@/components/ui/image";
import { AuthForm } from "./_components/auth-form";

export function LoginPage() {
  return (
    <>
      <MetaTags title="Login" description="Login to your account" />
      <div className="fixed inset-0 h-svh w-full overflow-hidden bg-background">
        <div className="pointer-events-none absolute inset-0">
          <div className="absolute inset-0 bg-[radial-gradient(1200px_600px_at_10%_10%,rgba(99,102,241,0.20),transparent_60%)]" />
          <div className="absolute inset-0 bg-[radial-gradient(900px_500px_at_90%_20%,rgba(236,72,153,0.18),transparent_60%)]" />
          <div className="absolute inset-0 bg-[radial-gradient(700px_400px_at_50%_100%,rgba(34,197,94,0.16),transparent_60%)]" />
          <div className="absolute inset-0 bg-background/60 dark:bg-black/80" />
        </div>
        <div className="relative flex h-full flex-col items-center justify-center gap-6 p-6 md:p-10">
          <LazyImage
            src={logoRainbow}
            alt="Trenova Logo"
            className="size-14 object-contain drop-shadow-[0_4px_24px_rgba(255,255,255,0.25)]"
          />
          <AuthForm />
        </div>
      </div>
    </>
  );
}
