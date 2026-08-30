import { useState } from "react";
import { NavLink, Outlet, useLocation, useNavigate } from "react-router-dom";
import { LogOut, MessageSquareText } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { useAuth } from "@/features/auth/auth-context";

const navigationItems = [
  { path: "/", label: "首页" },
  { path: "/interviews", label: "我的面试" },
];

export function RootLayout() {
  const navigate = useNavigate();
  const location = useLocation();
  const { user, isAuthenticated, logout } = useAuth();
  const [isLoggingOut, setIsLoggingOut] = useState(false);
  const isInterviewWorkspace = location.pathname.startsWith("/interviews");

  async function handleLogout() {
    if (isLoggingOut) return;

    setIsLoggingOut(true);
    try {
      await logout();
      toast.success("已安全退出");
    } catch {
      toast.warning("已清除本机登录状态", {
        description: "服务器会话暂时无法同步退出。",
      });
    } finally {
      navigate("/", { replace: true });
      setIsLoggingOut(false);
    }
  }

  if (isInterviewWorkspace) {
    return <Outlet />;
  }

  return (
    <div className="min-h-svh bg-white text-neutral-950">
      <header className="sticky top-0 z-50 border-b border-neutral-200/80 bg-white/90 backdrop-blur">
        <div className="mx-auto flex h-14 max-w-6xl items-center justify-between px-4 sm:px-6">
          <NavLink to="/" className="flex items-center gap-2 font-semibold">
            <span className="flex size-8 items-center justify-center rounded-lg bg-neutral-950 text-white">
              <MessageSquareText className="size-4" />
            </span>
            <span>AI Meeting</span>
          </NavLink>

          <nav className="flex items-center gap-1">
            {navigationItems
              .filter((item) => item.path === "/" || isAuthenticated)
              .map((item) => (
              <NavLink
                key={item.path}
                to={item.path}
                className={({ isActive }) =>
                  [
                    "rounded-lg px-3 py-2 text-sm transition-colors",
                    isActive
                      ? "bg-neutral-100 font-medium text-neutral-950"
                      : "text-neutral-500 hover:bg-neutral-100 hover:text-neutral-950",
                  ].join(" ")
                }
              >
                {item.label}
              </NavLink>
              ))}

            {isAuthenticated ? (
              <div className="ml-2 flex items-center gap-2 border-l border-neutral-200 pl-3">
                <span className="hidden text-sm text-neutral-500 sm:inline">
                  {user?.username}
                </span>
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  disabled={isLoggingOut}
                  onClick={handleLogout}
                >
                  <LogOut data-icon="inline-start" />
                  {isLoggingOut ? "退出中" : "退出"}
                </Button>
              </div>
            ) : (
              <Button asChild variant="ghost" size="sm" className="ml-1">
                <NavLink to="/auth">登录</NavLink>
              </Button>
            )}
          </nav>
        </div>
      </header>

      <main>
        <Outlet />
      </main>
    </div>
  );
}
