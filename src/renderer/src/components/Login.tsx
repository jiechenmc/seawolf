import { useState } from 'react'
import Logo from '../assets/logo.png'

function Login(): JSX.Element {
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    console.log('Logging in with: ', username, password)
  }

  return (
    <div className="flex items-center justify-center h-screen bg-[#F5EFED]">
      <div className="bg-[#CEDADA] p-8 rounded-3xl shadow-md w-full max-w-2xl">
        <div className="flex items-center justify-center mb-7">
          <img src={Logo} alt="Logo" className="w-2/12 h-2/12 mr-7" />
          <div className="text-center text-3xl font-bold">Welcome To SeaWolf Exchange</div>
        </div>
        <form onSubmit={handleSubmit} className="space-y-6">
          <input
            type="text"
            placeholder="Wallet Address"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
          <input
            type="password"
            placeholder="Private Key"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
          <div className="flex justify-center">
            <button
              type="submit"
              className="w-6/12 px-4 py-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600 focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              Login
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}

export default Login
