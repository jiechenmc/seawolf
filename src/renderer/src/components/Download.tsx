import SideNav from './SideNav'
import React, { useState, useEffect, useRef } from 'react'
import { AppContext } from '../AppContext'
import { FaRegTrashAlt } from 'react-icons/fa'
import { FaRegCirclePause } from 'react-icons/fa6'
import { FaRegPlayCircle } from 'react-icons/fa'
import {
  discoverFile,
  getDownloadedFiles,
  getFile,
  getSessionInfo,
  pauseDownload,
  resumeDownload
} from '@renderer/rpcUtils'
import { BsDisplayFill } from 'react-icons/bs'

type downloadType = {
  size: number
  price: number
  file_name: string
  data_cid: string
  provider_id: string
  session_id: number
  download_status?: string
  download_progress: number
  // 'Downloading' | 'Paused' | 'Done' | 'Cancelled'
}

type providerType = {
  peer_id: string
  price: number
  file_name: string
  wallet_address: string
}

type discoverType = {
  size: number
  data_cid: string
  providers: providerType[]
}

type sessionType = {
  session_id: number
  req_cid: string
  rx_bytes: number
  total_bytes: number
  paused: number
  is_complete: boolean
  result: number
}

function Download(): JSX.Element {
  const [searchHash, setSearchHash] = useState<string>('')

  const { user, pathForDownload, filesDownloading, proxy, balance, history } =
    React.useContext(AppContext)
  const [peerId] = user
  const [downloadPath] = pathForDownload
  const [downloadingFiles, setDownloadingFiles] = filesDownloading
  const [currProxy, setCurrProxy] = proxy
  const [walletBalance, setWalletBalance] = balance
  const [, setHistoryView] = history

  const [completedDownloads, setCompletedDownloads] = useState([])

  const [showProviders, setShowProviders] = useState<boolean>(false)
  const [providers, setProviders] = useState<discoverType>()

  // useEffect(() => {
  //   const fetchData = async () => {
  //     try {
  //       const data = await getDownloadedFiles()
  //       setCompletedDownloads(data)
  //     } catch (error) {
  //       console.error('Error fetching downloaded files:', error)
  //     }
  //   }

  //   fetchData()
  // }, [])

  useEffect(() => {
    const interval = setInterval(() => {
      setDownloadingFiles((prevFiles: downloadType[]) =>
        prevFiles.map((file) => {
          if (file.download_status === 'Downloading') {
            getSessionInfo(file.session_id).then((data) => {
              const updatedFile = {
                ...file,
                download_progress: data.rx_bytes,
                size: data.total_bytes,
                download_status: data.paused
                  ? 'Paused'
                  : data.is_complete
                    ? 'Done'
                    : data.result
                      ? 'Error'
                      : file.download_status
              }
              setDownloadingFiles((prevFiles: downloadType[]) =>
                prevFiles.map((f) => (f.session_id === file.session_id ? updatedFile : f))
              )
            })
          }
          return file
        })
      )
    }, 3000)

    return () => clearInterval(interval)
  }, [])

  const handleSearchFile = async () => {
    try {
      const data = await discoverFile(searchHash)
      setProviders(
        data
        // .filter((provider: providerType) => {
        //   provider.peer_id !== peerId
        // })
      )
      setShowProviders(true)
    } catch (error) {
      console.log('Error finding providers: ', error)
    }
  }

  const handleProviderClick = async (provider: providerType) => {
    try {
      let data_cid = providers?.data_cid || ''
      const normalizedPath = downloadPath.endsWith('/') ? downloadPath : downloadPath + '/'
      const data = await getFile(provider.peer_id, data_cid, normalizedPath + provider.file_name)
      let newFile = {
        size: providers?.size || 0,
        price: provider.price,
        file_name: provider.file_name,
        data_cid: data_cid,
        provider_id: provider.peer_id,
        session_id: data,
        download_status: 'Downloading',
        download_progress: 0
      }
      setDownloadingFiles((prevList: downloadType[]) => [...prevList, newFile])
    } catch (error) {
      console.log('Error downloading file: ', error)
    }
    setShowProviders(false)
    setSearchHash('')
  }

  const handlePausePlay = async (file: downloadType) => {
    if (file.download_status === 'Downloading') {
      try {
        await pauseDownload(file.session_id)
        setDownloadingFiles((prevList: downloadType[]) => {
          prevList.map((thisFile) => {
            thisFile.data_cid === file.data_cid
          })
        })
        file.download_status = 'Paused'
      } catch (error) {
        console.log('Error pausing download: ', error)
      }
    } else {
      try {
        await resumeDownload(file.session_id)
        file.download_status = 'Downloading'
      } catch (error) {
        console.log('Error resuming download: ', error)
      }
    }
    // setDownloadedFiles((prevFiles) => {
    //   const updatedFiles = [...prevFiles]
    //   const updatedFile = { ...updatedFiles[index] }
    //   if (updatedFile.status === 'Downloading') {
    //     updatedFile.status = 'Paused'
    //   } else if (updatedFile.status === 'Paused') {
    //     updatedFile.status = 'Downloading'
    //   }
    //   updatedFiles[index] = updatedFile
    //   return updatedFiles
    // })
  }

  const handleCancelDownload = (file: downloadType, index: number) => {
    let confirmed = window.confirm(`Are you sure want to cancel download for: ${file.file_name}`)
    if (confirmed) {
      // setDownloadedFiles((prevFiles) => {
      //   setWalletBalance((prevBalance: number) => prevBalance + downloadedFile.file.cost)
      //   const updatedFiles = [...prevFiles]
      //   updatedFiles.splice(index, 1)
      //   return updatedFiles
      // })
    }
  }

  return (
    <div className="flex ml-52">
      <SideNav />
      <div className="flex-1 p-6">
        <h1 className="text-2xl font-bold mb-4">Downloads</h1>

        <div className="flex justify-between mb-16 w-7/12">
          <div className="bg-white p-4 rounded-lg shadow-md w-full">
            <h2 className="text-xl font-semibold">Get A File</h2>

            <div className="mt-7">
              <label className="block text-sm font-medium text-gray-700 mb-2">File Hash ID</label>
              <input
                type="text"
                className="mt-1 block w-11/12 border border-gray-300 rounded-md p-2"
                placeholder="Hash ID"
                value={searchHash}
                onChange={(e) => setSearchHash(e.target.value)}
              />
            </div>
            <button
              className="mt-7 px-4 py-2 bg-[#737fa3] text-white font-semibold rounded-md hover:bg-[#7c85a3]"
              onClick={handleSearchFile}
            >
              Get File
            </button>
          </div>
        </div>
        <h2 className="text-xl font-semibold mb-2">Downloaded Files</h2>
        <div className="overflow-x-auto border border-gray-300 rounded-lg">
          <div className="flex items-center p-2 border-b border-gray-300">
            <span className="flex-[4.5] font-semibold">File</span>
            <span className="flex-1 font-semibold">Bytes (MB)</span>
            <span className="flex-1 font-semibold">Cost (SWE)</span>
            <span className="flex-1 font-semibold">Progress</span>
            <span className="flex-1 font-semibold">Status</span>
          </div>
        </div>
        {downloadingFiles.map((file: downloadType, index: number) => (
          <div
            key={index}
            className="flex items-center px-2 py-1 border-b border-gray-300 rounded-md"
          >
            <div className="flex-[4.5]">
              <div>
                <span className="block font-semibold">
                  {file.file_name.length > 60
                    ? `${file.file_name.slice(0, 57)}...`
                    : file.file_name}
                </span>
                <span className="block text-gray-500">{file.data_cid}</span>{' '}
              </div>
            </div>
            <span className="flex-1 ">
              {(file.size / 1e6).toLocaleString(undefined, {
                minimumFractionDigits: 0,
                maximumFractionDigits: 6
              })}
            </span>
            <span className="flex-1 ">{file.price}</span>
            <span className="flex-1 ">
              {file.download_status !== 'Done' ? `${file.download_progress} / ${file.size}` : ''}
            </span>
            <span className="flex-1 flex justify-between items-center">
              <span style={{ color: file.download_status === 'Done' ? 'green' : 'black' }}>
                {file.download_status}
              </span>
              <div className="flex items-center ml-auto space-x-4">
                {file.download_status === 'Downloading' && (
                  <button
                    className="text-2xl text-black hover:text-gray-600"
                    onClick={() => handlePausePlay(file)}
                  >
                    <FaRegCirclePause />
                  </button>
                )}
                {file.download_status === 'Paused' && (
                  <button
                    className="text-2xl text-black hover:text-gray-600"
                    onClick={() => handlePausePlay(file)}
                  >
                    <FaRegPlayCircle />
                  </button>
                )}
                {/* {file.download_status !== 'Done' && (
                  <button
                    onClick={() => handleCancelDownload(file, index)}
                    className="text-red-500 hover:text-red-700"
                  >
                    <FaRegTrashAlt />
                  </button>
                )} */}
              </div>
            </span>
          </div>
        ))}
        {showProviders && (
          <div className="absolute top-0 left-0 w-full h-full bg-black bg-opacity-50 flex justify-center items-center">
            <div className="bg-white w-2/3 p-6 rounded-lg shadow-lg max-h-[80%] overflow-y-auto">
              <h2 className="text-xl font-bold mb-4 text-center">Available Providers</h2>
              <ul className="space-y-2">
                {providers?.providers.map((provider: providerType, index) => (
                  <li
                    key={index}
                    className="p-4 rounded-md hover:bg-gray-200 cursor-pointer border border-gray-300"
                    onClick={() => handleProviderClick(provider)}
                  >
                    <div>
                      <p className="font-medium text-base">Peer: {provider.peer_id}</p>
                      <p className="text-gray-600">
                        File Name:{' '}
                        {provider.file_name.length > 60
                          ? `${provider.file_name.slice(0, 57)}...`
                          : provider.file_name}
                      </p>
                      <p className="text-gray-600">Price: {provider.price} SWE</p>
                    </div>
                  </li>
                ))}
                {providers?.providers.length === 0 && (
                  <p className="text-gray-600 text-center mt-4">
                    File currently not being provided
                  </p>
                )}
              </ul>
              <button
                className="mt-4 w-full px-4 py-2 font-semibold rounded-md bg-gray-300 hover:bg-gray-400 text-gray-800"
                onClick={() => setShowProviders(false)}
              >
                Cancel
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}

export default Download
