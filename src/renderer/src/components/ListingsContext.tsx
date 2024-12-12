import React, { useState, useEffect } from 'react'
import { FaRegFile, FaRegFilePdf } from 'react-icons/fa'
import { LuFileText } from 'react-icons/lu'
import { BsFiletypeMp4, BsFiletypeMp3, BsFiletypePng, BsFiletypeJpg } from 'react-icons/bs'
import { ChevronDown } from 'lucide-react'
import { AppContext } from '../AppContext'
import ChatMenu from './ChatMenu'
import {
  discoverFiles,
  getChat,
  getFile,
  getOutgoingChatRequests,
  sendChatRequest
} from '@renderer/rpcUtils'
import LoadingModal from './LoadingModal'
import { tranferMoney } from '@renderer/walletUtils'

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

type requestType = {
  chat_id?: number
  request_id: number
  peer_id: string
  file_cid: string
  status: string
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

type listingType = {
  size: number
  data_cid: string
  peer_id: string
  price: number
  file_name: string
  wallet_address: string
  request_chat_status: string
  request_id?: number
  chat?: chatType
}

const ListingsContent = () => {
  const {
    user: [peerId],
    pathForDownload: [downloadPath],
    wallet: [walletAddress],
    balance: [walletBalance, setWalletBalance],
    history: [, setHistoryView]
  } = React.useContext(AppContext)

  const [discoveredFiles, setDiscoveredFiles] = useState<listingType[]>([])
  const [searchTerm, setSearchTerm] = useState('')
  const [displayedListings, setDisplayedListings] = useState<listingType[]>([])
  const [sortBy, setSortBy] = useState('lowest_price')
  const [showSortOptions, setShowSortOptions] = useState(false)

  const [isChatOpen, setIsChatOpen] = useState(false)
  const [currentChat, setCurrentChat] = useState<chatType>()

  const [loading, setLoading] = useState<boolean>(false)

  useEffect(() => {
    const fetchData = async () => {
      try {
        const data = await discoverFiles()
        const data2 = await getOutgoingChatRequests()

        const statusMap = new Map<
          string,
          { chat_id: number | undefined; request_id: number; status: string }
        >()
        data2.forEach((request: requestType) => {
          let key = `${request.peer_id}+${request.file_cid}`
          statusMap.set(key, {
            chat_id: request.chat_id,
            request_id: request.request_id,
            status: request.status
          })
        })

        let arr: listingType[] = await Promise.all(
          data.flatMap((discover: discoverType) =>
            discover.providers.map(async (provider: providerType) => {
              let key = `${provider.peer_id}+${discover.data_cid}`
              let requestData = statusMap.get(key)
              let chatID = requestData?.chat_id || -1
              let newListing = {
                size: discover.size,
                data_cid: discover.data_cid,
                peer_id: provider.peer_id,
                price: provider.price,
                file_name: provider.file_name,
                wallet_address: provider.wallet_address
              }
              if (requestData) {
                if (chatID) {
                  return {
                    ...newListing,
                    request_chat_status: requestData.status,
                    request_id: requestData.request_id
                  }
                } else {
                  return {
                    ...newListing,
                    request_chat_status: requestData.status,
                    request_id: requestData.request_id,
                    chat: await getChat(provider.peer_id, chatID)
                  }
                }
              } else {
                return {
                  ...newListing,
                  request_chat_status: 'not yet'
                }
              }
            })
          )
        )
        arr = arr.filter((listing: listingType) => listing.peer_id !== peerId)
        setDiscoveredFiles(arr)
        setDisplayedListings(arr)
        setLoading(false)
      } catch (error) {
        console.error('Error discovering all files on network: ', error)
      }
    }
    setLoading(true)
    fetchData()
  }, [])

  useEffect(() => {
    const interval = setInterval(() => {
      getOutgoingChatRequests()
        .then((data) => {
          const statusMap = new Map<number, string>()
          data.forEach((request: requestType) => {
            statusMap.set(request.request_id, request.status)
          })
          setDiscoveredFiles((prevList: listingType[]) =>
            prevList.map((eachListing: listingType) => {
              if (eachListing.request_id) {
                const status = statusMap.get(eachListing.request_id)
                if (status) {
                  if (data.chat_id) {
                    getChat(eachListing.peer_id, data.chat_id).then((chatData) => {
                      return {
                        ...eachListing,
                        request_chat_status: status,
                        chat: chatData
                      }
                    })
                  } else {
                    return {
                      ...eachListing,
                      request_chat_status: status
                    }
                  }
                }
              }
              return eachListing
            })
          )
          setDisplayedListings((prevList: listingType[]) =>
            prevList.map((eachListing: listingType) => {
              if (eachListing.request_id) {
                const status = statusMap.get(eachListing.request_id)
                if (status) {
                  if (data.chat_id) {
                    getChat(eachListing.peer_id, data.chat_id).then((chatData) => {
                      return {
                        ...eachListing,
                        request_chat_status: status,
                        chat: chatData
                      }
                    })
                  } else {
                    return {
                      ...eachListing,
                      request_chat_status: status
                    }
                  }
                }
              }
              return eachListing
            })
          )
        })
        .catch((error) => {
          console.error('Error getting outgoing chat requests: ', error)
        })
    }, 5000)

    return () => clearInterval(interval)
  }, [])

  const handleChatOpen = (listing: listingType) => {
    setCurrentChat(listing.chat)
    setIsChatOpen(true)
  }

  const handleChatClose = () => {
    setIsChatOpen(false)
  }

  const handleRequestChat = async (listing: listingType) => {
    try {
      const data = await sendChatRequest(listing.peer_id, listing.data_cid)

      setDiscoveredFiles((prevList: listingType[]) =>
        prevList.map((eachListing: listingType) => {
          if (
            eachListing.data_cid === listing.data_cid &&
            eachListing.peer_id === listing.peer_id
          ) {
            return {
              ...eachListing,
              request_chat_status: 'pending',
              request_id: data.request_id
            }
          }
          return eachListing
        })
      )
      setDisplayedListings((prevList: listingType[]) =>
        prevList.map((eachListing: listingType) => {
          if (
            eachListing.data_cid === listing.data_cid &&
            eachListing.peer_id === listing.peer_id
          ) {
            return {
              ...eachListing,
              request_chat_status: 'pending',
              request_id: data.request_id
            }
          }
          return eachListing
        })
      )
    } catch (error) {
      console.log('Error sending chat request: ', error)
    }
  }

  // useEffect(() => {
  //   setDisplayedListings(marketListings)
  // }, [marketListings])

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

  const handleBuyFile = async (listing: listingType) => {
    try {
      let msg: string = `Name:  ${listing.file_name}\nSize:  ${listing.size} MB\nCost:  ${listing.price} SWE`
      if (listing.price > walletBalance) {
        alert(`${msg}\n\nNot enough SWE. Your current balance: ${walletBalance}.`)
      } else {
        const confirmed = window.confirm(
          `${msg}\n\nYou currently have ${walletBalance} SWE. Would like to proceed with the purchase?`
        )
        if (confirmed) {
          let data_cid = listing.data_cid
          const normalizedPath = downloadPath.endsWith('/') ? downloadPath : downloadPath + '/'
          const data = await getFile(listing.peer_id, data_cid, normalizedPath + listing.file_name)

          tranferMoney(listing.wallet_address, listing.price)
          setWalletBalance((prevBalance) => prevBalance - listing.price)

          let newHistory = {
            date: new Date(),
            file_name: listing.file_name,
            file_cid: listing.data_cid,
            file_size: listing.size,
            file_cost: listing.price,
            type: 'buy'
          }
          setHistoryView((prevList) => [...prevList, newHistory])
        }
      }
    } catch (error) {
      console.log('Error downloading file: ', error)
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

      {displayedListings.map((listing: listingType, index: number) => (
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
              maximumFractionDigits: 4
            })}{' '}
            MB
          </span>
          <span className="flex-1">
            <button
              onClick={() => handleBuyFile(listing)}
              className="bg-[#737fa3] hover:bg-[#7c85a3] text-white px-3 py-1 mr-5 rounded"
            >
              Buy
            </button>
            {listing.request_chat_status === 'not yet' ? (
              <button
                onClick={() => handleRequestChat(listing)}
                className="bg-[#737fa3] text-white px-3 py-1 rounded"
              >
                Request Chat
              </button>
            ) : listing.request_chat_status === 'pending' ? (
              <button className="bg-[#919191] text-white px-3 py-1 rounded">
                Pending Chat Request
              </button>
            ) : listing.request_chat_status === 'declined' ? (
              <button className="bg-[#ffcece] text-white px-3 py-1 rounded">
                Request Declined
              </button>
            ) : (
              <button
                onClick={() => handleChatOpen(listing)}
                className="bg-[#839eb9] text-white px-3 py-1 rounded"
              >
                Chat now
              </button>
            )}
          </span>
        </div>
      ))}

      {isChatOpen && <ChatMenu onClose={handleChatClose} chat={currentChat} />}

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
      <LoadingModal isVisible={loading} />
    </div>
  )
}

export default ListingsContent
