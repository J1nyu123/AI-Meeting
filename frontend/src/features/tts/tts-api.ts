import { apiRequest, getAccessToken } from "@/api/client";
import type { TtsTask } from "./types";

export const ttsApi = {
  synthesize(text: string, idempotencyKey: string, signal?: AbortSignal) {
    return apiRequest<TtsTask>("/media/tts/synthesize", {
      method: "POST",
      signal,
      headers: {
        "Idempotency-Key": idempotencyKey,
      },
      body: JSON.stringify({
        text,
        language: "zh",
        speed: 50,
        volume: 60,
        pitch: 50,
        audioEncoding: "lame",
        sampleRate: 16_000,
        timeoutSeconds: 60,
        pollIntervalMs: 1_000,
      }),
    });
  },

  async audioBlob(audioPath: string, signal?: AbortSignal) {
    const accessToken = getAccessToken();
    const response = await fetch(audioPath, {
      signal,
      headers: accessToken
        ? {
            Authorization: `Bearer ${accessToken}`,
          }
        : undefined,
    });

    if (!response.ok) {
      throw new Error("合成音频加载失败");
    }

    return response.blob();
  },
};
