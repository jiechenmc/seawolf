import React, { useState } from 'react'
import { AppContext } from '../AppContext'

type FileType = {
  cid: number
  name: string
  size: number
  cost: number
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

const CreateListingContent = () => {
  const { allFiles, downloadFiles } = React.useContext(AppContext)
  const {
    marketListing: [marketListings, setMarketListings],
    userListing: [userListings, setUserListings]
  } = React.useContext(AppContext)
  const [listOfFiles] = allFiles
  const [downloadedFiles] = downloadFiles

  const filesForMarket = listOfFiles.concat(
    downloadedFiles.filter((file) => file.status === 'Done').map((file) => file.file)
  )

  const [selectedFile, setSelectedFile] = useState<FileType | null>(null)
  const [cost, setCost] = useState<number>(0)
  const [endDate, setEndDate] = useState<string>('')
  const [listingType, setListingType] = useState<'sale' | 'auction'>('sale')
  const [formError, setFormError] = useState<string>('')

  // min end ddate the day after
  const getTomorrowDate = () => {
    const tomorrow = new Date()
    tomorrow.setDate(tomorrow.getDate() + 1)
    return tomorrow.toISOString().split('T')[0]
  }

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    setFormError('')

    if (!selectedFile) {
      setFormError('Please select a file to list')
      return
    }

    if (cost <= 0) {
      setFormError('Cost must be greater than 0')
      return
    }

    if (!endDate) {
      setFormError('Please select an end date')
      return
    }

    const isAlreadyListed = marketListings.some(
      (listing) => listing.cid === selectedFile.cid && listing.status === 'active'
    )

    if (isAlreadyListed) {
      setFormError('This file is already listed in the market')
      return
    }

    const newListing: ListingType = {
      cid: selectedFile.cid,
      name: selectedFile.name,
      size: selectedFile.size,
      cost: cost,
      endDate: endDate,
      type: listingType,
      status: 'active'
    }

    setMarketListings((prevListings) => [...prevListings, newListing])
    setUserListings((prevListings) => [...prevListings, newListing])
    // Reset form
    setSelectedFile(null)
    setCost(0)
    setEndDate('')
    setListingType('sale')
    alert('Listing created successfully!')
  }

  return (
    <div className="w-full max-w-2xl mx-auto">
      <div className="bg-white p-6 rounded-lg shadow-md">
        <form onSubmit={handleSubmit}>
          {/* File Selection */}
          <div className="mb-6">
            <label className="block text-sm font-medium text-gray-700 mb-2">Select File</label>
            <select
              className="w-full p-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
              value={selectedFile?.cid || ''}
              onChange={(e) => {
                const file = filesForMarket.find((f) => f.cid.toString() === e.target.value)
                setSelectedFile(file || null)
              }}
            >
              <option value="">Select a file</option>
              {filesForMarket.map((file: FileType) => (
                <option key={file.cid} value={file.cid}>
                  {file.name} ({file.size} MB)
                </option>
              ))}
            </select>
          </div>

          {/* Auction or Sale */}
          {/* <div className="mb-6">
            <label className="block text-sm font-medium text-gray-700 mb-2">Listing Type</label>
            <div className="flex gap-4">
              <label className="flex items-center">
                <input
                  type="radio"
                  value="sale"
                  checked={listingType === 'sale'}
                  onChange={(e) => setListingType(e.target.value as 'sale' | 'auction')}
                  className="mr-2"
                />
                Sale
              </label>
              <label className="flex items-center">
                <input
                  type="radio"
                  value="auction"
                  checked={listingType === 'auction'}
                  onChange={(e) => setListingType(e.target.value as 'sale' | 'auction')}
                  className="mr-2"
                />
                Auction
              </label>
            </div>
          </div> */}

          {/* Cost/Starting Bid */}
          <div className="mb-6">
            <label className="block text-sm font-medium text-gray-700 mb-2">
              {listingType === 'sale' ? 'Price (SWE)' : 'Starting Bid (SWE)'}
            </label>
            <div className="flex border border-gray-300 rounded-md p-2 items-center">
              <input
                type="number"
                value={cost}
                onChange={(e) => {
                  const newValue = parseFloat(e.target.value)
                  setCost(newValue < 0 ? 0 : newValue)
                }}
                className="block border-none outline-none w-full pr-4"
                placeholder="0"
                min="0"
                step="any"
              />
              <span className="text-gray-500">SWE</span>
            </div>
          </div>

          {/* End Date */}
          <div className="mb-6">
            <label className="block text-sm font-medium text-gray-700 mb-2">End Date</label>
            <input
              type="date"
              value={endDate}
              onChange={(e) => setEndDate(e.target.value)}
              min={getTomorrowDate()}
              className="w-full p-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
            />
          </div>

          {/* Error */}
          {formError && <div className="mb-4 text-red-500 text-sm">{formError}</div>}

          {/* Submit Button */}
          <button
            type="submit"
            className="w-full bg-[#737fa3] hover:bg-[#7c85a3] text-white px-4 py-2 rounded-lg"
          >
            Create Listing
          </button>
        </form>
      </div>

      {/* File preview */}
      {selectedFile && (
        <div className="mt-6 bg-white p-6 rounded-lg shadow-md">
          <h3 className="text-lg font-semibold mb-4">Selected File Details</h3>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <p className="text-sm text-gray-500">File Name</p>
              <p className="font-medium">{selectedFile.name}</p>
            </div>
            <div>
              <p className="text-sm text-gray-500">Size</p>
              <p className="font-medium">{selectedFile.size.toFixed(2)} MB</p>
            </div>
            <div>
              <p className="text-sm text-gray-500">File ID</p>
              <p className="font-medium">{selectedFile.cid}</p>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

export default CreateListingContent
