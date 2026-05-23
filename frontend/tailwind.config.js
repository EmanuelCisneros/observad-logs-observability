/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["./src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        // The palette is a deep slate base with a single signal-amber accent —
        // chosen so log severity colours (the real information) stand out
        // instead of competing with chrome.
        ink: {
          900: "#0a0c10",
          800: "#10131a",
          700: "#171b24",
          600: "#1f2530",
          500: "#2b323f",
          400: "#3a4350",
        },
        mist: {
          100: "#f4f6f8",
          200: "#d6dbe2",
          300: "#9aa3b2",
          400: "#6b7585",
        },
        signal: {
          DEFAULT: "#f5a524",
          dim: "#9c6a1c",
        },
        level: {
          debug: "#6b7585",
          info: "#3b82f6",
          warn: "#f5a524",
          error: "#ef4444",
          fatal: "#a855f7",
        },
      },
      fontFamily: {
        mono: ["'JetBrains Mono'", "ui-monospace", "SFMono-Regular", "monospace"],
        sans: ["'Inter Tight'", "system-ui", "sans-serif"],
      },
      keyframes: {
        "fade-in": {
          from: { opacity: "0", transform: "translateY(4px)" },
          to: { opacity: "1", transform: "translateY(0)" },
        },
        "pulse-dot": {
          "0%, 100%": { opacity: "1" },
          "50%": { opacity: "0.3" },
        },
      },
      animation: {
        "fade-in": "fade-in 0.2s ease-out",
        "pulse-dot": "pulse-dot 1.6s ease-in-out infinite",
      },
    },
  },
  plugins: [],
};
