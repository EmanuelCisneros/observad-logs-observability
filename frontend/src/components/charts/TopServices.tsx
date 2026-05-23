"use client";

import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  Tooltip,
  ResponsiveContainer,
} from "recharts";
import { compact } from "@/lib/utils";
import { Card } from "@/components/ui/primitives";
import type { Stats } from "@/types";

interface TopServicesProps {
  stats?: Stats;
}

const tooltipStyle = {
  backgroundColor: "#10131a",
  border: "1px solid #2b323f",
  borderRadius: "6px",
  fontFamily: "'JetBrains Mono', monospace",
  fontSize: 12,
};

export function TopServices({ stats }: TopServicesProps) {
  if (!stats?.top_services?.length) {
    return (
      <Card className="flex h-48 items-center justify-center text-sm text-mist-400">
        No services detected
      </Card>
    );
  }

  const data = stats.top_services.slice(0, 8).map((s) => ({
    name: s.service.length > 18 ? s.service.slice(0, 17) + "…" : s.service,
    count: s.count,
  }));

  return (
    <Card className="pt-3">
      <p className="mb-3 font-mono text-[11px] uppercase tracking-widest text-mist-400">
        Top services
      </p>
      <ResponsiveContainer width="100%" height={180}>
        <BarChart data={data} layout="vertical" barCategoryGap="30%">
          <XAxis
            type="number"
            tick={{ fill: "#6b7585", fontSize: 11, fontFamily: "monospace" }}
            axisLine={{ stroke: "#2b323f" }}
            tickLine={false}
            tickFormatter={compact}
          />
          <YAxis
            type="category"
            dataKey="name"
            tick={{ fill: "#9aa3b2", fontSize: 11, fontFamily: "monospace" }}
            axisLine={false}
            tickLine={false}
            width={100}
          />
          <Tooltip
            contentStyle={tooltipStyle}
            labelStyle={{ color: "#9aa3b2" }}
            cursor={{ fill: "rgba(255,255,255,0.04)" }}
          />
          <Bar dataKey="count" fill="#f5a524" opacity={0.85} radius={[0, 3, 3, 0]} />
        </BarChart>
      </ResponsiveContainer>
    </Card>
  );
}
