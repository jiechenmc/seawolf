import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import Logo from '../assets/logo.png'
import React from 'react'
import { AppContext } from '../AppContext'
import LoadingModal from './LoadingModal'
import { loginUser } from '../rpcUtils'

function Login(): JSX.Element {
  const { user } = React.useContext(AppContext)

  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')

  const [loading, setLoading] = useState<boolean>(false)

  const navigate = useNavigate()

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)

    try {
      const data = await loginUser(username, password)
      if (data) {
        setLoading(false)
        navigate('upload')
      }
    } catch (error) {
      setLoading(false)
      console.log(error)
    }
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
            placeholder="Username"
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
      <LoadingModal isVisible={loading} />
    </div>
  )
}

export default Login
