import { cn, levelBg } from "@/lib/utils";
import type { Level } from "@/types";
import type { ButtonHTMLAttributes, InputHTMLAttributes, ReactNode } from "react";

interface BadgeProps {
  level: Level;
  className?: string;
}
export function LevelBadge({ level, className }: BadgeProps) {
  return (
    <span
      className={cn(
        "inline-flex items-center rounded px-1.5 py-0.5 font-mono text-[11px] font-semibold uppercase tracking-wider",
        levelBg(level),
        className,
      )}
    >
      {level}
    </span>
  );
}

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: "primary" | "ghost" | "danger";
  size?: "sm" | "md";
  children: ReactNode;
}
export function Button({
  variant = "ghost",
  size = "md",
  className,
  children,
  ...rest
}: ButtonProps) {
  return (
    <button
      className={cn(
        "inline-flex items-center gap-1.5 rounded font-sans font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-signal/50 disabled:pointer-events-none disabled:opacity-40",
        size === "sm" && "px-2.5 py-1 text-xs",
        size === "md" && "px-3.5 py-1.5 text-sm",
        variant === "primary" &&
          "bg-signal text-ink-900 hover:bg-signal/90 active:bg-signal/80",
        variant === "ghost" &&
          "border border-ink-500 text-mist-200 hover:border-ink-400 hover:bg-ink-600",
        variant === "danger" &&
          "border border-red-500/40 text-red-400 hover:bg-red-500/10",
        className,
      )}
      {...rest}
    >
      {children}
    </button>
  );
}

interface InputProps extends InputHTMLAttributes<HTMLInputElement> {
  className?: string;
}
export function Input({ className, ...rest }: InputProps) {
  return (
    <input
      className={cn(
        "w-full rounded border border-ink-500 bg-ink-700 px-3 py-1.5 font-mono text-sm text-mist-100 placeholder:text-mist-400 focus:border-signal/60 focus:outline-none focus:ring-1 focus:ring-signal/30",
        className,
      )}
      {...rest}
    />
  );
}

interface SelectProps {
  value: string;
  onChange: (v: string) => void;
  options: { label: string; value: string }[];
  className?: string;
}
export function Select({ value, onChange, options, className }: SelectProps) {
  return (
    <select
      value={value}
      onChange={(e) => onChange(e.target.value)}
      className={cn(
        "rounded border border-ink-500 bg-ink-700 px-2.5 py-1.5 font-mono text-sm text-mist-100 focus:border-signal/60 focus:outline-none",
        className,
      )}
    >
      {options.map((o) => (
        <option key={o.value} value={o.value}>
          {o.label}
        </option>
      ))}
    </select>
  );
}

export function Spinner({ className }: { className?: string }) {
  return (
    <svg
      className={cn("animate-spin", className)}
      viewBox="0 0 24 24"
      fill="none"
      aria-hidden
    >
      <circle
        className="opacity-20"
        cx="12" cy="12" r="10"
        stroke="currentColor" strokeWidth="3"
      />
      <path
        className="opacity-80"
        fill="currentColor"
        d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"
      />
    </svg>
  );
}

export function Card({
  className,
  children,
}: {
  className?: string;
  children: ReactNode;
}) {
  return (
    <div
      className={cn(
        "rounded-lg border border-ink-500 bg-ink-700 p-4",
        className,
      )}
    >
      {children}
    </div>
  );
}

export function LiveDot({ active }: { active: boolean }) {
  return (
    <span className="inline-flex items-center gap-1.5 text-xs font-mono">
      <span
        className={cn(
          "inline-block h-2 w-2 rounded-full",
          active
            ? "animate-pulse-dot bg-green-400"
            : "bg-mist-400",
        )}
      />
      {active ? "LIVE" : "OFFLINE"}
    </span>
  );
}
