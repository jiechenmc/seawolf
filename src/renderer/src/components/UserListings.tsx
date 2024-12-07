import React, { useState, useEffect } from 'react'
import { FaRegFile, FaRegFilePdf, FaRegTrashAlt } from 'react-icons/fa'
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

const UserListings = () => {
  const { 
    userListing: [userListings, setUserListings],
    marketListing: [marketListings, setMarketListings]
  } = React.useContext(AppContext)
  
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
    setDisplayedListings(userListings)
  }, [userListings])

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
      setDisplayedListings(userListings)
    } else {
      const searchList = userListings.filter((listing) => 
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

  const handleRemoveListing = (listing: ListingType) => {
    const confirmed = window.confirm(`Are you sure you want to remove the follwing listing: ${listing.name}?`)
    
    if (confirmed) {
      setUserListings(prevListings => 
        prevListings.filter(item => item.cid !== listing.cid)
      )
      
      // Remove from market listings
      setMarketListings(prevListings => 
        prevListings.filter(item => item.cid !== listing.cid)
      )
    }
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
          <span className="flex-1 font-semibold text-left">Status</span>
        </div>
      </div>
      
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
          <span className="flex-1 text-left flex items-center justify-between">
            <span className={`px-2 py-1 rounded ${
              listing.status === 'active' ? 'bg-green-100 text-green-800' : 'bg-gray-100 text-gray-800'
            }`}>
              {listing.status}
            </span>
            <button onClick={handleChatOpen} className="bg-[#737fa3] text-white px-3 py-1 rounded">
              Chat
            </button>
            <button
              onClick={() => handleRemoveListing(listing)}
              className="text-red-500 hover:text-red-700 ml-2"
            >
              <FaRegTrashAlt />
            </button>
          </span>
        </div>
      ))}

      {isChatOpen && <ChatMenu onClose={handleChatClose} otherUserName="Test"/>}

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

export default UserListings