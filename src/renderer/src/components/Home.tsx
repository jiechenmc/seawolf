import React, { useState, useRef, useEffect } from 'react'
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

type fileType = {
  cid: number
  name: string
  size: number
  cost: number
}

function Home(): JSX.Element {
  const fileInputRef = useRef<HTMLInputElement | null>(null)

  const { totalFiles, totalBytes, allFiles, viewFiles, search } = React.useContext(AppContext)

  const [numFiles, setNumFiles] = totalFiles
  const [numBytes, setNumBytes] = totalBytes
  const [listOfFiles, setListOfFiles] = allFiles
  const [filesToView, setFilesToView] = viewFiles
  const [searchHash, setSearchHash] = search

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
      setFilesToView(listOfFiles)
    } else {
      let searchList: fileType[] = []
      listOfFiles.forEach((file: fileType) => {
        if (file.name.includes(searchHash)) {
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
      let fileCount: number = 0
      let byteCount: number = 0
      let filesToAppend: fileType[] = listOfFiles
      filesArr.forEach((file) => {
        byteCount += file.size / 1e6
        fileCount++
        let newFile: fileType = {
          cid: generateRandom10DigitNumber(),
          name: file.name,
          size: file.size / 1e6,
          cost: 0
        }
        filesToAppend.push(newFile)
      })
      setNumBytes((prevBytes: number) => prevBytes + byteCount)
      setNumFiles((prevFiles: number) => prevFiles + fileCount)
      setListOfFiles(() => filesToAppend)
      setFilesToView(() => filesToAppend)
    }
  }

  const handleUploadButtonClick = () => {
    fileInputRef.current?.click()
  }

  const handleRemoveFile = (file: fileType) => {
    let newByteSize = numBytes - file.size
    setNumBytes(() => (newByteSize === 0 ? 0 : newByteSize))
    setNumFiles((prevFiles: number) => prevFiles - 1)
    let fileIndex = listOfFiles.indexOf(file)
    console.log(fileIndex)
    let newFileList = listOfFiles
    newFileList.splice(fileIndex, 1)
    console.log(newFileList)
    setListOfFiles(() => newFileList)
    setFilesToView(() => newFileList)
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
              <p className="text-lg">{numBytes === 0 ? '0 MB' : numBytes.toFixed(6) + ' MB'}</p>
            </div>
          </div>
        </div>

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
                {getFileIcon(file.name)}
              </div>
              <div className="ml-2">
                <span className="block font-semibold">{file.name}</span>
                <span className="block text-gray-500">{file.cid}</span>{' '}
              </div>
            </div>
            <span className="flex-1 text-left">{file.size.toFixed(4)} MB</span>
            <span className="flex-1 text-left flex justify-between items-center">
              <span>{file.cost} SWE</span>
              <button
                onClick={() => handleRemoveFile(file)}
                className="text-red-500 hover:text-red-700"
              >
                <FaRegTrashAlt />
              </button>
            </span>
          </div>
        ))}
      </div>
    </div>
  )
}

export default Home
