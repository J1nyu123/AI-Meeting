import { useState } from "react";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { AnswerComposer } from "./AnswerComposer";

function ComposerHarness({
  onSubmit,
  onVoiceToggle,
}: {
  onSubmit: () => void;
  onVoiceToggle: () => void;
}) {
  const [value, setValue] = useState("");

  return (
    <AnswerComposer
      value={value}
      disabled={false}
      isSubmitting={false}
      voiceStatus="idle"
      voiceSupported
      voiceError={null}
      onChange={setValue}
      onSubmit={onSubmit}
      onVoiceToggle={onVoiceToggle}
    />
  );
}

describe("AnswerComposer", () => {
  it("keeps the final submission under user control", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn();
    render(
      <ComposerHarness onSubmit={onSubmit} onVoiceToggle={vi.fn()} />,
    );

    await user.type(screen.getByRole("textbox"), "这是我的最终回答");
    expect(onSubmit).not.toHaveBeenCalled();

    await user.click(screen.getByRole("button", { name: "提交回答" }));
    expect(onSubmit).toHaveBeenCalledOnce();
  });

  it("locks editing and submission while recording", async () => {
    const user = userEvent.setup();
    const onVoiceToggle = vi.fn();
    render(
      <AnswerComposer
        value="实时转写内容"
        disabled={false}
        isSubmitting={false}
        voiceStatus="listening"
        voiceSupported
        voiceError={null}
        onChange={vi.fn()}
        onSubmit={vi.fn()}
        onVoiceToggle={onVoiceToggle}
      />,
    );

    expect(screen.getByRole("textbox")).toBeDisabled();
    expect(screen.getByRole("button", { name: "提交回答" })).toBeDisabled();

    await user.click(screen.getByRole("button", { name: "停止录音" }));
    expect(onVoiceToggle).toHaveBeenCalledOnce();
  });

  it("starts voice input only after the user clicks the microphone control", async () => {
    const user = userEvent.setup();
    const onVoiceToggle = vi.fn();
    render(
      <ComposerHarness onSubmit={vi.fn()} onVoiceToggle={onVoiceToggle} />,
    );

    expect(onVoiceToggle).not.toHaveBeenCalled();
    await user.click(screen.getByRole("button", { name: "语音输入" }));
    expect(onVoiceToggle).toHaveBeenCalledOnce();
  });
});
