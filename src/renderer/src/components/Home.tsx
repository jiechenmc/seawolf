import React, { useState } from 'react'
import SideNav from './SideNav'
import { FaFolderOpen, FaRegFilePdf, FaRegFile, FaCloudUploadAlt } from 'react-icons/fa'
import { LuFileText } from 'react-icons/lu'
import { BsFiletypeMp4, BsFiletypeMp3, BsFiletypePng, BsFiletypeJpg } from 'react-icons/bs'

function Home(): JSX.Element {
  const [listOfFiles, setListOfFiles] = useState([])
  const [filesToView, setFilesToView] = useState([])
  const [fileHash, setFileHash] = useState('')

  const getFileIcon = (fileType: string) => {
    switch (fileType) {
      case '.dir':
        return <FaFolderOpen />
      case '.txt':
        return <LuFileText />
      case '.pdf':
        return <FaRegFilePdf />
      case '.mp3':
        return <BsFiletypeMp3 />
      case '.mp4':
        return <BsFiletypeMp4 />
      case '.png':
        return <BsFiletypePng />
      case '.jpg':
        return <BsFiletypeJpg />
      default:
        return <FaRegFile />
    }
  }

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault()

    console.log(fileHash)
    let searchList = []

    listOfFiles.forEach((file) => {
      console.log(file)
    })

    setFilesToView(searchList)
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
              <p className="text-lg">0</p>
            </div>
            <div className="flex-1 text-center">
              <h3 className="font-bold">Total Bytes</h3>
              <p className="text-lg">0 GB</p>
            </div>
          </div>
        </div>

        <div className="flex justify-between mb-4">
          <div className="flex w-2/3">
            <input
              type="text"
              placeholder="Search for file name or hash"
              value={fileHash}
              onChange={(e) => setFileHash(e.target.value)}
              className="border border-gray-300 rounded-lg p-2 flex-1 mr-2"
            />
            <button
              className="bg-[#737fa3] hover:bg-[#7c85a3] text-white px-4 py-2 rounded-lg"
              onClick={handleSearch}
            >
              Search
            </button>
          </div>

          <label htmlFor="file-upload">
            <button className="bg-[#737fa3] hover:bg-[#7c85a3] text-white px-4 py-2 rounded-lg flex items-center">
              <FaCloudUploadAlt className="mr-2" />
              <span>Upload File</span>
            </button>
            <input
              id="file-upload"
              type="file"
              // onChange={handleFileChange}
              className="hidden"
              multiple
            />
          </label>
        </div>

        <div className="overflow-x-auto border border-gray-300 rounded-lg">
          <div className="flex items-center p-2 border-b border-gray-300">
            <input type="checkbox" className="mr-10" />
            <span className="flex-1 font-semibold text-left">File</span>
            <span className="flex-1 font-semibold text-center">Bytes</span>
          </div>
        </div>
      </div>
    </div>
  )
}

export default Home
