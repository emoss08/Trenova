import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useAuthStore } from "@/stores/auth-store";
import { m } from "motion/react";
import { useState } from "react";
import { useNavigate } from "react-router";

export function DashLoginPage() {
  const navigate = useNavigate();
  const login = useAuthStore((state) => state.login);
  const [emailAddress, setEmailAddress] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [pending, setPending] = useState(false);

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault();
    if (!emailAddress || !password) {
      setError("Enter your email and password.");
      return;
    }
    setPending(true);
    setError(null);
    try {
      await login({ emailAddress, password });
      void navigate("/dash", { replace: true });
    } catch {
      setError("We couldn't sign you in. Check your email and password and try again.");
    } finally {
      setPending(false);
    }
  };

  return (
    <div className="flex min-h-dvh flex-col justify-center bg-background px-6 text-foreground">
      <m.div
        initial={{ opacity: 0, y: 12 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.25, ease: "easeOut" }}
        className="mx-auto w-full max-w-sm"
      >
        <div className="mb-8">
          <h1 className="text-3xl font-semibold tracking-tight">Dash</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Your loads, settlements, and pay — in one place.
          </p>
        </div>

        <form onSubmit={handleSubmit} className="flex flex-col gap-4" noValidate>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="dash-email">Email</Label>
            <Input
              id="dash-email"
              type="email"
              autoComplete="email"
              inputMode="email"
              placeholder="you@example.com"
              value={emailAddress}
              onChange={(event) => setEmailAddress(event.target.value)}
            />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="dash-password">Password</Label>
            <Input
              id="dash-password"
              type="password"
              autoComplete="current-password"
              placeholder="••••••••"
              value={password}
              onChange={(event) => setPassword(event.target.value)}
            />
          </div>

          {error ? <p className="text-sm text-destructive">{error}</p> : null}

          <Button type="submit" className="mt-2 h-11 w-full" disabled={pending}>
            {pending ? "Signing in..." : "Sign in"}
          </Button>
        </form>

        <p className="mt-6 text-center text-xs text-muted-foreground">
          No account yet? Ask your carrier to send you a Dash invitation.
        </p>
        <p className="mt-2 text-center text-xs text-muted-foreground">
          Office or dispatch?{" "}
          <a href="/login" className="text-foreground underline underline-offset-4">
            Sign in to Trenova
          </a>
        </p>
      </m.div>
    </div>
  );
}
