import type { TokenPair } from "@/features/auth/types";
import type { ApiResponse } from "@/types/api";

const API_BASE_URL = "/api/v1";
const ACCESS_TOKEN_KEY = "ai-meeting-access-token";
const REFRESH_TOKEN_KEY = "ai-meeting-refresh-token";

let refreshPromise: Promise<string> | null = null;

export class ApiError extends Error {
  readonly status: number;
  readonly code: string;
  readonly requestId?: string;

  constructor(
    status: number,
    code: string,
    message: string,
    requestId?: string,
  ) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
    this.requestId = requestId;
  }
}

export function getAccessToken() {
  return localStorage.getItem(ACCESS_TOKEN_KEY);
}

export function getRefreshToken() {
  return localStorage.getItem(REFRESH_TOKEN_KEY);
}

export function saveTokens(accessToken: string, refreshToken: string) {
  localStorage.setItem(ACCESS_TOKEN_KEY, accessToken);
  localStorage.setItem(REFRESH_TOKEN_KEY, refreshToken);
}

export function clearTokens() {
  localStorage.removeItem(ACCESS_TOKEN_KEY);
  localStorage.removeItem(REFRESH_TOKEN_KEY);
}

type ApiRequestOptions = RequestInit & {
  auth?: boolean;
};

async function readResponse<T>(response: Response): Promise<ApiResponse<T>> {
  try {
    return (await response.json()) as ApiResponse<T>;
  } catch {
    throw new ApiError(
      response.status,
      "INVALID_RESPONSE",
      "服务器返回了无法识别的数据",
    );
  }
}

async function refreshAccessToken(): Promise<string> {
  if (refreshPromise) {
    return refreshPromise;
  }

  const refreshToken = getRefreshToken();

  if (!refreshToken) {
    throw new ApiError(401, "UNAUTHORIZED", "登录状态已失效");
  }

  refreshPromise = apiRequest<TokenPair>(
    "/auth/refresh",
    {
      method: "POST",
      auth: false,
      body: JSON.stringify({ refreshToken }),
    },
    true,
  )
    .then((tokenPair) => {
      saveTokens(tokenPair.accessToken, tokenPair.refreshToken);
      return tokenPair.accessToken;
    })
    .finally(() => {
      refreshPromise = null;
    });

  return refreshPromise;
}

async function fetchApiResponse(
  path: string,
  options: ApiRequestOptions = {},
  hasRetried = false,
): Promise<Response> {
  const { auth = true, headers: initialHeaders, ...requestOptions } = options;
  const headers = new Headers(initialHeaders);

  if (
    requestOptions.body &&
    !(requestOptions.body instanceof FormData) &&
    !headers.has("Content-Type")
  ) {
    headers.set("Content-Type", "application/json");
  }

  const accessToken = getAccessToken();

  if (auth && accessToken) {
    headers.set("Authorization", `Bearer ${accessToken}`);
  }

  const response = await fetch(`${API_BASE_URL}${path}`, {
    ...requestOptions,
    headers,
  });

  if (
    response.status === 401 &&
    auth &&
    !hasRetried &&
    getRefreshToken()
  ) {
    try {
      const nextAccessToken = await refreshAccessToken();
      headers.set("Authorization", `Bearer ${nextAccessToken}`);

      return fetchApiResponse(
        path,
        {
          ...options,
          headers,
        },
        true,
      );
    } catch {
      clearTokens();
    }
  }

  return response;
}

export async function apiRawRequest(
  path: string,
  options: ApiRequestOptions = {},
) {
  const response = await fetchApiResponse(path, options);

  if (!response.ok) {
    const payload = await readResponse<never>(response);
    throw new ApiError(
      response.status,
      payload.code,
      payload.message || "请求失败",
      payload.requestId,
    );
  }

  return response;
}

export async function apiRequest<T>(
  path: string,
  options: ApiRequestOptions = {},
  hasRetried = false,
): Promise<T> {
  const response = await fetchApiResponse(path, options, hasRetried);

  const payload = await readResponse<T>(response);

  if (!response.ok || !payload.success) {
    throw new ApiError(
      response.status,
      payload.code,
      payload.message || "请求失败",
      payload.requestId,
    );
  }

  return payload.data;
}
