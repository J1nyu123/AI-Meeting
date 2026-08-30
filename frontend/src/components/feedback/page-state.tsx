import type { LucideIcon } from "lucide-react";
import { AlertCircle, AlertTriangle, LoaderCircle } from "lucide-react";
import type { ReactNode } from "react";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { cn } from "@/lib/utils";

interface PageLoadingProps {
  title?: string;
  description?: string;
  fullPage?: boolean;
  className?: string;
}

export function PageLoading({
  title = "正在加载",
  description = "请稍候，马上就好。",
  fullPage = false,
  className,
}: PageLoadingProps) {
  return (
    <div
      role="status"
      aria-live="polite"
      className={cn(
        "flex min-h-64 flex-col items-center justify-center px-4 text-center",
        fullPage && "min-h-[calc(100svh-3.5rem)]",
        className,
      )}
    >
      <LoaderCircle className="size-6 animate-spin text-neutral-400" />
      <p className="mt-4 text-sm font-medium text-neutral-700">{title}</p>
      <p className="mt-1 text-sm text-neutral-500">{description}</p>
    </div>
  );
}

interface EmptyStateProps {
  icon: LucideIcon;
  title: string;
  description: string;
  action?: ReactNode;
  className?: string;
}

export function EmptyState({
  icon: Icon,
  title,
  description,
  action,
  className,
}: EmptyStateProps) {
  return (
    <Card className={cn("border-dashed bg-neutral-50/50 shadow-none", className)}>
      <CardHeader className="items-center pt-10 text-center">
        <div className="flex size-12 items-center justify-center rounded-full bg-white ring-1 ring-neutral-200">
          <Icon className="size-5 text-neutral-500" />
        </div>
        <CardTitle className="mt-3">{title}</CardTitle>
        <CardDescription className="max-w-md">{description}</CardDescription>
      </CardHeader>
      {action && (
        <CardContent className="flex justify-center pb-10">{action}</CardContent>
      )}
    </Card>
  );
}

interface ErrorStateProps {
  title?: string;
  message: string;
  onRetry?: () => void;
  isRetrying?: boolean;
  secondaryAction?: ReactNode;
  className?: string;
}

export function ErrorState({
  title = "加载失败",
  message,
  onRetry,
  isRetrying = false,
  secondaryAction,
  className,
}: ErrorStateProps) {
  return (
    <Card
      role="alert"
      className={cn("border-red-200 bg-red-50/70 shadow-none", className)}
    >
      <CardHeader>
        <div className="flex size-10 items-center justify-center rounded-full bg-white text-red-600 ring-1 ring-red-200">
          <AlertTriangle className="size-5" />
        </div>
        <CardTitle className="mt-2 text-red-950">{title}</CardTitle>
        <CardDescription className="text-red-700">{message}</CardDescription>
      </CardHeader>
      {(onRetry || secondaryAction) && (
        <CardContent className="flex flex-wrap gap-3">
          {onRetry && (
            <Button variant="outline" disabled={isRetrying} onClick={onRetry}>
              {isRetrying && <LoaderCircle className="animate-spin" />}
              {isRetrying ? "正在重试" : "重新加载"}
            </Button>
          )}
          {secondaryAction}
        </CardContent>
      )}
    </Card>
  );
}

export function InlineError({ message }: { message: string }) {
  return (
    <div
      role="alert"
      className="flex items-start gap-2 rounded-xl border border-red-200 bg-red-50 px-3 py-2.5 text-sm text-red-700"
    >
      <AlertCircle className="mt-0.5 size-4 shrink-0" />
      <span>{message}</span>
    </div>
  );
}

interface ContentEmptyStateProps {
  icon: LucideIcon;
  title: string;
  description?: string;
  className?: string;
}

export function ContentEmptyState({
  icon: Icon,
  title,
  description,
  className,
}: ContentEmptyStateProps) {
  return (
    <div className={cn("flex flex-col items-center px-4 py-10 text-center", className)}>
      <div className="flex size-10 items-center justify-center rounded-full bg-neutral-100 text-neutral-500">
        <Icon className="size-4" />
      </div>
      <p className="mt-3 text-sm font-medium text-neutral-700">{title}</p>
      {description && (
        <p className="mt-1 max-w-sm text-sm text-neutral-500">{description}</p>
      )}
    </div>
  );
}
