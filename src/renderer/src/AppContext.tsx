import { useState, createContext } from 'react'

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

const AppContext = createContext<any>(null)

const AppProvider = ({ children }) => {
  const [user, setUser] = useState(null)

  const [numFiles, setNumFiles] = useState(0)
  const [numBytes, setNumBytes] = useState(0)

  const [listOfFiles, setListOfFiles] = useState([])
  const [filesToView, setFilesToView] = useState([])
  const [searchHash, setSeearchHash] = useState('')

  const [currProxy, setCurrProxy] = useState<proxyType | null>(null)
  const [listOfProxies, setListOfProxies] = useState<proxyType[]>(testList)

  return (
    <AppContext.Provider
      value={{
        user: [user, setUser],
        totalFiles: [numFiles, setNumFiles],
        totalBytes: [numBytes, setNumBytes],
        allFiles: [listOfFiles, setListOfFiles],
        viewFiles: [filesToView, setFilesToView],
        search: [searchHash, setSeearchHash],
        proxy: [currProxy, setCurrProxy],
        proxies: [listOfProxies, setListOfProxies]
      }}
    >
      {children}
    </AppContext.Provider>
  )
}

export { AppContext, AppProvider }
