import { ElectronAPI } from '@electron-toolkit/preload'

export {}

declare global {
  interface ElectronAPI {
    getPlatform: () => Promise<string>
  }

  interface Window {
    electron: ElectronAPI
    api: unknown
  }
}
