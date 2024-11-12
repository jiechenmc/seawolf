import { BrowserRouter as Router, Route, Routes } from 'react-router-dom'
import Login from './components/Login'
import Register from './components/Register'
import Upload from './components/Upload'
import Download from './components/Download'
import Mining from './components/Mining'
import Proxy from './components/Proxy'
import Wallet from './components/Wallet'
import Account from './components/Account'
import Market from './components/Market'
import { AppProvider } from './AppContext'

function App(): JSX.Element {
  return (
    <AppProvider>
      <Router>
        <Routes>
          <Route path="/" element={<Login />} />
          <Route path="/register" element={<Register />} />
          <Route path="/upload" element={<Upload />} />
          <Route path="/download" element={<Download />} />
          <Route path="/mining" element={<Mining />} />
          <Route path="/proxy" element={<Proxy />} />
          <Route path="/wallet" element={<Wallet />} />
          <Route path="/account" element={<Account />} />
          <Route path="/market" element={<Market />} />
        </Routes>
      </Router>
    </AppProvider>
  )
}

export default App
