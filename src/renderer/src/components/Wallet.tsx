import SideNav from './SideNav'
import { FaRegClipboard } from 'react-icons/fa'
import NavBar from './NavBar'
import React from 'react'
import { AppContext } from '../AppContext'

function Wallet(): JSX.Element {
  const { user, balance } = React.useContext(AppContext)

  const [walletAddress, setWalletAddress] = user
  const [walletBalance, setWalletBalance] = balance

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

  return (
    <div className="flex ml-52">
      <SideNav />
      {/* <NavBar /> */}

      <div className="flex-1 p-6">
        <h1 className="text-2xl font-bold mb-4">Wallet</h1>

        <div className="flex justify-between mb-16">
          <div className="bg-white p-4 rounded-lg shadow-md w-1/2">
            <div className="flex items-center mb-3">
              <h2 className="text-xl font-semibold">Wallet ID: {walletAddress}</h2>
              <button
                onClick={handleCopyToClipboard}
                className="ml-2 p-1 hover:bg-gray-200 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
                aria-label="Copy value to clipboard"
              >
                <FaRegClipboard className="text-xl" />
              </button>
            </div>
            <div className="flex-1 text-left">
              <h2 className="font-bold">Current Balance</h2>
              <p className="text-lg">{walletBalance} SWE</p>
            </div>
          </div>
        </div>

        <h2 className="text-xl font-semibold mb-2">History</h2>
        <div className="overflow-x-auto border border-gray-300 rounded-lg">
          <div className="flex items-center p-2 border-b border-gray-300">
            <span className="flex-1 font-semibold">Date</span>
            <span className="flex-1 font-semibold">Transaction ID</span>
            <span className="flex-1 font-semibold">File</span>
            <span className="flex-1 font-semibold">Amount</span>
          </div>
        </div>
      </div>
    </div>
  )
}

export default Wallet
