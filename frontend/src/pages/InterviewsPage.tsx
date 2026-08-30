import { ArrowRight, FilePlus2, MessageSquareText, Mic, Sparkles } from "lucide-react";
import { Link } from "react-router-dom";
import { Button } from "@/components/ui/button";

export function InterviewsPage() {
  return (
    <div className="flex min-h-[calc(100svh-3rem)] flex-col md:min-h-svh">
      <header className="hidden h-14 shrink-0 items-center border-b border-neutral-200 px-6 md:flex">
        <div className="flex items-center gap-2 text-sm text-neutral-500">
          <MessageSquareText className="size-4" />
          面试工作区
        </div>
      </header>

      <div className="flex flex-1 items-center justify-center px-5 py-12">
        <div className="w-full max-w-xl text-center">
          <div className="mx-auto flex size-14 items-center justify-center rounded-2xl bg-neutral-950 text-white shadow-sm">
            <Sparkles className="size-6" />
          </div>
          <h1 className="mt-6 text-3xl font-semibold tracking-tight">
            准备好开始模拟面试了吗？
          </h1>
          <p className="mx-auto mt-3 max-w-md text-sm leading-7 text-neutral-500">
            从左侧选择一场历史面试继续对话或查看报告，也可以上传简历创建一场新的个性化面试。
          </p>

          <Button asChild size="lg" className="mt-7">
            <Link to="/interviews/new">
              <FilePlus2 data-icon="inline-start" />
              新建面试
              <ArrowRight data-icon="inline-end" />
            </Link>
          </Button>

          <div className="mt-12 grid gap-3 text-left sm:grid-cols-2">
            <div className="rounded-2xl border border-neutral-200 p-4">
              <div className="flex size-9 items-center justify-center rounded-xl bg-neutral-100">
                <Mic className="size-4" />
              </div>
              <p className="mt-3 text-sm font-medium">沉浸式面试对话</p>
              <p className="mt-1 text-xs leading-5 text-neutral-500">
                在右侧区域完成问答，进度会自动保留。
              </p>
            </div>
            <div className="rounded-2xl border border-neutral-200 p-4">
              <div className="flex size-9 items-center justify-center rounded-xl bg-neutral-100">
                <MessageSquareText className="size-4" />
              </div>
              <p className="mt-3 text-sm font-medium">历史面试集中管理</p>
              <p className="mt-1 text-xs leading-5 text-neutral-500">
                快速切换面试、查看报告或删除记录。
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
