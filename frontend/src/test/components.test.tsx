import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { LevelBadge } from "@/components/ui/primitives";
import { StatsCards } from "@/components/charts/StatsCards";
import type { Stats } from "@/types";
import { LEVELS } from "@/types";

describe("LevelBadge", () => {
  LEVELS.forEach((lvl) => {
    it(`renders ${lvl} badge`, () => {
      render(<LevelBadge level={lvl} />);
      expect(screen.getByText(lvl)).toBeInTheDocument();
    });
  });
});

const mockStats: Stats = {
  total: 42_300,
  by_level: { debug: 100, info: 40_000, warn: 1_800, error: 400, fatal: 0 },
  top_services: [
    { service: "api", count: 30_000 },
    { service: "worker", count: 12_300 },
  ],
  histogram: [],
};

describe("StatsCards", () => {
  it("shows spinner while loading with no data", () => {
    const { container } = render(
      <StatsCards stats={undefined} isLoading={true} />,
    );
    expect(container.querySelector("svg")).toBeInTheDocument();
  });

  it("renders total count in compact form", () => {
    render(<StatsCards stats={mockStats} isLoading={false} />);
    expect(screen.getByText("42.3K")).toBeInTheDocument();
  });

  it("renders a card for every level", () => {
    render(<StatsCards stats={mockStats} isLoading={false} />);
    LEVELS.forEach((lvl) => {
      expect(screen.getByText(lvl)).toBeInTheDocument();
    });
  });

  it("renders zero-count levels without crashing", () => {
    const statsWithZeros: Stats = {
      ...mockStats,
      by_level: { debug: 0, info: 0, warn: 0, error: 0, fatal: 0 },
    };
    expect(() =>
      render(<StatsCards stats={statsWithZeros} isLoading={false} />),
    ).not.toThrow();
  });
});
