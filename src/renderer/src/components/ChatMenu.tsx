import React, { useState, useEffect, useRef } from 'react'
import { IoClose } from 'react-icons/io5'

type Message = {
  text: string
  sender: 'user' | 'other'
}

interface ChatMenuProps {
  onClose: () => void
  otherUserName?: string
}

const ChatMenu: React.FC<ChatMenuProps> = ({ onClose, otherUserName = 'User' }) => {
  const [message, setMessage] = useState('')
  const [messages, setMessages] = useState<Message[]>([])
  const messagesEndRef = useRef<HTMLDivElement | null>(null)

  // Scroll to the bottom when new messages are added
  useEffect(() => {
    if (messagesEndRef.current) {
      messagesEndRef.current.scrollIntoView({ behavior: 'smooth' })
    }
  }, [messages])

  const handleSend = () => {
    if (message.trim()) {
      setMessages([...messages, { text: message, sender: 'user' }])
      setMessage('') // Clear input after sending
    }
  }

  return (
    <div className="fixed bottom-10 right-10 w-80 h-96 bg-white rounded-lg shadow-lg p-4 flex flex-col">
      {/* Header with close button */}
      <div className="flex justify-between items-center border-b-2 pb-2 mb-2">
        <span className="font-semibold">{otherUserName}</span>
        <button onClick={onClose}>
          <IoClose size={20} />
        </button>
      </div>

      {/* Messages */}
      <div className="flex-1 overflow-y-auto p-2 space-y-2 max-h-64">
        {messages.map((msg, index) => (
          <div
            key={index}
            className={`p-2 rounded-lg ${
              msg.sender === 'user' ? 'bg-blue-500 text-white ml-auto' : 'bg-gray-200 text-black'
            }`}
            style={{
              maxWidth: '75%', // Limit message bubble width
              wordWrap: 'break-word', // Allow word wrapping for long messages
            }}
          >
            {msg.text}
          </div>
        ))}
        <div ref={messagesEndRef} /> {/* Scroll to this element */}
      </div>

      {/* Text input */}
      <div className="flex items-center mt-auto">
        <input
          type="text"
          value={message}
          onChange={(e) => setMessage(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter') {
              handleSend()
            }
          }}
          className="flex-1 border border-gray-300 p-2 rounded-lg"
          placeholder="Type a message"
        />
        <button onClick={handleSend} className="ml-2 bg-blue-500 text-white p-2 rounded-lg">
          Send
        </button>
      </div>
    </div>
  )
}

export default ChatMenu
