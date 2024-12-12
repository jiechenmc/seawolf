import { ElectronAPI } from '@electron-toolkit/preload'

export {}

declare global {
  interface ElectronAPI {
    process: any
    getPlatform: () => Promise<string>
    getDownloadPath: () => Promise<string>
  }

  interface Window {
    electron: ElectronAPI
    api: unknown
  }
}
