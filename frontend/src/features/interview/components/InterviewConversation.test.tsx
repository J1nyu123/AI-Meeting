import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { InterviewConversation } from "./InterviewConversation";

describe("InterviewConversation", () => {
  it("exposes playback controls for the current question", async () => {
    const user = userEvent.setup();
    const onToggle = vi.fn();
    render(
      <InterviewConversation
        turns={[]}
        currentQuestion={{
          number: "1",
          content: "请介绍一个你解决过的复杂问题。",
          isFollowUp: false,
        }}
        isSubmitting={false}
        speech={{
          status: "idle",
          error: null,
          disabled: false,
          onToggle,
        }}
      />,
    );

    expect(screen.getByText("请介绍一个你解决过的复杂问题。")).toBeVisible();
    await user.click(screen.getByRole("button", { name: "播放题目" }));
    expect(onToggle).toHaveBeenCalledOnce();
  });

  it("shows the stop action while the interviewer is speaking", () => {
    render(
      <InterviewConversation
        turns={[]}
        currentQuestion={{
          number: "2",
          content: "为什么选择这个技术方案？",
          isFollowUp: true,
        }}
        isSubmitting={false}
        speech={{
          status: "playing",
          error: null,
          disabled: false,
          onToggle: vi.fn(),
        }}
      />,
    );

    expect(screen.getByText("动态追问")).toBeVisible();
    expect(screen.getByRole("button", { name: "停止播报" })).toBeEnabled();
  });
});
