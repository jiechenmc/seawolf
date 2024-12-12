import React, { useState, useRef, useEffect } from 'react'
import SideNav from './SideNav'
import {
  FaFolderOpen,
  FaRegFilePdf,
  FaRegFile,
  FaCloudUploadAlt,
  FaRegTrashAlt
} from 'react-icons/fa'
import { LuFileText } from 'react-icons/lu'
import { BsFiletypeMp4, BsFiletypeMp3, BsFiletypePng, BsFiletypeJpg } from 'react-icons/bs'
import { AppContext } from '../AppContext'
import { deleteFile, getUploadedFiles, uploadFile } from '@renderer/rpcUtils'
import LoadingModal from './LoadingModal'

// type fileType = {
//   cid: number
//   uploadPath: string
//   name: string
//   size: number
//   cost: number
//   uploadEta?: number
//   uploadStatus?: 'uploading' | 'completed' | 'cancelled' | 'error' | null
//   downloadPath?: string
//   downloadEta?: number
//   downloadStatus?: 'downloading' | 'completed' | 'cancelled' | 'error' | null
//   selectStatus?: boolean
// }

type fileType = {
  size: number
  price: number
  file_name: string
  data_cid: string
  provider_id: string
  uploadPath: string
  select_status?: boolean
}

function Upload(): JSX.Element {
  const fileInputRef = useRef<HTMLInputElement | null>(null)

  const { user, sysPlatform, numUploadFiles, numUploadBytes, history } =
    React.useContext(AppContext)

  const [peerId] = user
  const [platform] = sysPlatform

  const [historyView, setHistoryView] = history

  const [numFiles, setNumFiles] = useState<number>(0)
  const [numBytes, setNumBytes] = useState<number>(0)

  const [uploadedFiles, setUploadedFiles] = useState([])
  const [filesToView, setFilesToView] = useState<fileType[]>([])
  const [searchHash, setSearchHash] = useState<string>('')

  const [fileQueue, setfileQueue] = useState<fileType[]>([])
  const [showCostModal, setShowCostModal] = useState<boolean>(false)

  const [byteCount, setByteCount] = useState<number>(0)
  const [fileCount, setFileCount] = useState<number>(0)

  const [loading, setLoading] = useState<boolean>(false)

  function normalizePath(filePath: string) {
    let isWin = platform === 'win32' ? true : false
    if (isWin) {
      const drive = filePath[0].toLowerCase()
      const unixPath = filePath.replace(/^([a-zA-Z]):\\/, `/mnt/${drive}/`).replace(/\\/g, '/')
      return unixPath
    }
    return filePath
  }

  useEffect(() => {
    const fetchData = async () => {
      try {
        const data = await getUploadedFiles()
        setUploadedFiles(data)
        setFilesToView(data)
        setNumFiles(data.length)
        setNumBytes(data.reduce((sum: number, file: fileType) => sum + file.size, 0))
      } catch (error) {
        console.error('Error fetching uploaded files: ', error)
      }
    }

    fetchData()
  }, [])

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
        if (file.file_name.includes(searchHash)) {
          searchList.push(file)
        }
        if (file.data_cid.includes(searchHash)) {
          searchList.push(file)
        }
      })

      setFilesToView(() => searchList)
    }
  }

  const handleUploadButtonClick = () => {
    fileInputRef.current?.click()
  }

  const handleFileUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files) {
      const filesArr = Array.from(e.target.files)
      setFileCount(filesArr.length)

      let byteCount: number = 0
      let filesToAppend = filesArr.map((file) => {
        byteCount += file.size
        let newFile: fileType = {
          size: file.size,
          price: 0,
          file_name: file.name,
          data_cid: '',
          provider_id: peerId,
          uploadPath: normalizePath(file.path),
          select_status: true
        }
        return newFile
      })

      setByteCount(byteCount)
      setfileQueue(filesToAppend)
      setShowCostModal(true)
    }
  }

  const handleCostConfirm = async (fileQueue: fileType[]) => {
    setNumBytes(byteCount)
    setNumFiles(fileCount)
    await Promise.all(
      fileQueue.map(async (file) => {
        try {
          file.data_cid = await uploadFile(file.uploadPath, file.price)
        } catch (error) {
          console.log(error)
        }
      })
    )
    setFilesToView((prevList) => prevList.concat(fileQueue))
    setShowCostModal(false)
  }

  const handleCostCancelAll = () => {
    setShowCostModal(false)
  }

  const handleRemoveFile = async (index: number, file: fileType) => {
    let confirmed = window.confirm(`Are you sure you want to delete: ${file.file_name}?`)
    if (confirmed) {
      try {
        await deleteFile(file.data_cid)
        setNumBytes((prevBytes: number) =>
          prevBytes - file.size === 0 ? 0 : prevBytes - file.size
        )
        setNumFiles((prevFiles: number) => prevFiles - 1)
        filesToView.splice(index, 1)
        setFilesToView(() => filesToView)
      } catch (error) {
        console.error('Error fetching uploaded files:', error)
      }

      try {
        const data = await getUploadedFiles()
        setUploadedFiles(data)
      } catch (error) {
        console.error('Error fetching uploaded files:', error)
      }
    }
  }

  return (
    <div className="flex ml-52">
      <SideNav />

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
                {Math.round(numBytes / 1e6) === 0
                  ? '0 MB'
                  : (numBytes / 1e6).toLocaleString(undefined, {
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
            <span className="flex-[3] font-semibold text-left">File</span>
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
            <div className="flex flex-[3] items-center">
              <div className="flex flex-col items-center justify-center h-full mr-5">
                {getFileIcon(file.file_name)}
              </div>
              <div className="ml-2">
                <span className="block font-semibold">
                  {file.file_name.length > 60
                    ? `${file.file_name.slice(0, 57)}...`
                    : file.file_name}
                </span>
                <span className="block text-gray-500">{file.data_cid}</span>{' '}
              </div>
            </div>
            <span className="flex-1 text-left">
              {(file.size / 1e6).toLocaleString(undefined, {
                minimumFractionDigits: 0,
                maximumFractionDigits: 4
              })}{' '}
              MB
            </span>
            <span className="flex-1 text-left flex justify-between items-center">
              <span>{file.price} SWE</span>
              <button
                onClick={() => handleRemoveFile(index, file)}
                className="text-red-500 hover:text-red-700"
              >
                <FaRegTrashAlt />
              </button>
            </span>
          </div>
        ))}
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
                          checked={fileQueue.every((file: fileType) => file.select_status)}
                          onChange={(e) =>
                            setfileQueue(
                              fileQueue.map((file: fileType) => ({
                                ...file,
                                select_status: e.target.checked
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
                    {fileQueue.map((file: fileType, index: number) => (
                      <tr key={index} className="border-b">
                        <td className="px-2 py-2">
                          <input
                            type="checkbox"
                            checked={file.select_status ?? true}
                            onChange={(e) =>
                              setfileQueue((prevQueue) =>
                                prevQueue.map((f, i) =>
                                  i === index ? { ...f, select_status: e.target.checked } : f
                                )
                              )
                            }
                          />
                        </td>
                        <td className="px-4 py-2 relative group">
                          {file.file_name.length > 60
                            ? `${file.file_name.slice(0, 57)}...`
                            : file.file_name}
                          <div className="absolute bottom-3 left-2 transform translate-y-full bg-gray-700 text-white text-sm rounded-lg p-2 opacity-0 group-hover:opacity-100 transition-opacity duration-100 delay-50 pointer-events-none">
                            {file.file_name.length > 60
                              ? `${file.file_name.slice(0, 57)}...`
                              : file.file_name}
                          </div>
                        </td>
                        <td className="px-4 py-2">
                          {(file.size / 1e6).toLocaleString(undefined, {
                            minimumFractionDigits: 0,
                            maximumFractionDigits: 6
                          })}
                        </td>
                        <td className="px-4 py-2">
                          <input
                            type="number"
                            value={file.price}
                            onChange={(e) => {
                              const updatedCost = parseFloat(e.target.value) || 0
                              setfileQueue((prevQueue) =>
                                prevQueue.map((f, i) =>
                                  i === index ? { ...f, price: updatedCost } : f
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
                  onClick={() => handleCostConfirm(fileQueue.filter((file) => file.select_status))}
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
