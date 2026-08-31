import tsParser from '@typescript-eslint/parser'
import vue from 'eslint-plugin-vue'
import vueParser from 'vue-eslint-parser'

const s2vPagesPlugin = {
  rules: {
    'no-page-arbitrary-visual': {
      meta: {
        type: 'problem',
        docs: { description: 'Pages must not use arbitrary visual values or raw brand colors' },
      },
      create(context) {
        const pattern = /rounded-\[[^\]]+\]|shadow-\[[^\]]+\]|(?:bg|text)-\[[#]|\bbg-indigo-|\btext-indigo-|\baccent-indigo-|\brounded-(?:xl|lg)\b|\bbg-slate-|\bbg-emerald-50\b|\bbg-amber-/
        return {
          Program() {
            const lines = context.sourceCode.getText().split(/\n/)
            lines.forEach((line, index) => {
              const match = line.match(pattern)
              if (!match) return
              context.report({
                loc: { start: { line: index + 1, column: 0 }, end: { line: index + 1, column: Math.max(line.length, 1) } },
                message: `Do not use '${match[0]}' in pages. Use design tokens or shared/ui.`,
              })
            })
          },
        }
      },
    },
  },
}

export default [
  { ignores: ['dist/**', 'node_modules/**', '**/*.vue.js', '**/*.d.ts'] },
  {
    files: ['**/*.ts'],
    languageOptions: { parser: tsParser, parserOptions: { ecmaVersion: 'latest', sourceType: 'module' } },
    rules: {
      'no-debugger': 'error',
      'no-constant-condition': 'error',
    },
  },
  {
    files: ['**/*.vue'],
    languageOptions: {
      parser: vueParser,
      parserOptions: { parser: tsParser, ecmaVersion: 'latest', sourceType: 'module' },
    },
    plugins: { vue },
    rules: {
      'vue/no-unused-vars': 'error',
      'vue/no-unused-components': 'warn',
    },
  },
  {
    files: ['src/pages/**/*.vue'],
    plugins: { s2v: s2vPagesPlugin },
    rules: {
      's2v/no-page-arbitrary-visual': 'error',
    },
  },
]
