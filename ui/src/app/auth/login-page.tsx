import logo from "@/assets/logo.webp";
import { MetaTags } from "@/components/meta-tags";
import { LazyImage } from "@/components/ui/image";
import { AuthForm } from "./_components/auth-form";

export function LoginPage() {
  return (
    <>
      <MetaTags title="Login" description="Login to your account" />
      {/* <div className="absolute top-0 z-[-2] h-screen w-screen bg-background bg-[radial-gradient(ellipse_80%_80%_at_50%_-20%,rgba(120,119,198,0.3),rgba(255,255,255,0))]"></div> */}
      <div className="flex min-h-svh flex-col items-center justify-center gap-6 p-6 md:p-10">
        <div className="flex w-full max-w-sm flex-col items-center gap-6">
          <LazyImage
            src={logo}
            alt="Trenova Logo"
            width={40}
            height={40}
            layout="constrained"
          />
          <AuthForm />
        </div>
      </div>
    </>
  );
}
