import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@/config/env", () => ({
  resolveAppEnv: vi.fn(() => ({
    apiBaseUrl: "/api",
    apiTarget: "http://localhost:8080",
    wsBaseUrl: "ws://localhost:8080",
  })),
  resolveApiBaseUrl: vi.fn(() => "/api"),
  resolveRuntimeWsBaseUrl: vi.fn(() => "ws://localhost:8080"),
  resolveWsBaseUrl: vi.fn(() => "ws://localhost:8080"),
}));

import { AudioToTextWebSocket } from "@/services/audioToTextWs";

describe("AudioToTextWebSocket message handling", () => {
  let instance: AudioToTextWebSocket;

  beforeEach(() => {
    instance = new AudioToTextWebSocket();
  });

  it("ignores out-of-order transcription packets", () => {
    const onTranscription = vi.fn();
    instance.onTranscription = onTranscription;

    (
      instance as unknown as {
        handleMessage: (message: Record<string, unknown>) => void;
      }
    ).handleMessage({
      type: "transcription",
      displayText: "最新快照",
      revision: 20,
    });
    (
      instance as unknown as {
        handleMessage: (message: Record<string, unknown>) => void;
      }
    ).handleMessage({
      type: "transcription",
      displayText: "旧快照",
      revision: 10,
    });

    expect(onTranscription).toHaveBeenCalledTimes(1);
    expect(onTranscription).toHaveBeenCalledWith("最新快照");
  });

  it("deduplicates identical packets with the same timestamp", () => {
    const onTranscription = vi.fn();
    instance.onTranscription = onTranscription;

    const message = {
      type: "transcription",
      displayText: "重复快照",
      revision: 30,
    };
    (
      instance as unknown as {
        handleMessage: (incoming: typeof message) => void;
      }
    ).handleMessage(message);
    (
      instance as unknown as {
        handleMessage: (incoming: typeof message) => void;
      }
    ).handleMessage(message);

    expect(onTranscription).toHaveBeenCalledTimes(1);
  });

  it("publishes a final atomic snapshot", () => {
    const onTranscription = vi.fn();
    const onSnapshot = vi.fn();
    instance.onTranscription = onTranscription;
    instance.onSnapshot = onSnapshot;

    (
      instance as unknown as {
        handleMessage: (message: Record<string, unknown>) => void;
      }
    ).handleMessage({
      type: "final",
      displayText: "最终文本",
      committedText: "最终文本",
      liveText: "",
      revision: 40,
    });

    expect(onSnapshot).toHaveBeenCalledWith(
      expect.objectContaining({ status: "final", committedText: "最终文本" }),
    );
  });
});
