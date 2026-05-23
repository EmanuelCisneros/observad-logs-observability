import { describe, it, expect } from "vitest";
import {
  compact,
  fmtDate,
  levelBg,
  levelColor,
  LEVEL_HEX,
  cn,
} from "@/lib/utils";
import type { Level } from "@/types";

describe("cn", () => {
  it("merges class names", () => {
    expect(cn("a", "b")).toBe("a b");
  });

  it("resolves Tailwind conflicts in favour of the last value", () => {
    const result = cn("text-red-400", "text-blue-400");
    expect(result).toBe("text-blue-400");
  });

  it("drops falsy values", () => {
    expect(cn("a", false && "b", undefined, "c")).toBe("a c");
  });
});

describe("compact", () => {
  it("leaves small numbers unchanged", () => {
    expect(compact(0)).toBe("0");
    expect(compact(999)).toBe("999");
  });

  it("formats thousands with K suffix", () => {
    expect(compact(1_000)).toBe("1.0K");
    expect(compact(42_500)).toBe("42.5K");
  });

  it("formats millions with M suffix", () => {
    expect(compact(1_000_000)).toBe("1.0M");
    expect(compact(2_345_678)).toBe("2.3M");
  });
});

describe("levelColor", () => {
  const levels: Level[] = ["debug", "info", "warn", "error", "fatal"];

  it("returns a non-empty Tailwind class for every level", () => {
    levels.forEach((lvl) => {
      const cls = levelColor(lvl);
      expect(cls).toBeTruthy();
      expect(cls).toMatch(/^text-/);
    });
  });

  it("returns distinct classes for different levels", () => {
    const colors = levels.map(levelColor);
    expect(new Set(colors).size).toBe(levels.length);
  });
});

describe("levelBg", () => {
  it("returns bg and text classes for every level", () => {
    const levels: Level[] = ["debug", "info", "warn", "error", "fatal"];
    levels.forEach((lvl) => {
      const cls = levelBg(lvl);
      expect(cls).toMatch(/bg-/);
      expect(cls).toMatch(/text-/);
    });
  });
});

describe("LEVEL_HEX", () => {
  it("has a valid hex entry for every level", () => {
    const levels: Level[] = ["debug", "info", "warn", "error", "fatal"];
    levels.forEach((lvl) => {
      expect(LEVEL_HEX[lvl]).toMatch(/^#[0-9a-f]{6}$/i);
    });
  });
});

describe("fmtDate", () => {
  it("returns a string in HH:MM format", () => {
    const result = fmtDate("2024-06-01T12:34:00Z");
    expect(result).toMatch(/\d{1,2}:\d{2}/);
  });
});
