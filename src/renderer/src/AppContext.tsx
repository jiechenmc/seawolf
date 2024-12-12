import { useState, createContext, useEffect, useMemo } from 'react'

type fileType = {
  cid: number
  fileUploadPath?: string
  fileName: string
  fileSize: number
  fileCost: number
  uploadEta?: number
  uploadStatus?: 'uploading' | 'completed' | 'cancelled' | 'error' | null
  fileDownloadPath?: string
  downloadEta?: number
  downloadStatus?: 'downloading' | 'completed' | 'cancelled' | 'error' | null
  selectStatus?: boolean
}

type downloadType = {
  size: number
  price: number
  file_name: string
  data_cid: string
  provider_id: string
  session_id: number
  download_status?: string
  download_progress: number
  // 'Downloading' | 'Paused' | 'Done' | 'Error'
}

type proxyType = {
  ip: string
  location: string
  cost: number
}

type historyType = {
  date: Date
  file_name: string
  file_cid: string
  file_size: number
  file_cost: number
  type: string
}

const AppContext = createContext<any>(null)

const AppProvider = ({ children }) => {
  const [peerId, setPeerId] = useState<string>('')

  const [walletAddress, setWalletAddress] = useState<string>('')

  const [downloadingFiles, setDownloadingFiles] = useState<downloadType[]>([])

  const [proxy, setProxy] = useState<proxyType | null>(null)
  const [proxies, setProxies] = useState<proxyType[]>([])

  const [walletBalance, setWalletBalance] = useState<number>(0)

  const [historyView, setHistoryView] = useState<historyType[]>([])

  const [platform, setPlatform] = useState<string>('')
  const [downloadPath, setDownloadPath] = useState<string>('')

  useEffect(() => {
    window.electron.getPlatform().then((platform) => {
      setPlatform(platform)

      window.electron.getDownloadPath().then((path) => {
        const isWin = platform === 'win32' ? true : false
        let pathToSet = path
        if (isWin) {
          const drive = path[0].toLowerCase()
          pathToSet = path.replace(/^([a-zA-Z]):\\/, `/mnt/${drive}/`).replace(/\\/g, '/')
        }
        setDownloadPath(pathToSet)
      })
    })

    fetch('http://localhost:8080/balance?q=default', {
      headers: {
        'Content-Type': 'application/json'
      }
    }).then(async (r) => {
      const data = await r.json()
      setWalletBalance(parseInt(data))
    })

    fetch('http://localhost:8080/account', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({ account: 'default' })
    }).then(async (r) => {
      const data = await r.json()
      setWalletAddress(data.message)
    })
  }, [])

  const contextValue = useMemo(
    () => ({
      user: [peerId, setPeerId],
      sysPlatform: [platform, setPlatform],
      pathForDownload: [downloadPath, setDownloadPath],
      filesDownloading: [downloadingFiles, setDownloadingFiles],
      proxy: [proxy, setProxy],
      proxies: [proxies, setProxies],
      wallet: [walletAddress, setWalletAddress],
      balance: [walletBalance, setWalletBalance],
      history: [historyView, setHistoryView]
    }),
    [
      peerId,
      platform,
      downloadPath,
      downloadingFiles,
      proxy,
      proxies,
      walletAddress,
      walletBalance,
      historyView
    ]
  )

  return <AppContext.Provider value={contextValue}>{children}</AppContext.Provider>
}

export { AppContext, AppProvider }
