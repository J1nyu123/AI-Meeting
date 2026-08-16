import { act, renderHook } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { useAudioTranscriptionController } from "./useAudioTranscriptionController";

class FakeRecognition {
  static latest: FakeRecognition | null = null;
  lang = "";
  continuous = false;
  interimResults = false;
  onresult: ((event: unknown) => void) | null = null;
  onerror: ((event: unknown) => void) | null = null;
  onend: (() => void) | null = null;
  start = vi.fn();
  stop = vi.fn();
  abort = vi.fn();
  constructor() {
    FakeRecognition.latest = this;
  }
}

const user = { id: 1, username: "candidate" };

describe("useAudioTranscriptionController browser provider", () => {
  afterEach(() => {
    delete (window as Window & { webkitSpeechRecognition?: unknown })
      .webkitSpeechRecognition;
    FakeRecognition.latest = null;
  });

  it("degrades to text input when browser speech is unsupported", async () => {
    const { result } = renderHook(() =>
      useAudioTranscriptionController(user),
    );
    await act(async () => result.current.startRecording());
    expect(result.current.isRecording).toBe(false);
    expect(result.current.error).toContain("文字输入");
    expect(result.current.isSpeechSupported).toBe(false);
  });

  it("merges partial and final browser speech results", async () => {
    (
      window as Window & { webkitSpeechRecognition?: typeof FakeRecognition }
    ).webkitSpeechRecognition = FakeRecognition;
    const { result } = renderHook(() =>
      useAudioTranscriptionController(user),
    );
    await act(async () => result.current.startRecording());
    expect(result.current.isRecording).toBe(true);
    const recognition = FakeRecognition.latest;
    expect(recognition).not.toBeNull();

    act(() => {
      recognition?.onresult?.({
        resultIndex: 0,
        results: {
          length: 1,
          0: { isFinal: false, length: 1, 0: { transcript: "Redis 保存" } },
        },
      });
    });
    expect(result.current.currentSentence).toBe("Redis 保存");

    act(() => {
      recognition?.onresult?.({
        resultIndex: 0,
        results: {
          length: 1,
          0: {
            isFinal: true,
            length: 1,
            0: { transcript: "Redis 保存会话状态" },
          },
        },
      });
    });
    expect(result.current.transcription).toBe("Redis 保存会话状态");
    act(() => result.current.stopRecording());
    expect(recognition?.stop).toHaveBeenCalledTimes(1);
  });

  it("requires a logged-in user before starting", async () => {
    const { result } = renderHook(() => useAudioTranscriptionController(null));
    await expect(result.current.startRecording()).rejects.toThrow(
      "User is not logged in",
    );
  });
});
