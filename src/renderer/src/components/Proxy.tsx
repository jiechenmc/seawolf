import React, { useContext, useState } from 'react'
import SideNav from './SideNav'
// import Navbar from './NavBar'
import { AppContext } from '../AppContext'

type Proxy = {
  ip: string
  location: string
  cost: number
}

const testList: Proxy[] = [
  { ip: '192.168.1.1', location: 'New York, USA', cost: 10 },
  { ip: '192.168.1.2', location: 'London, UK', cost: 12 },
  { ip: '192.168.1.3', location: 'Tokyo, Japan', cost: 8 }
]

function Proxy(): JSX.Element {
  // const [currProxy, setCurrProxy] = useState<Proxy | null>(null)
  // const [listOfProxies, setListOfProxies] = useState<Proxy[]>(testList)
  const { proxy, proxies } = React.useContext(AppContext)
  const [currProxy, setCurrProxy] = proxy
  const [listOfProxies, setListOfProxies] = proxies

  const [serveAsProxy, setServeAsProxy] = useState(false)

  const handleChooseProxy = (proxy: Proxy) => {
    const isConfirmed = window.confirm(`Are you sure you want to connect to IP: ${proxy.ip}?`)

    if (isConfirmed) {
      setCurrProxy(proxy)
    }
  }

  const handleRemoveProxy = () => {
    const isConfirmed = window.confirm(
      `Are you sure you want to disconnect from IP: ${currProxy?.ip}?`
    )

    if (isConfirmed) {
      setCurrProxy(null)
    }
  }

  const handleToggle = () => {
    setServeAsProxy((prev) => !prev)
  }

  return (
    <div className="flex ml-52">
      {/* <Navbar /> */}
      <SideNav />

      <div className="flex-1 p-6">
        <h1 className="text-2xl font-bold mb-4">Proxy</h1>

        <div className="bg-white p-4 rounded-lg shadow-md mb-8">
          <h2 className="text-xl font-semibold pb-5">Serve As Proxy</h2>
          <p className=" text-gray-700 mb-4">
            {serveAsProxy
              ? `Currently serving as a proxy on IP: 192.1.213.72`
              : 'Currently not serving as a proxy'}
          </p>
          <div
            onClick={handleToggle}
            className={`${
              serveAsProxy ? 'bg-green-500' : 'bg-gray-300'
            } relative inline-flex h-6 w-11 items-center rounded-full cursor-pointer transition-colors`}
          >
            <span
              className={`${
                serveAsProxy ? 'translate-x-6' : 'translate-x-1'
              } inline-block w-4 h-4 transform bg-white rounded-full transition-transform`}
            />
          </div>
        </div>

        <div className="bg-white p-4 rounded-lg shadow-md mb-16">
          <h2 className="text-xl font-semibold pb-5">Currently Connected To:</h2>
          <div>
            {currProxy ? (
              <div>
                <div className="flex justify-between mt-2">
                  <div className="flex-1 text-center">
                    <h3 className="font-bold">IP Address</h3>
                    <p className="text-lg">{currProxy.ip}</p>
                  </div>
                  <div className="flex-1 text-center">
                    <h3 className="font-bold">Location</h3>
                    <p className="text-lg">{currProxy.location}</p>
                  </div>
                  <div className="flex-1 text-center">
                    <h3 className="font-bold">Cost</h3>
                    <p className="text-lg">${currProxy.cost}</p>
                  </div>
                </div>
                <button
                  className="mt-7 px-4 py-2 bg-[#737fa3] text-white font-semibold rounded-md hover:bg-[#7c85a3]"
                  onClick={handleRemoveProxy}
                >
                  Disconnect
                </button>
              </div>
            ) : (
              <p className="text-l">None. Please choose one from below to get started.</p>
            )}
          </div>
        </div>
        <h2 className="text-xl font-semibold mb-2">Choose A Proxy</h2>
        <div className="overflow-x-auto border border-gray-300 rounded-lg">
          <div className="flex items-center p-2 border-b border-gray-300">
            <span className="flex-1 font-semibold text-left">IP Address</span>
            <span className="flex-1 font-semibold text-left">Location</span>
            <span className="flex-1 font-semibold text-left">Cost / MB</span>
          </div>
          <div>
            {listOfProxies.map((proxy, index) => (
              <button
                key={index}
                onClick={() => handleChooseProxy(proxy)}
                className="flex items-center p-2 w-full hover:bg-gray-200 border-b border-gray-300"
              >
                <span className="flex-1 text-left">{proxy.ip}</span>
                <span className="flex-1 text-left">{proxy.location}</span>
                <span className="flex-1 text-left">${proxy.cost}</span>
              </button>
            ))}
          </div>
        </div>
      </div>
    </div>
  )
}

export default Proxy
