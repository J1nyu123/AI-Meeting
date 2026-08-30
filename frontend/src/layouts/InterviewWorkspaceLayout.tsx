import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  FilePlus2,
  Home,
  LoaderCircle,
  LogOut,
  Menu,
  MessageSquareText,
  Trash2,
  X,
} from "lucide-react";
import { useState } from "react";
import { Link, Outlet, useLocation, useNavigate } from "react-router-dom";
import { toast } from "sonner";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import { useAuth } from "@/features/auth/auth-context";
import { interviewApi } from "@/features/interview/interview-api";
import type {
  InterviewSession,
  InterviewStatus,
} from "@/features/interview/types";
import { getErrorMessage } from "@/lib/errors";
import { cn } from "@/lib/utils";

const statusLabels: Record<InterviewStatus, string> = {
  CREATED: "待上传",
  ANALYZING: "分析中",
  READY: "待开始",
  IN_PROGRESS: "进行中",
  COMPLETED: "已完成",
  FAILED: "失败",
};

const statusDots: Record<InterviewStatus, string> = {
  CREATED: "bg-neutral-400",
  ANALYZING: "bg-blue-500",
  READY: "bg-emerald-500",
  IN_PROGRESS: "bg-amber-500",
  COMPLETED: "bg-neutral-500",
  FAILED: "bg-red-500",
};

function recordPath(record: InterviewSession) {
  const sessionPath = `/interviews/${encodeURIComponent(record.sessionId)}`;
  return record.status === "COMPLETED" ? `${sessionPath}/report` : sessionPath;
}

function formatSidebarDate(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "时间未知";

  const now = new Date();
  if (date.toDateString() === now.toDateString()) {
    return new Intl.DateTimeFormat("zh-CN", {
      hour: "2-digit",
      minute: "2-digit",
    }).format(date);
  }

  return new Intl.DateTimeFormat("zh-CN", {
    month: "numeric",
    day: "numeric",
  }).format(date);
}

