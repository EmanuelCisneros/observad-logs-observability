"use client";

import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  Tooltip,
  ResponsiveContainer,
  CartesianGrid,
} from "recharts";
import { LEVEL_HEX, fmtDate } from "@/lib/utils";
import { Card } from "@/components/ui/primitives";
import type { Stats } from "@/types";
import { LEVELS } from "@/types";

interface HistogramProps {
  stats?: Stats;
}

interface ChartRow {
  t: string;
  [key: string]: number | string;
}

function pivot(stats: Stats): ChartRow[] {
  return stats.histogram.map((b) => {
    const row: ChartRow = { t: fmtDate(b.bucket) };
    for (const lvl of LEVELS) {
      row[lvl] = b.counts[lvl] ?? 0;
    }
    return row;
  });
}

const tooltipStyle = {
  backgroundColor: "#10131a",
  border: "1px solid #2b323f",
  borderRadius: "6px",
  fontFamily: "'JetBrains Mono', monospace",
  fontSize: 12,
};

export function VolumeHistogram({ stats }: HistogramProps) {
  if (!stats?.histogram?.length) {
    return (
      <Card className="flex h-48 items-center justify-center text-sm text-mist-400">
        No data for selected range
      </Card>
    );
  }

  const data = pivot(stats);

  return (
    <Card className="pt-3">
      <p className="mb-3 font-mono text-[11px] uppercase tracking-widest text-mist-400">
        Log volume · 1-min buckets
      </p>
      <ResponsiveContainer width="100%" height={180}>
        <BarChart data={data} barCategoryGap="20%">
          <CartesianGrid
            strokeDasharray="3 3"
            stroke="#2b323f"
            vertical={false}
          />
          <XAxis
            dataKey="t"
            tick={{ fill: "#6b7585", fontSize: 11, fontFamily: "monospace" }}
            axisLine={{ stroke: "#2b323f" }}
            tickLine={false}
          />
          <YAxis
            tick={{ fill: "#6b7585", fontSize: 11, fontFamily: "monospace" }}
            axisLine={false}
            tickLine={false}
            width={36}
          />
          <Tooltip
            contentStyle={tooltipStyle}
            labelStyle={{ color: "#9aa3b2" }}
            itemStyle={{ color: "#d6dbe2" }}
            cursor={{ fill: "rgba(255,255,255,0.04)" }}
          />
          {LEVELS.map((lvl) => (
            <Bar
              key={lvl}
              dataKey={lvl}
              stackId="a"
              fill={LEVEL_HEX[lvl]}
              opacity={0.85}
              radius={lvl === "fatal" ? [3, 3, 0, 0] : undefined}
            />
          ))}
        </BarChart>
      </ResponsiveContainer>
    </Card>
  );
}
