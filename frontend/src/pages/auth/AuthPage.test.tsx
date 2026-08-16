import { render } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import AuthPage from "@/pages/auth/AuthPage";
import { AUTH_BACKGROUND_VIDEO_SRC } from "@/pages/auth/authPage.constants";

vi.mock("@/hooks/auth/useAuthPageController", () => ({
  useAuthPageController: () => ({
    mode: "login",
    formData: { username: "", password: "", confirmPassword: "" },
    loading: false,
    error: "",
    registerLoading: false,
    localError: "",
    switchMode: vi.fn(),
    handleInputChange: vi.fn(),
    handleSubmit: vi.fn(),
  }),
}));

vi.mock("@/components/auth/AuthMarketingPanel", () => ({
  default: () => <div>marketing</div>,
}));

vi.mock("@/components/auth/AuthFormCard", () => ({
  default: () => <div>form</div>,
}));

describe("AuthPage background", () => {
  it("uses the replaceable login video with safe playback attributes", () => {
    const { container } = render(<AuthPage />);
    const video = container.querySelector("video");

    expect(video?.getAttribute("src")).toBe(AUTH_BACKGROUND_VIDEO_SRC);
    expect(video?.hasAttribute("autoplay")).toBe(true);
    expect(video?.hasAttribute("loop")).toBe(true);
    expect(video?.hasAttribute("playsinline")).toBe(true);
    expect(video?.muted).toBe(true);
  });
});
