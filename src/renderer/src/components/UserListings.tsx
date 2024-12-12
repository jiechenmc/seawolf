import React, { useState, useEffect } from 'react'
import { FaRegFile, FaRegFilePdf, FaRegTrashAlt } from 'react-icons/fa'
import { LuFileText } from 'react-icons/lu'
import { BsFiletypeMp4, BsFiletypeMp3, BsFiletypePng, BsFiletypeJpg } from 'react-icons/bs'
import { ChevronDown } from 'lucide-react'
import { AppContext } from '../AppContext'
import ChatMenu from './ChatMenu'
import {
  acceptChatRequest,
  declineChatRequest,
  discoverFiles,
  finishChat,
  getIncomingChatRequests
} from '@renderer/rpcUtils'

type listingType = {
  size: number
  data_cid: string
  peer_id: string
  price: number
  file_name: string
  wallet_address: string
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

type messageType = {
  timestamp: string
  from: string
  text: string
}

type chatType = {
  chat_id: number
  buyer: string
  seller: string
  file_cid: string
  status: string
  messages: messageType[]
}

type chatRequestType = {
  request_id: number
  peer_id: string
  file_cid: string
  status: string
  chat?: chatType
}

const UserListings = () => {
  const {
    user: [peerId]
  } = React.useContext(AppContext)

  const [searchTerm, setSearchTerm] = useState('')
  const [discoveredFiles, setDiscoveredFiles] = useState<listingType[]>([])
  const [displayedListings, setDisplayedListings] = useState<listingType[]>([])
  const [sortBy, setSortBy] = useState('lowest_price')
  const [showSortOptions, setShowSortOptions] = useState(false)

  const [isChatOpen, setIsChatOpen] = useState(false)
  const [showChatRequests, setShowChatRequests] = useState<boolean>(false)
  const [chatRequests, setChatRequests] = useState<chatRequestType[]>([])

  const [currentChat, setCurrentChat] = useState<chatType>()

  useEffect(() => {
    const fetchData = async () => {
      try {
        const data = await discoverFiles()
        let arr: listingType[] = data
          .flatMap((discover: discoverType) =>
            discover.providers.map((provider: providerType) => ({
              size: discover.size,
              data_cid: discover.data_cid,
              peer_id: provider.peer_id,
              price: provider.price,
              file_name: provider.file_name,
              wallet_address: provider.wallet_address
            }))
          )
          .filter((listing: listingType) => listing.peer_id === peerId)
        setDiscoveredFiles(arr)
        setDisplayedListings(arr)
      } catch (error) {
        console.error('Error discovering all files on network: ', error)
      }
    }
    fetchData()
  }, [])

  const handleChatOpen = (chatRequest: chatRequestType) => {
    setCurrentChat(chatRequest.chat)
    setIsChatOpen(true)
  }

  const handleChatClose = () => {
    setIsChatOpen(false)
  }

  const handleShowChatRequests = async (listing: listingType) => {
    try {
      const data = await getIncomingChatRequests()
      setChatRequests(
        data.filter(
          (chatRequest: chatRequestType) =>
            chatRequest.file_cid === listing.data_cid && chatRequest.status !== 'declined'
        )
      )
      setShowChatRequests(true)
    } catch (error) {
      console.log('Error getting chat requests: ', error)
    }
  }

  const handleAcceptChat = async (chatRequest: chatRequestType) => {
    try {
      const data = await acceptChatRequest(chatRequest.peer_id, chatRequest.request_id)
      setChatRequests((prevRequests: chatRequestType[]) =>
        prevRequests.map((request) =>
          request.request_id === chatRequest.request_id
            ? {
                ...chatRequest,
                status: 'accepted',
                chat: data
              }
            : request
        )
      )
    } catch (error) {
      console.log('Error accepting chat request: ', error)
    }
  }

  const handleDeclineChat = async (chatRequest: chatRequestType) => {
    try {
      await declineChatRequest(chatRequest.peer_id, chatRequest.request_id)
      setChatRequests((prevRequests: chatRequestType[]) =>
        prevRequests.map((request) =>
          request.request_id === chatRequest.request_id
            ? {
                ...chatRequest,
                status: 'declined'
              }
            : request
        )
      )
    } catch (error) {
      console.log('Error declining chat request: ', error)
    }
  }

  const handleFinishChat = async (chatRequest: chatRequestType) => {
    try {
      let chatID = chatRequest.chat?.chat_id || -1
      const data = await finishChat(chatRequest.peer_id, chatID)
      setChatRequests((prevRequests: chatRequestType[]) =>
        prevRequests.map((request) =>
          request.chat?.chat_id === data.chat_id
            ? {
                ...chatRequest,
                status: 'finished',
                chat: data
              }
            : request
        )
      )
    } catch (error) {
      console.log('Error closing up chat: ', error)
    }
  }

  const sortOptions = [
    { id: 'lowest_price', label: 'Lowest Price' },
    { id: 'highest_price', label: 'Highest Price' }
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
      setDisplayedListings(discoveredFiles)
    } else {
      const searchList = discoveredFiles.filter(
        (listing: listingType) =>
          listing.file_name.toLowerCase().includes(searchTerm.toLowerCase()) ||
          listing.data_cid.toString().includes(searchTerm)
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
        sortedListings.sort((a, b) => a.price - b.price)
        break
      case 'highest_price':
        sortedListings.sort((a, b) => b.price - a.price)
        break
    }
    setDisplayedListings(sortedListings)
  }

  // const handleRemoveListing = (listing: listingType) => {
  //   const confirmed = window.confirm(
  //     `Are you sure you want to remove the follwing listing: ${listing.file_name}?`
  //   )

  //   if (confirmed) {
  //     setUserListings((prevListings) => prevListings.filter((item) => item.cid !== listing.cid))

  //     // Remove from market listings
  //     setMarketListings((prevListings) => prevListings.filter((item) => item.cid !== listing.cid))
  //   }
  // }

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
            {sortOptions.find((opt) => opt.id === sortBy)?.label}
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
          <span className="flex-[4.5] font-semibold text-left">File</span>
          <span className="flex-1 font-semibold text-left">Cost (SWE)</span>
          <span className="flex-1 font-semibold text-left">Size</span>
          <span className="flex-1 font-semibold text-left">Action</span>
        </div>
      </div>

      {displayedListings.map((listing, index) => (
        <div
          key={index}
          className="flex items-center px-2 py-1 border-b border-gray-300 rounded-md"
        >
          <div className="flex flex-[4.5] items-center">
            <div className="flex flex-col items-center justify-center h-full mr-5">
              {getFileIcon(listing.file_name)}
            </div>
            <div className="ml-2">
              <span className="block font-semibold">{listing.file_name}</span>
              <span className="block text-gray-500">{listing.data_cid}</span>
            </div>
          </div>
          <span className="flex-1 text-left">{listing.price}</span>
          <span className="flex-1 text-left">
            {(listing.size / 1e6).toLocaleString(undefined, {
              minimumFractionDigits: 0,
              maximumFractionDigits: 6
            })}{' '}
            MB
          </span>
          <span className="flex-1 text-left flex items-center justify-between">
            {/* <span
              className={`px-2 py-1 rounded ${
                listing.status === 'active'
                  ? 'bg-green-100 text-green-800'
                  : 'bg-gray-100 text-gray-800'
              }`}
            >
              {listing.status}
            </span> */}
            <button
              onClick={() => handleShowChatRequests(listing)}
              className="bg-[#737fa3] text-white px-3 py-1 rounded"
            >
              Show Chats
            </button>
            {/* <button
              onClick={() => handleRemoveListing(listing)}
              className="text-red-500 hover:text-red-700 ml-2"
            >
              <FaRegTrashAlt />
            </button> */}
          </span>
        </div>
      ))}

      {/* Pagination */}
      {/* <div className="mt-4 flex justify-center gap-2">
        {[1, 2, 3, 4, 5].map((page) => (
          <button
            key={page}
            className="px-3 py-1 bg-[#737fa3] hover:bg-[#7c85a3] text-white rounded"
          >
            {page}
          </button>
        ))}
      </div> */}
      {showChatRequests && (
        <div className="absolute top-0 left-0 w-full h-full bg-black bg-opacity-50 flex justify-center items-center">
          <div className="bg-white w-2/3 p-6 rounded-lg shadow-lg max-h-[80%] overflow-y-auto">
            <h2 className="text-xl font-bold mb-4 text-center">Incoming Chats</h2>
            <ul className="space-y-2">
              {chatRequests.map((chatRequest: chatRequestType, index) => (
                <li
                  key={index}
                  className="p-4 rounded-md hover:bg-gray-200 cursor-pointer border border-gray-300"
                >
                  <div>
                    <p className="font-medium text-base">Peer: {chatRequest.peer_id}</p>
                    <p className="text-gray-600">Request ID: {chatRequest.request_id}</p>
                    <p className="text-gray-600">Status: {chatRequest.status}</p>
                  </div>
                  {chatRequest.status === 'pending' && (
                    <div className="flex gap-2 mt-2">
                      <button
                        onClick={() => handleAcceptChat(chatRequest)}
                        className="bg-green-500 text-white px-4 py-2 rounded-md"
                      >
                        Accept
                      </button>
                      <button
                        onClick={() => handleDeclineChat(chatRequest)}
                        className="bg-red-500 text-white px-4 py-2 rounded-md"
                      >
                        Decline
                      </button>
                    </div>
                  )}
                  {chatRequest.status === 'accepted' && (
                    <div className="flex gap-2 mt-2">
                      <button
                        onClick={() => handleChatOpen(chatRequest)}
                        className="bg-blue-500 text-white px-4 py-2 rounded-md"
                      >
                        Chat Now
                      </button>
                      {chatRequest.chat?.status !== 'finished' && (
                        <button
                          onClick={() => handleFinishChat(chatRequest)}
                          className="bg-red-500 text-white px-4 py-2 rounded-md"
                        >
                          Finish Chat
                        </button>
                      )}
                    </div>
                  )}
                  {chatRequest.status === 'finished' && (
                    <div className="flex gap-2 mt-2">
                      <button
                        onClick={() => handleChatOpen(chatRequest)}
                        className="bg-blue-500 text-white px-4 py-2 rounded-md"
                      >
                        View Chat History
                      </button>
                    </div>
                  )}
                </li>
              ))}
              {chatRequests.length === 0 && (
                <p className="text-gray-600 text-center mt-4">
                  No incoming chat requests at this time
                </p>
              )}
            </ul>
            <button
              className="mt-4 w-full px-4 py-2 font-semibold rounded-md bg-gray-300 hover:bg-gray-400 text-gray-800"
              onClick={() => setShowChatRequests(false)}
            >
              Close
            </button>
          </div>
        </div>
      )}
      {isChatOpen && <ChatMenu onClose={handleChatClose} chat={currentChat} />}
    </div>
  )
}

export default UserListings
