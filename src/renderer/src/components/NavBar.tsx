import React, { useState } from 'react'
import Draggable from 'react-draggable'
import { useNavigate } from 'react-router-dom'
import Logo from '../assets/logo.png'
import { FaHome, FaExchangeAlt, FaCog, FaWallet } from 'react-icons/fa'
import { GiTwoCoins } from 'react-icons/gi'

function NavBar(): JSX.Element {
  const [showButtons, setShowButtons] = useState(false)
  const navigate = useNavigate()

  const handleMainClick = (e: React.MouseEvent<HTMLButtonElement>) => {
    e.preventDefault()
    setShowButtons(!showButtons)
  }

  const handleClickTab = (e: React.MouseEvent<HTMLButtonElement>, tab: string) => {
    e.preventDefault()
    e.stopPropagation() // Prevent triggering the main button's click event
    navigate(tab)
    setShowButtons(false) // Hide the buttons after navigation
  }

  const subButtons = [
    { icon: FaHome, position: 'top-0 left-1/2 -translate-x-1/2 -translate-y-full', tab: '/home' },
    {
      icon: FaExchangeAlt,
      position: 'top-1/2 right-0 -translate-y-1/2 translate-x-full',
      tab: '/exchange'
    },
    {
      icon: GiTwoCoins,
      position: 'bottom-0 left-1/2 -translate-x-1/2 translate-y-full',
      tab: '/mining'
    },
    {
      icon: FaWallet,
      position: 'top-1/2 left-0 -translate-y-1/2 -translate-x-full',
      tab: '/wallet'
    },
    { icon: FaCog, position: 'top-0 left-0 -translate-x-full -translate-y-full', tab: '/settings' }
  ]

  return (
    <Draggable defaultPosition={{ x: 0, y: 500 }}>
      <div className="fixed">
        <button
          className="relative z-10 bg-black hover:bg-gray-100 p-2 rounded-full shadow-lg cursor-move"
          onClick={handleMainClick}
        >
          <img src={Logo} alt="Logo" className="w-12 h-12" />
        </button>
        {showButtons && (
          <>
            {subButtons.map((button, index) => (
              <button
                key={index}
                className={`absolute ${button.position} bg-white hover:bg-gray-100 p-2 rounded-full shadow-lg`}
                onClick={(event) => handleClickTab(event, button.tab)}
              >
                <button.icon className="w-6 h-6 text-gray-600" />
              </button>
            ))}
          </>
        )}
      </div>
    </Draggable>
  )
}

export default NavBar
