import { Link } from "react-router-dom";
import {
  ArrowRight,
  ChartNoAxesCombined,
  FileText,
  MessageSquareText,
  Sparkles,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";

const features = [
  {
    title: "简历驱动",
    description: "根据你的 PDF 简历生成个性化面试问题。",
    icon: FileText,
  },
  {
    title: "动态追问",
    description: "根据回答内容和遗漏点继续深入提问。",
    icon: MessageSquareText,
  },
  {
    title: "逐题复盘",
    description: "查看评分、反馈、能力雷达和改进建议。",
    icon: ChartNoAxesCombined,
  },
];

export function HomePage() {
  return (
    <div className="mx-auto max-w-6xl px-4 py-20 sm:px-6 sm:py-28">
      <section className="mx-auto flex max-w-3xl flex-col items-center text-center">
        <div className="mb-6 flex items-center gap-2 rounded-full border border-neutral-200 bg-neutral-50 px-3 py-1.5 text-sm text-neutral-600">
          <Sparkles className="size-4" />
          AI 模拟面试
        </div>

        <h1 className="text-balance text-4xl font-semibold tracking-tight sm:text-6xl">
          让每一次面试练习
          <span className="block text-neutral-400">都有真实反馈</span>
        </h1>

        <p className="mt-6 max-w-2xl text-pretty text-lg leading-8 text-neutral-500">
          上传简历，完成一场支持动态追问和刷新恢复的模拟面试，
          最后通过报告逐题复盘。
        </p>

        <div className="mt-8 flex flex-wrap justify-center gap-3">
          <Button asChild size="lg" className="rounded-xl px-5">
            <Link to="/auth">
              开始面试
              <ArrowRight data-icon="inline-end" />
            </Link>
          </Button>

          <Button
            asChild
            size="lg"
            variant="outline"
            className="rounded-xl px-5"
          >
            <Link to="/interviews">查看历史</Link>
          </Button>
        </div>
      </section>

      <section className="mt-24 grid gap-4 md:grid-cols-3">
        {features.map(({ title, description, icon: Icon }) => (
          <Card key={title} className="border-0 shadow-none ring-neutral-200">
            <CardHeader>
              <div className="mb-3 flex size-10 items-center justify-center rounded-xl bg-neutral-100">
                <Icon className="size-5 text-neutral-700" />
              </div>
              <CardTitle>{title}</CardTitle>
            </CardHeader>

            <CardContent>
              <p className="leading-6 text-neutral-500">{description}</p>
            </CardContent>
          </Card>
        ))}
      </section>
    </div>
  );
}