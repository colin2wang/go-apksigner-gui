// Author: colin2wang (colin2wang@gmail.com)
// Date: 2026-10-06

/// <reference types="vite/client" />

declare module '*.vue' {
  import type { DefineComponent } from 'vue'
  const component: DefineComponent<{}, {}, any>
  export default component
}
