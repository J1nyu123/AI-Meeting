import { describe, expect, it } from "vitest";
import { normalizeTaskResult } from "@/services/xunfeiTtsService";

describe("xunfeiTtsService", () => {
  it("treats numeric task status and string code as a completed successful task", () => {
    const result = normalizeTaskResult({
      taskStatus: 5 as unknown as string,
      code: "0" as unknown as number,
      message: "ok",
    });

    expect(result.taskStatus).toBe("5");
    expect(result.code).toBe(0);
    expect(result.completed).toBe(true);
    expect(result.success).toBe(true);
  });

  it("maps pybuf fields into playable audio fields", () => {
    const result = normalizeTaskResult({
      taskStatus: "5",
      pybufContent: "QQ==",
      pybufUrl: "https://example.com/audio.mp3",
    });

    expect(result.audioBase64).toBe("QQ==");
    expect(result.audioUrl).toBe("https://example.com/audio.mp3");
    expect(result.completed).toBe(true);
    expect(result.success).toBe(true);
  });

  it("prefers the authenticated application audio path", () => {
    const result = normalizeTaskResult({
      taskStatus: "5",
      code: 0,
      audioPath: "/api/v1/media/tts/tasks/task-1/audio",
      audioUrl: "https://provider.example/audio",
    });
    expect(result.audioUrl).toBe("/api/v1/media/tts/tasks/task-1/audio");
  });
});
