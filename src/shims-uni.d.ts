/// <reference types="@dcloudio/types" />

export {}

declare module 'uview-plus' {
  const uviewPlus: object
  export default uviewPlus
}

declare module '@vue/runtime-core' {
  type Hooks = App.AppInstance & Page.PageInstance
  interface ComponentCustomOptions extends Hooks {}
}
