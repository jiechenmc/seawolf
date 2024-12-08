import React from 'react'

interface LoadingModalProps {
  isVisible: boolean
  message?: string
}

const LoadingModal: React.FC<LoadingModalProps> = ({ isVisible, message = 'Loading...' }) => {
  if (!isVisible) return null

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white p-6 rounded-lg shadow-lg flex flex-col items-center space-y-4">
        <div className="w-10 h-10 border-4 border-blue-500 border-t-transparent rounded-full animate-spin"></div>
        <span className="text-gray-700">{message}</span>
      </div>
    </div>
  )
}

export default LoadingModal
