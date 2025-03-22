import logo from "@/assets/logo.webp";
import { MetaTags } from "@/components/meta-tags";
import { LazyImage } from "@/components/ui/image";
import { AuthForm } from "./_components/auth-form";

export function LoginPage() {
  return (
    <>
      <MetaTags title="Login" description="Login to your account" />
      <div className="fixed inset-0 flex flex-col items-center justify-center px-4">
        <div className="absolute top-8 left-8">
          <LazyImage
            src={logo}
            alt="Trenova Logo"
            width={40}
            height={40}
            layout="constrained"
          />
        </div>
        <AuthForm />
        <footer className="mt-8 text-center text-sm text-muted-foreground">
          &copy; {new Date().getFullYear()} Trenova. All rights reserved.
        </footer>
      </div>
    </>
  );
}
