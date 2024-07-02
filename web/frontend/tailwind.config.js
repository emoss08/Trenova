import typographyPlugin from "@tailwindcss/typography";
import animatePlugin from "tailwindcss-animate";

/** @type {import("tailwindcss").Config} */
export default {
  darkMode: ["class"],
  content: [
    "./pages/**/*.{ts,tsx}",
    "./components/**/*.{ts,tsx}",
    "./app/**/*.{ts,tsx}",
    "./src/**/*.{ts,tsx}",
  ],
  theme: {
    container: {
      center: true,
      padding: "2rem",
      screens: {
        sm: "640px",
        // => @media (min-width: 640px) { ... }

        md: "768px",
        // => @media (min-width: 768px) { ... }

        lg: "1240px",
        // => @media (min-width: 1024px) { ... }

        xl: "1280px",
        // => @media (min-width: 1280px) { ... }

        "2xl": "1536px",
        // => @media (min-width: 1536px) { ... }
      },
    },
    fontSize: {
      xxs: "0.5rem",
      xs: "0.725rem",
      sm: "0.875rem",
      base: "1rem",
      xl: "1.25rem",
      "2xl": "1.563rem",
      "3xl": "1.875rem",
      "4xl": "2.25rem",
      "5xl": "3.052rem",
    },
    extend: {
      screens: {
        "3xl": "1920px",
        "4xl": "2560px",
      },
      transitionProperty: {
        height: "height",
        spacing: "margin, padding",
      },
      backgroundImage: {
        "rainbow-gradient-light":
          "linear-gradient(90deg, rgba(129,26,188,0.9), rgba(219,52,52,0.9), rgba(241, 196, 15, .9), rgba(34,230,171,0.9), rgba(0,60,211,0.9), rgba(129,26,188,0.9))",
        "rainbow-gradient-dark": "linear-gradient(90deg, white, gray)",
      },
      background: {
        "card-rainbow-gradient":
          "linear-gradient(49deg, #2d4de0 0, #9f71f0 30%, #fc6277 58%, #f8ef6f 95%)",
      },
      zIndex: {
        100: "100",
      },
      colors: {
        border: "hsl(var(--border))",
        input: "hsl(var(--input))",
        ring: "hsl(var(--ring))",
        background: "hsl(var(--background))",
        foreground: "hsl(var(--foreground))",
        success: "hsl(var(--success))",
        primary: {
          DEFAULT: "hsl(var(--primary))",
          foreground: "hsl(var(--primary-foreground))",
        },
        secondary: {
          DEFAULT: "hsl(var(--secondary))",
          foreground: "hsl(var(--secondary-foreground))",
        },
        destructive: {
          DEFAULT: "hsl(var(--destructive))",
          foreground: "hsl(var(--destructive-foreground))",
        },
        muted: {
          DEFAULT: "hsl(var(--muted))",
          foreground: "hsl(var(--muted-foreground))",
        },
        accent: {
          DEFAULT: "hsl(var(--accent))",
          foreground: "hsl(var(--accent-foreground))",
        },
        popover: {
          DEFAULT: "hsl(var(--popover))",
          foreground: "hsl(var(--popover-foreground))",
        },
        card: {
          DEFAULT: "hsl(var(--card))",
          foreground: "hsl(var(--card-foreground))",
        },
        alert: {
          DEFAULT: "hsl(var(--alert))",
          foreground: "hsl(var(--alert-foreground))",
        },
      },
      height: {
        82: "22rem",
      },
      backgroundSize: {
        "200%": "200% 100%",
        "size-200": "200% 200%",
      },
      borderRadius: {
        lg: "var(--radius)",
        md: "calc(var(--radius) - 2px)",
        sm: "calc(var(--radius) - 4px)",
      },
      backgroundPosition: {
        "pos-0": "0% 0%",
        "pos-100": "100% 100%",
      },
      keyframes: {
        shake: {
          "0%, 100%": { transform: "translateX(0)" },
          "10%, 30%, 50%, 70%, 90%": { transform: "translateX(-2px)" },
          "20%, 40%, 60%, 80%": { transform: "translateX(2px)" },
        },
        "accordion-down": {
          from: { height: 0 },
          to: { height: "var(--radix-accordion-content-height)" },
        },
        shine: {
          from: { backgroundPosition: "200% 0" },
          to: { backgroundPosition: "-200% 0" },
        },
        "accordion-up": {
          from: { height: "var(--radix-accordion-content-height)" },
          to: { height: 0 },
        },
        "topbar-flow": {
          "0%": { backgroundPosition: "0% 50%" },
          "100%": { backgroundPosition: "-200% 50%" },
        },
      },
      animation: {
        "accordion-down": "accordion-down 0.2s ease-out",
        "accordion-up": "accordion-up 0.2s ease-out",
        "rainbow-flow": "topbar-flow 5s ease infinite",
        shine: "shine 8s ease-in-out infinite",
      },
      fontFamily: {
        sans: ["var(--font-geist-sans)"],
        mono: ["var(--font-geist-mono)"],
      },
    },
  },
  plugins: [animatePlugin, typographyPlugin],
};
