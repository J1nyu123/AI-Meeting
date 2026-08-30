import { ChartNoAxesCombined } from "lucide-react";
import { ContentEmptyState } from "@/components/feedback/page-state";
import type { InterviewRadarMetric } from "@/features/interview/types";

interface ReportRadarChartProps {
  metrics: InterviewRadarMetric[];
}

const WIDTH = 360;
const HEIGHT = 320;
const CENTER_X = WIDTH / 2;
const CENTER_Y = 145;
const RADIUS = 100;

function getPoint(index: number, total: number, radius: number) {
  const angle = -Math.PI / 2 + (index * Math.PI * 2) / total;
  return {
    x: CENTER_X + Math.cos(angle) * radius,
    y: CENTER_Y + Math.sin(angle) * radius,
  };
}

function pointsToString(points: Array<{ x: number; y: number }>) {
  return points.map((point) => `${point.x},${point.y}`).join(" ");
}

export function ReportRadarChart({ metrics }: ReportRadarChartProps) {
  if (metrics.length < 3) {
    return (
      <ContentEmptyState
        className="h-72 justify-center"
        icon={ChartNoAxesCombined}
        title="暂无能力雷达"
        description="报告中的能力指标不足，暂时无法绘制雷达图。"
      />
    );
  }

  const normalizedMetrics = metrics.map((metric) => ({
    ...metric,
    value: Math.min(100, Math.max(0, Number(metric.value) || 0)),
  }));
  const total = normalizedMetrics.length;
  const outerPoints = normalizedMetrics.map((_, index) =>
    getPoint(index, total, RADIUS),
  );
  const valuePoints = normalizedMetrics.map((metric, index) =>
    getPoint(index, total, (RADIUS * metric.value) / 100),
  );

  return (
    <svg
      viewBox={`0 0 ${WIDTH} ${HEIGHT}`}
      role="img"
      aria-label="面试能力雷达图"
      className="mx-auto h-auto w-full max-w-md"
    >
      {[0.25, 0.5, 0.75, 1].map((level) => (
        <polygon
          key={level}
          points={pointsToString(
            normalizedMetrics.map((_, index) =>
              getPoint(index, total, RADIUS * level),
            ),
          )}
          fill="none"
          stroke="#e5e5e5"
          strokeWidth="1"
        />
      ))}

      {outerPoints.map((point, index) => (
        <line
          key={normalizedMetrics[index].label}
          x1={CENTER_X}
          y1={CENTER_Y}
          x2={point.x}
          y2={point.y}
          stroke="#e5e5e5"
          strokeWidth="1"
        />
      ))}

      <polygon
        points={pointsToString(valuePoints)}
        fill="rgba(23, 23, 23, 0.14)"
        stroke="#171717"
        strokeWidth="2"
      />

      {valuePoints.map((point, index) => (
        <circle
          key={normalizedMetrics[index].label}
          cx={point.x}
          cy={point.y}
          r="3.5"
          fill="#171717"
        />
      ))}

      {normalizedMetrics.map((metric, index) => {
        const labelPoint = getPoint(index, total, RADIUS + 34);
        const anchor =
          Math.abs(labelPoint.x - CENTER_X) < 8
            ? "middle"
            : labelPoint.x > CENTER_X
              ? "start"
              : "end";

        return (
          <g key={metric.label}>
            <text
              x={labelPoint.x}
              y={labelPoint.y}
              textAnchor={anchor}
              className="fill-neutral-600 text-[11px]"
            >
              {metric.label}
            </text>
            <text
              x={labelPoint.x}
              y={labelPoint.y + 16}
              textAnchor={anchor}
              className="fill-neutral-950 text-[12px] font-semibold"
            >
              {metric.value}
            </text>
          </g>
        );
      })}
    </svg>
  );
}
