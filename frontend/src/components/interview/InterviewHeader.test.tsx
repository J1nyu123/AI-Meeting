import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import InterviewHeader from "@/components/interview/InterviewHeader";

describe("InterviewHeader", () => {
  it("uses the generic interview room title", () => {
    render(
      <InterviewHeader
        isReady
        currentQuestionNumber="1"
        isCurrentQuestionFollowUp={false}
        currentFollowUpCount={0}
        isInterviewFinished={false}
        totalInterviewScore={null}
        isCameraOpen={false}
        isEndingInterview={false}
        onToggleCamera={vi.fn()}
        onOpenSketchpad={vi.fn()}
        onEndInterview={vi.fn()}
      />,
    );

    expect(screen.getByText("模拟面试室")).toBeTruthy();
    expect(screen.queryByText(/Java 高级开发工程师/)).toBeNull();
  });
});
