import React from 'react'
import Logo from '../assets/logo.png'
import { FaHome, FaExchangeAlt, FaCog, FaWallet, FaGlobeAmericas } from 'react-icons/fa'
import { GiTwoCoins } from 'react-icons/gi'
import { useNavigate } from 'react-router-dom'

function SideNav(): JSX.Element {
  const navigate = useNavigate()

  const handleClickTab = (e: React.FormEvent, tab: string) => {
    e.preventDefault()
    console.log(`clicked ${tab}`)

    navigate(tab)
  }

  return (
    <div className="fixed left-0 top-0 h-screen w-52 bg-[#CEDADA] text-[#071108] shadow-lg rounded-tr-sm rounded-br-sm">
      <div className="flex items-center py-4 pl-1 border-b-2 border-[#9ca6a6]">
        <img src={Logo} alt="Logo" className="w-16 h-16" />
        <span className="ml-3 text-xl font-bold ">SeaWolf Exchange</span>
      </div>

      <div className="mt-6 space-y-6">
        <button
          className="flex items-center px-6 py-2 w-11/12 hover:bg-[#9db6b6] ml-1 rounded-lg font-semibold"
          onClick={(event) => handleClickTab(event, '/home')}
        >
          <FaHome className="mr-5 text-3xl" />
          <span>Home</span>
        </button>

        <button
          className="flex items-center px-6 py-2 w-11/12 hover:bg-[#9db6b6] ml-1 rounded-lg font-semibold"
          onClick={(event) => handleClickTab(event, '/exchange')}
        >
          <FaExchangeAlt className="mr-5 text-3xl" />
          <span>Exchange</span>
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

        <button
          className="flex items-center px-6 py-2 w-11/12 hover:bg-[#9db6b6] ml-1 rounded-lg font-semibold"
          onClick={(event) => handleClickTab(event, '/wallet')}
        >
          <FaWallet className="mr-5 text-3xl" />
          <span>Wallet</span>
        </button>

        <button
          className="flex items-center px-6 py-2 w-11/12 hover:bg-[#9db6b6] ml-1 rounded-lg font-semibold"
          onClick={(event) => handleClickTab(event, '/settings')}
        >
          <FaCog className="mr-5 text-3xl" />
          <span>Settings</span>
        </button>
      </div>
    </div>
  )
}

export default SideNav
