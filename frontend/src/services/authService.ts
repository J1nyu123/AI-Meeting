import service from "@/lib/request";
import {
  clearAuthToken,
  getRefreshToken,
  setAuthToken,
  setRefreshToken,
} from "@/lib/authToken";
import { AppError, ErrorCode } from "@/lib/errors";
import type {
  AuthPayloadDTO,
  ResultBoolean,
  ResultVoid,
  UserActualRespDTO,
  UserLoginReqDTO,
  UserRegisterReqDTO,
  UserRespDTO,
} from "@/types/auth";

const isRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === "object" && value !== null;

const toNumber = (value: unknown): number | undefined => {
  if (typeof value === "number") return Number.isNaN(value) ? undefined : value;
  if (typeof value === "string" && value.trim() !== "") {
    const parsed = Number(value);
    return Number.isNaN(parsed) ? undefined : parsed;
  }
  return undefined;
};

const toString = (value: unknown): string | undefined => {
  if (typeof value === "string") return value;
  if (typeof value === "number" || typeof value === "boolean")
    return String(value);
  return undefined;
};

const normalizeUser = (raw: unknown): UserRespDTO | null => {
  if (!isRecord(raw)) return null;

  const username = toString(raw.username) || "";
  if (!username) return null;

  return {
    id: toNumber(raw.id),
    username,
    realName: toString(raw.realName ?? raw.real_name),
    phone: toString(raw.phone),
    mail: toString(raw.mail),
    avatar: toString(raw.avatar),
    deletionTime: toNumber(raw.deletionTime ?? raw.deletion_time),
    createTime: toString(raw.createTime ?? raw.create_time ?? raw.createdAt),
    updateTime: toString(raw.updateTime ?? raw.update_time ?? raw.updatedAt),
    delFlag: toNumber(raw.delFlag ?? raw.del_flag) as 0 | 1 | undefined,
  };
};

const extractUserFromAuthPayload = (payload: unknown): UserRespDTO | null => {
  const direct = normalizeUser(payload);
  if (direct) return direct;
  if (!isRecord(payload)) return null;
  return normalizeUser(payload.user) || normalizeUser(payload.currentUser);
};

const extractTokenFromAuthPayload = (payload: unknown): string | null => {
  if (!isRecord(payload)) return null;

  const directToken = toString(
    payload.token ?? payload.accessToken ?? payload.authToken,
  )?.trim();
  if (directToken) return directToken;

  const nestedData = payload.data;
  if (isRecord(nestedData)) {
    const nestedToken = toString(
      nestedData.token ?? nestedData.accessToken ?? nestedData.authToken,
    )?.trim();
    if (nestedToken) return nestedToken;
  }

  return null;
};

const extractRefreshToken = (payload: unknown): string | null => {
  if (!isRecord(payload)) return null;
  return (
    toString(payload.refreshToken ?? payload.refresh_token)?.trim() || null
  );
};

export const authService = {
  login: async (data: UserLoginReqDTO) => {
    const payload = await service.post<AuthPayloadDTO>("/v1/auth/login", data);
    const token = extractTokenFromAuthPayload(payload);
    if (token) {
      setAuthToken(token);
    }
    const refreshToken = extractRefreshToken(payload);
    if (refreshToken) setRefreshToken(refreshToken);

    const user = extractUserFromAuthPayload(payload);
    if (!user) {
      throw new Error("Login succeeded but user info is missing");
    }
    if (!token) {
      throw new Error("Login succeeded but token is missing");
    }
    return user;
  },

  register: (data: UserRegisterReqDTO) => {
    return service.post<ResultVoid>("/v1/auth/register", data);
  },

  checkLogin: async () => {
    const payload = await service.get<AuthPayloadDTO>("/v1/auth/me");
    const token = extractTokenFromAuthPayload(payload);
    if (token) {
      setAuthToken(token);
    }

    const user = extractUserFromAuthPayload(payload);
    if (!user) {
      throw new AppError(ErrorCode.UNAUTHORIZED, "User is not logged in");
    }
    return user;
  },

  logout: async () => {
    try {
      return await service.post<ResultVoid>("/v1/auth/logout", {
        refreshToken: getRefreshToken(),
      });
    } finally {
      clearAuthToken();
    }
  },

  getUser: (username: string) => {
    void username;
    return service.get<UserRespDTO>("/v1/auth/me");
  },

  getUserActual: (username: string) => {
    void username;
    return service.get<UserActualRespDTO>("/v1/auth/me");
  },

  hasUsername: (username: string) => {
    void username;
    return Promise.resolve(false as ResultBoolean);
  },
};
