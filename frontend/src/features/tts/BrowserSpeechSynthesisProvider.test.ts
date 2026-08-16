import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { BrowserSpeechSynthesisProvider } from "./BrowserSpeechSynthesisProvider";

type MockUtterance = {
  text: string;
  lang: string;
  rate: number;
  pitch: number;
  onend: (() => void) | null;
  onerror: ((event: { error?: string }) => void) | null;
};

describe("BrowserSpeechSynthesisProvider", () => {
  let active: MockUtterance | null;
  const speak = vi.fn((utterance: MockUtterance) => {
    active = utterance;
  });
  const cancel = vi.fn();
  const resume = vi.fn();

  beforeEach(() => {
    active = null;
    speak.mockClear();
    cancel.mockClear();
    resume.mockClear();
    class Utterance {
      text: string;
      lang = "";
      rate = 1;
      pitch = 1;
      onend: (() => void) | null = null;
      onerror: ((event: { error?: string }) => void) | null = null;
      constructor(text: string) {
        this.text = text;
      }
    }
    vi.stubGlobal("SpeechSynthesisUtterance", Utterance);
    Object.defineProperty(window, "speechSynthesis", {
      configurable: true,
      value: { speak, cancel, resume },
    });
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("speaks Chinese text and resolves when playback ends", async () => {
    const provider = new BrowserSpeechSynthesisProvider();
    const playback = provider.speak("追问内容");
    expect(provider.isSupported()).toBe(true);
    expect(speak).toHaveBeenCalledOnce();
    expect(active?.text).toBe("追问内容");
    expect(active?.lang).toBe("zh-CN");
    active?.onend?.();
    await expect(playback).resolves.toBeUndefined();
  });

  it("cancels playback when the request is aborted", async () => {
    const provider = new BrowserSpeechSynthesisProvider();
    const controller = new AbortController();
    const playback = provider.speak("question", { signal: controller.signal });
    controller.abort();
    await expect(playback).rejects.toMatchObject({ name: "AbortError" });
    expect(cancel).toHaveBeenCalled();
  });
});
