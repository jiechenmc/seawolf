import React, { useState, useEffect } from 'react'
import { FaRegFile, FaRegFilePdf } from 'react-icons/fa'
import { LuFileText } from 'react-icons/lu'
import { BsFiletypeMp4, BsFiletypeMp3, BsFiletypePng, BsFiletypeJpg } from 'react-icons/bs'
import { ChevronDown } from 'lucide-react'
import { AppContext } from '../AppContext'
import ChatMenu from './ChatMenu'

type ListingType = {
  cid: number
  name: string
  size: number
  cost: number
  endDate: string
  type: 'sale' | 'auction'
  status: 'active' | 'ended'
}
const ListingsContent = () => {
  const { marketListing: [marketListings, setMarketListings] } = React.useContext(AppContext)
  const [searchTerm, setSearchTerm] = useState('')
  const [displayedListings, setDisplayedListings] = useState<ListingType[]>([]) 
  const [sortBy, setSortBy] = useState('lowest_price')
  const [showSortOptions, setShowSortOptions] = useState(false)

  const [isChatOpen, setIsChatOpen] = useState(false)

  const handleChatOpen = () => {
    setIsChatOpen(true)
  }

  const handleChatClose = () => {
    setIsChatOpen(false)
  }

  useEffect(() => {
    setDisplayedListings(marketListings)
  }, [marketListings])

  const sortOptions = [
    { id: 'lowest_price', label: 'Lowest Price' },
    { id: 'highest_price', label: 'Highest Price' },
    { id: 'recent', label: 'Most Recent' },
    { id: 'oldest', label: 'Oldest' },
  ]

  const getFileIcon = (fileName: string) => {
    const fileType = fileName.split('.').pop()

    switch (fileType) {
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

    if (searchTerm === '') {
      setDisplayedListings(marketListings)
    } else {
      const searchList = marketListings.filter((listing) => 
        listing.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
        listing.cid.toString().includes(searchTerm)
      )
      setDisplayedListings(searchList)
    }
  }

  const handleSortChange = (option: string) => {
    setSortBy(option)
    setShowSortOptions(false)
    const sortedListings = [...displayedListings]
    switch (option) {
      case 'lowest_price':
        sortedListings.sort((a, b) => a.cost - b.cost)
        break
      case 'highest_price':
        sortedListings.sort((a, b) => b.cost - a.cost)
        break
      case 'recent':
        sortedListings.sort((a, b) => new Date(b.endDate).getTime() - new Date(a.endDate).getTime())
        break
      case 'oldest':
        sortedListings.sort((a, b) => new Date(a.endDate).getTime() - new Date(b.endDate).getTime())
        break
    }
    setDisplayedListings(sortedListings)
  }

  return (
    <div className="w-full">
      {/* Search and Filter */}
      <div className="flex justify-between mb-4">
        <div className="flex w-2/3">
          <input
            type="text"
            placeholder="Search for file name or hash"
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="border border-gray-300 rounded-lg p-2 flex-1 mr-2"
            onKeyDown={(e) => {
              if (e.key === 'Enter') {
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

        <div className="relative">
          <button
            onClick={() => setShowSortOptions(!showSortOptions)}
            className="bg-[#737fa3] hover:bg-[#7c85a3] text-white px-4 py-2 rounded-lg flex items-center gap-2"
          >
            {sortOptions.find(opt => opt.id === sortBy)?.label}
            <ChevronDown className="w-4 h-4" />
          </button>
          {showSortOptions && (
            <div className="absolute right-0 mt-2 w-48 bg-white rounded-lg shadow-lg border border-gray-200 z-10">
              {sortOptions.map((option) => (
                <button
                  key={option.id}
                  onClick={() => handleSortChange(option.id)}
                  className="w-full text-left px-4 py-2 hover:bg-gray-100 first:rounded-t-lg last:rounded-b-lg"
                >
                  {option.label}
                </button>
              ))}
            </div>
          )}
        </div>
      </div>

      {/* Listings Table */}
      <div className="overflow-x-auto border border-gray-300 rounded-lg">
        <div className="flex items-center p-2 border-b border-gray-300">
          <span className="flex-1 font-semibold text-left">File</span>
          <span className="flex-1 font-semibold text-left">Cost</span>
          <span className="flex-1 font-semibold text-left">End Date</span>
          <span className="flex-1 font-semibold text-left">Size</span>
          <span className="flex-1 font-semibold text-left">Action</span>
        </div>
      </div>
        
      {isChatOpen && <ChatMenu onClose={handleChatClose} otherUserName="Test"/>}

      {displayedListings.map((listing, index) => (
        <div
          key={index}
          className="flex items-center px-2 py-1 border-b border-gray-300 rounded-md"
        >
          <div className="flex flex-1 items-center">
            <div className="flex flex-col items-center justify-center h-full mr-5">
              {getFileIcon(listing.name)}
            </div>
            <div className="ml-2">
              <span className="block font-semibold">{listing.name}</span>
              <span className="block text-gray-500">{listing.cid}</span>
            </div>
          </div>
          <span className="flex-1 text-left">{listing.cost} SWE</span>
          <span className="flex-1 text-left">{listing.endDate}</span>
          <span className="flex-1 text-left">
            {listing.size.toLocaleString(undefined, {
              minimumFractionDigits: 0,
              maximumFractionDigits: 6
            })}{' '}
            MB
          </span>
          <span className="flex-1">
            <button
              className="bg-[#737fa3] hover:bg-[#7c85a3] text-white px-3 py-1 mr-5 rounded"
            >
              {listing.type === 'auction' ? 'Bid' : 'Buy'}
            </button>
            <button onClick={handleChatOpen} className="bg-[#737fa3] text-white px-3 py-1 rounded">
              Chat
            </button>
          </span>
        </div>
      ))}

      {/* Pagination */}
      <div className="mt-4 flex justify-center gap-2">
        {[1, 2, 3, 4, 5].map((page) => (
          <button
            key={page}
            className="px-3 py-1 bg-[#737fa3] hover:bg-[#7c85a3] text-white rounded"
          >
            {page}
          </button>
        ))}
      </div>
    </div>
  )
}

export default ListingsContent