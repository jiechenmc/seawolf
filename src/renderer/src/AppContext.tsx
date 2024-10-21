import { useState, createContext } from 'react'

type fileType = {
  cid: number
  name: string
  size: number
  cost: number
}

type proxyType = {
  ip: string
  location: string
  cost: number
}

const testList: proxyType[] = [
  { ip: '192.168.1.1', location: 'New York, USA', cost: 10 },
  { ip: '192.168.1.2', location: 'London, UK', cost: 12 },
  { ip: '192.168.1.3', location: 'Tokyo, Japan', cost: 8 }
]

type downloadType = {
  file: fileType
  eta: number
  status: string
}

type historyType = {
  date: Date
  file: fileType
  type: string
  proxy: string
}

const testList2: downloadType[] = [
  { file: { cid: 2657828461, name: 'something.pdf', size: 10, cost: 14 }, eta: 0, status: 'Done' },
  { file: { cid: 9477837364, name: 'another.pdf', size: 25, cost: 20 }, eta: 10, status: 'Paused' }
]

const testList3: historyType[] = [
  {
    date: new Date(),
    file: { cid: 2657828461, name: 'something.pdf', size: 10, cost: 14 },
    type: 'downloaded',
    proxy: '192.168.1.1'
  }
]

const AppContext = createContext<any>(null)

const AppProvider = ({ children }) => {
  const [walletAddress, setWalletAddress] = useState(null)

  const [numFiles, setNumFiles] = useState(0)
  const [numBytes, setNumBytes] = useState(0)

  const [listOfFiles, setListOfFiles] = useState([])
  const [filesToView, setFilesToView] = useState([])
  const [searchHash, setSeearchHash] = useState('')

  const [downloadedFiles, setDownloadedFiles] = useState<downloadType[]>(testList2)

  const [currProxy, setCurrProxy] = useState<proxyType | null>(null)
  const [listOfProxies, setListOfProxies] = useState<proxyType[]>(testList)

  const [walletBalance, setWalletBalance] = useState<number>(100)

  const [historyView, setHistoryView] = useState<historyType[]>(testList3)

  return (
    <AppContext.Provider
      value={{
        user: [walletAddress, setWalletAddress],
        totalFiles: [numFiles, setNumFiles],
        totalBytes: [numBytes, setNumBytes],
        allFiles: [listOfFiles, setListOfFiles],
        viewFiles: [filesToView, setFilesToView],
        search: [searchHash, setSeearchHash],
        proxy: [currProxy, setCurrProxy],
        proxies: [listOfProxies, setListOfProxies],
        balance: [walletBalance, setWalletBalance],
        downloadFiles: [downloadedFiles, setDownloadedFiles],
        history: [historyView, setHistoryView]
      }}
    >
      {children}
    </AppContext.Provider>
  )
}

export { AppContext, AppProvider }
