import React, { useState, useRef } from 'react'
import SideNav from './SideNav'
// import Navbar from './NavBar'
import {
  FaFolderOpen,
  FaRegFilePdf,
  FaRegFile,
  FaCloudUploadAlt,
  FaRegTrashAlt
} from 'react-icons/fa'
import { LuFileText, LuFileType } from 'react-icons/lu'
import { BsFiletypeMp4, BsFiletypeMp3, BsFiletypePng, BsFiletypeJpg } from 'react-icons/bs'
import { AppContext } from '../AppContext'

// type fileType = {
//   cid: number
//   name: string
//   size: number
//   cost: number
//   selectStatus: boolean
// }

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

function Upload(): JSX.Element {
  const fileInputRef = useRef<HTMLInputElement | null>(null)

  const { uploadFiles, numUploadFiles, numUploadBytes, history } = React.useContext(AppContext)

  const [uploadedFiles, setUploadedFiles] = uploadFiles
  const [numFiles, setNumFiles] = numUploadFiles
  const [numBytes, setNumBytes] = numUploadBytes

  const [historyView, setHistoryView] = history

  const [filesToView, setFilesToView] = useState<fileType[]>(uploadedFiles)
  const [searchHash, setSearchHash] = useState<string>('')

  const [fileQueue, setfileQueue] = useState<fileType[]>([])
  const [fileQueueIndex, setFileQueueIndex] = useState<number>(0)
  const [showCostModal, setShowCostModal] = useState(false)
  const [fileCost, setFileCost] = useState<number>(0)
  const [byteCount, setByteCount] = useState<number>(0)
  const [fileCount, setFileCount] = useState<number>(0)

  const getFileIcon = (fileName: string) => {
    const fileType = fileName.split('.').pop()

    switch (fileType) {
      case 'dir':
        return <FaFolderOpen />
      case 'txt':
        return <LuFileText />
      case 'pdf':
        return <FaRegFilePdf />
      case 'mp3':
        return <BsFiletypeMp3 />
      case 'mp4':
        return <BsFiletypeMp4 />
      case 'png':
        return <BsFiletypePng />
      case 'jpg':
        return <BsFiletypeJpg />
      default:
        return <FaRegFile />
    }
  }

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault()

    if (searchHash === '') {
      setFilesToView(uploadedFiles)
    } else {
      let searchList: fileType[] = []
      uploadedFiles.forEach((file: fileType) => {
        if (file.fileName.includes(searchHash)) {
          searchList.push(file)
        }
        if (file.cid.toString().includes(searchHash)) {
          searchList.push(file)
        }
      })

      // setSearchHash(() => '')
      setFilesToView(() => searchList)
    }
  }

  const generateRandom10DigitNumber = () => {
    // Generate a random number between 1000000000 (1 billion) and 9999999999 (just under 10 billion)
    return Math.floor(Math.random() * 9000000000) + 1000000000
  }

  const handleFileUpload = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files) {
      const filesArr = Array.from(e.target.files)
      let countFile: number = 0
      let countByte: number = 0
      // let filesToAppend: fileType[] = listOfFiles
      let filesToAppend: fileType[] = []
      filesArr.forEach((file) => {
        countByte += file.size / 1e6
        countFile++
        let newFile: fileType = {
          // cid: generateRandom10DigitNumber(),
          cid: generateRandom10DigitNumber(),
          fileName: file.name,
          fileSize: file.size / 1e6,
          fileCost: 0,
          selectStatus: true
        }
        filesToAppend.push(newFile)
      })
      setFileCount(countFile)
      setByteCount(countByte)

      setfileQueue(filesToAppend)
      setShowCostModal(true)
    }
  }

  const handleUploadButtonClick = () => {
    fileInputRef.current?.click()
  }

  const handleRemoveFile = (index: number, file: fileType) => {
    let confirmed = window.confirm(`Are you sure you want to delete: ${file.fileName}?`)
    if (confirmed) {
      let newByteSize = numBytes - file.fileSize
      setNumBytes(() => (newByteSize === 0 ? 0 : newByteSize))
      setNumFiles((prevFiles: number) => prevFiles - 1)
      let newFileList = uploadedFiles
      newFileList.splice(index, 1)
      setUploadedFiles(() => newFileList)
      setFilesToView(() => newFileList)
    }
  }

  const handleCostConfirm = (fileQueue) => {
    // let idx: number = fileQueueIndex
    // fileQueue[idx].cost = fileCost
    // setFileQueueIndex(idx + 1)

    // if (idx + 1 === fileQueue.length) {
    //   setNumBytes((prevBytes: number) => prevBytes + byteCount)
    //   setNumFiles((prevFiles: number) => prevFiles + fileCount)
    //   setListOfFiles((prevList: fileType[]) => prevList.concat(fileQueue))
    //   setFilesToView((prevList: fileType[]) => prevList.concat(fileQueue))

    //   setHistoryView((prevView) => {
    //     const newHistory = fileQueue.map((file) => ({
    //       date: new Date(),
    //       file: file,
    //       type: 'uploaded',
    //       proxy: 'self'
    //     }))

    //     return [...newHistory, ...prevView]
    //   })

    //   handleCostCancelAll()
    // }

    // setFileCost(0)
    setNumBytes(() => byteCount)
    setNumFiles(() => fileCount)
    setUploadedFiles((prevList: fileType[]) => prevList.concat(fileQueue))
    setFilesToView((prevList: fileType[]) => prevList.concat(fileQueue))
    setFileCost(0)
    setShowCostModal(false)
  }

  const handleCostCancelAll = () => {
    setFileQueueIndex(0)
    setFileCost(0)
    setShowCostModal(false)
  }

  const handleCostCancelOne = () => {
    setFileCount((prevCount) => prevCount - 1)
    let currFileSize = fileQueue[fileQueueIndex].fileSize
    setByteCount((prevCount) => prevCount - currFileSize)
    let queue = fileQueue
    queue.splice(fileQueueIndex, 1)
    setfileQueue(() => queue)
    setFileCost(0)
  }

  return (
    <div className="flex ml-52">
      <SideNav />
      {/* <Navbar /> */}

      <div className="flex-1 p-6">
        <h1 className="text-2xl font-bold mb-4">Welcome!</h1>

        <div className="bg-white p-4 rounded-lg shadow-md mb-16">
          <h2 className="text-xl font-semibold">Overview</h2>
          <div className="flex justify-between mt-2">
            <div className="flex-1 text-center">
              <h3 className="font-bold">Number of Files</h3>
              <p className="text-lg">{numFiles}</p>
            </div>
            <div className="flex-1 text-center">
              <h3 className="font-bold">Total Bytes</h3>
              <p className="text-lg">
                {Math.round(numBytes * 1e6) / 1e6 === 0
                  ? '0 MB'
                  : numBytes.toLocaleString(undefined, {
                      minimumFractionDigits: 0,
                      maximumFractionDigits: 4
                    }) + ' MB'}
              </p>
            </div>
          </div>
        </div>

        <h1 className="text-xl font-bold mb-4">Uploaded Files</h1>
        <div className="flex justify-between mb-4">
          <div className="flex w-2/3">
            <input
              type="text"
              placeholder="Search for file name or hash"
              value={searchHash}
              onChange={(e) => setSearchHash(e.target.value)}
              className="border border-gray-300 rounded-lg p-2 flex-1 mr-2"
              onKeyDown={(e) => {
                if (e.key == 'Enter') {
                  handleSearch(e)
                }
              }}
            />
            <button
              className="bg-[#737fa3] hover:bg-[#7c85a3] text-white px-4 py-2 rounded-lg"
              onClick={handleSearch}
            >
              Search
            </button>
          </div>

          <div>
            <button
              type="button"
              className="bg-[#737fa3] hover:bg-[#7c85a3] text-white px-4 py-2 rounded-lg flex items-center"
              onClick={handleUploadButtonClick}
            >
              <FaCloudUploadAlt className="mr-2" />
              <span>Upload File</span>
            </button>
            <input
              ref={fileInputRef}
              id="file-upload"
              type="file"
              onChange={handleFileUpload}
              className="hidden"
              multiple
            />
          </div>
        </div>

        <div className="overflow-x-auto border border-gray-300 rounded-lg">
          <div className="flex items-center p-2 border-b border-gray-300">
            <input type="checkbox" className="mr-10" />
            <span className="flex-1 font-semibold text-left">File</span>
            <span className="flex-1 font-semibold text-left">Bytes</span>
            <span className="flex-1 font-semibold text-left">Cost</span>
          </div>
        </div>
        {filesToView.map((file: fileType, index: number) => (
          <div
            key={index}
            className="flex items-center px-2 py-1 border-b border-gray-300 rounded-md"
          >
            <input type="checkbox" className="mr-10" />
            <div className="flex flex-1 items-center">
              <div className="flex flex-col items-center justify-center h-full mr-5">
                {getFileIcon(file.fileName)}
              </div>
              <div className="ml-2">
                <span className="block font-semibold">{file.fileName}</span>
                <span className="block text-gray-500">{file.cid}</span>{' '}
              </div>
            </div>
            <span className="flex-1 text-left">
              {file.fileSize.toLocaleString(undefined, {
                minimumFractionDigits: 0,
                maximumFractionDigits: 6
              })}{' '}
              MB
            </span>
            <span className="flex-1 text-left flex justify-between items-center">
              <span>{file.fileCost} SWE</span>
              <button
                onClick={() => handleRemoveFile(index, file)}
                className="text-red-500 hover:text-red-700"
              >
                <FaRegTrashAlt />
              </button>
            </span>
          </div>
        ))}

        {/* {showCostModal && (
          <div className="fixed inset-0 bg-black bg-opacity-50 flex justify-center items-center">
            <div className="bg-white p-6 rounded-lg shadow-lg w-1/3">
              <h2 className="text-l font-bold mb-4">
                ({fileQueueIndex + 1}/{fileQueue.length}) Set Cost For:{' '}
                {fileQueue[fileQueueIndex].name}
              </h2>
              <input
                type="number"
                value={fileCost}
                onChange={(e) => {
                  const newValue = parseFloat(e.target.value)
                  if (newValue < 0) {
                    setFileCost(0)
                  } else {
                    setFileCost(newValue)
                  }
                }}
                placeholder="Enter cost in SWE"
                className="border border-gray-300 rounded-lg p-2 w-full mb-4"
              />
              <div className="flex justify-end">
                <button
                  onClick={handleCostCancelAll}
                  className="bg-gray-300 hover:bg-gray-400 text-gray-800 px-4 py-2 rounded-lg mr-2"
                >
                  Cancel All
                </button>
                <button
                  onClick={handleCostCancelOne}
                  className="bg-gray-300 hover:bg-gray-400 text-gray-800 px-4 py-2 rounded-lg mr-2"
                >
                  Skip File
                </button>
                <button
                  onClick={handleCostConfirm}
                  className="bg-[#737fa3] hover:bg-[#7c85a3] text-white px-4 py-2 rounded-lg"
                >
                  Confirm Cost
                </button>
              </div>
            </div>
          </div>
        )} */}
        {showCostModal && (
          <div className="fixed inset-0 bg-black bg-opacity-50 flex justify-center items-center">
            <div className="bg-white p-6 rounded-lg shadow-lg w-1/2">
              <h2 className="text-xl font-bold mb-4">Set Costs for Uploaded Files</h2>
              <div className="overflow-y-auto max-h-80">
                <table className="w-full text-left">
                  <thead>
                    <tr>
                      <th className="px-2 py-2">
                        <input
                          type="checkbox"
                          checked={fileQueue.every((file) => file.selectStatus)}
                          onChange={(e) =>
                            setfileQueue(
                              fileQueue.map((file) => ({
                                ...file,
                                selectStatus: e.target.checked
                              }))
                            )
                          }
                        />
                      </th>
                      <th className="px-4 py-2">File Name</th>
                      <th className="px-4 py-2">Size (MB)</th>
                      <th className="px-4 py-2">Cost (SWE)</th>
                    </tr>
                  </thead>
                  <tbody>
                    {fileQueue.map((file, index) => (
                      <tr key={index} className="border-b">
                        <td className="px-2 py-2">
                          <input
                            type="checkbox"
                            checked={file.selectStatus ?? true}
                            onChange={(e) =>
                              setfileQueue((prevQueue) =>
                                prevQueue.map((f, i) =>
                                  i === index ? { ...f, selectStatus: e.target.checked } : f
                                )
                              )
                            }
                          />
                        </td>
                        <td className="px-4 py-2 relative group">
                          {/* Truncate file name */}
                          {file.fileName.length > 20
                            ? `${file.fileName.slice(0, 20)}...`
                            : file.fileName}

                          {/* Tooltip */}
                          <div className="absolute bottom-3 left-2 transform translate-y-full bg-gray-700 text-white text-sm rounded-lg p-2 opacity-0 group-hover:opacity-100 transition-opacity duration-100 delay-50 pointer-events-none">
                            {file.fileName}
                          </div>
                        </td>
                        <td className="px-4 py-2">
                          {file.fileSize.toLocaleString(undefined, {
                            minimumFractionDigits: 0,
                            maximumFractionDigits: 6
                          })}
                        </td>
                        <td className="px-4 py-2">
                          <input
                            type="number"
                            value={file.fileCost}
                            onChange={(e) => {
                              const updatedCost = parseFloat(e.target.value) || 0
                              setfileQueue((prevQueue) =>
                                prevQueue.map((f, i) =>
                                  i === index ? { ...f, cost: updatedCost } : f
                                )
                              )
                            }}
                            placeholder="Enter cost"
                            className="border border-gray-300 rounded-lg p-2 w-full"
                          />
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
              <div className="flex justify-end mt-4">
                <button
                  onClick={handleCostCancelAll}
                  className="bg-gray-300 hover:bg-gray-400 text-gray-800 px-4 py-2 rounded-lg mr-2"
                >
                  Cancel All
                </button>
                <button
                  onClick={() => handleCostConfirm(fileQueue.filter((file) => file.selectStatus))}
                  className="bg-[#737fa3] hover:bg-[#7c85a3] text-white px-4 py-2 rounded-lg"
                >
                  Confirm Selected
                </button>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}

export default Upload
