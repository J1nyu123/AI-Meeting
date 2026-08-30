import { describe, expect, it } from "vitest";
import { resampleTo16k } from "./pcm-microphone-source";

describe("resampleTo16k", () => {
  it("keeps 16 kHz input unchanged", () => {
    const input = new Float32Array([0, 0.25, -0.5, 1]);

    expect(resampleTo16k(input, 16_000)).toBe(input);
  });

  it("downsamples browser audio to the expected 16 kHz length", () => {
    const input = Float32Array.from(
      { length: 480 },
      (_, index) => index / 480,
    );

    const output = resampleTo16k(input, 48_000);

    expect(output).toHaveLength(160);
    expect(output[0]).toBeCloseTo(0);
    expect(output[159]).toBeGreaterThan(0.98);
  });
});
