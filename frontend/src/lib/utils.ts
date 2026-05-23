import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";
import type { Level } from "@/types";

export function cn(...inputs: ClassValue[]): string {
  return twMerge(clsx(inputs));
}

export function levelColor(level: Level): string {
  switch (level) {
    case "debug": return "text-mist-400";
    case "info":  return "text-blue-400";
    case "warn":  return "text-signal";
    case "error": return "text-red-400";
    case "fatal": return "text-purple-400";
  }
}

export function levelBg(level: Level): string {
  switch (level) {
    case "debug": return "bg-mist-400/10 text-mist-300";
    case "info":  return "bg-blue-500/10 text-blue-400";
    case "warn":  return "bg-signal/10 text-signal";
    case "error": return "bg-red-500/10 text-red-400";
    case "fatal": return "bg-purple-500/10 text-purple-400";
  }
}

export const LEVEL_HEX: Record<Level, string> = {
  debug: "#6b7585",
  info:  "#3b82f6",
  warn:  "#f5a524",
  error: "#ef4444",
  fatal: "#a855f7",
};

export function fmtTimestamp(iso: string): string {
  const d = new Date(iso);
  return d.toLocaleTimeString("en-GB", {
    hour:   "2-digit",
    minute: "2-digit",
    second: "2-digit",
    fractionalSecondDigits: 3,
  });
}

export function fmtDate(iso: string): string {
  return new Date(iso).toLocaleTimeString("en-GB", {
    hour:   "2-digit",
    minute: "2-digit",
  });
}

export function compact(n: number): string {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`;
  if (n >= 1_000)     return `${(n / 1_000).toFixed(1)}K`;
  return String(n);
}
