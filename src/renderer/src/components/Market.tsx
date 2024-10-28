import React, { useState } from 'react'
import SideNav from './SideNav'
import ListingsContent from './ListingsContext'
import CreateListingContent from './CreateListingContent'
import UserListings from './UserListings'


function Market(): JSX.Element {
  
  const [activeTab, setActiveTab] = useState<'Listings' | 'YourListings' | 'CreateListing'>('Listings')    
  return (
    <div className="flex ml-52">
      <SideNav />
      <div className="flex-1 p-6">
        <h1 className="text-2xl font-bold mb-4">Market</h1>
        <div className="flex mb-6">
          <button
            className={`mr-4 px-4 py-2 ${activeTab === 'Listings' ? 'bg-[#737fa3] hover:bg-[#7c85a3] text-white' : 'bg-gray-200'} rounded-lg`}
            onClick={() => setActiveTab('Listings')}
          >
            Listings
          </button>
          <button
            className={`mr-4 px-4 py-2 ${activeTab === 'YourListings' ? 'bg-[#737fa3] hover:bg-[#7c85a3] text-white' : 'bg-gray-200'} rounded-lg`}
            onClick={() => setActiveTab('YourListings')}
          >
            Your Listings
          </button>
          <button
            className={`px-4 py-2 ${activeTab === 'CreateListing' ? 'bg-[#737fa3] hover:bg-[#7c85a3] text-white' : 'bg-gray-200'} rounded-lg`}
            onClick={() => setActiveTab('CreateListing')}
          >
            Create Listing
          </button>
        </div>

        {activeTab === 'Listings' && (
          <div>
            <h2 className="text-xl font-semibold mb-2">All Listings</h2>
            <div className="border-b border-gray-300 mb-4"></div>

            <ListingsContent />
          </div>
        )}

        {activeTab === 'YourListings' && (
          <div>
            <h2 className="text-xl font-semibold mb-2">Your Listings</h2>
            <div className="border-b border-gray-300 mb-4"></div>
            <UserListings />
          </div>
        )}

        {activeTab === 'CreateListing' && (
          <div>
            <h2 className="text-xl font-semibold mb-2">Create New Listing</h2>
            <div className="border-b border-gray-300 mb-4"></div>
            <CreateListingContent />
          </div>
        )}
      </div>
    </div>
  )
}

export default Market
