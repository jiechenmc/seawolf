import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import Logo from '../assets/logo.png'
import React from 'react'
import { AppContext } from '../AppContext'

function Login(): JSX.Element {
  const { user } = React.useContext(AppContext)

  const [, setWalletAddress] = user

  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')

  const navigate = useNavigate()

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault()
    console.log('Cliked on button to login')

    const loginRequest = {
      jsonrpc: '2.0',
      method: 'p2p_Login',
      params: [username, password],
      id: 1
    }

    const response = await fetch('http://localhost:8080/rpc', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(loginRequest)
    })

    if (response.ok) {
      const data = await response.json()
      if (data.error) {
        console.log('Got error logging in')
      } else {
        console.log('Login successful')
      }
    } else {
      console.error('Error calling p2p.Login')
    }

    setWalletAddress(username)
    navigate('upload')
  }

  return (
    <div className="flex items-center justify-center h-screen bg-[#F5EFED]">
      <div className="bg-[#CEDADA] p-8 rounded-3xl shadow-lg w-full max-w-2xl">
        <div className="flex items-center justify-center mb-7">
          <img src={Logo} alt="Logo" className="w-2/12 h-2/12 mr-2" />
          <div className="text-center text-3xl font-bold">SeaWolf Exchange</div>
        </div>
        <form onSubmit={handleLogin} className="space-y-6">
          <input
            type="text"
            placeholder="Wallet Address"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 shadow-md"
          />
          <input
            type="password"
            placeholder="Password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 shadow-md"
          />
          <div className="flex justify-center">
            <button
              type="submit"
              className="w-6/12 px-4 py-2 bg-[#737fa3] text-white rounded-lg hover:bg-[#7c85a3] focus:outline-none focus:ring-2 focus:ring-[#7c85a3] shadow-md"
            >
              Login
            </button>
          </div>
          <div className="text-center mt-4">
            <span className="text-gray-600">Don't have an account? </span>
            <a href="/register" className="text-blue-600 hover:underline">
              Register here
            </a>
          </div>
        </form>
      </div>
    </div>
  )
}

export default Login
