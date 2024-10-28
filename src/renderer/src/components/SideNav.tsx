import React from 'react'
import Logo from '../assets/logo.png'
import { FaMale, FaGlobeAmericas, FaSignOutAlt, FaShopify } from 'react-icons/fa'
import { IoCloudUploadSharp } from 'react-icons/io5'
import { IoIosCloudDownload } from 'react-icons/io'
import { GiTwoCoins } from 'react-icons/gi'
import { useNavigate } from 'react-router-dom'

function SideNav(): JSX.Element {
  const navigate = useNavigate()

  const handleClickTab = (e: React.FormEvent, tab: string) => {
    e.preventDefault()

    navigate(tab)
  }

  return (
    <div className="fixed left-0 top-0 h-screen w-52 bg-[#CEDADA] text-[#071108] shadow-lg rounded-tr-sm rounded-br-sm">
      <div className="flex items-center py-4 pl-1 border-b-2 border-[#9ca6a6]">
        <img src={Logo} alt="Logo" className="w-16 h-16" />
        <span className="ml-3 text-xl font-bold ">SeaWolf Exchange</span>
      </div>

      <div className="mt-3 space-y-6">
        <button
          className="flex items-center px-6 py-2 w-11/12 hover:bg-[#9db6b6] ml-1 rounded-lg font-semibold"
          onClick={(event) => handleClickTab(event, '/upload')}
        >
          <IoCloudUploadSharp className="mr-5 text-3xl" />
          <span>Upload</span>
        </button>

        <button
          className="flex items-center px-6 py-2 w-11/12 hover:bg-[#9db6b6] ml-1 rounded-lg font-semibold"
          onClick={(event) => handleClickTab(event, '/download')}
        >
          <IoIosCloudDownload className="mr-5 text-3xl" />
          <span>Download</span>
        </button>

        <button
          className="flex items-center px-6 py-2 w-11/12 hover:bg-[#9db6b6] ml-1 rounded-lg font-semibold"
          onClick={(event) => handleClickTab(event, '/mining')}
        >
          <GiTwoCoins className="mr-5 text-3xl" />
          <span>Mining</span>
        </button>

        <button
          className="flex items-center px-6 py-2 w-11/12 hover:bg-[#9db6b6] ml-1 rounded-lg font-semibold"
          onClick={(event) => handleClickTab(event, '/proxy')}
        >
          <FaGlobeAmericas className="mr-5 text-3xl" />
          <span>Proxy</span>
        </button>

        {/* <button
          className="flex items-center px-6 py-2 w-11/12 hover:bg-[#9db6b6] ml-1 rounded-lg font-semibold"
          onClick={(event) => handleClickTab(event, '/wallet')}
        >
          <FaWallet className="mr-5 text-3xl" />
          <span>Wallet</span>
        </button> */}

        <button
          className="flex items-center px-6 py-2 w-11/12 hover:bg-[#9db6b6] ml-1 rounded-lg font-semibold"
          onClick={(event) => handleClickTab(event, '/account')}
        >
          <FaMale className="mr-5 text-3xl" />
          <span>Account</span>
        </button>

        <button
          className="flex items-center px-6 py-2 w-11/12 hover:bg-[#9db6b6] ml-1 rounded-lg font-semibold"
          onClick={(event) => handleClickTab(event, '/market')}
        >
          <FaShopify className="mr-5 text-3xl" />
          <span>Market</span>
        </button>
      </div>

      <button
        className="flex items-center px-6 py-2 w-11/12 hover:bg-[#9db6b6] ml-1 rounded-lg font-semibold mt-32"
        onClick={(event) => handleClickTab(event, '/')}
      >
        <FaSignOutAlt className="mr-5 text-3xl" />
        <span>Logout</span>
      </button>
    </div>
  )
}

export default SideNav
