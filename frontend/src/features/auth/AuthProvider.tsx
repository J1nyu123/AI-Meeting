import { useEffect, useMemo, useState, type PropsWithChildren } from "react";
import { clearTokens, getAccessToken, getRefreshToken } from "@/api/client";
import { authApi } from "./auth-api";
import { AuthContext, type AuthContextValue } from "./auth-context";
import type { Credentials, User } from "./types";

export function AuthProvider({ children }: PropsWithChildren) {
  const [user, setUser] = useState<User | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    let active = true;

    async function restoreSession() {
      if (!getAccessToken() && !getRefreshToken()) {
        setIsLoading(false);
        return;
      }

      try {
        const currentUser = await authApi.me();

        if (active) {
          setUser(currentUser);
        }
      } catch {
        clearTokens();

        if (active) {
          setUser(null);
        }
      } finally {
        if (active) {
          setIsLoading(false);
        }
      }
    }

    void restoreSession();

    return () => {
      active = false;
    };
  }, []);

  const value = useMemo<AuthContextValue>(
    () => ({
      user,
      isLoading,
      isAuthenticated: user !== null,

      async login(credentials: Credentials) {
        const currentUser = await authApi.login(credentials);
        setUser(currentUser);
      },

      async register(credentials: Credentials) {
        await authApi.register(credentials);
        const currentUser = await authApi.login(credentials);
        setUser(currentUser);
      },

      async logout() {
        try {
          await authApi.logout();
        } finally {
          setUser(null);
        }
      },
    }),
    [isLoading, user],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}
