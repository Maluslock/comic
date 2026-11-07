/// <reference types="@dcloudio/types" />

export {}

declare module '@vue/runtime-core' {
  type Hooks = App.AppInstance & Page.PageInstance
  interface ComponentCustomOptions extends Hooks {}
}
