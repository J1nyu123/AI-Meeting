import { Navigate, Outlet, useLocation } from "react-router-dom";
import { PageLoading } from "@/components/feedback/page-state";
import { useAuth } from "./auth-context";

export function RequireAuth() {
  const location = useLocation();
  const { isLoading, isAuthenticated } = useAuth();

  if (isLoading) {
    return (
      <PageLoading
        fullPage
        title="正在恢复登录状态"
        description="正在确认你的账号信息。"
      />
    );
  }

  if (!isAuthenticated) {
    return (
      <Navigate
        to="/auth"
        replace
        state={{ from: `${location.pathname}${location.search}` }}
      />
    );
  }

  return <Outlet />;
}
