import { MetaTags } from "@/components/meta-tags";
import { AuthForm } from "./_components/auth-form";

export function LoginPage() {
  return (
    <>
      <MetaTags title="Login" description="Login to your account" />
      <div className="fixed inset-0 flex flex-col items-center justify-center px-4">
        <AuthForm />
        <footer className="mt-8 text-center text-sm text-muted-foreground">
          &copy; {new Date().getFullYear()} Trenova. All rights reserved.
        </footer>
      </div>
    </>
  );
}
