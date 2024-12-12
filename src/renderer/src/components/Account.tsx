import SideNav from './SideNav'
import { FaRegClipboard } from 'react-icons/fa'
import React, { useEffect, useState } from 'react'
import { AppContext } from '@renderer/AppContext'

function formatDateTime(date: Date): string {
  const padToTwoDigits = (num: number) => (num < 10 ? `0${num}` : num)

  const month = padToTwoDigits(date.getMonth() + 1)
  const day = padToTwoDigits(date.getDate())
  const year = date.getFullYear()

  const hours = padToTwoDigits(date.getHours())
  const minutes = padToTwoDigits(date.getMinutes())
  const seconds = padToTwoDigits(date.getSeconds())

  return `${month}/${day}/${year} ${hours}:${minutes}:${seconds}`
}

function Account(): JSX.Element {
  const handleCopyToClipboard = () => {
    navigator.clipboard
      .writeText('hello')
      .then(() => {
        console.log('copied to clipboard')
      })
      .catch((err) => {
        console.error('failed to copy due to: ', err)
      })
  }

  const [walletBalance, setWalletBalance] = useState(0)
  const [walletAddress, setWalletAddress] = useState('')
  const [sentTxId, setTxId] = useState('')

  const [activeTab, setActiveTab] = useState<'Wallet' | 'History' | 'Settings'>('Wallet')

  useEffect(() => {
    fetch("http://localhost:8080/balance?q=default", {
      headers: {
        'Content-Type': 'application/json'
      },
    }).then(async (r) => {
      const data = await r.json()
      setWalletBalance(parseInt(data))
    })

    fetch("http://localhost:8080/account", {
      method: "POST",
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({ account: "default" })
    }
    ).then(async (r) => {
      const data = await r.json()
      setWalletAddress(data.message)
    })

    // tranferMoney("1P3JSQhXCj2iUeNb1rDzrxSNry7PukwXKJ", 1).then(d => setTxId(d))
  }, [])

  console.log(sentTxId)

  // const [walletAddress] = user
  // const [walletBalance] = balance
  // const [historyView] = history

  return (
    <div className="flex ml-52">
      <SideNav />
      <div className="flex-1 p-6">
        <h1 className="text-2xl font-bold mb-4">Account</h1>
        <div className="flex mb-6">
          <button
            className={`mr-4 px-4 py-2 ${activeTab === 'Wallet' ? 'bg-[#737fa3] hover:bg-[#7c85a3] text-white' : 'bg-gray-200'} rounded-lg`}
            onClick={() => setActiveTab('Wallet')}
          >
            Wallet
          </button>
          <button
            className={`mr-4 px-4 py-2 ${activeTab === 'History' ? 'bg-[#737fa3] hover:bg-[#7c85a3] text-white' : 'bg-gray-200'} rounded-lg`}
            onClick={() => setActiveTab('History')}
          >
            History
          </button>
          <button
            className={`px-4 py-2 ${activeTab === 'Settings' ? 'bg-[#737fa3] hover:bg-[#7c85a3] text-white' : 'bg-gray-200'} rounded-lg`}
            onClick={() => setActiveTab('Settings')}
          >
            Settings
          </button>
        </div>

        {activeTab === 'Wallet' && (
          <div>
            <h2 className="text-xl font-semibold mb-2">Wallet</h2>
            <div className="border-b border-gray-300 mb-4"></div>
            <div className="flex justify-between mb-16">
              <div className="bg-white p-4 rounded-lg shadow-md w-1/2">
                <div className="flex items-center mb-3">
                  <h3 className="text-lg font-semibold">Wallet Address: {walletAddress}</h3>
                  <button
                    onClick={handleCopyToClipboard}
                    className="ml-2 p-1 hover:bg-gray-200 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
                    aria-label="Copy value to clipboard"
                  >
                    <FaRegClipboard className="text-xl" />
                  </button>
                </div>
                <div className="flex-1 text-left">
                  <h3 className="font-bold">Current Balance</h3>
                  <p className="text-lg">{walletBalance} BTC</p>
                </div>
              </div>
            </div>
          </div>
        )}

        {activeTab === 'History' && (
          <div>
            <h3 className="text-xl font-semibold mb-2">History</h3>
            <div className="border-b border-gray-300 mb-6"></div>
            <div className="overflow-x-auto border border-gray-300 rounded-lg mb-2">
              <div className="flex items-center p-2 border-b border-gray-300">
                <span className="flex-1 font-semibold">Date</span>
                <span className="flex-1 font-semibold">File</span>
                <span className="flex-1 font-semibold">Cost (SWE)</span>
                <span className="flex-1 font-semibold">Transaction Type</span>
                <span className="flex-1 font-semibold">Proxy</span>
              </div>
            </div>
            {/* {historyView.map((historyItem, index: number) => (
              <div
                key={index}
                className="flex items-center px-2 py-1 border-b border-gray-300 rounded-md"
              >
                <span className="flex-1 ">{formatDateTime(historyItem.date)}</span>
                <div className="flex flex-1 items-center">
                  <div>
                    <span className="block font-semibold">{historyItem.file.name}</span>
                    <span className="block text-gray-500">{historyItem.file.cid}</span>{' '}
                  </div>
                </div>
                <span className="flex-1 ">{historyItem.file.cost}</span>
                <span className="flex-1 ">{historyItem.type}</span>
                <span className="flex-1 ">{historyItem.proxy}</span>
              </div>
            ))} */}
          </div>
        )}

        {activeTab === 'Settings' && (
          <div>
            <h2 className="text-xl font-semibold mb-2">Transfer Settings</h2>
            <div className="border-b border-gray-300 mb-4 w-1/6"></div>

            <div className="mb-4">
              <label
                htmlFor="transfer-name"
                className="block text-sm font-medium text-gray-700 mb-1"
              >
                Save Folder
              </label>
              <input
                type="text"
                id="transfer-name"
                className="mt-1 block w-1/3 border border-gray-300 rounded-md p-2"
                placeholder="Enter new save directory"
              />
            </div>

            <div className="mb-4">
              <label
                htmlFor="upload-limit"
                className="block text-sm font-medium text-gray-700 mb-1"
              >
                Upload Limit
              </label>
              <div className="flex border border-gray-300 rounded-md p-2 w-1/3 items-center">
                <input
                  type="number"
                  id="upload-limit"
                  className="block border-none outline-none w-full pr-4"
                  placeholder="0"
                  min="0"
                  step="any"
                />
                <span className="text-gray-500">Mbps</span>
              </div>
            </div>

            <button className="mt-4 px-4 py-2 bg-[#737fa3] text-white font-semibold rounded-md hover:bg-[#7c85a3]">
              Save Changes
            </button>
          </div>
        )}
      </div>
    </div>
  )
}

export default Account
