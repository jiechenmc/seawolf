import { getMessages, sendMessage } from '@renderer/rpcUtils'
import React, { useState, useEffect, useRef } from 'react'
import { IoClose } from 'react-icons/io5'
import { AppContext } from '@renderer/AppContext'

type messageType = {
  timestamp: string
  from: string
  text: string
}

type chatType = {
  chat_id: number
  buyer: string
  seller: string
  file_cid: string
  status: string
  messages: messageType[]
}

interface ChatMenuProps {
  onClose: () => void
  chat?: chatType
}

const ChatMenu: React.FC<ChatMenuProps> = ({ onClose, chat }) => {
  if (!chat) {
    return null
  }

  const { user } = React.useContext(AppContext)
  const [peerId] = user

  const [message, setMessage] = useState<string>('')
  const [messages, setMessages] = useState<messageType[]>([])
  const messagesEndRef = useRef<HTMLDivElement | null>(null)

  useEffect(() => {
    const interval = setInterval(() => {
      getMessages(chat.buyer, chat.chat_id)
        .then((data) => {
          setMessages(data)
        })
        .catch((error) => {
          console.error('Error fetching messages: ', error)
        })
    }, 3000)

    return () => clearInterval(interval)
  }, [])

  useEffect(() => {
    if (messagesEndRef.current) {
      messagesEndRef.current.scrollIntoView({ behavior: 'smooth' })
    }
  }, [messages])

  const handleSend = async () => {
    if (message.trim()) {
      try {
        const data = await sendMessage(chat.buyer, chat.chat_id, message)
        setMessages((prevlist: messageType[]) => [...prevlist, data])
        setMessage('')
      } catch (error) {}

      setMessage('')
    }
  }

  return (
    <div className="fixed bottom-10 right-10 w-2/5 h-96 bg-white rounded-lg shadow-lg p-4 flex flex-col">
      <div className="flex justify-between items-center border-b-2 pb-2 mb-2">
        <span className="font-semibold">{chat.buyer}</span>
        <button onClick={onClose}>
          <IoClose size={20} />
        </button>
      </div>

      <div className="flex-1 overflow-y-auto p-2 space-y-2 max-h-64">
        {messages.map((msg, index) => (
          <div
            key={index}
            className={`p-2 rounded-lg ${
              msg.from === peerId ? 'bg-blue-500 text-white ml-auto' : 'bg-gray-200 text-black'
            }`}
            style={{
              maxWidth: '75%',
              wordWrap: 'break-word'
            }}
          >
            {msg.text}
          </div>
        ))}
        <div ref={messagesEndRef} />
      </div>

      {chat.status === 'ongoing' && (
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
      )}
    </div>
  )
}

export default ChatMenu
