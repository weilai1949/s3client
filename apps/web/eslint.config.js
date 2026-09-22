import js from '@eslint/js'
import pluginVue from 'eslint-plugin-vue'
import tseslint from 'typescript-eslint'
import vueParser from 'vue-eslint-parser'

/**
 * Soft ESLint baseline for Vue + TS. Essential/recommended only;
 * no type-aware rules (avoids rewriting the whole app).
 */
export default tseslint.config(
  { ignores: ['dist/**', 'node_modules/**', '*.config.*'] },
  js.configs.recommended,
  ...tseslint.configs.recommended,
  ...pluginVue.configs['flat/essential'],
  {
    // 覆盖 src 与两套 E2E（e2e/ 为 mock 版、e2e-real/ 为真实联调版，todolist #37）。
    // 此前只 lint src/**，E2E 源码零静态检查。
    files: ['src/**/*.{ts,vue}', 'e2e/**/*.ts', 'e2e-real/**/*.ts'],
    languageOptions: {
      parser: vueParser,
      parserOptions: {
        parser: tseslint.parser,
        ecmaVersion: 'latest',
        sourceType: 'module',
        extraFileExtensions: ['.vue'],
      },
    },
    rules: {
      'vue/multi-word-component-names': 'off',
      // 收紧为 error：当前 src 已 0 警告（新增 any 会直接失败）；
      // 新代码请用具体类型或 unknown + 收窄，不要用 any 掩盖不变量。
      '@typescript-eslint/no-explicit-any': 'error',
      '@typescript-eslint/no-unused-vars': ['warn', { argsIgnorePattern: '^_', varsIgnorePattern: '^_' }],
      'no-unused-vars': 'off',
      // TS / vue-tsc already check undefined identifiers; browser globals trip no-undef in .vue scripts.
      'no-undef': 'off',
      'no-console': 'off',
      'prefer-const': 'warn',
      'no-empty': ['warn', { allowEmptyCatch: true }],
    },
  },
  {
    // E2E 运行在 Node（进程 / Buffer / node:crypto），声明其全局，避免误报。
    files: ['e2e/**/*.ts', 'e2e-real/**/*.ts'],
    languageOptions: {
      globals: { process: 'readonly', Buffer: 'readonly', console: 'readonly' },
    },
  },
)
