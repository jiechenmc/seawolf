import { contextBridge, ipcRenderer } from 'electron'
import { electronAPI } from '@electron-toolkit/preload'

// Custom APIs for renderer
const api = {
  getPlatform: () => ipcRenderer.invoke('get-platform'),
  getDownloadPath: () => ipcRenderer.invoke('get-download-path')
}

const combinedAPI = {
  ...electronAPI,
  ...api
}

// Use `contextBridge` APIs to expose Electron APIs to
// renderer only if context isolation is enabled, otherwise
// just add to the DOM global.
if (process.contextIsolated) {
  try {
    // contextBridge.exposeInMainWorld('electron', electronAPI)
    contextBridge.exposeInMainWorld('electron', combinedAPI)
    contextBridge.exposeInMainWorld('api', api)
  } catch (error) {
    console.error(error)
  }
} else {
  // @ts-ignore (define in dts)
  // window.electron = electronAPI
  window.electron = combinedAPI
  // @ts-ignore (define in dts)
  window.api = api
}
