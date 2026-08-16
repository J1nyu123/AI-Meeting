import { describe, expect, it } from "vitest";
import { AppError, ErrorCode } from "@/lib/errors";
import { getInterviewReportErrorMessage } from "@/hooks/interview/report/useInterviewReportData";

describe("getInterviewReportErrorMessage", () => {
  it("explains that a missing report can belong to an unfinished interview", () => {
    expect(
      getInterviewReportErrorMessage(
        new AppError(
          ErrorCode.RESOURCE_NOT_FOUND,
          "Requested resource not found",
        ),
      ),
    ).toBe("面试尚未完成或报告尚未生成，请返回继续面试。");
  });
});
