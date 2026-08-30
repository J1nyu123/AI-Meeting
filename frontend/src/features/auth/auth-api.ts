import {
  apiRequest,
  clearTokens,
  getRefreshToken,
  saveTokens,
} from "@/api/client";
import type { Credentials, TokenPair, User } from "./types";

export const authApi = {
  register(credentials: Credentials) {
    return apiRequest<User>("/auth/register", {
      method: "POST",
      auth: false,
      body: JSON.stringify(credentials),
    });
  },

  async login(credentials: Credentials) {
    const tokenPair = await apiRequest<TokenPair>("/auth/login", {
      method: "POST",
      auth: false,
      body: JSON.stringify(credentials),
    });

    saveTokens(tokenPair.accessToken, tokenPair.refreshToken);
    return tokenPair.user;
  },

  me() {
    return apiRequest<User>("/auth/me");
  },

  async logout() {
    const refreshToken = getRefreshToken();

    try {
      if (refreshToken) {
        await apiRequest<{ loggedOut: boolean }>("/auth/logout", {
          method: "POST",
          auth: false,
          body: JSON.stringify({ refreshToken }),
        });
      }
    } finally {
      clearTokens();
    }
  },
};
