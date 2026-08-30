import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  apiRawRequest,
  apiRequest,
  clearTokens,
  getAccessToken,
  getRefreshToken,
  saveTokens,
} from "./client";

function jsonResponse(
  data: unknown,
  options: { status?: number; success?: boolean; code?: string } = {},
) {
  const status = options.status ?? 200;
  return new Response(
    JSON.stringify({
      success: options.success ?? true,
      code: options.code ?? "OK",
      message: options.success === false ? "登录状态已失效" : null,
      data,
      requestId: "request-test",
    }),
    {
      status,
      headers: { "Content-Type": "application/json" },
    },
  );
}

describe("api client", () => {
  const fetchMock = vi.fn<typeof fetch>();

  beforeEach(() => {
    clearTokens();
    vi.stubGlobal("fetch", fetchMock);
  });

  afterEach(() => {
    clearTokens();
    vi.unstubAllGlobals();
  });

  it("refreshes an expired access token and retries the original request", async () => {
    saveTokens("expired-access", "valid-refresh");
    fetchMock
      .mockResolvedValueOnce(
        jsonResponse(null, {
          status: 401,
          success: false,
          code: "UNAUTHORIZED",
        }),
      )
      .mockResolvedValueOnce(
        jsonResponse({
          accessToken: "next-access",
          refreshToken: "next-refresh",
          expiresIn: 900,
          user: { id: 7, username: "tester" },
        }),
      )
      .mockResolvedValueOnce(
        jsonResponse({ id: 7, username: "tester" }),
      );

    const user = await apiRequest<{ id: number; username: string }>("/auth/me");

    expect(user.username).toBe("tester");
    expect(fetchMock).toHaveBeenCalledTimes(3);
    expect(fetchMock.mock.calls[1][0]).toBe("/api/v1/auth/refresh");
    expect(fetchMock.mock.calls[2][0]).toBe("/api/v1/auth/me");
    expect(
      (fetchMock.mock.calls[2][1]?.headers as Headers).get("Authorization"),
    ).toBe("Bearer next-access");
    expect(getAccessToken()).toBe("next-access");
    expect(getRefreshToken()).toBe("next-refresh");
  });

  it("returns authenticated raw responses for PDF and audio requests", async () => {
    saveTokens("pdf-access", "pdf-refresh");
    fetchMock.mockResolvedValueOnce(
      new Response("%PDF-test", {
        status: 200,
        headers: { "Content-Type": "application/pdf" },
      }),
    );

    const response = await apiRawRequest("/interviews/session-1/resume");

    expect(await response.text()).toBe("%PDF-test");
    expect(
      (fetchMock.mock.calls[0][1]?.headers as Headers).get("Authorization"),
    ).toBe("Bearer pdf-access");
  });

  it("turns invalid JSON responses into a typed ApiError", async () => {
    fetchMock.mockResolvedValueOnce(
      new Response("not-json", { status: 502 }),
    );

    await expect(apiRequest("/broken")).rejects.toMatchObject({
      name: "ApiError",
      code: "INVALID_RESPONSE",
      status: 502,
    });
  });
});