function InterviewHistoryItem({
  record,
  active,
  deleting,
  onNavigate,
  onDelete,
}: {
  record: InterviewSession;
  active: boolean;
  deleting: boolean;
  onNavigate: () => void;
  onDelete: () => void;
}) {
  return (
    <div
      className={cn(
        "group relative flex items-center rounded-xl text-sm transition-colors",
        active
          ? "bg-neutral-200/75 text-neutral-950"
          : "text-neutral-700 hover:bg-neutral-200/55",
      )}
    >
      <Link
        to={recordPath(record)}
        className="min-w-0 flex-1 px-3 py-2.5 pr-9"
        onClick={onNavigate}
      >
        <div className="truncate font-medium">
          {record.direction || "AI 模拟面试"}
        </div>
        <div className="mt-1 flex items-center gap-1.5 text-xs text-neutral-500">
          <span className={cn("size-1.5 rounded-full", statusDots[record.status])} />
          <span>{statusLabels[record.status]}</span>
          <span>·</span>
          <span>{formatSidebarDate(record.createdAt)}</span>
        </div>
      </Link>

      <AlertDialog>
        <AlertDialogTrigger asChild>
          <Button
            type="button"
            variant="ghost"
            size="icon-sm"
            aria-label="删除面试记录"
            className={cn(
              "absolute right-1.5 size-7 text-neutral-500 hover:bg-white hover:text-red-600",
              active
                ? "opacity-100"
                : "opacity-0 group-hover:opacity-100 focus-visible:opacity-100",
            )}
          >
            <Trash2 className="size-3.5" />
          </Button>
        </AlertDialogTrigger>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>删除这场面试？</AlertDialogTitle>
            <AlertDialogDescription>
              面试进度、回答记录、简历和报告都会被永久删除，此操作无法撤销。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              disabled={deleting}
              onClick={onDelete}
            >
              {deleting ? "删除中" : "确认删除"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}

export function InterviewWorkspaceLayout() {
  const location = useLocation();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { user, logout } = useAuth();
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [isLoggingOut, setIsLoggingOut] = useState(false);

  const interviewsQuery = useQuery({
    queryKey: ["interviews"],
    queryFn: () => interviewApi.list(),
  });

  const deleteMutation = useMutation({
    mutationFn: (sessionId: string) => interviewApi.delete(sessionId),
    onSuccess: async (_, sessionId) => {
      await queryClient.invalidateQueries({ queryKey: ["interviews"] });
      toast.success("面试记录已删除");

      if (location.pathname.includes(sessionId)) {
        navigate("/interviews", { replace: true });
      }
    },
    onError: (error) => {
      toast.error("删除失败", {
        description: getErrorMessage(error, "请稍后再试。"),
      });
    },
  });

  async function handleLogout() {
    if (isLoggingOut) return;

    setIsLoggingOut(true);
    try {
      await logout();
      toast.success("已安全退出");
    } catch {
      toast.warning("已清除本机登录状态");
    } finally {
      navigate("/", { replace: true });
      setIsLoggingOut(false);
    }
  }

  const records = interviewsQuery.data?.records ?? [];

  const sidebar = (
    <aside className="flex h-full w-72 shrink-0 flex-col bg-neutral-100/90">
      <div className="flex h-14 items-center justify-between px-3">
        <Link
          to="/interviews"
          className="flex min-w-0 items-center gap-2 rounded-lg px-2 py-1.5 font-semibold hover:bg-neutral-200/60"
          onClick={() => setSidebarOpen(false)}
        >
          <span className="flex size-7 items-center justify-center rounded-lg bg-neutral-950 text-white">
            <MessageSquareText className="size-3.5" />
          </span>
          <span className="truncate">AI Meeting</span>
        </Link>
        <Button
          type="button"
          variant="ghost"
          size="icon-sm"
          aria-label="关闭侧栏"
          className="md:hidden"
          onClick={() => setSidebarOpen(false)}
        >
          <X />
        </Button>
      </div>

      <div className="px-3 pt-2">
        <Button asChild variant="outline" className="w-full justify-start bg-white">
          <Link to="/interviews/new" onClick={() => setSidebarOpen(false)}>
            <FilePlus2 />
            新建面试
          </Link>
        </Button>
      </div>

      <div className="mt-6 flex min-h-0 flex-1 flex-col">
        <div className="flex items-center justify-between px-5 pb-2">
          <p className="text-xs font-medium text-neutral-500">历史面试</p>
          {interviewsQuery.data && (
            <span className="text-xs text-neutral-400">{interviewsQuery.data.total}</span>
          )}
        </div>

        <div className="min-h-0 flex-1 overflow-y-auto px-2 pb-4">
          {interviewsQuery.isLoading && (
            <div className="flex items-center gap-2 px-3 py-4 text-sm text-neutral-500">
              <LoaderCircle className="size-4 animate-spin" />
              加载记录中
            </div>
          )}

          {interviewsQuery.error && (
            <div className="mx-1 rounded-xl border border-red-200 bg-red-50 p-3 text-xs text-red-700">
              <p>历史记录加载失败</p>
              <button
                type="button"
                className="mt-2 font-medium underline underline-offset-2"
                onClick={() => void interviewsQuery.refetch()}
              >
                重新加载
              </button>
            </div>
          )}

          {!interviewsQuery.isLoading && !interviewsQuery.error && records.length === 0 && (
            <p className="px-3 py-4 text-sm leading-6 text-neutral-500">
              暂无历史记录，创建一场面试后会显示在这里。
            </p>
          )}

          <div className="space-y-0.5">
            {records.map((record) => (
              <InterviewHistoryItem
                key={record.sessionId}
                record={record}
                active={location.pathname.includes(record.sessionId)}
                deleting={
                  deleteMutation.isPending &&
                  deleteMutation.variables === record.sessionId
                }
                onNavigate={() => setSidebarOpen(false)}
                onDelete={() => deleteMutation.mutate(record.sessionId)}
              />
            ))}
          </div>
        </div>
      </div>

      <div className="border-t border-neutral-200 p-2">
        <Link
          to="/"
          className="flex items-center gap-2 rounded-xl px-3 py-2 text-sm text-neutral-600 hover:bg-neutral-200/60 hover:text-neutral-950"
        >
          <Home className="size-4" />
          返回首页
        </Link>
        <div className="mt-1 flex items-center gap-2 rounded-xl px-3 py-2">
          <div className="flex size-7 items-center justify-center rounded-full bg-neutral-900 text-xs font-medium text-white">
            {user?.username?.slice(0, 1).toUpperCase() || "U"}
          </div>
          <span className="min-w-0 flex-1 truncate text-sm">{user?.username}</span>
          <Button
            type="button"
            variant="ghost"
            size="icon-sm"
            aria-label="退出登录"
            disabled={isLoggingOut}
            onClick={handleLogout}
          >
            {isLoggingOut ? (
              <LoaderCircle className="animate-spin" />
            ) : (
              <LogOut />
            )}
          </Button>
        </div>
      </div>
    </aside>
  );

  return (
    <div className="flex h-svh overflow-hidden bg-white text-neutral-950">
      <div className="hidden md:block">{sidebar}</div>

      {sidebarOpen && (
        <div className="fixed inset-0 z-50 flex md:hidden">
          <button
            type="button"
            aria-label="关闭侧栏"
            className="absolute inset-0 bg-black/20 backdrop-blur-[1px]"
            onClick={() => setSidebarOpen(false)}
          />
          <div className="relative h-full shadow-xl">{sidebar}</div>
        </div>
      )}

      <main className="flex min-w-0 flex-1 flex-col overflow-hidden">
        <div className="sticky top-0 z-40 flex h-12 items-center justify-between border-b border-neutral-200 bg-white/90 px-3 backdrop-blur md:hidden">
          <Button
            type="button"
            variant="ghost"
            size="icon-sm"
            aria-label="打开历史面试侧栏"
            onClick={() => setSidebarOpen(true)}
          >
            <Menu />
          </Button>
          <span className="text-sm font-medium">AI Meeting</span>
          <Button asChild variant="ghost" size="icon-sm">
            <Link to="/interviews/new" aria-label="新建面试">
              <FilePlus2 />
            </Link>
          </Button>
        </div>
        <div className="min-h-0 flex-1 overflow-y-auto">
          <Outlet />
        </div>
      </main>
    </div>
  );
}
