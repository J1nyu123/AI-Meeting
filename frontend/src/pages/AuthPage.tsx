import { useEffect, useState, type FormEvent } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { LoaderCircle, MessageSquareText } from "lucide-react";
import { toast } from "sonner";
import { ApiError } from "@/api/client";
import { InlineError } from "@/components/feedback/page-state";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useAuth } from "@/features/auth/auth-context";

type AuthMode = "login" | "register";

export function AuthPage() {
  const [mode, setMode] = useState<AuthMode>("login");
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState("");

  const navigate = useNavigate();
  const location = useLocation();
  const { login, register, isAuthenticated } = useAuth();

  const isLogin = mode === "login";
  const destination =
    (location.state as { from?: string } | null)?.from || "/interviews";

  useEffect(() => {
    if (isAuthenticated) {
      navigate(destination, { replace: true });
    }
  }, [destination, isAuthenticated, navigate]);

  function switchMode() {
    setMode(isLogin ? "register" : "login");
    setError("");
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");

    const normalizedUsername = username.trim();

    if (normalizedUsername.length < 3 || normalizedUsername.length > 64) {
      setError("用户名长度必须为 3 到 64 个字符");
      return;
    }

    if (password.length < 8 || password.length > 72) {
      setError("密码长度必须为 8 到 72 个字符");
      return;
    }

    if (!isLogin && password !== confirmPassword) {
      setError("两次输入的密码不一致");
      return;
    }

    setIsSubmitting(true);

    try {
      const credentials = { username: normalizedUsername, password };

      if (isLogin) {
        await login(credentials);
      } else {
        await register(credentials);
      }

      toast.success(isLogin ? "登录成功" : "账号创建成功", {
        description: isLogin ? "欢迎回来。" : "已自动为你登录。",
      });
      navigate(destination, { replace: true });
    } catch (submitError) {
      const message =
        submitError instanceof ApiError
          ? submitError.message
          : "无法连接服务器，请稍后重试";
      setError(message);
      toast.error(isLogin ? "登录失败" : "注册失败", {
        description: message,
      });
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <div className="flex min-h-[calc(100svh-3.5rem)] items-center justify-center bg-neutral-50/60 px-4 py-12">
      <Card className="w-full max-w-md bg-white shadow-sm">
        <CardHeader className="items-center text-center">
          <div className="mb-3 flex size-11 items-center justify-center rounded-xl bg-neutral-950 text-white">
            <MessageSquareText className="size-5" />
          </div>

          <CardTitle className="text-2xl">
            {isLogin ? "欢迎回来" : "创建账号"}
          </CardTitle>

          <CardDescription>
            {isLogin
              ? "登录后继续你的模拟面试"
              : "注册一个 AI Meeting 账号"}
          </CardDescription>
        </CardHeader>

        <CardContent>
          <form className="space-y-5" onSubmit={handleSubmit}>
            <div className="space-y-2">
              <Label htmlFor="username">用户名</Label>
              <Input
                id="username"
                value={username}
                onChange={(event) => setUsername(event.target.value)}
                placeholder="请输入用户名"
                autoComplete="username"
                disabled={isSubmitting}
                required
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="password">密码</Label>
              <Input
                id="password"
                type="password"
                value={password}
                onChange={(event) => setPassword(event.target.value)}
                placeholder="请输入密码"
                autoComplete={isLogin ? "current-password" : "new-password"}
                disabled={isSubmitting}
                required
              />
            </div>

            {!isLogin && (
              <div className="space-y-2">
                <Label htmlFor="confirm-password">确认密码</Label>
                <Input
                  id="confirm-password"
                  type="password"
                  value={confirmPassword}
                  onChange={(event) => setConfirmPassword(event.target.value)}
                  placeholder="请再次输入密码"
                  autoComplete="new-password"
                  disabled={isSubmitting}
                  required
                />
              </div>
            )}

            {error && (
              <InlineError message={error} />
            )}

            <Button type="submit" className="w-full" disabled={isSubmitting}>
              {isSubmitting && <LoaderCircle className="animate-spin" />}
              {isSubmitting ? "请稍候..." : isLogin ? "登录" : "注册"}
            </Button>

            <p className="text-center text-sm text-neutral-500">
              {isLogin ? "还没有账号？" : "已经有账号？"}

              <Button
                type="button"
                variant="link"
                className="px-1"
                onClick={switchMode}
                disabled={isSubmitting}
              >
                {isLogin ? "立即注册" : "返回登录"}
              </Button>
            </p>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
