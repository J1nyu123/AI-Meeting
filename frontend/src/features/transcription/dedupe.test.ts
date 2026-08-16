import { describe, expect, it } from "vitest";
import { mergeTranscript } from "./dedupe";

describe("mergeTranscript", () => {
  it("removes exact and contained snapshots", () => {
    expect(mergeTranscript("我负责后端", "我负责后端")).toBe("我负责后端");
    expect(mergeTranscript("我负责后端接口", "负责后端")).toBe("我负责后端接口");
  });
  it("merges suffix-prefix overlap", () => {
    expect(mergeTranscript("系统使用 Redis 保存", "Redis 保存会话状态")).toBe(
      "系统使用 Redis 保存会话状态",
    );
  });
});
