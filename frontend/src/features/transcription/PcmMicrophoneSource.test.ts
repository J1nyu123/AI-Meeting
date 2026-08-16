import { describe, expect, it } from "vitest";
import { resampleTo16k } from "./PcmMicrophoneSource";

describe("resampleTo16k", () => {
  it("keeps 16 kHz samples unchanged", () => {
    const input = new Float32Array([0, 0.5, -0.5]);
    expect(resampleTo16k(input, 16_000)).toBe(input);
  });

  it("resamples 48 kHz input to one third of the samples", () => {
    const input = new Float32Array(1_920);
    const output = resampleTo16k(input, 48_000);
    expect(output).toHaveLength(640);
  });
});
