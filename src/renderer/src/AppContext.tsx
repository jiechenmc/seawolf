import { useState, createContext } from 'react'

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

type ListingType = {
  cid: number
  name: string
  size: number
  cost: number
  endDate: string
  type: 'sale' | 'auction'
  status: 'active' | 'ended'
}

type proxyType = {
  ip: string
  location: string
  cost: number
}

type historyType = {
  date: Date
  file: fileType
  type: string
  proxy: string
}

const proxyTest: proxyType[] = [
  { ip: '192.168.1.1', location: 'New York, USA', cost: 10 },
  { ip: '192.168.1.2', location: 'London, UK', cost: 12 },
  { ip: '192.168.1.3', location: 'Tokyo, Japan', cost: 8 }
]

const historyTest: historyType[] = [
  {
    date: new Date(),
    file: {
      cid: 2657828461,
      fileName: 'something.pdf',
      fileSize: 10,
      fileCost: 14,
      fileDownloadPath: 'downloadedFiles',
      downloadEta: 0,
      downloadStatus: 'completed'
    },
    type: 'downloaded',
    proxy: '192.168.1.1'
  }
]

const AppContext = createContext<any>(null)

const AppProvider = ({ children }) => {
  const [peerId, setPeerId] = useState<string>('')

  const [walletAddress, setWalletAddress] = useState<string | null>(null)

  const [numUploadedFiles, setNumUploadedFiles] = useState<number>(0)
  const [numUploadedBytes, setNumUploadedBytes] = useState<number>(0)

  const [currProxy, setCurrProxy] = useState<proxyType | null>(null)
  const [listOfProxies, setListOfProxies] = useState<proxyType[]>(proxyTest)

  const [walletBalance, setWalletBalance] = useState<number>(100)

  const [historyView, setHistoryView] = useState<historyType[]>(historyTest)

  const [marketListings, setMarketListings] = useState<ListingType[]>([])
  const [userMarketListings, setUserMarketListings] = useState<ListingType[]>([])

  return (
    <AppContext.Provider
      value={{
        user: [peerId, setPeerId],
        numUploadFiles: [numUploadedFiles, setNumUploadedFiles],
        numUploadBytes: [numUploadedBytes, setNumUploadedBytes],
        proxy: [currProxy, setCurrProxy],
        proxies: [listOfProxies, setListOfProxies],
        balance: [walletBalance, setWalletBalance],
        history: [historyView, setHistoryView],
        marketListing: [marketListings, setMarketListings],
        userListing: [userMarketListings, setUserMarketListings]
      }}
    >
      {children}
    </AppContext.Provider>
  )
}

export { AppContext, AppProvider }
