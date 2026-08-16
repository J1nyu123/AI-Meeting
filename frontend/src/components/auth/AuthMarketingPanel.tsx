import { CheckCircle2 } from "lucide-react";
import { APP_BRAND_NAME, APP_MARKETING_TAGLINE } from "@/lib/branding";

const highlights = [
  "简历驱动的面试问题生成",
  "长会话恢复与动态评分追问",
  "面试报告与逐题复盘",
];

export default function AuthMarketingPanel() {
  return (
    <div className="space-y-6">
      <div className="inline-flex items-center gap-2 rounded-full border border-white/25 bg-white/10 px-3 py-1 text-xs text-slate-100 backdrop-blur-sm">
        {APP_BRAND_NAME} · {APP_MARKETING_TAGLINE}
      </div>
      <div className="space-y-3">
        <h1 className="text-4xl font-semibold tracking-tight text-white">
          登录后开启高效 AI 面试体验
        </h1>
        <p className="text-lg text-slate-200">
          从简历分析、模拟问答到报告复盘，完整记录每一次面试练习。
        </p>
      </div>
      <div className="space-y-3">
        {highlights.map((item) => (
          <div
            key={item}
            className="flex items-center gap-3 text-sm text-slate-100"
          >
            <CheckCircle2 className="h-4 w-4 text-emerald-500" />
            <span>{item}</span>
          </div>
        ))}
      </div>
    </div>
  );
}
