import type { Config } from "tailwindcss";

const config: Config = {
  content: [
    "./app/**/*.{js,ts,jsx,tsx,mdx}",
    "./components/**/*.{js,ts,jsx,tsx,mdx}",
  ],
  theme: {
    extend: {
      colors: {
        bg: { primary: "#080c14", card: "#0d1320" },
        border: { DEFAULT: "#1e2d45" },
      },
    },
  },
  plugins: [],
};

export default config;
