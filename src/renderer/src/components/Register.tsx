import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import Logo from '../assets/logo.png'
import React from 'react'
import { AppContext } from '../AppContext'

function Register(): JSX.Element {
  const { user } = React.useContext(AppContext)

  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')

  const navigate = useNavigate()

  const handleGoBackLogin = (e: React.FormEvent) => {
    e.preventDefault()

    navigate('/')
  }

  const handleRegisterAccount = (e: React.FormEvent) => {
    e.preventDefault()

    navigate('/')
  }

  return (
    <div className="flex items-center justify-center h-screen bg-[#F5EFED]">
      <div className="bg-[#CEDADA] p-8 rounded-3xl shadow-lg w-full max-w-2xl">
        <div className="flex items-center justify-center mb-3">
          <img src={Logo} alt="Logo" className="w-2/12 h-2/12 mr-2" />
          <div className="text-center text-3xl font-bold">SeaWolf Exchange</div>
        </div>
        <div className="text-center text-2xl font-semibold text-gray-800 mb-7">
          Register for a new account
        </div>
        <form>
          <label className="block text-lg font-semibold text-gray-700 mb-1">
            Create a new username
          </label>
          <input
            type="text"
            placeholder="Username"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 shadow-md"
          />
          <label className="mt-6 block text-lg font-semibold text-gray-700 mb-1">
            Choose a password
          </label>
          <input
            type="password"
            placeholder="Password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 shadow-md"
          />
          <div className="flex justify-center mt-6">
            <button
              type="button"
              onClick={handleGoBackLogin}
              className="w-5/12 mx-4 px-4 py-2 bg-[#9498a5] text-white rounded-lg hover:bg-[#7c85a3] focus:outline-none focus:ring-2 focus:ring-[#7c85a3] shadow-md"
            >
              Back to Login
            </button>
            <button
              type="submit"
              onClick={handleRegisterAccount}
              className="w-5/12 mx-4 px-4 py-2 bg-[#737fa3] text-white rounded-lg hover:bg-[#7c85a3] focus:outline-none focus:ring-2 focus:ring-[#7c85a3] shadow-md"
            >
              Create Account
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}

export default Register
