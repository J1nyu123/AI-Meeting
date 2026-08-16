import { fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";
import SidebarNav from "@/components/layout/sidebar/SidebarNav";
import SidebarUserMenu from "@/components/layout/sidebar/SidebarUserMenu";

describe("sidebar available features", () => {
  it("shows AI interview without the unimplemented chat entry", () => {
    render(
      <MemoryRouter>
        <SidebarNav />
      </MemoryRouter>,
    );

    expect(screen.getByText("AI 面试")).toBeTruthy();
    expect(screen.queryByText("新对话")).toBeNull();
  });

  it("opens read-only account information", () => {
    render(
      <SidebarUserMenu
        username="demo_candidate_04"
        createTime="2026-08-15T10:00:00Z"
        userInitials="DE"
        onLogout={vi.fn()}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: /demo_candidate_04/ }));
    fireEvent.click(screen.getByText("账号信息"));

    expect(screen.getByText("当前账号的只读信息")).toBeTruthy();
    expect(screen.getByText("注册时间")).toBeTruthy();
  });
});
