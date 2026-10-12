import type { Config } from 'tailwindcss'

export default {
  content: ['./index.html', './src/**/*.{vue,ts}'],
  theme: {
    extend: {
      colors: {
        s2v: {
          canvas: 'rgb(var(--s2v-canvas) / <alpha-value>)',
          surface: 'rgb(var(--s2v-surface) / <alpha-value>)',
          ink: 'rgb(var(--s2v-ink) / <alpha-value>)',
          body: 'rgb(var(--s2v-body) / <alpha-value>)',
          muted: 'rgb(var(--s2v-muted) / <alpha-value>)',
          label: 'rgb(var(--s2v-label) / <alpha-value>)',
          nav: 'rgb(var(--s2v-nav) / <alpha-value>)',
          'nav-hover': 'rgb(var(--s2v-nav-hover) / <alpha-value>)',
          line: 'rgb(var(--s2v-line) / <alpha-value>)',
          'line-strong': 'rgb(var(--s2v-line-strong) / <alpha-value>)',
          'control-line': 'rgb(var(--s2v-control-line) / <alpha-value>)',
          primary: 'rgb(var(--s2v-primary) / <alpha-value>)',
          'primary-hover': 'rgb(var(--s2v-primary-hover) / <alpha-value>)',
          'primary-soft': 'rgb(var(--s2v-primary-soft) / <alpha-value>)',
          'primary-accent': 'rgb(var(--s2v-primary-accent) / <alpha-value>)',
          'primary-foreground': 'rgb(var(--s2v-primary-foreground) / <alpha-value>)',
          'media-overlay': 'rgb(var(--s2v-media-overlay) / <alpha-value>)',
          success: 'rgb(var(--s2v-success) / <alpha-value>)',
          warning: 'rgb(var(--s2v-warning) / <alpha-value>)',
          danger: 'rgb(var(--s2v-danger) / <alpha-value>)',
        },
      },
      fontFamily: {
        sans: ['Segoe UI Variable Text', 'Segoe UI', 'PingFang SC', 'Hiragino Sans GB', 'Microsoft YaHei UI', 'Microsoft YaHei', 'sans-serif'],
        mono: ['JetBrains Mono', 'SFMono-Regular', 'Consolas', 'monospace'],
      },
      boxShadow: {
        subtle: 'var(--s2v-shadow)',
        xs: '0 1px 2px 0 rgba(0, 0, 0, 0.04)',
      },
      borderRadius: {
        panel: 'var(--s2v-radius)',
        control: 'var(--s2v-radius-control)',
        nav: 'var(--s2v-radius-nav)',
      },
    },
  },
  plugins: [],
} satisfies Config
