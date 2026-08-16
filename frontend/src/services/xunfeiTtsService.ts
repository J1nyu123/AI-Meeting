import service from "@/lib/request";

const TERMINAL_TASK_STATUSES = new Set(["2", "4", "5"]);

export type CreateXunfeiTtsTaskParams = {
  text: string;
  vcn?: string;
  language?: string;
  speed?: number;
  volume?: number;
  pitch?: number;
  rhy?: number;
  audioEncoding?: string;
  sampleRate?: number;
  timeoutSeconds?: number;
  pollIntervalMs?: number;
};

export type XunfeiTtsTaskResult = {
  sid?: string;
  taskId?: string;
  taskStatus?: string;
  code?: number;
  message?: string;
  audioBase64?: string | null;
  audioUrl?: string | null;
  audioPath?: string | null;
  pybufContent?: string | null;
  pybufUrl?: string | null;
  completed?: boolean;
  success?: boolean;
};

type RequestOptions = {
  signal?: AbortSignal;
  idempotencyKey?: string;
};

const toTrimmedString = (value: unknown) => {
  if (typeof value !== "string" && typeof value !== "number") {
    return undefined;
  }

  const normalized = String(value).trim();
  return normalized.length > 0 ? normalized : undefined;
};

const toOptionalNumber = (value: unknown) => {
  if (typeof value === "number" && Number.isFinite(value)) {
    return value;
  }

  if (typeof value === "string" && value.trim() !== "") {
    const parsed = Number(value);
    return Number.isFinite(parsed) ? parsed : undefined;
  }

  return undefined;
};

export const normalizeTaskResult = (
  payload: XunfeiTtsTaskResult,
): XunfeiTtsTaskResult => {
  const taskStatus = toTrimmedString(payload.taskStatus);
  const code = toOptionalNumber(payload.code);
  const audioBase64 =
    toTrimmedString(payload.audioBase64) ??
    toTrimmedString(payload.pybufContent) ??
    null;
  const audioUrl =
    toTrimmedString(payload.audioPath) ??
    toTrimmedString(payload.audioUrl) ??
    toTrimmedString(payload.pybufUrl) ??
    null;
  const completed =
    typeof payload.completed === "boolean"
      ? payload.completed
      : Boolean(taskStatus && TERMINAL_TASK_STATUSES.has(taskStatus));
  const success =
    typeof payload.success === "boolean"
      ? payload.success
      : taskStatus === "5" || (completed && code === 0);

  return {
    sid: toTrimmedString(payload.sid),
    taskId: toTrimmedString(payload.taskId),
    taskStatus,
    code,
    message: toTrimmedString(payload.message),
    audioBase64,
    audioUrl,
    audioPath: toTrimmedString(payload.audioPath) ?? null,
    pybufContent: toTrimmedString(payload.pybufContent) ?? null,
    pybufUrl: toTrimmedString(payload.pybufUrl) ?? null,
    completed,
    success,
  };
};

const toTaskError = (task: XunfeiTtsTaskResult, fallback: string) =>
  new Error(task.message || fallback);

export const xunfeiTtsService = {
  async createTask(
    params: CreateXunfeiTtsTaskParams,
    options?: RequestOptions,
  ) {
    const response = await service.post<
      XunfeiTtsTaskResult,
      CreateXunfeiTtsTaskParams
    >("/v1/media/tts/tasks", params, {
      signal: options?.signal,
      headers: { "Idempotency-Key": options?.idempotencyKey || "" },
    });
    return normalizeTaskResult(response);
  },

  async queryTask(taskId: string, options?: RequestOptions) {
    const response = await service.get<XunfeiTtsTaskResult>(
      `/v1/media/tts/tasks/${encodeURIComponent(taskId)}`,
      {
        signal: options?.signal,
      },
    );
    return normalizeTaskResult(response);
  },

  async synthesize(
    params: CreateXunfeiTtsTaskParams,
    options?: RequestOptions,
  ) {
    return this.createTaskAndWait(params, options);
  },

  async createTaskAndWait(
    params: CreateXunfeiTtsTaskParams,
    options?: RequestOptions,
  ) {
    let task = await this.createTask(params, options);
    const timeoutMs = Math.max(10, params.timeoutSeconds || 90) * 1000;
    const intervalMs = Math.max(500, params.pollIntervalMs || 1500);
    const deadline = Date.now() + timeoutMs;
    while (!task.completed && Date.now() < deadline) {
      await new Promise<void>((resolve, reject) => {
        const timer = window.setTimeout(resolve, intervalMs);
        options?.signal?.addEventListener(
          "abort",
          () => {
            window.clearTimeout(timer);
            reject(new DOMException("Aborted", "AbortError"));
          },
          { once: true },
        );
      });
      if (!task.taskId) throw new Error("TTS task id is missing");
      task = await this.queryTask(task.taskId, options);
    }
    if (!task.completed || !task.success) {
      throw toTaskError(task, task.completed ? "TTS synthesis failed" : "TTS synthesis timed out");
    }
    if (!task.audioUrl) throw toTaskError(task, "TTS synthesis completed without audio content");
    return task;
  },
};
