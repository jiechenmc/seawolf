/* eslint-disable prettier/prettier */
import { useContext, useState, useEffect } from 'react'
import SideNav from './SideNav'
import { AppContext } from '../AppContext'
import LoadingModal from './LoadingModal'
import {
    Proxy,
    getAllProxies,
    connectToProxy,
    disconnectFromProxy,
    registerAsProxy,
    unregisterAsProxy,
    getProxyBytes,
    discoverFiles
} from '../rpcUtils'
import { tranferMoney } from '@renderer/walletUtils'

function ProxyComponent(): JSX.Element {
    const { proxy, proxies } = useContext(AppContext)
    const [currProxy, setCurrProxy] = proxy
    const [listOfProxies, setListOfProxies] = proxies

    const [serveAsProxy, setServeAsProxy] = useState(false)
    const [walletAddress, setWalletAddress] = useState('')
    const [price, setPrice] = useState(0)
    const [loading, setLoading] = useState<boolean>(false)

    useEffect(() => {
        setLoading(true)
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
            setLoading(false)
        })
    }, [])


    useEffect(() => {
        // Fetch the list of proxies from the backend
        const fetchProxies = async (): Promise<void> => {
            try {
                const lmfao = await discoverFiles()
                if (lmfao) {
                    console.log("lmfao")
                }
                const proxies = await getAllProxies()
                setListOfProxies(proxies)
            } catch (error) {
                console.error('Failed to fetch proxies:', error)
            }
        }

        fetchProxies()
    }, [setListOfProxies])

    const handleChooseProxy = async (proxy: Proxy): Promise<void> => {
        const isConfirmed = window.confirm(
            `Are you sure you want to connect to peer ID: ${proxy.peer_id}?`
        )

        if (isConfirmed) {
            try {
                await connectToProxy(proxy.peer_id)
                setCurrProxy(proxy)
            } catch (error) {
                console.error('Failed to connect to proxy:', error)
            }
        }
        console.log(proxy)
    }

    const handleRemoveProxy = async (): Promise<void> => {
        if (currProxy) {
            const isConfirmed = window.confirm(
                `Are you sure you want to disconnect from peer ID: ${currProxy.peer_id}?`
            )

            if (isConfirmed) {
                try {
                    await disconnectFromProxy()
                    const bytes = await getProxyBytes(currProxy.peer_id)
                    const totalBytes = bytes.rx_bytes + bytes.tx_bytes
                    tranferMoney(currProxy.wallet_address, totalBytes * currProxy.price / 1024.0 / 1024.0)
                    setCurrProxy(null)
                } catch (error) {
                    console.error('Failed to disconnect from proxy:', error)
                }
            }
        }
    }

    const handleToggle = async (): Promise<void> => {
        if (!serveAsProxy) {
            // Register as proxy
            if (!walletAddress || !price) {
                alert('Please enter a valid wallet address and price.')
                return
            }
            try {
                await registerAsProxy(price, walletAddress)
                setServeAsProxy(true)
            } catch (error) {
                console.error('Failed to register as proxy:', error)
            }
        } else {
            // Unregister as proxy
            try {
                await unregisterAsProxy()
                setServeAsProxy(false)
            } catch (error) {
                console.error('Failed to unregister as proxy:', error)
            }
        }
    }

    return (
        <div className="flex ml-52">
            <SideNav />

            <div className="flex-1 p-6">
                <h1 className="text-2xl font-bold mb-4">Proxy</h1>

                <div className="bg-white p-4 rounded-lg shadow-md mb-8">
                    <h2 className="text-xl font-semibold pb-5">Serve As Proxy</h2>
                    <p className="text-gray-700 mb-4">
                        {serveAsProxy ? 'Currently serving as a proxy' : 'Currently not serving as a proxy'}
                    </p>

                    {!serveAsProxy && (
                        <div className="mb-4">
                            {/* <label className="block text-gray-700">Wallet Address:</label>
                            <input
                                type="text"
                                value={walletAddress}
                                onChange={(e) => setWalletAddress(e.target.value)}
                                className="mt-1 block w-full border border-gray-300 rounded-md shadow-sm p-2"
                            /> */}
                            <label className="block text-gray-700 mt-4">Price (SWE per MB):</label>
                            <input
                                type="number"
                                value={price}
                                onChange={(e) => setPrice(parseFloat(e.target.value))}
                                className="mt-1 block w-full border border-gray-300 rounded-md shadow-sm p-2"
                            />
                        </div>
                    )}

                    <div
                        onClick={handleToggle}
                        className={`${serveAsProxy ? 'bg-green-500' : 'bg-gray-300'
                            } relative inline-flex h-6 w-11 items-center rounded-full cursor-pointer transition-colors`}
                    >
                        <span
                            className={`${serveAsProxy ? 'translate-x-6' : 'translate-x-1'
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
                                        <h3 className="font-bold">Peer ID</h3>
                                        <p className="text-lg">{currProxy.peer_id}</p>
                                    </div>
                                    <div className="flex-1 text-center">
                                        <h3 className="font-bold">Cost</h3>
                                        <p className="text-lg">{currProxy.price} SWE</p>
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
                        <span className="flex-1 font-semibold text-left">Peer ID</span>
                        <span className="flex-1 font-semibold text-left">Cost / MB</span>
                    </div>
                    <div>
                        <LoadingModal isVisible={loading} />
                        {listOfProxies && listOfProxies.length > 0 ? (
                            listOfProxies.map((proxy: Proxy, index: number) => (
                                <button
                                    key={index}
                                    onClick={() => handleChooseProxy(proxy)}
                                    className="flex items-center p-2 w-full hover:bg-gray-200 border-b border-gray-300"
                                >
                                    <span className="flex-1 text-left">{proxy.peer_id}</span>
                                    <span className="flex-1 text-left">{proxy.price} SWE</span>
                                </button>
                            ))
                        ) : (
                            <p className="text-center p-4">No proxies available.</p>
                        )}
                    </div>
                </div>
            </div>
        </div>
    )
}

export default ProxyComponent
