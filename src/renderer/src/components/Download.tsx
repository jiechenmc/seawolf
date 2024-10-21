import SideNav from './SideNav'
import NavBar from './NavBar'
import React, { useState } from 'react'
import { AppContext } from '../AppContext'
import { FaRegTrashAlt } from 'react-icons/fa'
import { FaRegCirclePause } from 'react-icons/fa6'
import { FaRegPlayCircle } from 'react-icons/fa'

type fileType = {
  cid: number
  name: string
  size: number
  cost: number
}

const testDB: fileType[] = [
  { cid: 3158518221, name: 'document1.pdf', size: 1, cost: 500 },
  { cid: 7263573340, name: 'image2.png', size: 2, cost: 3 },
  { cid: 9780383347, name: 'audio3.mp3', size: 3, cost: 10 },
  { cid: 3529260449, name: 'video4.mp4', size: 4, cost: 15 },
  { cid: 6043729820, name: 'spreadsheet5.xlsx', size: 5, cost: 7 }
]

function Download(): JSX.Element {
  const [searchHash, setSearchHash] = useState<string>('')

  const { proxy, balance, downloadFiles, history } = React.useContext(AppContext)
  const [currProxy, setCurrProxy] = proxy
  const [walletBalance, setWalletBalance] = balance
  const [downloadedFiles, setDownloadedFiles] = downloadFiles
  const [, setHistoryView] = history

  const handleSearchFile = () => {
    const found = testDB.find((file) => file.cid === Number(searchHash))

    if (found) {
      let msg: string = `Name:  ${found.name}\nSize:  ${found.size} MB\nCost:  ${found.cost} SWE`

      if (walletBalance < found.cost) {
        alert(`${msg}\n\nNot enough SWE. Your current balance: ${walletBalance}.`)
      } else {
        const confirmed = window.confirm(
          `${msg}\n\nYou currently have ${walletBalance} SWE. Would like to proceed with the purchase?`
        )
        if (confirmed) {
          setWalletBalance((currBalance: number) => currBalance - found.cost)
          let newFileDownloaded = {
            file: found,
            eta: Math.floor(Math.random() * 29 + 1),
            status: 'Downloading'
          }
          downloadedFiles.unshift(newFileDownloaded)
          setDownloadedFiles(downloadedFiles)

          setHistoryView((prevView) => {
            const newHistory = [
              {
                date: new Date(),
                file: newFileDownloaded.file,
                type: 'downloaded',
                proxy: currProxy ? currProxy.ip : 'self'
              }
            ]

            return [...newHistory, ...prevView]
          })
        }
      }
    } else {
      alert(`No file found with the hash: ${searchHash}`)
    }

    setSearchHash('')
  }

  const handlePausePlay = (index: number) => {
    setDownloadedFiles((prevFiles) => {
      const updatedFiles = [...prevFiles]
      const updatedFile = { ...updatedFiles[index] }

      if (updatedFile.status === 'Downloading') {
        updatedFile.status = 'Paused'
      } else if (updatedFile.status === 'Paused') {
        updatedFile.status = 'Downloading'
      }

      updatedFiles[index] = updatedFile

      return updatedFiles
    })
  }

  const handleCancelDownload = (downloadedFile, index: number) => {
    let confirmed = window.confirm(
      `Are you sure want to cancel download for: ${downloadedFile.file.name}`
    )
    if (confirmed) {
      setDownloadedFiles((prevFiles) => {
        setWalletBalance((prevBalance: number) => prevBalance + downloadedFile.file.cost)
        const updatedFiles = [...prevFiles]
        updatedFiles.splice(index, 1)

        return updatedFiles
      })
    }
  }

  return (
    <div className="flex ml-52">
      <SideNav />
      {/* <NavBar /> */}
      <div className="flex-1 p-6">
        <h1 className="text-2xl font-bold mb-4">Downloads</h1>

        <div className="flex justify-between mb-16 w-1/2">
          <div className="bg-white p-4 rounded-lg shadow-md w-full">
            <h2 className="text-xl font-semibold">Get A File</h2>

            <div className="mt-7">
              <label className="block text-sm font-medium text-gray-700 mb-2">File Hash ID</label>
              <input
                type="text"
                className="mt-1 block w-3/4 border border-gray-300 rounded-md p-2"
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
            <span className="flex-1 font-semibold">File</span>
            <span className="flex-1 font-semibold">Bytes (MB)</span>
            <span className="flex-1 font-semibold">Cost (SWE)</span>
            <span className="flex-1 font-semibold">ETA (Mins)</span>
            <span className="flex-1 font-semibold">Status</span>
          </div>
        </div>
        {downloadedFiles.map((downloadedFile, index: number) => (
          <div
            key={index}
            className="flex items-center px-2 py-1 border-b border-gray-300 rounded-md"
          >
            <div className="flex-1">
              <div>
                <span className="block font-semibold">{downloadedFile.file.name}</span>
                <span className="block text-gray-500">{downloadedFile.file.cid}</span>{' '}
              </div>
            </div>
            <span className="flex-1 ">
              {downloadedFile.file.size.toLocaleString(undefined, {
                minimumFractionDigits: 0,
                maximumFractionDigits: 6
              })}
            </span>
            <span className="flex-1 ">{downloadedFile.file.cost}</span>
            <span className="flex-1 ">{downloadedFile.eta}</span>
            <span className="flex-1 text-left flex justify-between items-center">
              <span>{downloadedFile.status}</span>
              <div className="flex items-center ml-auto space-x-4">
                {downloadedFile.status === 'Downloading' && (
                  <button
                    className="text-2xl text-black hover:text-gray-600"
                    onClick={() => handlePausePlay(index)}
                  >
                    <FaRegCirclePause />
                  </button>
                )}
                {downloadedFile.status === 'Paused' && (
                  <button
                    className="text-2xl text-black hover:text-gray-600"
                    onClick={() => handlePausePlay(index)}
                  >
                    <FaRegPlayCircle />
                  </button>
                )}
                {downloadedFile.status !== 'Done' && (
                  <button
                    onClick={() => handleCancelDownload(downloadedFile, index)}
                    className="text-red-500 hover:text-red-700"
                  >
                    <FaRegTrashAlt />
                  </button>
                )}
              </div>
            </span>
          </div>
        ))}
      </div>
    </div>
  )
}

export default Download
