import { Metadata } from "@/components/metadata";
import { Button } from "@/components/ui/button";
import { useAuthStore } from "@/stores/auth-store";
import { useNavigate } from "react-router";

export function Home() {
  const navigate = useNavigate();
  const { user, logout } = useAuthStore();

  const handleLogout = async () => {
    await logout();
    void navigate("/login", { replace: true });
  };

  return (
    <div className="flex min-h-screen flex-col items-center justify-center gap-4">
      <Metadata title="Home" description="Trenova dashboard" />
      <h1 className="text-4xl font-bold">Trenova</h1>
      {user && <p className="text-muted-foreground">Welcome, {user.name}</p>}
      <Button variant="outline" onClick={handleLogout}>
        Sign out
      </Button>
    </div>
  );
}
